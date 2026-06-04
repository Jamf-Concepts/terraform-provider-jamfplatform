// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_invitation

import (
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildComputerInvitationInput converts the Terraform plan model into the SDK
// ComputerInvitation payload used for Create. The endpoint has no update
// operation, so this is only ever called from Create.
//
//   - `sshPassword` is supplied separately because it is a WriteOnly attribute
//     (null in the plan model) and must be read from req.Config by the caller.
//   - `siteID` is supplied separately: it is the user's `enroll_into_site_id`
//     value (a Jamf Pro site ID). A non-empty string is parsed to an int and
//     sent as `<enroll_into_site><id>N</id></enroll_into_site>`. Empty means no
//     site — the block is omitted so the server defaults to `-1`/`NONE`.
//   - Server-derived fields (id, invitation, invitation_status, times_used,
//     invited_user_uuid, expiration_date_epoch/_utc, and the read-only <site>
//     element) are intentionally omitted; they are populated on read only.
func buildComputerInvitationInput(plan ComputerInvitationResourceModel, sshPassword *string, siteID string) *proclassic.ComputerInvitation {
	in := &proclassic.ComputerInvitation{
		InvitationType:              helpers.OptionalStringPointer(plan.InvitationType),
		ExpirationDate:              helpers.OptionalStringPointer(plan.ExpirationDate),
		KeepExistingSiteMembership:  optionalBoolPointer(plan.KeepExistingSiteMembership),
		MultipleUsesAllowed:         optionalBoolPointer(plan.MultipleUsesAllowed),
		CreateAccountIfDoesNotExist: optionalBoolPointer(plan.CreateAccountIfDoesNotExist),
		HideAccount:                 optionalBoolPointer(plan.HideAccount),
		LockDownSsh:                 optionalBoolPointer(plan.LockDownSSH),
		SshUsername:                 helpers.OptionalStringPointer(plan.SSHUsername),
		SshPassword:                 sshPassword,
	}

	if siteID != "" {
		if id, err := strconv.Atoi(siteID); err == nil {
			in.EnrollIntoSite = &proclassic.ComputerInvitationEnrollIntoSite{ID: &id}
		}
	}

	return in
}

// optionalBoolPointer mirrors helpers.OptionalStringPointer for types.Bool.
// Returns nil for null/unknown so omitted Optional bools are not serialised as
// `false`.
func optionalBoolPointer(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}
