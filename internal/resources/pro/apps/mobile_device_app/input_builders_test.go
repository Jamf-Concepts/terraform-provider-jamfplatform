// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

func idSet(ids ...string) types.Set {
	vals := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, types.StringValue(id))
	}
	return types.SetValueMust(types.StringType, vals)
}

func TestBuildMobileAppGeneral(t *testing.T) {
	m := &MobileAppGeneralModel{
		Name:           types.StringValue("Maps"),
		Version:        types.StringValue("1.0"),
		BundleID:       types.StringValue("com.apple.Maps"),
		OsType:         types.StringValue(osTypeIOS),
		IsFree:         types.BoolValue(true),
		DeploymentType: types.StringValue(deploymentTypeAutomatic),
		ExternalURL:    types.StringValue("https://example.com/app.ipa"),
		ItunesStoreURL: types.StringValue("https://apps.apple.com/app/id915056765"),
		ItunesSyncTime: types.Int64Value(1700000000),
		HostExternally: types.BoolValue(true),
		CategoryID:     types.StringValue("7"),
		SiteID:         types.StringValue("3"),
	}
	g := buildMobileAppGeneral(m)
	if g.Name == nil || *g.Name != "Maps" {
		t.Errorf("name not mapped: %+v", g.Name)
	}
	if g.OsType == nil || *g.OsType != osTypeIOS {
		t.Errorf("os_type not mapped: %+v", g.OsType)
	}
	if g.Free == nil || !*g.Free {
		t.Errorf("is_free → free not mapped")
	}
	if g.ExternalURL == nil || *g.ExternalURL != "https://example.com/app.ipa" {
		t.Errorf("external_url not mapped")
	}
	if g.ItunesStoreURL == nil {
		t.Errorf("itunes_store_url not mapped")
	}
	if g.ItunesSyncTime == nil || *g.ItunesSyncTime != 1700000000 {
		t.Errorf("itunes_sync_time not mapped: %+v", g.ItunesSyncTime)
	}
	if g.HostExternally == nil || !*g.HostExternally {
		t.Errorf("host_externally not mapped")
	}
	if g.Category == nil || g.Category.ID == nil || *g.Category.ID != 7 {
		t.Errorf("category id not mapped: %+v", g.Category)
	}
	if g.Site == nil || g.Site.ID == nil || *g.Site.ID != 3 {
		t.Errorf("site id not mapped: %+v", g.Site)
	}
	// Server-managed fields are never written from the model.
	if g.DisplayName != nil || g.Description != nil || g.InternalApp != nil {
		t.Errorf("server-managed fields must not be written: %+v %+v %+v", g.DisplayName, g.Description, g.InternalApp)
	}
}

func TestBuildMobileNotification(t *testing.T) {
	if n := buildMobileNotification(types.BoolNull()); n != nil {
		t.Errorf("expected nil notification when unconfigured, got %+v", n)
	}
	n := buildMobileNotification(types.BoolValue(true))
	if n == nil || n.Enabled == nil || !*n.Enabled {
		t.Fatalf("notification enabled not assembled: %+v", n)
	}
	// Mobile carries only the bool form — no method leg.
	if n.Method != nil {
		t.Errorf("mobile notification must not set a method, got %+v", n.Method)
	}
}

func TestBuildMobileAppScope_TargetsAndCollapse(t *testing.T) {
	ctx := context.Background()

	empty := &scope.MobileScopeModelNoIbeacons{
		AllMobileDevices:     types.BoolNull(),
		AllJssUsers:          types.BoolNull(),
		MobileDeviceIDs:      types.SetNull(types.StringType),
		MobileDeviceGroupIDs: types.SetNull(types.StringType),
		BuildingIDs:          types.SetNull(types.StringType),
		DepartmentIDs:        types.SetNull(types.StringType),
		UserIDs:              types.SetNull(types.StringType),
		UserGroupIDs:         types.SetNull(types.StringType),
	}
	s, diags := buildMobileAppScope(ctx, empty)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if s != nil {
		t.Errorf("expected nil scope for empty model, got %+v", s)
	}

	m := &scope.MobileScopeModelNoIbeacons{
		AllMobileDevices:     types.BoolValue(false),
		AllJssUsers:          types.BoolNull(),
		MobileDeviceIDs:      idSet("11", "12"),
		MobileDeviceGroupIDs: idSet("5"),
		BuildingIDs:          types.SetNull(types.StringType),
		DepartmentIDs:        types.SetNull(types.StringType),
		UserIDs:              types.SetNull(types.StringType),
		UserGroupIDs:         types.SetNull(types.StringType),
		Limitations: &scope.MobileScopeLimitationsModelNoIbeacons{
			NetworkSegmentIDs:                idSet("2"),
			DirectoryServiceOrLocalUserNames: types.SetNull(types.StringType),
			DirectoryServiceUserGroupNames:   types.SetNull(types.StringType),
		},
	}
	s, diags = buildMobileAppScope(ctx, m)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if s == nil || s.MobileDevices == nil || s.MobileDevices.MobileDevice == nil || len(*s.MobileDevices.MobileDevice) != 2 {
		t.Fatalf("mobile_devices not mapped: %+v", s)
	}
	if s.MobileDeviceGroups == nil || s.MobileDeviceGroups.MobileDeviceGroup == nil || len(*s.MobileDeviceGroups.MobileDeviceGroup) != 1 {
		t.Errorf("mobile_device_groups not mapped")
	}
	if s.Limitations == nil || s.Limitations.NetworkSegments == nil {
		t.Errorf("limitations network_segments not mapped")
	}
}

