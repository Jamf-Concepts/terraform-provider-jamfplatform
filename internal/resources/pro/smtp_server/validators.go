// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package smtp_server

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// authBlockConfigValidator enforces the discriminator contract between
// `authentication_type` and the connection / credential blocks at plan time:
//
//   - NONE        → `connection_settings` required; all credential blocks forbidden.
//   - BASIC       → `connection_settings` + `basic_auth_credentials` required; graph/google forbidden.
//   - GRAPH_API   → `graph_api_credentials` required; connection_settings/basic/google forbidden.
//   - GOOGLE_MAIL → `google_mail_credentials` required; connection_settings/basic/graph forbidden.
//
// Off-the-shelf framework validators express "exactly one of" but not the
// value-discriminated "the block matching authentication_type must be present
// (and the foreign ones absent)", so this is a custom resource.ConfigValidator
// (mirrors cloud_identity_provider.providerBlockConfigValidator and
// directory_binding.typeBlockConfigValidator). See STYLE_GUIDE §Cross-field
// validation.
//
// Block presence is read from typed Object values via GetAttribute so an unknown
// block (sourced from a variable / another resource) defers rather than
// false-erroring (STYLE_GUIDE §"Config-time validators MUST defer on unknown
// values").
type authBlockConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (authBlockConfigValidator) Description(context.Context) string {
	return "authentication_type selects which blocks apply: NONE requires connection_settings only; BASIC requires connection_settings + basic_auth_credentials; GRAPH_API requires graph_api_credentials only; GOOGLE_MAIL requires google_mail_credentials only. All non-matching blocks are forbidden."
}

// MarkdownDescription returns the markdown description.
func (v authBlockConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
func (authBlockConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var authType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("authentication_type"), &authType)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if authType.IsNull() || authType.IsUnknown() {
		return
	}

	var connection, basic, graph, google types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("connection_settings"), &connection)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("basic_auth_credentials"), &basic)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("graph_api_credentials"), &graph)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("google_mail_credentials"), &google)...)
	if resp.Diagnostics.HasError() {
		return
	}

	at := authType.ValueString()
	switch at {
	case authNone:
		requireBlock(resp, path.Root("connection_settings"), connection, at, "connection_settings")
		forbidBlock(resp, path.Root("basic_auth_credentials"), basic, at, "basic_auth_credentials")
		forbidBlock(resp, path.Root("graph_api_credentials"), graph, at, "graph_api_credentials")
		forbidBlock(resp, path.Root("google_mail_credentials"), google, at, "google_mail_credentials")
	case authBasic:
		requireBlock(resp, path.Root("connection_settings"), connection, at, "connection_settings")
		requireBlock(resp, path.Root("basic_auth_credentials"), basic, at, "basic_auth_credentials")
		forbidBlock(resp, path.Root("graph_api_credentials"), graph, at, "graph_api_credentials")
		forbidBlock(resp, path.Root("google_mail_credentials"), google, at, "google_mail_credentials")
	case authGraphAPI:
		requireBlock(resp, path.Root("graph_api_credentials"), graph, at, "graph_api_credentials")
		forbidBlock(resp, path.Root("connection_settings"), connection, at, "connection_settings")
		forbidBlock(resp, path.Root("basic_auth_credentials"), basic, at, "basic_auth_credentials")
		forbidBlock(resp, path.Root("google_mail_credentials"), google, at, "google_mail_credentials")
	case authGoogleMail:
		requireBlock(resp, path.Root("google_mail_credentials"), google, at, "google_mail_credentials")
		forbidBlock(resp, path.Root("connection_settings"), connection, at, "connection_settings")
		forbidBlock(resp, path.Root("basic_auth_credentials"), basic, at, "basic_auth_credentials")
		forbidBlock(resp, path.Root("graph_api_credentials"), graph, at, "graph_api_credentials")
	}
}

// requireBlock errors when the block matching authentication_type is genuinely
// null. Defers on unknown (not resolvable yet).
func requireBlock(resp *resource.ValidateConfigResponse, p path.Path, block types.Object, authType, blockName string) {
	if block.IsUnknown() {
		return
	}
	if block.IsNull() {
		resp.Diagnostics.AddAttributeError(
			p,
			fmt.Sprintf("%s block required when authentication_type = %q", blockName, authType),
			fmt.Sprintf("`authentication_type = %q` requires the `%s` block to be set.", authType, blockName),
		)
	}
}

// forbidBlock errors when a non-matching block is present. Safe on unknown (a
// forbidden-when check fires on presence; unknown-treated-as-absent defers).
func forbidBlock(resp *resource.ValidateConfigResponse, p path.Path, block types.Object, authType, blockName string) {
	if block.IsNull() || block.IsUnknown() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		p,
		fmt.Sprintf("%s block forbidden when authentication_type = %q", blockName, authType),
		fmt.Sprintf("`%s` cannot be set when `authentication_type = %q`. Remove the block, or change `authentication_type`.", blockName, authType),
	)
}

// Compile-time interface assertion.
var _ resource.ConfigValidator = authBlockConfigValidator{}

// validateSenderSettingsWhenEnabled enforces, at plan time, the sender-identity
// rules Jamf Pro applies only to an enabled connection.
//
// Every sender-settings check on the wire is gated on `enabled` (probed
// 2026-09-05, Jamf Pro 11.31, EU gateway). While the connection is disabled both
// fields accept an empty string and round-trip as one, which is exactly how a
// tenant that has never set up mail reads back — so an empty address is a real
// state this resource has to be able to hold, and neither field carries a
// minimum-length validator. Enabling is where the server turns strict, and it
// attributes each refusal independently:
//
//	emailAddress not in address format → 400 [INVALID_EMAIL]
//	displayName empty, or omitted      → 400 [INVALID_DISPLAY_NAME]
//
// Only the empty cases are checked here. Address *format* is left to the server:
// it publishes no pattern, and a provider-side guess at one would refuse
// addresses Jamf Pro accepts.
//
// enabled is Optional+Computed, so this must run against the resolved plan
// rather than the configuration — a value carried forward by
// UseStateForUnknown is what the apply will actually send. An unknown value
// defers to the server on the same reasoning as
// app_request_settings.validateEnabledRequiresRequesterGroup: a first apply that
// leaves display_name out has no prior state to resolve from, so the check
// cannot fire and smtpServerWriteErrorDiagnostic translates the refusal instead.
func validateSenderSettingsWhenEnabled(enabled types.Bool, sender *smtpSenderSettingsModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if enabled.IsNull() || enabled.IsUnknown() || !enabled.ValueBool() || sender == nil {
		return diags
	}

	if !sender.EmailAddress.IsUnknown() && sender.EmailAddress.ValueString() == "" {
		diags.AddAttributeError(
			path.Root("sender_settings").AtName("email_address"),
			"Sender email address required to enable the SMTP server",
			"Jamf Pro accepts an empty sender email address only while the connection is disabled, and a tenant "+
				"that has never set up mail reads back with an empty one. Set sender_settings.email_address to the "+
				"address Jamf Pro should send from, or leave enabled false.",
		)
	}

	if !sender.DisplayName.IsUnknown() && sender.DisplayName.ValueString() == "" {
		diags.AddAttributeError(
			path.Root("sender_settings").AtName("display_name"),
			"Sender display name required to enable the SMTP server",
			"Jamf Pro requires a non-empty sender display name on an enabled connection, and refuses the write "+
				"when it is empty or absent. Set sender_settings.display_name to the name recipients should see, "+
				"or leave enabled false.",
		)
	}

	return diags
}
