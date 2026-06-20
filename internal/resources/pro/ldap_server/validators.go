// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ldap_server

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// accountAuthConfigValidator enforces the relationship between
// `connection_settings.authentication_type` and the `connection_settings.account` block at plan
// time, derived from the 2026-05-31 wire probe:
//
//   - authentication_type = "none" → anonymous bind; the `account` block must
//     be absent.
//   - any other authentication_type → a bind account is required; the
//     `account` block must be present with a non-empty distinguished_username.
//
// When authentication_type is null/unknown (Optional+Computed, server-defaulted
// to "none") the validator stays silent — there is nothing reliable to check.
type accountAuthConfigValidator struct{}

// Description returns a plain-text description of the validator.
func (accountAuthConfigValidator) Description(context.Context) string {
	return "connection_settings.account must be present (with distinguished_username) when authentication_type is not \"none\", and absent when it is \"none\""
}

// MarkdownDescription returns the markdown description.
func (v accountAuthConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements the plan-time cross-field check.
//
// Every attribute it reads is fetched as its typed value via GetAttribute and
// guarded on IsUnknown before absence is treated as an error. A decode into the
// Go model (`req.Config.Get`) would collapse an unknown `account` block to a nil
// pointer, false-erroring "account required" whenever the block is sourced from
// a variable / another resource. (STYLE_GUIDE §"Config-time validators MUST
// defer on unknown values".)
func (accountAuthConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	connPath := path.Root("connection_settings")
	accountPath := connPath.AtName("account")

	var authAttr types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, connPath.AtName("authentication_type"), &authAttr)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if authAttr.IsNull() || authAttr.IsUnknown() {
		return
	}
	auth := authAttr.ValueString()

	var account types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, accountPath, &account)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Defer when the account block itself is unknown — its eventual presence is
	// unknowable at plan time, so neither the "required" nor the "forbidden"
	// branch can fire safely.
	if account.IsUnknown() {
		return
	}
	hasAccount := !account.IsNull()

	if auth == authTypeNone {
		if hasAccount {
			resp.Diagnostics.AddAttributeError(
				accountPath,
				"account forbidden for anonymous bind",
				"`connection_settings.account` must not be set when `connection_settings.authentication_type = \"none\"` (anonymous bind). Remove the account block or choose an authentication type that requires credentials.",
			)
		}
		return
	}

	if !hasAccount {
		resp.Diagnostics.AddAttributeError(
			accountPath,
			"account required for authenticated bind",
			"`connection_settings.account` is required when `connection_settings.authentication_type` is not `\"none\"`. Supply the account block with a `distinguished_username` (and a `password` + `password_wo_version = 1` on first create).",
		)
		return
	}

	var dn types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, accountPath.AtName("distinguished_username"), &dn)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if dn.IsUnknown() {
		return
	}
	if dn.IsNull() || dn.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			accountPath.AtName("distinguished_username"),
			"distinguished_username required for authenticated bind",
			"`connection_settings.account.distinguished_username` must be set when `connection_settings.authentication_type` is not `\"none\"`.",
		)
	}
}

// Compile-time interface assertion.
var _ resource.ConfigValidator = accountAuthConfigValidator{}
