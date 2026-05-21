// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package network_segment

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildNetworkSegmentInput converts the Terraform plan model into the SDK
// NetworkSegmentPost payload used for both Create and Update. The four server-
// derived fields (distribution_point, distribution_server, swu_server, url) are
// intentionally omitted — they are populated on read only. ID is omitted on
// write — Create uses path id="0" and Update derives it from state.
func buildNetworkSegmentInput(plan NetworkSegmentResourceModel) *proclassic.NetworkSegmentPost {
	return &proclassic.NetworkSegmentPost{
		Name:                helpers.OptionalStringPointer(plan.Name),
		StartingAddress:     helpers.OptionalStringPointer(plan.StartingAddress),
		EndingAddress:       helpers.OptionalStringPointer(plan.EndingAddress),
		Building:            helpers.OptionalStringPointer(plan.Building),
		Department:          helpers.OptionalStringPointer(plan.Department),
		OverrideBuildings:   optionalBoolPointer(plan.OverrideBuildings),
		OverrideDepartments: optionalBoolPointer(plan.OverrideDepartments),
	}
}

// optionalBoolPointer mirrors helpers.OptionalStringPointer for types.Bool. Returns
// nil for null/unknown so omitted Optional bools do not get serialised as `false`.
func optionalBoolPointer(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}
