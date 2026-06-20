// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   pro.CreateApiIntegrationV1
//   pro.GetApiIntegrationV1
//   pro.UpdateApiIntegrationV1
//   pro.DeleteApiIntegrationV1
//   pro.RotateApiIntegrationClientCredentialsV1   (client secret generation / rotation)
//   pro.ListApiIntegrationsV1                      (data source / list resource)
//
// Server invariants (wire-probed 2026-06-01):
//   - The client ID (clientId) is assigned at creation and stable for the life
//     of the client. app_type is NONE until a secret is generated, then
//     CLIENT_CREDENTIALS.
//   - Each rotation POST mints a brand-new secret and invalidates the previous
//     one; the secret is never readable via GET.
//   - Disabling the client (enabled=false) revokes its credentials (app_type
//     reverts to NONE). Rotation requires the client to be enabled.
//
// Status: current. Last reviewed 2026-06-01.

package api_client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create creates a new Jamf Pro API client, optionally generating its first
// client secret when credential_rotation is set.
func (r *ApiClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApiClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultCreateTimeout, plan.Timeouts.Create)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	input, diags := buildApiClientInput(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResp, err := r.client.CreateApiIntegrationV1(createCtx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro API client", err.Error())
		return
	}
	id := idString(createResp.ID)

	secret := types.StringNull()
	if rotate := !plan.CredentialRotation.IsNull() && !plan.CredentialRotation.IsUnknown(); rotate {
		if !createResp.Enabled {
			resp.Diagnostics.AddError(
				"Cannot generate client secret on a disabled API client",
				"credential_rotation was set but the API client is not enabled. Set enabled = true so Jamf Pro can mint the client secret.",
			)
			return
		}
		creds, rotErr := r.client.RotateApiIntegrationClientCredentialsV1(createCtx, id)
		if rotErr != nil {
			resp.Diagnostics.AddError("Error generating Jamf Pro API client secret", rotErr.Error())
			return
		}
		secret = types.StringValue(creds.ClientSecret)
	}

	got, err := r.client.GetApiIntegrationV1(createCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro API client", err.Error())
		return
	}
	resp.Diagnostics.Append(assignApiClientServerFields(ctx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ClientSecret = secret

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, apiClientIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created Jamf Pro API client", map[string]any{"id": id})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest API client representation.
func (r *ApiClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApiClientResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this API client without existing state or identity data, so the provider cannot determine which client to read.",
			)
			return
		}
		var identity apiClientIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing API client ID",
				"The resource identity did not include an 'id' attribute, so the provider cannot refresh the API client.",
			)
			return
		}
		state.ID = identity.ID
		state.ClientSecret = types.StringNull()
		state.CredentialRotation = types.StringNull()
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(apiClientTimeoutAttributeTypes)
	} else {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultReadTimeout, state.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro API client without ID.")
		return
	}

	got, err := r.client.GetApiIntegrationV1(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro API client not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, apiClientIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro API client", err.Error())
		return
	}

	priorSecret := state.ClientSecret
	resp.Diagnostics.Append(assignApiClientServerFields(ctx, &state, got)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ClientSecret = resolveStoredSecret(got.AppType, priorSecret)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, apiClientIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates a Jamf Pro API client. Update is a full-replace PUT; when
// credential_rotation carries a new value the client secret is re-minted.
func (r *ApiClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApiClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state ApiClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultUpdateTimeout, plan.Timeouts.Update)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	input, diags := buildApiClientInput(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.UpdateApiIntegrationV1(updateCtx, plan.ID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro API client", err.Error())
		return
	}

	rotated := false
	newSecret := types.StringNull()
	if shouldRotateCredentials(plan.CredentialRotation, state.CredentialRotation) {
		creds, rotErr := r.client.RotateApiIntegrationClientCredentialsV1(updateCtx, plan.ID.ValueString())
		if rotErr != nil {
			resp.Diagnostics.AddError("Error rotating Jamf Pro API client secret", rotErr.Error())
			return
		}
		newSecret = types.StringValue(creds.ClientSecret)
		rotated = true
	}

	got, err := r.client.GetApiIntegrationV1(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro API client", err.Error())
		return
	}
	resp.Diagnostics.Append(assignApiClientServerFields(ctx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if rotated {
		plan.ClientSecret = newSecret
	} else {
		plan.ClientSecret = resolveStoredSecret(got.AppType, state.ClientSecret)
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, apiClientIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Pro API client.
func (r *ApiClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApiClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultDeleteTimeout, state.Timeouts.Delete)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro API client without ID.")
		return
	}

	if err := r.client.DeleteApiIntegrationV1(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro API client already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro API client", fmt.Sprintf("API error: %v", err))
	}
}
