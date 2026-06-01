// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func sampleDeployment() *pro.AppInstallerDeployment {
	return &pro.AppInstallerDeployment{
		ID:                              "177",
		Name:                            "tf-acc-app-installer",
		Enabled:                         true,
		AppTitleID:                      "027",
		DeploymentType:                  "SELF_SERVICE",
		UpdateBehavior:                  "AUTOMATIC",
		SelectedVersion:                 "",
		LatestAvailableVersion:          "14.2",
		TitleAvailableInAis:             true,
		VersionRemoved:                  false,
		CategoryID:                      "-1",
		SiteID:                          "-1",
		SmartGroupID:                    "-1",
		InstallPredefinedConfigProfiles: false,
		TriggerAdminNotifications:       true,
	}
}

func TestAssignAppInstallerResourceModel_FlatScalars(t *testing.T) {
	state := AppInstallerResourceModel{}
	assignAppInstallerResourceModel(&state, sampleDeployment())

	if state.ID.ValueString() != "177" {
		t.Errorf("id: got %q", state.ID.ValueString())
	}
	if state.LatestAvailableVersion.ValueString() != "14.2" {
		t.Errorf("latest_available_version: got %q", state.LatestAvailableVersion.ValueString())
	}
	if !state.TitleAvailableInAis.ValueBool() {
		t.Errorf("title_available_in_ais should be true")
	}
	if state.CategoryID.ValueString() != "-1" {
		t.Errorf("category_id should be -1 verbatim, got %q", state.CategoryID.ValueString())
	}
	if !state.TriggerAdminNotifications.ValueBool() {
		t.Errorf("trigger_admin_notifications should be true")
	}
}

func TestAssignAppInstallerResourceModel_NestedBlocksGated(t *testing.T) {
	// The server echoes both blocks populated; an unmanaged (nil) block in state
	// must stay nil so the framework consistency check passes.
	d := sampleDeployment()
	d.NotificationSettings = &pro.AppInstallerNotificationSettings{NotificationMessage: new("hi")}
	d.SelfServiceSettings = &pro.AppInstallerSelfServiceSettings{Description: new("desc")}

	state := AppInstallerResourceModel{} // both blocks nil (unmanaged)
	assignAppInstallerResourceModel(&state, d)
	if state.NotificationSettings != nil {
		t.Errorf("unmanaged notification_settings must stay nil")
	}
	if state.SelfServiceSettings != nil {
		t.Errorf("unmanaged self_service_settings must stay nil")
	}
}

func TestAssignAppInstallerResourceModel_NestedBlocksManaged(t *testing.T) {
	d := sampleDeployment()
	d.NotificationSettings = &pro.AppInstallerNotificationSettings{
		NotificationMessage: new("hi"),
		Deadline:            new(48),
		Suppress:            new(true),
	}
	d.SelfServiceSettings = &pro.AppInstallerSelfServiceSettings{
		Description:               new("desc"),
		IncludeInFeaturedCategory: new(true),
		Categories: &[]pro.SelfServiceCategory{
			{ID: new("58"), Featured: new(true)},
		},
	}

	state := AppInstallerResourceModel{
		NotificationSettings: &NotificationSettingsModel{}, // managed
		SelfServiceSettings:  &SelfServiceSettingsModel{},  // managed
	}
	assignAppInstallerResourceModel(&state, d)

	if state.NotificationSettings == nil {
		t.Fatalf("managed notification_settings must be refreshed")
	}
	if state.NotificationSettings.NotificationMessage.ValueString() != "hi" {
		t.Errorf("notification_message: got %q", state.NotificationSettings.NotificationMessage.ValueString())
	}
	if state.NotificationSettings.Deadline.ValueInt64() != 48 {
		t.Errorf("deadline: got %d", state.NotificationSettings.Deadline.ValueInt64())
	}
	if !state.NotificationSettings.Suppress.ValueBool() {
		t.Errorf("suppress should be true")
	}

	if state.SelfServiceSettings == nil {
		t.Fatalf("managed self_service_settings must be refreshed")
	}
	if len(state.SelfServiceSettings.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(state.SelfServiceSettings.Categories))
	}
	c := state.SelfServiceSettings.Categories[0]
	if c.CategoryID.ValueString() != "58" || !c.Featured.ValueBool() {
		t.Errorf("category mismatch: %+v", c)
	}
}

func TestAssignAppInstallerResourceModel_CategoriesEmptyNotNil(t *testing.T) {
	d := sampleDeployment()
	d.SelfServiceSettings = &pro.AppInstallerSelfServiceSettings{Categories: nil}
	state := AppInstallerResourceModel{SelfServiceSettings: &SelfServiceSettingsModel{}}
	assignAppInstallerResourceModel(&state, d)
	if state.SelfServiceSettings.Categories == nil {
		t.Errorf("categories must be empty slice, not nil")
	}
	if len(state.SelfServiceSettings.Categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(state.SelfServiceSettings.Categories))
	}
}

func TestAssignAppInstallerResourceModel_NilSafe(t *testing.T) {
	state := AppInstallerResourceModel{ID: types.StringValue("keep")}
	assignAppInstallerResourceModel(&state, nil)
	if state.ID.ValueString() != "keep" {
		t.Errorf("nil response must not clobber state")
	}
}

func TestAssignAppInstallerDataSourceModel(t *testing.T) {
	data := AppInstallerDataSourceModel{}
	assignAppInstallerDataSourceModel(&data, sampleDeployment())
	if data.ID.ValueString() != "177" || data.Name.ValueString() != "tf-acc-app-installer" {
		t.Errorf("data source flat mapping mismatch: %+v", data)
	}
	if data.DeploymentType.ValueString() != "SELF_SERVICE" {
		t.Errorf("deployment_type: got %q", data.DeploymentType.ValueString())
	}
}
