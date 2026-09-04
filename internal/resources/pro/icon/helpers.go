// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package icon

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
