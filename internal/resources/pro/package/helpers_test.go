// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"context"
	"crypto/sha3"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyPollTick(t *testing.T) {
	t.Parallel()

	const (
		expectHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		prevHash   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		corrupt    = "ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
		expectSize = int64(11806221)
	)
	expectSizeStr := "11806221"
	oldSizeStr := "11774769"

	cases := []struct {
		name      string
		cur       string
		hashType  string
		status    string
		curSize   string
		want      pollDecision
		wantNotes string
	}{
		{
			name:     "converged",
			cur:      expectHash,
			hashType: "SHA3_512",
			status:   "READY",
			curSize:  expectSizeStr,
			want:     pollDecisionConverged,
		},
		{
			name:      "still recomputing — previous hash",
			cur:       prevHash,
			hashType:  "SHA3_512",
			status:    "READY",
			curSize:   oldSizeStr,
			want:      pollDecisionContinue,
			wantNotes: "previousHash branch — JCDS still serving old bytes",
		},
		{
			name:      "transient hash flip — size still old",
			cur:       expectHash,
			hashType:  "SHA3_512",
			status:    "READY",
			curSize:   oldSizeStr,
			want:      pollDecisionContinue,
			wantNotes: "size guard — JCDS in commit window; hash flipped but size hasn't",
		},
		{
			name:      "corruption — bogus hash, size matches upload",
			cur:       corrupt,
			hashType:  "SHA3_512",
			status:    "READY",
			curSize:   expectSizeStr,
			want:      pollDecisionCorruption,
			wantNotes: "server stored bytes whose hash differs from what we sent",
		},
		{
			name:      "corruption skipped while size still in flight",
			cur:       corrupt,
			hashType:  "SHA3_512",
			status:    "READY",
			curSize:   oldSizeStr,
			want:      pollDecisionContinue,
			wantNotes: "do not flag corruption during the transient size window",
		},
		{
			name:     "empty server response — still warming up",
			cur:      "",
			hashType: "",
			status:   "",
			curSize:  "",
			want:     pollDecisionContinue,
		},
		{
			name:      "legacy SHA_512 algorithm tag — not converged",
			cur:       expectHash,
			hashType:  "SHA_512",
			status:    "READY",
			curSize:   expectSizeStr,
			want:      pollDecisionContinue,
			wantNotes: "pre-upload default — must wait for hashType to flip to SHA3_512",
		},
		{
			name:     "status not READY — not converged",
			cur:      expectHash,
			hashType: "SHA3_512",
			status:   "IN_PROGRESS",
			curSize:  expectSizeStr,
			want:     pollDecisionContinue,
		},
		{
			name:      "upload failed — size=0 with expected hash",
			cur:       expectHash,
			hashType:  "SHA3_512",
			status:    "READY",
			curSize:   "0",
			want:      pollDecisionUploadFailed,
			wantNotes: "definitive failed-upload signal, checked before the hash/size match branches",
		},
		{
			name:      "upload failed — size=0 with a bogus hash",
			cur:       corrupt,
			hashType:  "SHA3_512",
			status:    "IN_PROGRESS",
			curSize:   "0",
			want:      pollDecisionUploadFailed,
			wantNotes: "size=0 wins over corruption — do not wait out the poll budget to report the wrong failure mode",
		},
		{
			name:      "size=0 but hash has not changed yet — still warming up",
			cur:       prevHash,
			hashType:  "SHA3_512",
			status:    "READY",
			curSize:   "0",
			want:      pollDecisionContinue,
			wantNotes: "hash still equals previousHash — not yet a signal either way",
		},
		{
			name:      "empty size is not zero size",
			cur:       expectHash,
			hashType:  "SHA3_512",
			status:    "READY",
			curSize:   "",
			want:      pollDecisionContinue,
			wantNotes: "\"\" means not-yet-computed, not a failed upload",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyPollTick(tc.cur, expectHash, prevHash, tc.hashType, tc.status, tc.curSize, expectSize)
			if got != tc.want {
				t.Fatalf("decision = %v, want %v (%s)", got, tc.want, tc.wantNotes)
			}
		})
	}
}

