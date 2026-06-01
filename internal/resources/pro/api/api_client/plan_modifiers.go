// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// shouldRotateCredentials reports whether an Update should mint a fresh
// client secret. Rotation happens only when `credential_rotation` carries a
// new, non-null value relative to prior state:
//   - null / unknown plan trigger: never rotate (leave the stored secret alone,
//     including when the user removes the trigger).
//   - state had no trigger and the plan now sets one: rotate.
//   - the trigger value changed: rotate.
func shouldRotateCredentials(planTrigger, stateTrigger types.String) bool {
	if planTrigger.IsNull() || planTrigger.IsUnknown() {
		return false
	}
	if stateTrigger.IsNull() || stateTrigger.IsUnknown() {
		return true
	}
	return planTrigger.ValueString() != stateTrigger.ValueString()
}
