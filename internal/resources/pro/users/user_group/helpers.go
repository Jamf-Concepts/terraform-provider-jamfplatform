// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// validateUserGroupPlan enforces the smart/static cross-field rules:
//   - smart  ⇒ criteria required, members forbidden
//   - static ⇒ members may be set (or omitted to leave membership alone), criteria forbidden
//
// Mirrors device_group/helpers.go validateDeviceGroupPlan. We do not use
// boolvalidator.AlsoRequires/ConflictsWith because those validators fire on
// any value, not on a specific value of the discriminator — see STYLE_GUIDE
// §Cross-field validation.
func validateUserGroupPlan(plan *UserGroupResourceModel) error {
	switch plan.GroupType.ValueString() {
	case "smart":
		if len(plan.Criteria) == 0 {
			return fmt.Errorf("criteria must be supplied for smart user groups")
		}
		if helpers.IsConfiguredValue(plan.Members) {
			return fmt.Errorf("members cannot be set for smart user groups — smart-group membership is derived from criteria")
		}
	case "static":
		if len(plan.Criteria) > 0 {
			return fmt.Errorf("criteria cannot be set for static user groups")
		}
		if plan.Members.IsUnknown() {
			return fmt.Errorf("members cannot be unknown when provided for a static user group")
		}
	case "":
		return fmt.Errorf("group_type must be provided")
	default:
		return fmt.Errorf("unsupported group_type %q", plan.GroupType.ValueString())
	}
	return nil
}
