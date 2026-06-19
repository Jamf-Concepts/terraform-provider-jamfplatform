// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"bytes"
	"context"
	"crypto/sha3"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/files"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// pollInterval is the cadence at which the verification poll nudges the
// cloud distribution point and re-reads the package record. Lifted from
// jamf-cli; package-level constant so the user controls only the upper
// bound via the `timeouts` block.
const pollInterval = 10 * time.Second

// Hash algorithm tags as the server reports them post-JCDS upload.
const (
	hashTypeSHA3512  = "SHA3_512"
	cloudStatusReady = "READY"
)

// errLocalChecksumMismatch is returned when the user-supplied
// `package_file_source_checksum` does not match the locally-computed SHA-3
// digest. The upload is aborted; nothing reaches the server.
var errLocalChecksumMismatch = errors.New("package_file_source_checksum did not match locally computed SHA-3-512")

// errCorruption is returned when the verification poll observes a non-empty
// server hash that matches neither the expected upload hash nor the
// pre-upload `previousHash`. This indicates the file uploaded differs from
// what we hashed locally.
var errCorruption = errors.New("server-computed package hash did not match locally computed SHA-3-512")

// errVerificationTimeout is returned when the verification poll's context
// deadline fires before convergence.
var errVerificationTimeout = errors.New("verification poll timed out waiting for JCDS hash convergence")

// HashFileSHA3 streams the local file through sha3.New512 once, returning
// the hex digest and byte count. The provider needs SHA-3-512 for the
// convergence check and the optional `package_file_source_checksum`
// integrity hint; MD5/SHA-256 are server-computed by JCDS.
func HashFileSHA3(path string) (string, int64, error) {
	f, err := os.Open(path) //nolint:gosec // user-supplied path is intentional
	if err != nil {
		return "", 0, fmt.Errorf("opening %q for hashing: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	return HashStreamSHA3(f)
}

// HashStreamSHA3 streams r through sha3.New512 once. Lets callers that
// already hold an open *os.File hash the bytes without reopening the
// file — critical for URL-source uploads where reopening would force a
// second download.
func HashStreamSHA3(r io.Reader) (string, int64, error) {
	h := sha3.New512()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", n, fmt.Errorf("hashing stream: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// PollPackageVerification nudges the cloud distribution point and polls
// the package record until JCDS has finished committing the uploaded
// binary. Convergence requires server-reported `hashValue`, `size`, AND
// `cloudTransferStatus=READY` to all line up with the locally-computed
// expectations. The size gate is critical: acceptance traces showed JCDS
// briefly returning a transient `hashValue` matching the new bytes while
// `size` was still the old binary's, then reverting — a hash-only
// convergence check would falsely declare success on that transient view.
//
// expectSha3 / expectSize describe the locally-computed digest + byte
// count of the file we just uploaded. previousHash is the SHA-3 digest
// stored before the upload so a "still-recomputing" view does not trip
// the corruption check.
//
// Each tick performs:
//
//  1. RefreshCloudDistributionPointInventoryV1(fileName) — slow (~4s) on
//     first call, fast (~200ms) afterwards.
//  2. GetPackageV1(id) — observe convergence.
//
// Returns the converged *pro.Package on success.
func PollPackageVerification(ctx context.Context, client *pro.Client, id, fileName, expectSha3 string, expectSize int64, previousHash string) (*pro.Package, error) {
	if client == nil {
		return nil, errors.New("PollPackageVerification: nil client")
	}
	expectLower := strings.ToLower(expectSha3)
	previousLower := strings.ToLower(previousHash)

	// First iteration runs immediately — no initial wait. Subsequent ticks
	// honour pollInterval.
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	tick := 0
	for {
		tick++
		// Honour ctx cancellation before each tick.
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, errVerificationTimeout
			}
			return nil, err
		}

		// Nudge the CDP. fileName drives a targeted recompute; ignore the
		// error and treat the subsequent GET as authoritative — if the
		// refresh fails transiently, the next tick will retry.
		_ = client.RefreshCloudDistributionPointInventoryV1(ctx, fileName)

		pkg, err := client.GetPackageV1(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("polling package %s: %w", id, err)
		}

		cur := strings.ToLower(helpers.DerefString(pkg.HashValue))
		hashType := helpers.DerefString(pkg.HashType)
		status := helpers.DerefString(pkg.CloudTransferStatus)
		sizeStr := helpers.DerefString(pkg.Size)

		tflog.Info(ctx, "package poll tick", map[string]any{
			"tick":           tick,
			"id":             id,
			"cur_hash":       cur,
			"expect_hash":    expectLower,
			"previous_hash":  previousLower,
			"hash_type":      hashType,
			"transfer_state": status,
			"cur_size":       sizeStr,
			"expect_size":    expectSize,
		})

		decision := classifyPollTick(cur, expectLower, previousLower, hashType, status, sizeStr, expectSize)
		switch decision {
		case pollDecisionConverged:
			return pkg, nil
		case pollDecisionCorruption:
			return nil, fmt.Errorf("%w: expected %s, server reported %s", errCorruption, expectLower, cur)
		}

		// Wait for the next tick. The first iteration's refresh+GET has
		// already fired above; subsequent iterations wait for the ticker
		// to honour pollInterval before retrying.
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, errVerificationTimeout
			}
			return nil, ctx.Err()
		case <-ticker.C:
			// next iteration
		}
	}
}

