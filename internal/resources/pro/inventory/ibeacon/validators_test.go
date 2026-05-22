// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ibeacon

import (
	"context"
	"testing"
)

// TestValidator_Descriptions confirms the ConfigValidator implements the
// Description / MarkdownDescription contract (framework requires both). The
// validator's value-based logic is exercised end-to-end by the acceptance
// tests in resource_acceptance_test.go (a plan-time check needs a live tfsdk
// Config to apply against; constructing one in unit tests with the right
// underlying tftypes value is brittle and adds no signal beyond the
// apply-time validateIbeaconPlan helper, which has full unit coverage in
// helpers_test.go). See user_group/ for the same pattern.
func TestValidator_Descriptions(t *testing.T) {
	v := includeAnyMajorMinorConfigValidator{}
	if v.Description(context.Background()) == "" {
		t.Errorf("Description must not be empty")
	}
	if v.MarkdownDescription(context.Background()) == "" {
		t.Errorf("MarkdownDescription must not be empty")
	}
}
