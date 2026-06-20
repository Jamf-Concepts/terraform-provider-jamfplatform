// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enrollment_customization

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/files"
)

// ModifyPlan implements provider-driven change detection for the local icon
// source. When `icon_source` is set the provider opens the source, reads the
// bytes, computes a SHA-256, and either:
//
//   - on Create, sets the computed `icon_source_hash` on the plan;
//   - on Update with a matching hash, leaves the plan untouched (no upload);
//   - on Update with a differing hash, marks `icon_source_hash` and the
//     branding `icon_url` Unknown so the framework re-derives them on apply.
//
// The destroy path returns early before touching the source.
func (r *EnrollmentCustomizationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan EnrollmentCustomizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.IconSource.IsNull() || plan.IconSource.IsUnknown() || plan.IconSource.ValueString() == "" {
		return
	}

	file, _, cleanup, openErr := files.OpenUploadSource(ctx, plan.IconSource.ValueString(), files.DefaultMaxBytes)
	if openErr != nil {
		resp.Diagnostics.AddError("Error opening enrollment customization icon source during plan", openErr.Error())
		return
	}
	defer cleanup()

	data, readErr := io.ReadAll(file)
	if readErr != nil {
		resp.Diagnostics.AddError("Error reading enrollment customization icon source during plan", readErr.Error())
		return
	}
	newHash := files.ComputeContentSHA256(data)

	// Create case — set computed icon_source_hash on the plan; leave icon_url
	// Unknown so Create can populate it from the upload response.
	if req.State.Raw.IsNull() {
		plan.IconSourceHash = types.StringValue(newHash)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("icon_source_hash"), plan.IconSourceHash)...)
		return
	}

	// Update case — compare against stored hash.
	var state EnrollmentCustomizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.IconSourceHash.ValueString() == newHash {
		return
	}

	// Hashes differ — re-upload on apply. Mark both Computed siblings Unknown
	// so the framework accepts the post-apply values.
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("icon_source_hash"), types.StringValue(newHash))...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("branding_settings").AtName("icon_url"), types.StringUnknown())...)
}
