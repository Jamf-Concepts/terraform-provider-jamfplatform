// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_invitation

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildComputerInvitationInput_FullPayload(t *testing.T) {
	plan := ComputerInvitationResourceModel{
		InvitationType:              types.StringValue("USER_INITIATED_URL"),
		ExpirationDate:              types.StringValue("2026-12-31 23:59:00"),
		KeepExistingSiteMembership:  types.BoolValue(true),
		MultipleUsesAllowed:         types.BoolValue(true),
		CreateAccountIfDoesNotExist: types.BoolValue(true),
		HideAccount:                 types.BoolValue(false),
		LockDownSSH:                 types.BoolValue(true),
		SSHUsername:                 types.StringValue("jamfmgmt"),
	}

	pw := "s3cret"
	got := buildComputerInvitationInput(plan, &pw, "7")

	if got.InvitationType == nil || *got.InvitationType != "USER_INITIATED_URL" {
		t.Errorf("invitation_type not propagated: %v", got.InvitationType)
	}
	if got.ExpirationDate == nil || *got.ExpirationDate != "2026-12-31 23:59:00" {
		t.Errorf("expiration_date not propagated: %v", got.ExpirationDate)
	}
	if got.SshPassword == nil || *got.SshPassword != "s3cret" {
		t.Errorf("ssh_password not propagated: %v", got.SshPassword)
	}
	if got.SshUsername == nil || *got.SshUsername != "jamfmgmt" {
		t.Errorf("ssh_username not propagated: %v", got.SshUsername)
	}
	if got.EnrollIntoSite == nil || got.EnrollIntoSite.ID == nil || *got.EnrollIntoSite.ID != 7 {
		t.Fatalf("enroll_into_site ID not set from enroll_into_site_id, got %+v", got.EnrollIntoSite)
	}
	if got.HideAccount == nil || *got.HideAccount {
		t.Errorf("hide_account=false not propagated: %v", got.HideAccount)
	}
}

func TestBuildComputerInvitationInput_OmitsSiteWhenEmpty(t *testing.T) {
	plan := ComputerInvitationResourceModel{
		InvitationType: types.StringValue("USER_INITIATED_EMAIL"),
	}
	got := buildComputerInvitationInput(plan, nil, "")
	if got.EnrollIntoSite != nil {
		t.Errorf("expected no enroll_into_site block when site id empty, got %+v", got.EnrollIntoSite)
	}
	if got.SshPassword != nil {
		t.Errorf("expected nil ssh_password when not provided, got %v", got.SshPassword)
	}
}

func TestBuildComputerInvitationInput_OmitsNullBools(t *testing.T) {
	plan := ComputerInvitationResourceModel{
		InvitationType:      types.StringValue("USER_INITIATED_URL"),
		MultipleUsesAllowed: types.BoolNull(),
		HideAccount:         types.BoolUnknown(),
	}
	got := buildComputerInvitationInput(plan, nil, "")
	if got.MultipleUsesAllowed != nil {
		t.Errorf("null bool must serialise as nil (omitted), got %v", got.MultipleUsesAllowed)
	}
	if got.HideAccount != nil {
		t.Errorf("unknown bool must serialise as nil (omitted), got %v", got.HideAccount)
	}
}
