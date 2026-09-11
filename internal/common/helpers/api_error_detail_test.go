// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/jamf/terraform-provider-jamfplatform/internal/testhelpers/gatewaystub"
)

// stubEgressIP binds the address lookup for one test, so nothing here reaches
// the network. Not t.Parallel-safe against the other tests in this file, which
// is why the tests that call it do not declare t.Parallel.
func stubEgressIP(t *testing.T, ip string) {
	t.Helper()

	original := egressIPLookup
	egressIPLookup = func() string { return ip }
	t.Cleanup(func() { egressIPLookup = original })
}

// TestAPIErrorDetail_OnlyEdgePagesGainGuidance pins the property that let this
// be applied to every call site at once: it changes nothing for an error a Jamf
// service produced. If it ever appended to those, 791 diagnostics would grow a
// paragraph of irrelevant network advice.
func TestAPIErrorDetail_OnlyEdgePagesGainGuidance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		reply gatewaystub.Reply
	}{
		{
			name: "pro json error",
			reply: gatewaystub.Reply{
				Status: http.StatusBadRequest, ContentType: "application/json",
				Body: `{"httpStatus":400,"errors":[{"code":"INVALID_FIELD","description":"name is required"}]}`,
			},
		},
		{
			name: "classic status page",
			reply: gatewaystub.Reply{
				Status: http.StatusNotFound, ContentType: "text/html", Body: gatewaystub.ClassicStatusPage,
			},
		},
		{
			name: "classic xml echo",
			reply: gatewaystub.Reply{
				Status: http.StatusBadRequest, ContentType: "application/xml",
				Body: `<?xml version="1.0" encoding="UTF-8"?><ebook><id>42</id></ebook>`,
			},
		},
		{
			name: "gateway unrouted",
			reply: gatewaystub.Reply{
				Status: http.StatusNotFound, ContentType: "text/plain; charset=utf-8", Body: "404 page not found\n",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := gatewaystub.ErrorFrom(t, tc.reply)
			if got, want := APIErrorDetail(err), err.Error(); got != want {
				t.Errorf("APIErrorDetail added guidance to an error Jamf produced:\ngot  %q\nwant %q", got, want)
			}
		})
	}
}

// TestAPIErrorDetail_SplitsBlockFromGatewayFailure pins the split the SDK's own
// godoc asks consumers to make. The two need opposite remedies: a block is
// standing and needs an allowlist change, a gateway error is transient and the
// SDK has already retried it, so telling an operator to chase their egress IP
// over a 504 costs a day and finds nothing.
func TestAPIErrorDetail_SplitsBlockFromGatewayFailure(t *testing.T) {

	t.Run("403 block reports the address it looked up", func(t *testing.T) {
		stubEgressIP(t, "203.0.113.10")

		detail := APIErrorDetail(gatewaystub.ErrorFrom(t, gatewaystub.Reply{
			Status: http.StatusForbidden, ContentType: "text/html", Body: gatewaystub.NginxBlockPage,
		}))

		if !strings.Contains(detail, "203.0.113.10") {
			t.Errorf("a blocked request does not report the address, which is the one thing "+
				"Jamf Support needs:\n%s", detail)
		}
		if strings.Contains(detail, EgressIPLookupURL) {
			t.Errorf("the address was looked up, so the operator should not also be told to run "+
				"the command:\n%s", detail)
		}
		if strings.Contains(detail, "Run the command again") {
			t.Errorf("a blocked request is described as transient, so the operator will re-run it "+
				"instead of fixing the allowlist:\n%s", detail)
		}
	})

	t.Run("403 block falls back to the command when the lookup fails", func(t *testing.T) {
		stubEgressIP(t, "")

		detail := APIErrorDetail(gatewaystub.ErrorFrom(t, gatewaystub.Reply{
			Status: http.StatusForbidden, ContentType: "text/html", Body: gatewaystub.NginxBlockPage,
		}))

		if !strings.Contains(detail, EgressIPLookupURL) {
			t.Errorf("a failed lookup left the operator with no way to find the address:\n%s", detail)
		}
	})

	t.Run("504 gateway failure asks for a re-run", func(t *testing.T) {
		stubEgressIP(t, "203.0.113.10")

		detail := APIErrorDetail(gatewaystub.ErrorFrom(t, gatewaystub.Reply{
			Status: http.StatusGatewayTimeout, ContentType: "text/html", Body: gatewaystub.CloudFrontPage(gatewaystub.CloudFrontGatewayTimeout),
		}))

		if !strings.Contains(detail, "Run the command again") {
			t.Errorf("a gateway failure does not say to run it again:\n%s", detail)
		}
		if strings.Contains(detail, "203.0.113.10") || strings.Contains(detail, EgressIPLookupURL) {
			t.Errorf("a gateway failure sends the operator after an egress IP, which the SDK has "+
				"already retried past and which is not the cause:\n%s", detail)
		}
	})
}

// TestAPIErrorDetail_KeepsTheOriginalMessage pins that guidance is appended
// rather than substituted. The status, endpoint and edge request id are what a
// support ticket is opened with, so losing them to a friendlier paragraph would
// be a worse diagnostic than the raw one this replaced.
func TestAPIErrorDetail_KeepsTheOriginalMessage(t *testing.T) {
	stubEgressIP(t, "203.0.113.10")

	err := gatewaystub.ErrorFrom(t, gatewaystub.Reply{
		Status: http.StatusForbidden, ContentType: "text/html", Body: gatewaystub.NginxBlockPage,
	})

	detail := APIErrorDetail(err)
	if !strings.HasPrefix(detail, err.Error()) {
		t.Errorf("the original error text is not the start of the detail:\n%s", detail)
	}
}

// TestAPIErrorDetail_NilAndNonAPI covers the two inputs a sweep of 791 call
// sites guarantees will reach this: a nil error on a path that reports one
// without checking, and an error that never came from the API at all.
func TestAPIErrorDetail_NilAndNonAPI(t *testing.T) {
	t.Parallel()

	if got := APIErrorDetail(nil); got != "" {
		t.Errorf("APIErrorDetail(nil) = %q, want empty", got)
	}

	plain := errors.New("converting plan to model: unhandled type")
	if got := APIErrorDetail(plain); got != plain.Error() {
		t.Errorf("APIErrorDetail changed a non-API error: got %q, want %q", got, plain.Error())
	}
}
