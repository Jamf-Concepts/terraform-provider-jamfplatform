// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_invitation

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// emailModeRequiresFieldsValidator enforces the server-side contract that
// distribution_method = "Send emails" requires the sender / subject / message
// fields. The classic endpoint otherwise 409s at apply with "Sender name is
// required"; this surfaces every missing field at plan time in one pass.
type emailModeRequiresFieldsValidator struct{}

var _ resource.ConfigValidator = emailModeRequiresFieldsValidator{}

func (v emailModeRequiresFieldsValidator) Description(_ context.Context) string {
	return fmt.Sprintf("when distribution_method is %q, sender_name, sender_email_address, subject, and message are required", distributionMethodSendEmails)
}

func (v emailModeRequiresFieldsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v emailModeRequiresFieldsValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg VPPInvitationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only enforce when distribution_method is known and equals "Send emails".
	if cfg.DistributionMethod.IsNull() || cfg.DistributionMethod.IsUnknown() {
		return
	}
	if cfg.DistributionMethod.ValueString() != distributionMethodSendEmails {
		return
	}

	required := []struct {
		val  types.String
		name string
	}{
		{cfg.SenderName, "sender_name"},
		{cfg.SenderEmailAddress, "sender_email_address"},
		{cfg.Subject, "subject"},
		{cfg.Message, "message"},
	}
	for _, r := range required {
		// Unknown (interpolated, not-yet-known) values cannot be validated at
		// plan time — only error on a definitively absent value.
		if r.val.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root(r.name),
				"Missing required field for email distribution",
				fmt.Sprintf("%s must be set when distribution_method is %q.", r.name, distributionMethodSendEmails),
			)
		}
	}
}
