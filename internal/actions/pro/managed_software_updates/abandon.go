// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   pro.AbandonManagedSoftwareUpdateFeatureToggleV1   (POST /plans/feature-toggle/abandon — 204)
//
// Break-glass: the Managed Software Updates feature toggle applies asynchronously, and a
// background enable/disable process can stall. This action force-stops it. It is not part
// of nominal usage — reach for it only when the feature-toggle resource reports a settle
// timeout. No input, no state.

package managed_software_updates

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
)

var _ action.Action = (*AbandonFeatureToggleAction)(nil)
var _ action.ActionWithConfigure = (*AbandonFeatureToggleAction)(nil)

// AbandonFeatureToggleAction force-stops a stuck Managed Software Updates feature-toggle
// process.
type AbandonFeatureToggleAction struct {
	msuAction
}

// NewAbandonFeatureToggleAction constructs the action.
func NewAbandonFeatureToggleAction() action.Action {
	return &AbandonFeatureToggleAction{}
}

func (a *AbandonFeatureToggleAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_managed_software_update_abandon"
}

func (a *AbandonFeatureToggleAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Force-stops a stuck Managed Software Updates enable/disable process. " +
			"Break-glass only — use this when `jamfplatform_pro_managed_software_update` reports that the feature did not finish turning on or off. Takes no input.",
		Attributes: map[string]actionschema.Attribute{},
	}
}

func (a *AbandonFeatureToggleAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *AbandonFeatureToggleAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: "Abandoning any stuck Managed Software Updates feature-toggle process"})

	if err := a.client.AbandonManagedSoftwareUpdateFeatureToggleV1(ctx); err != nil {
		resp.Diagnostics.AddError(
			"Abandon Feature Toggle Failed",
			"Unable to force-stop the Managed Software Updates feature-toggle process: "+err.Error(),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: "Feature-toggle process abandoned"})
}
