// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package files

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/sync/errgroup"
)

func TestURLSource(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"https://example.com/foo.pkg", true},
		{"http://example.com/foo.pkg", true},
		{"file:///tmp/foo.pkg", false},
		{"/tmp/foo.pkg", false},
		{"./relative.pkg", false},
		{"", false},
		{"HTTPS://example.com", false}, // case-sensitive on purpose, mirrors design doc
	}
	for _, c := range cases {
		if got := URLSource(c.in); got != c.want {
			t.Errorf("URLSource(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSanitizedTempPath(t *testing.T) {
	temp := os.TempDir()

	cases := []struct {
		name    string
		url     string
		wantErr bool
		wantSub string // expected basename substring on success
	}{
		{"plain pkg", "https://example.com/foo.pkg", false, "foo.pkg"},
		{"nested path", "https://example.com/dir/sub/foo.pkg", false, "foo.pkg"},
		{"missing basename", "https://example.com/", false, "download"},
		// path.Base strips the directory portion — "../../etc/passwd" reduces to
		// "passwd", which is a safe basename. The traversal defence guards
		// against pathological cases where the basename itself contains "..".
		{"basename collapses via path.Base", "https://example.com/../../etc/passwd", false, "passwd"},
		{"absolute basename ignored", "https://example.com/foo.pkg?x=y", false, "foo.pkg"},
		{"malformed url", "://not-a-url", true, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SanitizedTempPath(c.url)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got path %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasPrefix(got, temp) {
				t.Errorf("path %q is not under temp dir %q", got, temp)
			}
			if !strings.Contains(filepath.Base(got), c.wantSub) {
				t.Errorf("basename %q does not contain %q", filepath.Base(got), c.wantSub)
			}
		})
	}
}

func TestDownloadCapped_Success(t *testing.T) {
	body := []byte("hello-pkg-bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	filename, n, err := DownloadCapped(context.Background(), srv.URL+"/something.pkg", &buf, 0)
	if err != nil {
		t.Fatalf("DownloadCapped: %v", err)
	}
	if n != int64(len(body)) {
		t.Errorf("wrote %d bytes, want %d", n, len(body))
	}
	if !bytes.Equal(buf.Bytes(), body) {
		t.Errorf("body mismatch: got %q want %q", buf.String(), string(body))
	}
	if filename != "something.pkg" {
		t.Errorf("filename %q, want %q", filename, "something.pkg")
	}
}

func TestDownloadCapped_RedirectsCapHonoured(t *testing.T) {
	// Build a chain of 12 redirects; client cap is 10 — expect failure.
	mux := http.NewServeMux()
	for i := range 12 {
		mux.HandleFunc("/hop"+itoa(i), func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/hop"+itoa(i+1), http.StatusFound)
		})
	}
	mux.HandleFunc("/hop12", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("end"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var buf bytes.Buffer
	_, _, err := DownloadCapped(context.Background(), srv.URL+"/hop0", &buf, 0)
	if err == nil {
		t.Fatalf("expected redirect-cap error, got nil")
	}
	if !strings.Contains(err.Error(), "redirects") {
		t.Errorf("error %v does not mention redirects", err)
	}
}

func TestDownloadCapped_CapExceeded(t *testing.T) {
	// Server writes 100 bytes, cap is 10.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), 100))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	_, _, err := DownloadCapped(context.Background(), srv.URL+"/big.pkg", &buf, 10)
	if !errors.Is(err, ErrDownloadCapExceeded) {
		t.Fatalf("expected ErrDownloadCapExceeded, got %v", err)
	}
}

func TestDownloadCapped_HTTPErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	_, _, err := DownloadCapped(context.Background(), srv.URL+"/missing.pkg", &buf, 0)
	if err == nil {
		t.Fatalf("expected error on 404")
	}
}

func TestOpenUploadSource_LocalPath(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "src*.pkg")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	want := []byte("local-pkg-bytes")
	if _, err := tmp.Write(want); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("setup: %v", err)
	}

	file, filename, cleanup, err := OpenUploadSource(context.Background(), tmp.Name(), DefaultMaxBytes)
	if err != nil {
		t.Fatalf("OpenUploadSource: %v", err)
	}
	defer cleanup()

	if filename != filepath.Base(tmp.Name()) {
		t.Errorf("filename %q, want %q", filename, filepath.Base(tmp.Name()))
	}
	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("bytes mismatch: got %q want %q", got, want)
	}
}

func TestOpenUploadSource_URL(t *testing.T) {
	want := []byte("url-pkg-bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(want)
	}))
	defer srv.Close()

	file, filename, cleanup, err := OpenUploadSource(context.Background(), srv.URL+"/remote.pkg", DefaultMaxBytes)
	if err != nil {
		t.Fatalf("OpenUploadSource: %v", err)
	}

	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("bytes mismatch: got %q want %q", got, want)
	}
	if filename != "remote.pkg" {
		t.Errorf("filename %q, want %q", filename, "remote.pkg")
	}

	tempPath := file.Name()
	cleanup()
	if _, err := os.Stat(tempPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected temp file %q to be removed after cleanup, stat err=%v", tempPath, err)
	}
}

func TestOpenUploadSource_URLCapExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), 100))
	}))
	defer srv.Close()

	_, _, _, err := OpenUploadSource(context.Background(), srv.URL+"/big.pkg", 10)
	if !errors.Is(err, ErrDownloadCapExceeded) {
		t.Fatalf("expected ErrDownloadCapExceeded, got %v", err)
	}
}

func TestOpenUploadSource_URLConcurrentSameName(t *testing.T) {
	// Reproduces the parallel-plan race: N icon resources all download the same
	// URL (e.g. Apple's "512x512bb.png") concurrently; each must get a unique
	// tempfile. The start-gate releases all goroutines simultaneously to maximise
	// overlap, mirroring Terraform's default -parallelism=10 walk.
	const n = 20
	body := []byte("icon-bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	url := srv.URL + "/512x512bb.png"

	var (
		mu    sync.Mutex
		paths []string
	)
	gate := make(chan struct{})

	var eg errgroup.Group
	for range n {
		eg.Go(func() error {
			<-gate
			file, _, cleanup, err := OpenUploadSource(context.Background(), url, DefaultMaxBytes)
			if err != nil {
				return err
			}
			defer cleanup()
			mu.Lock()
			paths = append(paths, file.Name())
			mu.Unlock()
			return nil
		})
	}

	close(gate)
	if err := eg.Wait(); err != nil {
		t.Fatalf("concurrent OpenUploadSource: %v", err)
	}

	if len(paths) != n {
		t.Fatalf("got %d paths, want %d", len(paths), n)
	}
	seen := make(map[string]bool, n)
	for _, p := range paths {
		if seen[p] {
			t.Errorf("duplicate temp path: %q", p)
		}
		seen[p] = true
	}
}

// itoa is a tiny local helper avoiding strconv import noise in the redirect
// test fixture.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
