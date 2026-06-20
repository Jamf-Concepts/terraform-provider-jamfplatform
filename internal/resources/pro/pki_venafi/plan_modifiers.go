// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_venafi

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// shouldRotate reports whether an Int64 rotation trigger carries a new value
// relative to prior state, signalling the associated server-side rotation
// (regenerate the Jamf public key, or re-send the refresh token):
//   - null / unknown plan trigger: never rotate (leave the stored value alone,
//     including when the user removes the trigger).
//   - state had no trigger and the plan now sets one: rotate.
//   - the trigger value changed: rotate.
func shouldRotate(planTrigger, stateTrigger types.Int64) bool {
	if planTrigger.IsNull() || planTrigger.IsUnknown() {
		return false
	}
	if stateTrigger.IsNull() || stateTrigger.IsUnknown() {
		return true
	}
	return planTrigger.ValueInt64() != stateTrigger.ValueInt64()
}