// TestClassifyPollTick_ZeroByteUpload covers a genuinely empty
// package_file_source (expectSize == 0), which the "upload failed" fast path
// must NOT intercept — size "0" is the correct, converged value for a
// zero-byte binary, not a failure signal.
func TestClassifyPollTick_ZeroByteUpload(t *testing.T) {
	t.Parallel()

	const (
		expectHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		prevHash   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	got := classifyPollTick(expectHash, expectHash, prevHash, "SHA3_512", "READY", "0", 0)
	if got != pollDecisionConverged {
		t.Fatalf("decision = %v, want %v — a zero-byte upload must converge, not be flagged as a failed upload", got, pollDecisionConverged)
	}
}

func TestHashFileSHA3_KnownBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.pkg")
	payload := []byte("hello world")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, size, err := HashFileSHA3(path)
	if err != nil {
		t.Fatalf("HashFileSHA3: %v", err)
	}
	if size != int64(len(payload)) {
		t.Errorf("size = %d, want %d", size, len(payload))
	}

	h := sha3.New512()
	if _, err := h.Write(payload); err != nil {
		t.Fatalf("hash write: %v", err)
	}
	want := hex.EncodeToString(h.Sum(nil))
	if got != want {
		t.Errorf("hash mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestHashFileSHA3_MissingFile(t *testing.T) {
	_, _, err := HashFileSHA3(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatalf("expected error on missing file")
	}
}

func TestManifestBodiesEqual_LocalSourceMatches(t *testing.T) {
	dir := t.TempDir()
	body := []byte("<?xml version=\"1.0\"?><plist/>")
	path := filepath.Join(dir, "manifest.plist")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	equal, err := ManifestBodiesEqual(context.Background(), string(body), path)
	if err != nil {
		t.Fatalf("ManifestBodiesEqual: %v", err)
	}
	if !equal {
		t.Errorf("expected equal manifest bodies")
	}
}

func TestManifestBodiesEqual_LocalSourceDiffers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.plist")
	if err := os.WriteFile(path, []byte("new-body"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	equal, err := ManifestBodiesEqual(context.Background(), "old-body", path)
	if err != nil {
		t.Fatalf("ManifestBodiesEqual: %v", err)
	}
	if equal {
		t.Errorf("expected manifest bodies to differ")
	}
}

func TestManifestBodiesEqual_URLSourceMatches(t *testing.T) {
	body := []byte("<?xml version=\"1.0\"?><plist><dict/></plist>")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	equal, err := ManifestBodiesEqual(context.Background(), string(body), srv.URL+"/manifest.plist")
	if err != nil {
		t.Fatalf("ManifestBodiesEqual: %v", err)
	}
	if !equal {
		t.Errorf("expected equal URL-loaded manifest")
	}
}

func TestManifestBodiesEqual_EmptySrcMatchesEmptyStored(t *testing.T) {
	equal, err := ManifestBodiesEqual(context.Background(), "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equal {
		t.Errorf("empty src vs empty stored must be equal")
	}
}

func TestManifestBodiesEqual_EmptySrcVsNonEmptyStored(t *testing.T) {
	equal, err := ManifestBodiesEqual(context.Background(), "stored-body", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if equal {
		t.Errorf("empty src vs non-empty stored must not match")
	}
}

func TestIsZeroSize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		curSize string
		want    bool
	}{
		{name: "literal zero", curSize: "0", want: true},
		{name: "empty string is not zero", curSize: "", want: false},
		{name: "nonzero size", curSize: "11806221", want: false},
		{name: "unparseable is not zero", curSize: "not-a-number", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isZeroSize(tc.curSize); got != tc.want {
				t.Errorf("isZeroSize(%q) = %v, want %v", tc.curSize, got, tc.want)
			}
		})
	}
}

func TestRetryableUploadFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		err         error
		attempt     int
		maxAttempts int
		want        bool
	}{
		{name: "upload failed with attempts remaining", err: errUploadFailed, attempt: 1, maxAttempts: 5, want: true},
		{name: "upload failed on last attempt", err: errUploadFailed, attempt: 5, maxAttempts: 5, want: false},
		{name: "upload failed past budget", err: errUploadFailed, attempt: 6, maxAttempts: 5, want: false},
		{name: "corruption is never retried", err: errCorruption, attempt: 1, maxAttempts: 5, want: false},
		{name: "timeout is never retried", err: errVerificationTimeout, attempt: 1, maxAttempts: 5, want: false},
		{name: "nil error is never retried", err: nil, attempt: 1, maxAttempts: 5, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := retryableUploadFailure(tc.err, tc.attempt, tc.maxAttempts); got != tc.want {
				t.Errorf("retryableUploadFailure(%v, %d, %d) = %v, want %v", tc.err, tc.attempt, tc.maxAttempts, got, tc.want)
			}
		})
	}
}
