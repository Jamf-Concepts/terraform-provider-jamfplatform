// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func idSet(t *testing.T, ids ...string) types.Set {
	t.Helper()
	vals := make([]types.String, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, types.StringValue(id))
	}
	out, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("idSet: %v", diags)
	}
	return out
}

// TestBuildPatchPolicyInput_GeneralWritableOnly verifies only the writable
// general fields are emitted; the server-derived fields are never sent, and the
// software_title_configuration_id lands in the body.
func TestBuildPatchPolicyInput_GeneralWritableOnly(t *testing.T) {
	plan := PatchPolicyResourceModel{
		SoftwareTitleConfigurationID: types.StringValue("42"),
		Name:                         types.StringValue("8x8 Work latest"),
		TargetVersion:                types.StringValue("8.33.2.2"),
		Enabled:                      types.BoolValue(true),
		DistributionMethod:           types.StringValue("selfservice"),
		AllowDowngrade:               types.BoolValue(false),
		PatchUnknown:                 types.BoolValue(true),
		// Server-derived fields populated in state must NOT influence the payload.
		ReleaseDate:       types.Int64Value(123),
		IncrementalUpdate: types.BoolValue(true),
		Reboot:            types.BoolValue(true),
		MinimumOS:         types.StringValue("12.0"),
	}

	out, diags := buildPatchPolicyInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("build diags: %v", diags)
	}

	if out.SoftwareTitleConfigurationID == nil || *out.SoftwareTitleConfigurationID != 42 {
		t.Errorf("software_title_configuration_id must be set in body to 42")
	}
	if out.General == nil {
		t.Fatal("general must be emitted")
	}
	g := out.General
	if g.Name == nil || *g.Name != "8x8 Work latest" {
		t.Errorf("name not emitted")
	}
	if g.TargetVersion == nil || *g.TargetVersion != "8.33.2.2" {
		t.Errorf("target_version not emitted")
	}
	if g.Enabled == nil || !*g.Enabled {
		t.Errorf("enabled not emitted")
	}
	if g.DistributionMethod == nil || *g.DistributionMethod != "selfservice" {
		t.Errorf("distribution_method not emitted")
	}
	if g.PatchUnknown == nil || !*g.PatchUnknown {
		t.Errorf("patch_unknown not emitted")
	}
	// Server-derived fields are never sent.
	if g.ReleaseDate != nil {
		t.Errorf("release_date must NOT be emitted (server-derived)")
	}
	if g.IncrementalUpdate != nil {
		t.Errorf("incremental_update must NOT be emitted (server-derived)")
	}
	if g.Reboot != nil {
		t.Errorf("reboot must NOT be emitted (server-derived)")
	}
	if g.MinimumOs != nil {
		t.Errorf("minimum_os must NOT be emitted (server-derived)")
	}
	if g.KillApps != nil {
		t.Errorf("kill_apps must NOT be emitted (server-derived)")
	}
}

// TestBuildScope_EntityByID verifies the limited target / limitations /
// exclusions set round-trips by ID into the SDK shape.
func TestBuildScope_EntityByID(t *testing.T) {
	plan := PatchPolicyResourceModel{
		SoftwareTitleConfigurationID: types.StringValue("1"),
		Name:                         types.StringValue("p"),
		TargetVersion:                types.StringValue("1.0"),
		Scope: &PatchPolicyScopeModel{
			ComputerIDs:      idSet(t, "10"),
			ComputerGroupIDs: idSet(t, "20", "21"),
			BuildingIDs:      idSet(t, "30"),
			DepartmentIDs:    idSet(t, "40"),
			Limitations: &PatchPolicyScopeLimitationsModel{
				NetworkSegmentIDs: idSet(t, "50"),
				IbeaconIDs:        idSet(t, "60"),
			},
			Exclusions: &PatchPolicyScopeExclusionsModel{
				ComputerIDs:       idSet(t, "70"),
				ComputerGroupIDs:  idSet(t, "71"),
				BuildingIDs:       idSet(t, "72"),
				DepartmentIDs:     idSet(t, "73"),
				NetworkSegmentIDs: idSet(t, "74"),
				IbeaconIDs:        idSet(t, "75"),
			},
		},
	}

	out, diags := buildPatchPolicyInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("build diags: %v", diags)
	}
	s := out.Scope
	if s == nil {
		t.Fatal("scope must be emitted")
	}
	if s.Computers == nil || len(*s.Computers.Computer) != 1 || *(*s.Computers.Computer)[0].ID != 10 {
		t.Errorf("computer target not built")
	}
	if s.ComputerGroups == nil || len(*s.ComputerGroups.ComputerGroup) != 2 {
		t.Errorf("computer_group targets not built")
	}
	if s.Buildings == nil || *(*s.Buildings.Building)[0].ID != 30 {
		t.Errorf("building target not built")
	}
	if s.Departments == nil || *(*s.Departments.Department)[0].ID != 40 {
		t.Errorf("department target not built")
	}
	if s.Limitations == nil || s.Limitations.NetworkSegments == nil || *(*s.Limitations.NetworkSegments.NetworkSegment)[0].ID != 50 {
		t.Errorf("limitations network segment not built")
	}
	if s.Limitations.Ibeacons == nil || *(*s.Limitations.Ibeacons.Ibeacon)[0].ID != 60 {
		t.Errorf("limitations ibeacon not built")
	}
	if s.Exclusions == nil {
		t.Fatal("exclusions must be built")
	}
	e := s.Exclusions
	if e.Computers == nil || *(*e.Computers.Computer)[0].ID != 70 {
		t.Errorf("exclusion computer not built")
	}
	if e.NetworkSegments == nil || *(*e.NetworkSegments.NetworkSegment)[0].ID != 74 {
		t.Errorf("exclusion network segment not built")
	}
	if e.Ibeacons == nil || *(*e.Ibeacons.Ibeacon)[0].ID != 75 {
		t.Errorf("exclusion ibeacon not built")
	}
}

