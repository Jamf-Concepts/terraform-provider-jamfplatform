// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_identity_provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// createAzure provisions a Microsoft Entra ID Cloud Identity Provider:
// POST /v1/cloud-azure then GET to capture server-populated fields.
// The cfg arg is unused (no WriteOnly fields on Azure).
func (r *CloudIdentityProviderResource) createAzure(ctx context.Context, plan, _ CloudIdentityProviderResourceModel, resp *resource.CreateResponse) {
	if plan.Azure == nil {
		resp.Diagnostics.AddError(missingProviderBlockError(providerEntraID, "entra_id"))
		return
	}
	body := buildAzureCreateRequest(plan)

	created, err := r.client.CreateCloudAzureV1(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro Cloud Identity Provider (Azure)", err.Error())
		return
	}
	if created == nil || created.ID == "" {
		resp.Diagnostics.AddError(
			"Create response missing Cloud Identity Provider ID",
			"Jamf Pro returned 201 Created with no id; cannot persist state.",
		)
		return
	}

	// Pin the id from the create response so it survives even if the read-back
	// returns a response without cloudIdPCommon (assignAzureState only sets id
	// when that block is present). Mirrors the codebase create→GET convention.
	plan.ID = types.StringValue(created.ID)

	got, err := r.client.GetCloudAzureV1(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro Cloud Identity Provider (Azure)", err.Error())
		return
	}
	assignAzureState(&plan, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, cloudIdentityProviderIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "created Jamf Pro Cloud Identity Provider (Azure)", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// readAzure refreshes state from GET /v1/cloud-azure/{id}.
func (r *CloudIdentityProviderResource) readAzure(ctx context.Context, state *CloudIdentityProviderResourceModel, resp *resource.ReadResponse) {
	got, err := r.client.GetCloudAzureV1(ctx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro Cloud Identity Provider (Azure) not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro Cloud Identity Provider (Azure)", err.Error())
		return
	}

	assignAzureState(state, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, cloudIdentityProviderIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// updateAzure full-replaces via PUT /v1/cloud-azure/{id}, then GET.
// The state and cfg args are unused for Azure.
func (r *CloudIdentityProviderResource) updateAzure(ctx context.Context, plan, _, _ CloudIdentityProviderResourceModel, resp *resource.UpdateResponse) {
	if plan.Azure == nil {
		resp.Diagnostics.AddError(missingProviderBlockError(providerEntraID, "entra_id"))
		return
	}
	body := buildAzureUpdateRequest(plan)

	if _, err := r.client.UpdateCloudAzureV1(ctx, plan.ID.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro Cloud Identity Provider (Azure)", err.Error())
		return
	}

	got, err := r.client.GetCloudAzureV1(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro Cloud Identity Provider (Azure)", err.Error())
		return
	}
	assignAzureState(&plan, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, cloudIdentityProviderIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// deleteAzure removes the configuration via DELETE /v1/cloud-azure/{id}.
func (r *CloudIdentityProviderResource) deleteAzure(ctx context.Context, state CloudIdentityProviderResourceModel, resp *resource.DeleteResponse) {
	if err := r.client.DeleteCloudAzureV1(ctx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro Cloud Identity Provider (Azure) already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro Cloud Identity Provider (Azure)", err.Error())
	}
}
