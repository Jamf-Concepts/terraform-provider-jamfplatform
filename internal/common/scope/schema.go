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
func IDSetAttribute(attrLabel string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		MarkdownDescription: fmt.Sprintf("Set of Jamf Pro %s IDs.", attrLabel),
	}
}

// NameSetAttribute returns the canonical Set<String> schema for name-only
// classic-API scope target categories. Used for:
// directory_service_or_local_user_names, directory_service_user_group_names,
// limit_to_user_group_names.
func NameSetAttribute(attrLabel string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		MarkdownDescription: fmt.Sprintf("Set of %s names.", attrLabel),
	}
}