// TestBuildScope_OmittedCollapsesToNil verifies an all-null scope collapses to a
// nil scope pointer so the payload omits <scope> entirely.
func TestBuildScope_OmittedCollapsesToNil(t *testing.T) {
	plan := PatchPolicyResourceModel{
		SoftwareTitleConfigurationID: types.StringValue("1"),
		Name:                         types.StringValue("p"),
		TargetVersion:                types.StringValue("1.0"),
		Scope:                        &PatchPolicyScopeModel{},
	}
	out, diags := buildPatchPolicyInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("build diags: %v", diags)
	}
	if out.Scope != nil {
		t.Errorf("empty scope must collapse to nil, got %+v", out.Scope)
	}
}

// TestBuildUserInteraction_NestedRoundTrip verifies the declared user_interaction
// block (and its nested notifications/reminders/deadlines/grace_period) maps into
// the SDK shape, and that an omitted nested block is left nil.
func TestBuildUserInteraction_NestedRoundTrip(t *testing.T) {
	plan := PatchPolicyResourceModel{
		SoftwareTitleConfigurationID: types.StringValue("1"),
		Name:                         types.StringValue("p"),
		TargetVersion:                types.StringValue("1.0"),
		UserInteraction: &PatchPolicyUserInteractionModel{
			InstallButtonText:      types.StringValue("Install"),
			SelfServiceDescription: types.StringValue("desc"),
			SelfServiceIconID:      types.StringValue("99"),
			Notifications: &PatchPolicyUserInteractionNotificationsModel{
				Enabled: types.BoolValue(true),
				Subject: types.StringValue("subj"),
				Message: types.StringValue("msg"),
				Type:    types.StringValue("Self Service"),
				Reminders: &PatchPolicyUserInteractionNotificationsRemindersModel{
					Enabled:   types.BoolValue(true),
					Frequency: types.Int64Value(12),
				},
			},
			Deadlines: &PatchPolicyUserInteractionDeadlinesModel{
				Enabled: types.BoolValue(true),
				Period:  types.Int64Value(5),
			},
			// grace_period omitted → must stay nil in the payload.
		},
	}

	out, diags := buildPatchPolicyInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("build diags: %v", diags)
	}
	ui := out.UserInteraction
	if ui == nil {
		t.Fatal("user_interaction must be emitted")
	}
	if ui.InstallButtonText == nil || *ui.InstallButtonText != "Install" {
		t.Errorf("install_button_text not built")
	}
	if ui.SelfServiceIcon == nil || ui.SelfServiceIcon.ID == nil || *ui.SelfServiceIcon.ID != 99 {
		t.Errorf("self_service_icon id not built")
	}
	if ui.Notifications == nil || ui.Notifications.NotificationSubject == nil || *ui.Notifications.NotificationSubject != "subj" {
		t.Errorf("notifications subject not built")
	}
	if ui.Notifications.Reminders == nil || ui.Notifications.Reminders.NotificationReminderFrequency == nil || *ui.Notifications.Reminders.NotificationReminderFrequency != 12 {
		t.Errorf("reminders frequency not built")
	}
	if ui.Deadlines == nil || ui.Deadlines.DeadlinePeriod == nil || *ui.Deadlines.DeadlinePeriod != 5 {
		t.Errorf("deadlines period not built")
	}
	if ui.GracePeriod != nil {
		t.Errorf("omitted grace_period must stay nil, got %+v", ui.GracePeriod)
	}
}
