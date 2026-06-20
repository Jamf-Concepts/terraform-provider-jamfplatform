// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer_settings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildAppInstallerSettingsInput_NilBlocks verifies that a plan with no blocks
// produces a payload with both pointers nil (full-replace clear).
func TestBuildAppInstallerSettingsInput_NilBlocks(t *testing.T) {
	plan := AppInstallerSettingsResourceModel{}
	got := buildAppInstallerSettingsInput(plan)
	if got.DeploymentProcessControls != nil {
		t.Errorf("expected nil DeploymentProcessControls, got %+v", got.DeploymentProcessControls)
	}
	if got.EndUserExperienceSettings != nil {
		t.Errorf("expected nil EndUserExperienceSettings, got %+v", got.EndUserExperienceSettings)
	}
}

// TestBuildDeploymentSettings_NilFields verifies that null TF values produce
// nil SDK pointers, not zero-value integers.
func TestBuildDeploymentSettings_NilFields(t *testing.T) {
	m := &DeploymentSettingsModel{
		BatchSize:      types.Int64Null(),
		BatchFrequency: types.Int64Null(),
		Days:           types.SetNull(types.StringType),
		ServerTimeFrom: types.StringNull(),
		ServerTimeTo:   types.StringNull(),
	}
	got := buildDeploymentSettings(m)
	if got == nil {
		t.Fatal("expected non-nil struct for non-nil model")
	}
	if got.CommandsBatchSize != nil {
		t.Errorf("null BatchSize should produce nil pointer, got %d", *got.CommandsBatchSize)
	}
	if got.DaysOfWeek != nil {
		t.Errorf("null Days should produce nil pointer")
	}
}

// TestBuildDeploymentSettings_Populated verifies round-trip for all fields.
func TestBuildDeploymentSettings_Populated(t *testing.T) {
	days := types.SetValueMust(types.StringType, []attr.Value{
		types.StringValue("MONDAY"),
		types.StringValue("FRIDAY"),
	})

	m := &DeploymentSettingsModel{
		BatchSize:      types.Int64Value(500),
		BatchFrequency: types.Int64Value(120),
		Days:           days,
		ServerTimeFrom: types.StringValue("09:00:00Z"),
		ServerTimeTo:   types.StringValue("18:00:00Z"),
	}
	got := buildDeploymentSettings(m)
	if got == nil {
		t.Fatal("expected non-nil struct")
	}
	if *got.CommandsBatchSize != 500 {
		t.Errorf("CommandsBatchSize = %d, want 500", *got.CommandsBatchSize)
	}
	if *got.BatchFrequencyInMinutes != 120 {
		t.Errorf("BatchFrequencyInMinutes = %d, want 120", *got.BatchFrequencyInMinutes)
	}
	if *got.FromTimeOfDay != "09:00:00Z" {
		t.Errorf("FromTimeOfDay = %q, want 09:00:00Z", *got.FromTimeOfDay)
	}
	if got.DaysOfWeek == nil || len(*got.DaysOfWeek) != 2 {
		t.Errorf("DaysOfWeek: expected 2 elements")
	}
}

// TestBuildDays_NullVsEmpty verifies the nil/empty distinction is preserved.
func TestBuildDays_NullVsEmpty(t *testing.T) {
	nullResult := buildDays(types.SetNull(types.StringType))
	if nullResult != nil {
		t.Error("null Set should produce nil pointer")
	}

	emptyResult := buildDays(types.SetValueMust(types.StringType, nil))
	if emptyResult == nil {
		t.Fatal("empty Set should produce non-nil empty slice")
	}
	if len(*emptyResult) != 0 {
		t.Errorf("empty Set should produce empty slice, got len %d", len(*emptyResult))
	}
}

// TestBuildEndUserExperience_NilFields verifies null TF values produce nil pointers.
func TestBuildEndUserExperience_NilFields(t *testing.T) {
	m := &EndUserExperienceModel{
		NotificationFrequency: types.Int64Null(),
		NotificationMessage:   types.StringNull(),
		UpdateDeadline:        types.Int64Null(),
		ForceQuitMessage:      types.StringNull(),
		ForceQuitGracePeriod:  types.Int64Null(),
		UpdateCompleteMessage: types.StringNull(),
		Relaunch:              types.BoolNull(),
		Suppress:              types.BoolNull(),
	}
	got := buildEndUserExperience(m)
	if got == nil {
		t.Fatal("expected non-nil struct for non-nil model")
	}
	if got.NotificationInterval != nil {
		t.Error("null NotificationFrequency should produce nil pointer")
	}
	if got.Relaunch != nil {
		t.Error("null Relaunch should produce nil pointer")
	}
}
