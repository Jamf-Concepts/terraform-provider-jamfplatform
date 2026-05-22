// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ibeacon

import "fmt"

// validateIbeaconPlan enforces the include_any_*_value vs major/minor
// cross-field rules independently per axis:
//   - include_any_major_value = true  ⇒ major must be null
//   - include_any_major_value = false ⇒ major must be set
//   - include_any_minor_value = true  ⇒ minor must be null
//   - include_any_minor_value = false ⇒ minor must be set
//
// Apply-time helper, run from Create and Update — defence-in-depth alongside
// the plan-time includeAnyMajorMinorConfigValidator (which catches the same
// shape errors at plan time but cannot reason about values that only become
// known during apply).
func validateIbeaconPlan(plan *IbeaconResourceModel) error {
	includeAnyMajor := !plan.IncludeAnyMajorValue.IsNull() &&
		!plan.IncludeAnyMajorValue.IsUnknown() &&
		plan.IncludeAnyMajorValue.ValueBool()
	includeAnyMinor := !plan.IncludeAnyMinorValue.IsNull() &&
		!plan.IncludeAnyMinorValue.IsUnknown() &&
		plan.IncludeAnyMinorValue.ValueBool()

	majorSet := !plan.Major.IsNull() && !plan.Major.IsUnknown()
	minorSet := !plan.Minor.IsNull() && !plan.Minor.IsUnknown()

	if includeAnyMajor && majorSet {
		return fmt.Errorf("major must not be set when include_any_major_value = true")
	}
	if includeAnyMinor && minorSet {
		return fmt.Errorf("minor must not be set when include_any_minor_value = true")
	}
	if !includeAnyMajor && !majorSet {
		return fmt.Errorf("major must be set when include_any_major_value is unset or false; or set include_any_major_value = true to match any major value")
	}
	if !includeAnyMinor && !minorSet {
		return fmt.Errorf("minor must be set when include_any_minor_value is unset or false; or set include_any_minor_value = true to match any minor value")
	}
	return nil
}

// derefString returns the underlying string for a non-nil *string, or "" for
// nil. Used by the list resource for display-name rendering and by the
// classic-filter name accessor.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
