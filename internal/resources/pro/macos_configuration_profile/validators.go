// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// notificationCenterUnwritableValidator refuses the one Self Service
// notification pairing Jamf Pro's classic API cannot store: notifications on
// with the location set to Notification Center.
//
// The two attributes are a projection of a single wire element. `<notification>`
// takes either a bool or a location string, the LAST occurrence in a request
// wins, and applying either one resets the other to its default — the bool to
// false, the location to "Self Service". Sending both therefore always loses
// one of them, whichever came first, and no ordering, third element, extra
// occurrence or companion field changes that. Wire-probed on two tenants
// against Jamf Pro 11.31.1 on 2026-09-07; the full record is on
// buildSelfServiceNotification.
//
// The admin UI stores the pairing perfectly well, so this is a defect in the
// classic API rather than a state Jamf Pro cannot hold — which is exactly why
// refusing it at plan time is the honest answer. Applying it would report
// success and store "Self Service", and a practitioner comparing Terraform
// against the admin UI would find the provider had quietly changed a setting it
// claimed to have written. Tracked as Jamf PI-1662.
//
// Delete this validator when the API accepts the pair; the acceptance canary
// TestAccResource_MacOSConfigurationProfile_NotificationCenterIsUnwritable
// fails on that day and says so.
type notificationCenterUnwritableValidator struct{}

// Description returns a plain-text description of the validator.
func (notificationCenterUnwritableValidator) Description(context.Context) string {
	return `self_service.notification_location cannot be "` + notificationLocationSelfServiceAndCenter + `" while self_service.display_notifications is true`
}

// MarkdownDescription returns the markdown description.
func (v notificationCenterUnwritableValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
//
// It reads one attribute at a time rather than decoding the whole model: a
// profile configuration routinely carries unknown nested values (an
// interpolated scope id, a payload from a function), and Config.Get on the
// whole model fails outright on those, which would disable every validator on
// the resource.
func (notificationCenterUnwritableValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	locationPath := path.Root("self_service").AtName("notification_location")

	var location types.String
	if diags := req.Config.GetAttribute(ctx, locationPath, &location); diags.HasError() {
		return
	}
	if location.IsNull() || location.IsUnknown() || location.ValueString() != notificationLocationSelfServiceAndCenter {
		return
	}

	var display types.Bool
	if diags := req.Config.GetAttribute(ctx, path.Root("self_service").AtName("display_notifications"), &display); diags.HasError() {
		return
	}
	// Null or unknown is not a violation: null leaves the flag to Jamf Pro,
	// which defaults it off, and unknown resolves after apply.
	if display.IsNull() || display.IsUnknown() || !display.ValueBool() {
		return
	}

	resp.Diagnostics.AddAttributeError(
		locationPath,
		"Notification Center cannot be combined with display_notifications",
		"Jamf Pro stores a profile's notification flag and its notification location in one field, and setting either one resets the other. "+
			"With display_notifications = true the location always comes back as \""+notificationLocationSelfService+"\", so this configuration would report success and store something else.\n\n"+
			"Either set self_service.notification_location = \""+notificationLocationSelfService+"\", or drop self_service.display_notifications and set the location on its own.\n\n"+
			"To have both, set them under Self Service ▸ Notification in the Jamf Pro admin UI, which is not subject to this limitation, and leave both attributes out of the configuration. "+
			"Terraform reads the pairing back and preserves it.",
	)
}
