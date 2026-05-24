// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   pro.UploadIconV1
//   pro.GetIconV1
//   pro.DownloadIconV1
//
// Status: current. Last reviewed 2026-05-24.

package icon

import (
	"bytes"
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/files"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create uploads an icon to Jamf Pro and stores the resulting ID and URL in
// state. source_hash is already set on the plan by ModifyPlan, so Create
// does not need to read bytes a second time — it just opens the source and
// streams it to UploadIconV1.
func (r *IconResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IconResourceModel
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

	file, filename, cleanup, err := files.OpenUploadSource(createCtx, plan.IconFileSource.ValueString(), files.DefaultMaxBytes)
	if err != nil {
		resp.Diagnostics.AddError("Error opening icon source", err.Error())
		return
	}
	defer cleanup()

	iconResp, err := r.client.UploadIconV1(createCtx, filename, file)
	if err != nil {
		resp.Diagnostics.AddError("Error uploading Jamf Pro icon", err.Error())
		return
	}

	assignIconResourceModel(&plan, iconResp)
	// plan.SourceHash was set by ModifyPlan — do not overwrite.

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, iconIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created Jamf Pro icon", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state with the latest icon representation.
//
// Reading state:
//   - Normal refresh: load state from req.State.
//   - Import: req.State.Raw may be null, in which case the resource ID is
//     read from req.Identity. We do not rely on Raw.IsNull() alone — the
//     Plugin Framework's import flow can populate req.State with the ID
//     before calling Read, leaving Raw non-null but other attributes null.
//
// After loading, GetIconV1 refreshes the URL. We then ensure source_hash is
// populated: if state lacks one (post-import, post-corruption, or any case
// where Create's plan-time hash didn't make it into persisted state), we
// download the icon bytes and compute the canonical "sha256:<hex>" value.
// This makes the resource self-healing across import flows that strip
// Computed attrs from the persisted state.
func (r *IconResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IconResourceModel

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// If state didn't supply an ID, fall back to identity (import path).
	if state.ID.IsNull() || state.ID.IsUnknown() || state.ID.ValueString() == "" {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this icon without existing state or identity data, so the provider cannot determine which icon to read.",
			)
			return
		}
		var identity iconIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing icon ID",
				"The resource identity did not include an 'id' attribute, so the provider cannot refresh the icon.",
			)
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(iconTimeoutAttributeTypes)
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultReadTimeout, state.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro icon without ID.")
		return
	}

	iconResp, err := r.client.GetIconV1(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro icon not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro icon", err.Error())
		return
	}

	assignIconResourceModel(&state, iconResp)

	// Populate source_hash when missing. This fires on:
	//   - First Read after import (state has id but no source_hash).
	//   - Recovery from state corruption (source_hash got cleared somehow).
	// On normal refresh the hash is already set and we skip the download.
	if state.SourceHash.IsNull() || state.SourceHash.IsUnknown() || state.SourceHash.ValueString() == "" {
		data, dlErr := downloadIconBytes(readCtx, r.client, state.ID.ValueString(), iconResp.URL)
		if dlErr != nil {
			tflog.Warn(ctx, "icon Read: could not compute source_hash from server", map[string]any{
				"id":  state.ID.ValueString(),
				"err": dlErr.Error(),
			})
			state.SourceHash = types.StringNull()
		} else {
			state.SourceHash = types.StringValue(computeSourceHashString(data))
			tflog.Info(ctx, "icon Read populated source_hash from server bytes", map[string]any{
				"id":          state.ID.ValueString(),
				"source_hash": state.SourceHash.ValueString(),
			})
		}
		// icon_file_source is not server-derived; it cannot be inferred
		// from a refresh. Leave whatever state had (typically null on
		// import; a user path on subsequent refreshes).
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, iconIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a refresh-only no-op. Every "content change" routes through
// replacement via ModifyPlan's RequiresReplace, because Jamf Pro has no
// icon update endpoint. Update only runs for in-place config diffs that do
// NOT change source_hash — for example, after import when the user assigns
// icon_file_source to a local path whose bytes match the imported hash.
func (r *IconResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan IconResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	got, err := r.client.GetIconV1(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Jamf Pro icon", err.Error())
		return
	}
	assignIconResourceModel(&plan, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, iconIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Pro icon from Terraform state.
//
// No DeleteIconV1 endpoint exists. The icon record persists on the tenant;
// this operation removes it from Terraform state only.
func (r *IconResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IconResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Warn(ctx, "jamfplatform_pro_icon delete: no DeleteIconV1 API; icon removed from state only",
		map[string]any{"id": state.ID.ValueString()})
}

// downloadIconBytes fetches the raw bytes of a Jamf Pro icon. It tries
// iconURL (CDN download via files.DownloadCapped) first; if that fails or
// the URL is empty it falls back to DownloadIconV1 (/v1/icon/download/{id}),
// which may return HTTP 500 on some tenants.
func downloadIconBytes(ctx context.Context, client *pro.Client, id, iconURL string) ([]byte, error) {
	if iconURL != "" {
		var buf bytes.Buffer
		if _, _, err := files.DownloadCapped(ctx, iconURL, &buf, files.DefaultMaxBytes); err == nil {
			return buf.Bytes(), nil
		}
	}
	return client.DownloadIconV1(ctx, id, 0, "")
}
