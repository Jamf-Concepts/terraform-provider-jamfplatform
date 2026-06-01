// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func baseModel() AppInstallerResourceModel {
	return AppInstallerResourceModel{
		Name:           types.StringValue("tf-acc-app-installer"),
		AppTitleName:   types.StringValue("Adobe Lightroom Classic"),
		DeploymentType: types.StringValue(deploymentTypeSelfService),
		UpdateBehavior: types.StringValue(updateBehaviorManual),
	}
}

func TestBuildAppInstallerInput_RequiredFields(t *testing.T) {
	// app_title_id is the resolved catalog ID passed in by the caller (the plan's
	// app_title_id is Computed and not populated until the follow-up GET).
	out := buildAppInstallerInput(baseModel(), "027")
	if out.Name != "tf-acc-app-installer" {
		t.Errorf("name: got %q", out.Name)
	}
	if out.AppTitleID != "027" {
		t.Errorf("app_title_id: got %q", out.AppTitleID)
	}
	if out.DeploymentType != deploymentTypeSelfService {
		t.Errorf("deployment_type: got %q", out.DeploymentType)
	}
}

func TestBuildAppInstallerInput_UpdateBehaviorAlwaysSet(t *testing.T) {
	// update_behavior is server-required; it must always be emitted from the
	// known plan value (not routed through the Optional helper).
	out := buildAppInstallerInput(baseModel(), "027")
	if out.UpdateBehavior == nil {
		t.Fatalf("update_behavior must always be set")
	}
	if *out.UpdateBehavior != updateBehaviorManual {
		t.Errorf("update_behavior: got %q", *out.UpdateBehavior)
	}
}

func TestBuildAppInstallerInput_TriggerAdminNotifications(t *testing.T) {
	// trigger_admin_notifications is Optional+Computed: a configured value is
	// emitted; null/unknown collapses to nil so the wire omits it.
	m := baseModel()
	m.TriggerAdminNotifications = types.BoolValue(true)
	out := buildAppInstallerInput(m, "027")
	if out.TriggerAdminNotifications == nil || !*out.TriggerAdminNotifications {
		t.Errorf("configured trigger_admin_notifications must be emitted true, got %v", out.TriggerAdminNotifications)
	}

	m.TriggerAdminNotifications = types.BoolNull()
	out = buildAppInstallerInput(m, "027")
	if out.TriggerAdminNotifications != nil {
		t.Errorf("null trigger_admin_notifications must be nil, got %v", *out.TriggerAdminNotifications)
	}
}

func TestBuildAppInstallerInput_SentinelIDsOmittedWhenUnknown(t *testing.T) {
	// Optional+Computed IDs left unset (Unknown) must collapse to nil so the
	// wire omits them (server defaults to "-1").
	m := baseModel()
	m.CategoryID = types.StringUnknown()
	m.SiteID = types.StringNull()
	m.SmartGroupID = types.StringValue("-1")
	out := buildAppInstallerInput(m, "027")
	if out.CategoryID != nil {
		t.Errorf("unknown category_id should be nil, got %v", *out.CategoryID)
	}
	if out.SiteID != nil {
		t.Errorf("null site_id should be nil, got %v", *out.SiteID)
	}
	if out.SmartGroupID == nil || *out.SmartGroupID != "-1" {
		t.Errorf("smart_group_id -1 should be emitted verbatim, got %v", out.SmartGroupID)
	}
}

func TestBuildAppInstallerInput_NestedBlocksOmittedWhenAbsent(t *testing.T) {
	out := buildAppInstallerInput(baseModel(), "027")
	if out.NotificationSettings != nil {
		t.Errorf("notification_settings must be nil when block absent")
	}
	if out.SelfServiceSettings != nil {
		t.Errorf("self_service_settings must be nil when block absent")
	}
}

func TestBuildAppInstallerInput_NotificationOmitsUnsetFields(t *testing.T) {
	// Notification fields are individually optional: only the fields the user set
	// are emitted; unset fields are nil (omitted). The server rejects a blank
	// message or a non-positive interval/delay on any field that is present, so
	// zero-filling unset fields would 400.
	m := baseModel()
	m.NotificationSettings = &NotificationSettingsModel{
		NotificationMessage: types.StringValue("update available"),
		Deadline:            types.Int64Value(48),
	}
	out := buildAppInstallerInput(m, "027")
	n := out.NotificationSettings
	if n == nil {
		t.Fatalf("notification_settings must be emitted")
	}
	// Set fields are present.
	if n.NotificationMessage == nil || *n.NotificationMessage != "update available" {
		t.Errorf("notification_message: got %v", n.NotificationMessage)
	}
	if n.Deadline == nil || *n.Deadline != 48 {
		t.Errorf("deadline: got %v", n.Deadline)
	}
	// Unset fields are omitted (nil), not zero-filled.
	if n.DeadlineMessage != nil {
		t.Errorf("unset deadline_message must be nil, got %q", *n.DeadlineMessage)
	}
	if n.CompleteMessage != nil {
		t.Errorf("unset complete_message must be nil, got %q", *n.CompleteMessage)
	}
	if n.NotificationInterval != nil {
		t.Errorf("unset notification_interval must be nil, got %d", *n.NotificationInterval)
	}
	if n.QuitDelay != nil {
		t.Errorf("unset quit_delay must be nil, got %d", *n.QuitDelay)
	}
	if n.Relaunch != nil {
		t.Errorf("unset relaunch must be nil, got %v", *n.Relaunch)
	}
}

func TestBuildAppInstallerInput_CategoriesEmptyEmitsEmptySlice(t *testing.T) {
	// Within a managed self_service block, zero categories must emit an empty
	// slice (not nil) so clearing categories is deterministic (full-replace).
	m := baseModel()
	m.SelfServiceSettings = &SelfServiceSettingsModel{
		Description: types.StringValue("desc"),
		Categories:  nil,
	}
	out := buildAppInstallerInput(m, "027")
	ss := out.SelfServiceSettings
	if ss == nil {
		t.Fatalf("self_service_settings must be emitted")
	}
	if ss.Categories == nil {
		t.Fatalf("categories must be a non-nil pointer")
	}
	if len(*ss.Categories) != 0 {
		t.Errorf("categories must be empty slice, got %d", len(*ss.Categories))
	}
}

func TestBuildAppInstallerInput_CategoriesPopulated(t *testing.T) {
	m := baseModel()
	m.SelfServiceSettings = &SelfServiceSettingsModel{
		Categories: []SelfServiceCategoryModel{
			{CategoryID: types.StringValue("58"), Featured: types.BoolValue(true)},
			{CategoryID: types.StringValue("44"), Featured: types.BoolNull()},
		},
	}
	out := buildAppInstallerInput(m, "027")
	cats := *out.SelfServiceSettings.Categories
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
	if cats[0].ID == nil || *cats[0].ID != "58" || cats[0].Featured == nil || !*cats[0].Featured {
		t.Errorf("category 0 mismatch: %+v", cats[0])
	}
	if cats[1].Featured == nil || *cats[1].Featured {
		t.Errorf("category 1 unset featured must emit false, got %v", cats[1].Featured)
	}
}
