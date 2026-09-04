// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// validateAppTitleName is a plan-time preflight for app_title_name. The title is
// referenced by its catalog display name; this resolves it against the live
// App Catalog so an unknown name surfaces at plan time with a clear message
// instead of the apply-time resolution failure.
//
// Behaviour:
//   - nil resolver, or null/unknown/empty value: no-op (nothing to check, or the
//     provider is not yet configured, or the name derives from another resource).
//   - name resolves: OK.
//   - name not in the catalog under that exact spelling: an error diagnostic.
//   - any other resolve error, including a failed catalog read: a WARNING (not an
//     error). The preflight is best-effort and must not block plans when the
//     catalog API is unreachable; Create still resolves authoritatively on apply.
func validateAppTitleName(ctx context.Context, resolver titleCatalog, value types.String, attrPath path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if resolver == nil || !helpers.IsConfiguredValue(value) {
		return diags
	}
	name := value.ValueString()
	if name == "" {
		return diags
	}

	_, err := resolveTitleIDByName(ctx, resolver, name)
	if err == nil {
		return diags
	}
	if errors.Is(err, errTitleNotInCatalog) || helpers.IsNotFoundError(err) {
		diags.AddAttributeError(
			attrPath,
			"Unknown App Installer title",
			fmt.Sprintf("No App Catalog title is named exactly %q on this tenant. Names are matched exactly, including capitalisation. List the available titles with the jamfplatform_pro_app_installer_titles data source.", name),
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
func resolveAppTitleID(ctx context.Context, resolver titleCatalog, name string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	id, err := resolveTitleIDByName(ctx, resolver, name)
	if err != nil {
		if errors.Is(err, errTitleNotInCatalog) || helpers.IsNotFoundError(err) {
			diags.AddAttributeError(
				path.Root("app_title_name"),
				"Unknown App Installer title",
				fmt.Sprintf("No App Catalog title is named exactly %q on this tenant. Names are matched exactly, including capitalisation. List the available titles with the jamfplatform_pro_app_installer_titles data source.", name),
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
// This is safe from a perpetual diff because resolveTitleIDByName (used to
// resolve the configured app_title_name on apply) decides the match on byte
// equality: an off-casing name fails at plan time rather than resolving and then
// being rewritten here to the canonical TitleName. So the stored app_title_name
// always equals the canonical name this returns.
//
// It answers from the same cached catalog snapshot the forward resolver reads,
// rather than a per-deployment title GET: the list carries id and titleName (the
// wire shape is id, bundleId, titleName, publisher, iconUrl, version and
// installationPathShared), which is everything this needs, so a refresh of N
// deployments costs no catalog requests beyond the one snapshot.
//
// An id the snapshot does not hold returns ok=false, the same as a failed read: a
// title withdrawn from the catalog cannot be named, and the caller already treats
// that as "keep what state has" outside import.
func titleNameForID(ctx context.Context, catalog titleCatalog, id string) (string, bool) {
	if catalog == nil || id == "" {
		return "", false
	}
	titles, err := catalog.Titles(ctx)
	if err != nil {
		return "", false
	}
	for _, t := range titles {
		if t.ID == id {
			return t.TitleName, true
		}
	}
	return "", false
}
