// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Wire enum values for deployment_type and update_behavior. The server enums are
// stable, so these back static OneOf validators.
const (
	deploymentTypeInstallAutomatically = "INSTALL_AUTOMATICALLY"
	deploymentTypeSelfService          = "SELF_SERVICE"

	updateBehaviorAutomatic = "AUTOMATIC"
	updateBehaviorManual    = "MANUAL"
)

// boolPointerOrFalse returns the bool value, treating null/unknown as false.
// Used for the Self Service booleans, which the server defaults to false and so
// are always emitted.
func boolPointerOrFalse(value types.Bool) *bool {
	v := value.ValueBool()
	if !helpers.IsConfiguredValue(value) {
		v = false
	}
	return &v
}

// optionalIntPointer collapses null/unknown TF Int64 to nil so an unset field is
// dropped from the wire payload (the server rejects a non-positive value on any
// notification interval/delay that is present).
func optionalIntPointer(value types.Int64) *int {
	if !helpers.IsConfiguredValue(value) {
		return nil
	}
	v := int(value.ValueInt64())
	return &v
}
