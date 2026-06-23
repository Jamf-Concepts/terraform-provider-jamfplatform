// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_invitation

import (
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func bigInt(s string) *proclassic.BigInt {
	var b proclassic.BigInt
	b.SetString(s)
	return &b
}

func TestSiteValues_EchoVerbatim(t *testing.T) {
	// The `-1` id marker echoes verbatim, but the derived name nulls on the
	// sentinel (the classic GET nondeterministically echoes/omits "NONE").
	noneID := siteIDValue(&proclassic.ComputerInvitationEnrollIntoSite{ID: new(-1), Name: new("NONE")})
	if noneID.ValueString() != "-1" {
		t.Errorf("no-site id must echo -1, got %q", noneID.ValueString())
	}
	noneName := siteNameValue(&proclassic.ComputerInvitationEnrollIntoSite{ID: new(-1), Name: new("NONE")})
	if !noneName.IsNull() {
		t.Errorf("no-site name must be null on the sentinel, got %q", noneName.ValueString())
	}

	if !siteIDValue(nil).IsNull() {
		t.Errorf("nil site id must be null")
	}
	if !siteNameValue(nil).IsNull() {
		t.Errorf("nil site name must be null")
	}

	named := &proclassic.ComputerInvitationEnrollIntoSite{ID: new(7), Name: new("HQ")}
	if got := siteIDValue(named).ValueString(); got != "7" {
		t.Errorf("named site id must echo 7, got %q", got)
	}
	if got := siteNameValue(named).ValueString(); got != "HQ" {
		t.Errorf("named site name must echo HQ, got %q", got)
	}
}

func TestAssignComputerInvitationResourceModel_DriftAndSentinels(t *testing.T) {
	state := ComputerInvitationResourceModel{
		// User-authored value that the server will echo back drifted.
		ExpirationDate: types.StringValue("2026-12-31 23:59:00"),
	}
	src := &proclassic.ComputerInvitation{
		ID:                  new(280),
		Invitation:          bigInt("308000000000000000000000000000000000001"),
		InvitationType:      new("USER_INITIATED_URL"),
		ExpirationDate:      new("2026-12-31 23:58:59.306"),
		ExpirationDateEpoch: bigInt("1798761539306"),
		ExpirationDateUtc:   new("2026-12-31T23:58:59.306+0000"),
		EnrollIntoSite:      &proclassic.ComputerInvitationEnrollIntoSite{ID: new(-1), Name: new("NONE")},
		MultipleUsesAllowed: new(true),
		InvitationStatus:    new("MULTIPLE_USES"),
		TimesUsed:           new(0),
		SshUsername:         new("jamfmgmt"),
	}

	assignComputerInvitationResourceModel(&state, src)

	if state.ID.ValueString() != "280" {
		t.Errorf("id: got %q", state.ID.ValueString())
	}
	if state.ExpirationDate.ValueString() != "2026-12-31 23:59:00" {
		t.Errorf("expiration_date drift not reconciled: got %q", state.ExpirationDate.ValueString())
	}
	if state.EnrollIntoSiteID.ValueString() != "-1" {
		t.Errorf("no-site id must echo -1, got %q", state.EnrollIntoSiteID.ValueString())
	}
	if !state.EnrollIntoSiteName.IsNull() {
		t.Errorf("no-site name must be null on the sentinel, got %q", state.EnrollIntoSiteName.ValueString())
	}
	if state.Invitation.ValueString() != "308000000000000000000000000000000000001" {
		t.Errorf("invitation precision lost: %q", state.Invitation.ValueString())
	}
	if !state.MultipleUsesAllowed.ValueBool() {
		t.Errorf("multiple_uses_allowed not mapped")
	}
}
