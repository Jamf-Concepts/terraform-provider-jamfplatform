// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package pkg_test

import (
	"context"
	"crypto/sha3"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	pkg "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/package"
)

// Local fixtures committed under internal/resources/pro/inventory/package/test_fixtures/.
const (
	localFixture161 = "jamf-cli-1.16.1.pkg"
	localFixture150 = "jamf-cli-1.15.0.pkg"
)

// URL fixtures pulled at runtime from the public jamf-cli releases. Used by
// both the URL-disk-staging acceptance path and the streaming acceptance
// path so the same bytes flow through both code paths.
const (
	urlFixture170 = "https://github.com/Jamf-Concepts/jamf-cli/releases/download/v1.17.0/jamf-cli-1.17.0.pkg"
	urlFixture160 = "https://github.com/Jamf-Concepts/jamf-cli/releases/download/v1.16.0/jamf-cli-1.16.0.pkg"
)

// fixturePath returns an absolute path to a fixture file under the
// test_fixtures directory next to the test file.
func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("could not resolve caller path for fixture lookup")
	}
	dir := filepath.Dir(file)
	abs, err := filepath.Abs(filepath.Join(dir, "test_fixtures", name))
	if err != nil {
		t.Fatalf("resolving fixture path %q: %v", name, err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("fixture %q not present at %q: %v", name, abs, err)
	}
	return abs
}

// fixtureSHA3 returns the SHA-3-512 hex digest of a local fixture file.
// Used by testCheckPackageHashConverged closures to compare server-side
// hashes against the locally computed digest.
func fixtureSHA3(t *testing.T, path string) string {
	t.Helper()
	digest, _, err := pkg.HashFileSHA3(path)
	if err != nil {
		t.Fatalf("hashing fixture %q: %v", path, err)
	}
	return strings.ToLower(digest)
}

// urlFixtureCache memoises URL → (localPath, sha3) so each URL fixture is
// fetched at most once per test process. The streaming and disk paths
// both fetch from the same origin URL during the resource handler; this
// cache is only used to compute the EXPECTED hash for assertion.
var (
	urlFixtureMu    sync.Mutex
	urlFixtureCache = map[string]urlFixtureRecord{}
)

type urlFixtureRecord struct {
	path string
	sha3 string
}

// downloadAndHashURL fetches a URL into a tempfile (once per process) and
// returns the local path + SHA-3-512 digest. The path is only used to
// compute the expected hash for the test assertion — the resource handler
// fetches from the URL directly during the test.
func downloadAndHashURL(t *testing.T, url string) (string, string) {
	t.Helper()
	urlFixtureMu.Lock()
	defer urlFixtureMu.Unlock()

	if rec, ok := urlFixtureCache[url]; ok {
		return rec.path, rec.sha3
	}

	tmp, err := os.CreateTemp("", "tf-acc-url-fixture-*.pkg")
	if err != nil {
		t.Fatalf("creating tempfile for URL fixture: %v", err)
	}
	defer func() { _ = tmp.Close() }()

	resp, err := http.Get(url) //nolint:gosec,noctx // test-only fixture download
	if err != nil {
		_ = os.Remove(tmp.Name())
		t.Fatalf("fetching URL fixture %q: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = os.Remove(tmp.Name())
		t.Fatalf("URL fixture %q returned HTTP %d", url, resp.StatusCode)
	}

	hasher := sha3.New512()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		_ = os.Remove(tmp.Name())
		t.Fatalf("streaming URL fixture %q: %v", url, err)
	}

	digest := strings.ToLower(hex.EncodeToString(hasher.Sum(nil)))
	urlFixtureCache[url] = urlFixtureRecord{path: tmp.Name(), sha3: digest}
	return tmp.Name(), digest
}

// staticDigest produces the closure shape testCheckPackageHashConverged
// expects from a precomputed hex digest.
func staticDigest(d string) func() string {
	return func() string { return strings.ToLower(d) }
}

// renderHCL is a small sprintf wrapper that keeps the test bodies easy to
// scan when interpolating display_name / source / hash values.
func renderHCL(template string, args ...any) string {
	return fmt.Sprintf(template, args...)
}

// withContext returns a derived context for in-test SDK calls. Keeps the
// test bodies short.
func withContext() context.Context { return context.Background() }

// writeTempManifest stages a synthetic plist body in the test's tempdir.
// Returned path is auto-removed by the testing framework when the test
// finishes.
func writeTempManifest(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.plist")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing manifest fixture: %v", err)
	}
	return path
}
