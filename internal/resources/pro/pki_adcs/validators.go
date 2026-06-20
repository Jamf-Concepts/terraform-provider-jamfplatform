// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_adcs

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// connectorModeConfigValidator enforces the discriminator contract between
// `connector_mode` and the mode-specific fields at plan time (wire-proven — see
// spike/PKI_CERTIFICATES_SPIKE.md §4):
//
//   - INBOUND  → adcs_url + server_certificate + client_certificate required;
//     api_client_id forbidden.
//   - OUTBOUND → api_client_id required; adcs_url + server_certificate +
//     client_certificate forbidden.
//
// Off-the-shelf framework validators express "exactly one of" but not this
// value-discriminated shape, so this is a custom resource.ConfigValidator
// (mirrors smtp_server.authBlockConfigValidator). See STYLE_GUIDE §Cross-field
// validation.
//
// FOOTGUN (documented in the schema): a config validator only sees what is
// *declared* in config. Because Jamf Pro preserves omitted optional fields
// (merge-patch), a value left over from a previous apply is NOT re-validated here
// — this catches a both-declared conflict, not a preserved one. Unknown values
// defer (sourced from a variable / another resource) rather than false-error
// (STYLE_GUIDE §"Config-time validators MUST defer on unknown values").
type connectorModeConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (connectorModeConfigValidator) Description(context.Context) string {
	return "connector_mode selects which fields apply: INBOUND requires adcs_url + server_certificate + client_certificate and forbids api_client_id; OUTBOUND requires api_client_id and forbids adcs_url + server_certificate + client_certificate."
}

// MarkdownDescription returns the markdown description.
func (v connectorModeConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
func (connectorModeConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var mode types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("connector_mode"), &mode)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if mode.IsNull() || mode.IsUnknown() {
		return
	}

	var adcsURL, apiClientID types.String
	var serverCert, clientCert types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("adcs_url"), &adcsURL)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("api_client_id"), &apiClientID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("server_certificate"), &serverCert)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("client_certificate"), &clientCert)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := mode.ValueString()
	switch m {
	case connectorModeInbound:
		requireString(resp, path.Root("adcs_url"), adcsURL, m, "adcs_url")
		requireBlock(resp, path.Root("server_certificate"), serverCert, m, "server_certificate")
		requireBlock(resp, path.Root("client_certificate"), clientCert, m, "client_certificate")
		forbidString(resp, path.Root("api_client_id"), apiClientID, m, "api_client_id")
	case connectorModeOutbound:
		requireString(resp, path.Root("api_client_id"), apiClientID, m, "api_client_id")
		forbidString(resp, path.Root("adcs_url"), adcsURL, m, "adcs_url")
		forbidBlock(resp, path.Root("server_certificate"), serverCert, m, "server_certificate")
		forbidBlock(resp, path.Root("client_certificate"), clientCert, m, "client_certificate")
	}
}

// requireString errors when the required scalar is genuinely null; defers on unknown.
func requireString(resp *resource.ValidateConfigResponse, p path.Path, v types.String, mode, name string) {
	if v.IsUnknown() {
		return
	}
	if v.IsNull() {
		resp.Diagnostics.AddAttributeError(
			p,
			fmt.Sprintf("%s required when connector_mode=%q", name, mode),
			fmt.Sprintf("`connector_mode = %q` requires `%s` to be set.", mode, name),
		)
	}
}

// forbidString errors when a forbidden scalar is present; safe on unknown.
func forbidString(resp *resource.ValidateConfigResponse, p path.Path, v types.String, mode, name string) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		p,
		fmt.Sprintf("%s forbidden when connector_mode=%q", name, mode),
		fmt.Sprintf("`%s` cannot be set when `connector_mode = %q`. Remove it, or change `connector_mode`.", name, mode),
	)
}

// requireBlock errors when the required block is genuinely null; defers on unknown.
func requireBlock(resp *resource.ValidateConfigResponse, p path.Path, block types.Object, mode, name string) {
	if block.IsUnknown() {
		return
	}
	if block.IsNull() {
		resp.Diagnostics.AddAttributeError(
			p,
			fmt.Sprintf("%s block required when connector_mode=%q", name, mode),
			fmt.Sprintf("`connector_mode = %q` requires the `%s` block to be set.", mode, name),
		)
	}
}

// forbidBlock errors when a forbidden block is present; safe on unknown.
func forbidBlock(resp *resource.ValidateConfigResponse, p path.Path, block types.Object, mode, name string) {
	if block.IsNull() || block.IsUnknown() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		p,
		fmt.Sprintf("%s block forbidden when connector_mode=%q", name, mode),
		fmt.Sprintf("`%s` cannot be set when `connector_mode = %q`. Remove the block, or change `connector_mode`.", name, mode),
	)
}

// Compile-time interface assertion.
var _ resource.ConfigValidator = connectorModeConfigValidator{}
