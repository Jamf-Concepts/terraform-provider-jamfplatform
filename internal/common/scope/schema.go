// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IDSetAttribute returns the canonical Set<String> schema for ID-bearing
// classic-API scope target categories. Used for: computers, computer_groups,
// mobile_devices, mobile_device_groups, buildings, departments, jss_users,
// jss_user_groups, classes, network_segments, ibeacons.
//
// attrLabel is interpolated into the MarkdownDescription as the human-readable
// singular form (e.g. "computer", "computer group", "building").
//
// These sets are Optional-only — per-category granular ownership (wire-probed,
// see STYLE_GUIDE.md §Scope helper omission semantics):
//
//   - OMITTED (null): the category is not managed by Terraform. It never
//     enters state, and on update the input builder re-emits the live
//     server members for it (read-merge-write), so values maintained in the
//     admin UI are preserved.
//   - DECLARED (including `[]`): Terraform owns the category. `[]` clears it;
//     members drift-revert on refresh.
//
// The null/empty distinction is load-bearing, which is why these attributes
// must not be Computed: a Computed attribute cannot keep a null config null
// once the server echoes a value.
func IDSetAttribute(attrLabel string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		MarkdownDescription: fmt.Sprintf("Set of Jamf Pro %s IDs. Omit to leave this category as configured outside Terraform; set `[]` to clear it.", attrLabel),
	}
}

// NameSetAttribute returns the canonical Set<String> schema for name-only
// classic-API scope target categories. Used for:
// directory_service_or_local_user_names, directory_service_user_group_names,
// limit_to_user_group_names. Same granular-ownership semantics as
// IDSetAttribute: omit = not managed (preserved on the wire), `[]` = clear.
func NameSetAttribute(attrLabel string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		MarkdownDescription: fmt.Sprintf("Set of %s names. Omit to leave this category as configured outside Terraform; set `[]` to clear it.", attrLabel),
	}
}
