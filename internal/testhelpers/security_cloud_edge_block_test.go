// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"net/http"
	"testing"

	"github.com/jamf/terraform-provider-jamfplatform/internal/testhelpers/gatewaystub"
)

// TestIsMissingEntitlementRejectsAnEdgeBlock pins the edge exclusion on the
// bare-403 reading. A WAF or IP allowlist answers a catalogue read with a 403
// HTML page of its own; read as "this tenant is not entitled" it would skip the
// lane and report the run green with its egress blocked, and because each
// fixture caches the result in a sync.Once one misclassification silences every
// later consumer. That is the silent skip JAMFPLATFORM_ACC_REQUIRE exists to
// prevent.
//
// Tagged acceptance because the function under test is, so this runs in the
// acceptance lanes rather than in make test. It needs no credentials and no
// estate: gatewaystub drives the real SDK against a local stub, which is the
// only way to obtain an error the SDK itself marked as an edge page.
func TestIsMissingEntitlementRejectsAnEdgeBlock(t *testing.T) {
	err := gatewaystub.ErrorFrom(t, gatewaystub.Reply{
		Status:      http.StatusForbidden,
		ContentType: "text/html",
		Body:        gatewaystub.NginxBlockPage,
	})
	if isMissingEntitlement(err) {
		t.Fatalf("an edge 403 must fail the run, not skip it as an unentitled tenant: %v", err)
	}
}

// TestIsMissingEntitlementAcceptsABareJamfForbidden keeps the exclusion from
// swallowing the case it narrows. Jamf Security Cloud answers an unmapped route
// with BAD_PERMISSIONS and a tenant without the entitlement can present the
// same way, so a 403 a Jamf service produced must still skip.
func TestIsMissingEntitlementAcceptsABareJamfForbidden(t *testing.T) {
	err := gatewaystub.ErrorFrom(t, gatewaystub.Reply{
		Status:      http.StatusForbidden,
		ContentType: "application/json",
		Body:        `{"errors":[{"code":"BAD_PERMISSIONS"}]}`,
	})
	if !isMissingEntitlement(err) {
		t.Fatalf("a bare Jamf 403 must still skip as an unentitled tenant: %v", err)
	}
}
