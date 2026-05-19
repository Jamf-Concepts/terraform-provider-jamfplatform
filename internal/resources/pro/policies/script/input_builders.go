// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package script

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildScriptInput converts the Terraform plan model into an SDK Script payload.
// Null or Unknown plan values become omitted (nil) fields — Jamf Pro rejects an
// explicit empty string on `categoryId` and a wired-in `categoryId: ""` on an
// Optional+Computed attribute would surface as HTTP 500. helpers.OptionalStringPointer
// nils both Null *and* Unknown; see STYLE_GUIDE.md §Server-derived computed fields.
func buildScriptInput(plan ScriptResourceModel) *pro.Script {
	return &pro.Script{
		Name:           plan.Name.ValueString(),
		CategoryID:     helpers.OptionalStringPointer(plan.CategoryID),
		Info:           helpers.OptionalStringPointer(plan.Info),
		Notes:          helpers.OptionalStringPointer(plan.Notes),
		OsRequirements: helpers.OptionalStringPointer(plan.OsRequirements),
		Priority:       helpers.OptionalStringPointer(plan.Priority),
		Parameter4:     helpers.OptionalStringPointer(plan.Parameter4),
		Parameter5:     helpers.OptionalStringPointer(plan.Parameter5),
		Parameter6:     helpers.OptionalStringPointer(plan.Parameter6),
		Parameter7:     helpers.OptionalStringPointer(plan.Parameter7),
		Parameter8:     helpers.OptionalStringPointer(plan.Parameter8),
		Parameter9:     helpers.OptionalStringPointer(plan.Parameter9),
		Parameter10:    helpers.OptionalStringPointer(plan.Parameter10),
		Parameter11:    helpers.OptionalStringPointer(plan.Parameter11),
		ScriptContents: helpers.OptionalStringPointer(plan.ScriptContents),
	}
}
