// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package icon

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/files"
)

// hashLocalIconSource reads a local icon file and returns the canonical hash of
// its bytes. Only local paths reach it: a URL is read during apply, where the
// bytes that are hashed are the bytes that are uploaded.
func hashLocalIconSource(ctx context.Context, src string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	file, _, cleanup, err := files.OpenUploadSource(ctx, src, files.DefaultMaxBytes)
	if err != nil {
		diags.AddError("Error opening icon source during plan", err.Error())
		return "", diags
	}
	defer cleanup()

	data, err := io.ReadAll(file)
	if err != nil {
		diags.AddError("Error reading icon source during plan", err.Error())
		return "", diags
	}

	return files.ComputeContentSHA256(data), diags
}

// planIconReplacement plans a replacement of the icon, leaving every value the
// upload assigns unresolved so apply supplies it. trigger names the attribute
// Terraform reports as forcing the replacement, which is the changed URL where
// the source string moved and source_hash where the local bytes did.
//
// source_hash goes Unknown rather than carrying the hash computed at plan time,
// so that the value committed to state is always one Create read off the bytes
// it uploaded. Attribute plan modifiers run before this, so the Unknown wins
// over UseStateForUnknown.
func planIconReplacement(ctx context.Context, resp *resource.ModifyPlanResponse, plan *IconResourceModel, trigger path.Path) {
	plan.ID = types.StringUnknown()
	plan.URL = types.StringUnknown()
	plan.SourceHash = types.StringUnknown()
	resp.RequiresReplace = append(resp.RequiresReplace, trigger)
	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}
