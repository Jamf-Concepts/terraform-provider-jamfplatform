// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package appinstalleractions implements the fire-once Jamf Pro App Installer
// actions: jamfplatform_pro_retry_app_installer_installations (retry the failed
// installations of one deployment),
// jamfplatform_pro_retry_all_app_installer_installations (retry every failed
// App Installer installation in the tenant) and
// jamfplatform_pro_update_app_installer_version (move a MANUAL deployment to a
// newer version of its title).
//
// The retry pair is deliberately two actions rather than one with an optional
// deployment ID. Omitting an attribute would make the tenant-wide retry the
// accident you get by forgetting the deployment, and unlike the patch-policy
// retry this action is modelled on, the unbounded form is not scoped by a named
// object. The breadth is in the name instead.
package appinstalleractions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is empty: the App Installer endpoints predate the provider's
// overall floor.
const minJamfProVersion = ""

// appInstallerAction shares Configure logic across the App Installer actions.
type appInstallerAction struct {
	client *pro.Client
}

// configure binds the provider-supplied Jamf Pro client to the action via the shared
// providerdata.ConfigurePro helper.
func (a *appInstallerAction) configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "app_installers")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	a.client = client
}

// ensureClient guarantees Configure completed successfully before Invoke.
func (a *appInstallerAction) ensureClient(resp *action.InvokeResponse) bool {
	if a.client != nil {
		return true
	}

	resp.Diagnostics.AddError(
		"Provider Not Configured",
		"The Jamf Pro client was not configured. Re-run terraform init/apply so the provider can configure successfully.",
	)
	return false
}

// nothingToRetryDiagnostic explains a 404 from either retry endpoint.
//
// Wire-verified 2026-09-03: both answer `404` with an EMPTY `errors` array when
// there is nothing to retry, and the same empty 404 for a deployment that does
// not exist — the bodies are byte-identical, so the two cases cannot be told
// apart from here. Jamf Pro's unrouted tell in this namespace is
// `403 BAD_PERMISSIONS`, so a 404 is a routed request refused on state.
//
// It is a warning, not an error. A retry wired to a lifecycle action_trigger
// fires after every apply, and a healthy fleet with no failed installations is
// the normal case — failing the apply for it would break the workspace whenever
// nothing is wrong. The warning names both possible causes so the outcome is
// never silent.
func nothingToRetryDiagnostic(subject string) (string, string) {
	return "No Failed Installations Retried",
		"Jamf Pro answered 404 for " + subject + ". Either there are no failed installations to retry, " +
			"or the deployment does not exist — Jamf Pro returns the same empty response for both, so this " +
			"action cannot distinguish them. Nothing was retried. Check the deployment ID if you expected failures to be retried."
}