// ManifestBodiesEqual compares the manifest body stored in state with the
// content found at the supplied manifest file source. Returns true when
// stored already matches the file source content — re-upload is unnecessary.
//
// The src parameter accepts either a local path or an `http(s)://` URL.
// URL sources are downloaded into a sanitised tempfile via the common
// files helper (with the standard 8 GiB cap) and the tempfile is cleaned
// up before this function returns.
func ManifestBodiesEqual(ctx context.Context, stored, src string) (bool, error) {
	if src == "" {
		return stored == "", nil
	}

	data, err := readSourceBytes(ctx, src)
	if err != nil {
		return false, err
	}

	return bytes.Equal([]byte(stored), data), nil
}

// readSourceBytes loads the full contents of a manifest source into memory.
// Manifests are small plists — not worth streaming. Reuses
// files.OpenUploadSource for the URL-or-path branch.
func readSourceBytes(ctx context.Context, src string) ([]byte, error) {
	f, _, cleanup, err := files.OpenUploadSource(ctx, src, files.DefaultMaxBytes)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading manifest source %q: %w", src, err)
	}
	return data, nil
}

// pollDecision summarises the convergence-check outcome for a single
// poll tick. Pure-function extraction lets the decision logic be
// unit-tested without spinning up a Pro client.
type pollDecision int

const (
	pollDecisionContinue pollDecision = iota
	pollDecisionConverged
	pollDecisionCorruption
)

// classifyPollTick decides what to do with the server's response on a
// single verification poll tick. expectHash + expectSize describe the
// locally-known bytes; previousHash is the digest stored before the
// upload (used to swallow the "still recomputing" transient view).
//
//   - Converged: hash matches expected, hashType is SHA3_512, status is
//     READY, AND size matches expected. The size gate catches the
//     transient JCDS window where hash flips to new while size still
//     shows the old binary.
//   - Corruption: server reported a SHA3_512 hash that is neither
//     expected nor previousHash AND size matches expected (corrupt bytes
//     would still report their own size). Skip the corruption branch
//     when size has not yet caught up — that is the transient window,
//     not a real mismatch.
//   - Continue: everything else.
func classifyPollTick(cur, expectHash, previousHash, hashType, status, curSize string, expectSize int64) pollDecision {
	sizeMatch := curSize != "" && curSize == strconv.FormatInt(expectSize, 10)
	hashMatch := cur != "" && cur == expectHash && hashType == hashTypeSHA3512 && status == cloudStatusReady
	if hashMatch && sizeMatch {
		return pollDecisionConverged
	}
	if cur != "" && cur != expectHash && cur != previousHash && hashType == hashTypeSHA3512 && sizeMatch {
		return pollDecisionCorruption
	}
	return pollDecisionContinue
}
