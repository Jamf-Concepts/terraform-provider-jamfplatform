// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// titleResolver is the subset of *pro.Client used to look an App Catalog title up
// by name. Declaring it as an interface keeps the validator unit-testable without
// a live client.
type titleResolver interface {
	ResolveAppInstallerTitleV1IDByName(ctx context.Context, name string) (string, error)
}

// titleNamer is the subset of *pro.Client used to read a title's display name back
// from its ID (the deployment GET returns only the ID).
type titleNamer interface {
	GetAppInstallerTitleV1(ctx context.Context, id string) (*pro.AppInstallerTitle, error)
}

// validateAppTitleName is a plan-time preflight for app_title_name. The title is
// referenced by its catalog display name; this resolves it against the live
// App Catalog so an unknown name surfaces at plan time with a clear message
// instead of the apply-time resolution failure.
//
// Behaviour:
//   - nil resolver, or null/unknown/empty value: no-op (nothing to check, or the
//     provider is not yet configured, or the name derives from another resource).
//   - name resolves: OK.
//   - name not found (404, or Pro v1's 400 INVALID_ID): an error diagnostic.
//   - any other resolve error: a WARNING (not an error). The preflight is
//     best-effort and must not block plans when the catalog API is unreachable;
//     Create still resolves authoritatively on apply.
func validateAppTitleName(ctx context.Context, resolver titleResolver, value types.String, attrPath path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if resolver == nil || !helpers.IsConfiguredValue(value) {
		return diags
	}
	name := value.ValueString()
	if name == "" {
		return diags
	}

	_, err := resolver.ResolveAppInstallerTitleV1IDByName(ctx, name)
	if err == nil {
		return diags
	}
	if helpers.IsNotFoundError(err) {
		diags.AddAttributeError(
			attrPath,
			"Unknown App Installer title",
			fmt.Sprintf("No App Catalog title named %q exists on this tenant. List the available titles with the jamfplatform_pro_app_installer_titles data source.", name),
		)
		return diags
	}
	diags.AddAttributeWarning(
		attrPath,
		"Could not verify App Installer title",
		fmt.Sprintf("Skipping plan-time title validation for %q: %s. The title is still resolved against the catalog on apply.", name, err),
	)
	return diags
}

// resolveAppTitleID resolves an App Catalog title name to its ID. Used on apply
// (Create/Update) where resolution is authoritative — a failure aborts the
// operation with a clear diagnostic.
func resolveAppTitleID(ctx context.Context, resolver titleResolver, name string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	id, err := resolver.ResolveAppInstallerTitleV1IDByName(ctx, name)
	if err != nil {
		if helpers.IsNotFoundError(err) {
			diags.AddAttributeError(
				path.Root("app_title_name"),
				"Unknown App Installer title",
				fmt.Sprintf("No App Catalog title named %q exists on this tenant. List the available titles with the jamfplatform_pro_app_installer_titles data source.", name),
			)
			return "", diags
		}
		diags.AddAttributeError(
			path.Root("app_title_name"),
			"Unable to resolve App Installer title",
			fmt.Sprintf("Resolving App Catalog title %q failed: %s", name, err),
		)
		return "", diags
	}
	return id, diags
}

// titleNameForID reverse-resolves a title ID to its display name for Read/import
// (the deployment GET returns only app_title_id). Best-effort: on failure it
// returns ok=false so the caller can preserve any existing state value rather
// than fail a refresh over a transient catalog error.
//
// This is safe from a perpetual diff because ResolveAppInstallerTitleV1IDByName
// (used to resolve the configured app_title_name on apply) is an EXACT,
// case-sensitive match: an off-casing name 404s at plan time rather than
// resolving and then being rewritten here to the canonical TitleName. So the
// stored app_title_name always equals the canonical name this returns.
func titleNameForID(ctx context.Context, namer titleNamer, id string) (string, bool) {
	if namer == nil || id == "" {
		return "", false
	}
	title, err := namer.GetAppInstallerTitleV1(ctx, id)
	if err != nil || title == nil {
		return "", false
	}
	return title.TitleName, true
}
