// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package smtp_server

import (
	"context"
	"fmt"

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
