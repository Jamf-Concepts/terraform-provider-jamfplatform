// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package egressip

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// The lookup trims the trailing newline the echo service sends, so the address
// does not break the alignment of the support block it is rendered into.
func TestFetchFrom_TrimsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("203.0.113.10\n"))
	}))
	t.Cleanup(server.Close)

	if got := FetchFrom(server.URL); got != "203.0.113.10" {
		t.Errorf("FetchFrom() = %q, want %q", got, "203.0.113.10")
	}
}

// An intercepting proxy may answer the lookup with an arbitrarily long body. The
// read is capped so it cannot be pasted wholesale into a Terraform error.
func TestFetchFrom_CapsOversizedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("A", 4096)))
	}))
	t.Cleanup(server.Close)

	if got := FetchFrom(server.URL); len(got) > maxResponseBytes {
		t.Errorf("FetchFrom() returned %d bytes, want the read capped at %d", len(got), maxResponseBytes)
	}
}

// The network this diagnostic exists for intercepts requests, so the echo
// service is as likely to be answered by a captive portal as by AWS. A page is
// not an address: it must return "" so the caller renders the manual command
// instead of pasting proxy-controlled bytes into a support ticket.
func TestFetchFrom_RejectsNonAddressBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<!DOCTYPE html><html><head><title>Sign in</title></head>"))
	}))
	t.Cleanup(server.Close)

	if got := FetchFrom(server.URL); got != "" {
		t.Errorf("FetchFrom() = %q, want empty string for a body that is not an address", got)
	}
}

// A proxy refusing the request answers with a status and a body of its own —
// 407 with a sign-in page being the common one. The status is checked before
// the body so that body never reaches the diagnostic.
func TestFetchFrom_RejectsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusProxyAuthRequired)
		_, _ = w.Write([]byte("203.0.113.10\n"))
	}))
	t.Cleanup(server.Close)

	if got := FetchFrom(server.URL); got != "" {
		t.Errorf("FetchFrom() = %q, want empty string for a non-2xx status", got)
	}
}

// An unreachable lookup returns "" rather than an error, so the caller can
// substitute the manual command.
func TestFetchFrom_UnreachableReturnsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close() // nothing is listening now

	if got := FetchFrom(url); got != "" {
		t.Errorf("FetchFrom() = %q, want empty string for an unreachable host", got)
	}
}

// TestLookup_RequestsOnce pins the property that makes Lookup safe to call from
// a resource diagnostic. An edge block fails every in-flight resource at once,
// so without the cache one outage becomes ten identical calls to a third-party
// service, each adding its timeout to an error the operator is already waiting
// on.
//
// Lookup is a package variable so this can bind the cache to a stub server;
// production code reads it, never writes it.
func TestLookup_RequestsOnce(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte("203.0.113.10\n"))
	}))
	t.Cleanup(server.Close)

	original := Lookup
	Lookup = onceFrom(server.URL)
	t.Cleanup(func() { Lookup = original })

	for range 10 {
		if got := Lookup(); got != "203.0.113.10" {
			t.Fatalf("Lookup() = %q, want %q", got, "203.0.113.10")
		}
	}

	if got := calls.Load(); got != 1 {
		t.Errorf("the echo service was called %d times for 10 lookups, want 1 — an edge block "+
			"would multiply one outage into a burst of third-party requests", got)
	}
}

// TestLookup_CachesAFailure covers the other half of the cache. Whatever stopped
// the first attempt — no general internet egress, or a proxy permitting only the
// Jamf host — will stop the rest, so retrying per failed resource is the cost
// the cache exists to avoid.
func TestLookup_CachesAFailure(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close()

	original := Lookup
	Lookup = onceFrom(url)
	t.Cleanup(func() { Lookup = original })

	for range 5 {
		if got := Lookup(); got != "" {
			t.Fatalf("Lookup() = %q, want empty for an unreachable service", got)
		}
	}
	if got := calls.Load(); got != 0 {
		t.Errorf("unexpected calls to a closed server: %d", got)
	}
}