func TestBuildMobileAppVpp_Collapse(t *testing.T) {
	// Neither writable field present → nil.
	m := &MobileAppVppModel{
		AssignVppDeviceBasedLicenses: types.BoolNull(),
		VppAdminAccountID:            types.StringNull(),
	}
	if v := buildMobileAppVpp(m); v != nil {
		t.Errorf("expected nil vpp when no writable fields, got %+v", v)
	}

	m = &MobileAppVppModel{
		AssignVppDeviceBasedLicenses: types.BoolValue(true),
		VppAdminAccountID:            types.StringValue("4"),
	}
	v := buildMobileAppVpp(m)
	if v == nil || v.AssignVppDeviceBasedLicenses == nil || !*v.AssignVppDeviceBasedLicenses {
		t.Fatalf("assign not mapped: %+v", v)
	}
	if v.VppAdminAccountID == nil || *v.VppAdminAccountID != 4 {
		t.Errorf("vpp_admin_account_id not mapped")
	}
}

func TestBuildMobileAppSelfService_CategoriesAndIcon(t *testing.T) {
	m := &MobileAppSelfServiceModel{
		InstallButtonText:      types.StringValue("Install"),
		AfterInstallButtonText: types.StringValue("Open"),
		NotificationEnabled:    types.BoolValue(true),
		SelfServiceIcon:        &MobileAppSelfServiceIconModel{ID: types.StringValue("9")},
		SelfServiceCategories: []MobileAppSelfServiceCategoryModel{
			{ID: types.StringValue("3"), DisplayIn: types.BoolValue(true)},
		},
	}
	ss := buildMobileAppSelfService(m)
	if ss.SelfServiceInstallButtonText == nil || *ss.SelfServiceInstallButtonText != "Install" {
		t.Errorf("install_button_text not mapped")
	}
	if ss.SelfServiceAfterInstallButtonText == nil || *ss.SelfServiceAfterInstallButtonText != "Open" {
		t.Errorf("after_install_button_text not mapped")
	}
	if ss.Notification == nil || ss.Notification.Enabled == nil || !*ss.Notification.Enabled {
		t.Errorf("notification not assembled")
	}
	if ss.SelfServiceIcon == nil || ss.SelfServiceIcon.ID == nil || *ss.SelfServiceIcon.ID != 9 {
		t.Errorf("icon id not mapped: %+v", ss.SelfServiceIcon)
	}
	if ss.SelfServiceCategories == nil || ss.SelfServiceCategories.Category == nil || len(*ss.SelfServiceCategories.Category) != 1 {
		t.Fatalf("categories not mapped")
	}
	c := (*ss.SelfServiceCategories.Category)[0]
	if c.ID == nil || *c.ID != 3 || c.DisplayIn == nil || !*c.DisplayIn {
		t.Errorf("category not mapped: %+v", c)
	}
}

func TestBuildMobileAppAppConfiguration(t *testing.T) {
	// Unconfigured → nil.
	if ac := buildMobileAppAppConfiguration(&MobileAppAppConfigurationModel{Preferences: types.StringNull()}); ac != nil {
		t.Errorf("expected nil app_configuration when unconfigured, got %+v", ac)
	}
	ac := buildMobileAppAppConfiguration(&MobileAppAppConfigurationModel{Preferences: types.StringValue("<dict/>")})
	if ac == nil || ac.Preferences == nil || *ac.Preferences != "<dict/>" {
		t.Fatalf("preferences not mapped: %+v", ac)
	}
}

func TestNormalizeNewlines(t *testing.T) {
	if got := normalizeNewlines("a\r\nb\rc\nd"); got != "a\nb\nc\nd" {
		t.Errorf("normalizeNewlines = %q", got)
	}
}
