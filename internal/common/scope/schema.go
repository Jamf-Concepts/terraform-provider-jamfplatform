// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/planmodifiers"
)

// IDSetAttribute returns the canonical Set<String> schema for ID-bearing
// classic-API scope target categories. Used for: computers, computer_groups,
// mobile_devices, mobile_device_groups, buildings, departments, jss_users,
// jss_user_groups, classes, network_segments, ibeacons.
//
// attrLabel is interpolated into the MarkdownDescription as the human-readable
// singular form (e.g. "computer", "computer group", "building").
//
// These sets are Optional+Computed and carry the CanonicalEmptySet plan
// modifier. The canonical "no members" value is an empty set `[]`: the read
// path flattens an empty wire result to `[]` (EmptyStringSet), an explicit `[]`
// config plans and applies as `[]`, and an OMITTED attribute (null config) is
// planned as `[]` too. So a category is cleared either by omitting the
// attribute or by assigning `[]` — both clear the targets on the wire (the
// build path omits the empty wrapper). Computed is required so the modifier may
// set a value when the config is null; canonicalising on `[]` (not null) is
// forced by Terraform requiring a known non-null config (`[]`) to equal its
// plan.
func IDSetAttribute(attrLabel string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Set{planmodifiers.CanonicalEmptySet()},
		MarkdownDescription: fmt.Sprintf("Set of Jamf Pro %s IDs.", attrLabel),
	}
}

// NameSetAttribute returns the canonical Set<String> schema for name-only
// classic-API scope target categories. Used for:
// directory_service_or_local_user_names, directory_service_user_group_names,
// limit_to_user_group_names. Clear by omitting (null) or by assigning `[]` —
// see IDSetAttribute.
func NameSetAttribute(attrLabel string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Set{planmodifiers.CanonicalEmptySet()},
		MarkdownDescription: fmt.Sprintf("Set of %s names.", attrLabel),
	}
}
