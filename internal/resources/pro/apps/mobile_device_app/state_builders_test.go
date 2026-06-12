// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

func TestFlattenMobileAppGeneral_RoundTrip(t *testing.T) {
	state := &MobileAppGeneralModel{}
	g := &proclassic.MobileDeviceApplicationGeneral{
		ID:             new(42),
		Name:           new("Maps"),
		Version:        new("1.0"),
		BundleID:       new("com.apple.Maps"),
		OsType:         new(osTypeIOS),
		Description:    new("Apple Maps"),
		InternalApp:    new(true),
		Free:           new(true),
		DeploymentType: new(deploymentTypeSelfService),
		ExternalURL:    new("https://example.com/app.ipa"),
		ItunesStoreURL: new("https://apps.apple.com/app/id915056765"),
		ItunesSyncTime: new(1700000000),
		HostExternally: new(true),
		Category:       &proclassic.CategoryObject{ID: new(7), Name: new("Productivity")},
		Site:           &proclassic.SiteObject{ID: new(3), Name: new("HQ")},
	}
	flattenMobileAppGeneral(g, state)

	if state.ID.ValueString() != "42" {
		t.Errorf("id: got %q", state.ID.ValueString())
	}
	if state.OsType.ValueString() != osTypeIOS {
		t.Errorf("os_type: got %q", state.OsType.ValueString())
	}
	if state.Description.ValueString() != "Apple Maps" {
		t.Errorf("description not flattened: %q", state.Description.ValueString())
	}
	if !state.IsFree.ValueBool() {
		t.Errorf("is_free (free) not flattened")
	}
	if state.ExternalURL.ValueString() != "https://example.com/app.ipa" {
		t.Errorf("external_url not flattened")
	}
	if state.ItunesSyncTime.ValueInt64() != 1700000000 {
		t.Errorf("itunes_sync_time not flattened: %d", state.ItunesSyncTime.ValueInt64())
	}
	if !state.HostExternally.ValueBool() {
		t.Errorf("host_externally not flattened")
	}
	if state.CategoryID.ValueString() != "7" || state.CategoryName.ValueString() != "Productivity" {
		t.Errorf("category not flattened: %q / %q", state.CategoryID.ValueString(), state.CategoryName.ValueString())
	}
	if state.SiteID.ValueString() != "3" || state.SiteName.ValueString() != "HQ" {
		t.Errorf("site not flattened")
	}
}

// TestAssignMobileApp_GuardedBlocks is the core echo-guard test: a minimal app
// created without scope / self_service / vpp / app_configuration blocks must keep
// those blocks null in state even though the server echoes them on GET.
func TestAssignMobileApp_GuardedBlocks(t *testing.T) {
	state := &MobileAppResourceModel{
		General: &MobileAppGeneralModel{Name: types.StringValue("Maps")},
		// Scope / SelfService / Vpp / AppConfiguration intentionally nil (unmanaged).
	}
	server := &proclassic.MobileDeviceApplication{
		ID:      new(42),
		General: &proclassic.MobileDeviceApplicationGeneral{ID: new(42), Name: new("Maps"), OsType: new(osTypeIOS)},
		Scope:   &proclassic.MobileDeviceApplicationScope{AllMobileDevices: new(false), AllJssUsers: new(false)},
		SelfService: &proclassic.MobileDeviceApplicationSelfService{
			SelfServiceInstallButtonText: new("Install"),
		},
		Vpp:              &proclassic.MobileDeviceApplicationVpp{VppAdminAccountID: new(-1)},
		AppConfiguration: &proclassic.MobileDeviceApplicationAppConfiguration{Preferences: new("<dict/>")},
	}

	diags := assignMobileAppResourceModel(context.Background(), state, server)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if state.Scope != nil {
		t.Errorf("unmanaged scope was populated: %+v", state.Scope)
	}
	if state.SelfService != nil {
		t.Errorf("unmanaged self_service was populated: %+v", state.SelfService)
	}
	if state.Vpp != nil {
		t.Errorf("unmanaged vpp was populated: %+v", state.Vpp)
	}
	if state.AppConfiguration != nil {
		t.Errorf("unmanaged app_configuration was populated: %+v", state.AppConfiguration)
	}
	if state.ID.ValueString() != "42" {
		t.Errorf("id not set: %q", state.ID.ValueString())
	}
}

func TestFlattenMobileAppScope_RoundTrip(t *testing.T) {
	ctx := context.Background()
	state := &scope.MobileScopeModelNoIbeacons{
		Targets:     &scope.MobileScopeTargetsModel{},
		Limitations: &scope.MobileScopeLimitationsModelNoIbeacons{},
		Exclusions:  &scope.MobileScopeExclusionsModelNoIbeacons{},
	}
	s := &proclassic.MobileDeviceApplicationScope{
		AllMobileDevices: new(false),
		MobileDevices: &proclassic.MobileDeviceApplicationScopeMobileDevices{
			MobileDevice: &[]proclassic.MobileDeviceApplicationScopeMobileDevicesMobileDeviceItem{{ID: new(11)}, {ID: new(12)}},
		},
		MobileDeviceGroups: &proclassic.MobileDeviceApplicationScopeMobileDeviceGroups{
			MobileDeviceGroup: &[]proclassic.IDName{{ID: new(5)}},
		},
		Limitations: &proclassic.MobileDeviceApplicationScopeLimitations{
			NetworkSegments: &proclassic.MobileDeviceApplicationScopeLimitationsNetworkSegments{
				NetworkSegment: &[]proclassic.IDName{{ID: new(2)}},
			},
		},
		Exclusions: &proclassic.MobileDeviceApplicationScopeExclusions{
			Users: &proclassic.MobileDeviceApplicationScopeExclusionsUsers{
				User: &[]proclassic.MobileDeviceApplicationScopeExclusionsUsersUserItem{{Name: new("alice")}},
			},
		},
	}
	flattenMobileAppScope(ctx, s, state)

	var mdIDs []string
	state.Targets.MobileDeviceIDs.ElementsAs(ctx, &mdIDs, false)
	if len(mdIDs) != 2 {
		t.Errorf("mobile_device_ids: got %v", mdIDs)
	}
	var segIDs []string
	state.Limitations.NetworkSegmentIDs.ElementsAs(ctx, &segIDs, false)
	if len(segIDs) != 1 || segIDs[0] != "2" {
		t.Errorf("limitations network_segment_ids: got %v", segIDs)
	}
	var exclUsers []string
	state.Exclusions.DirectoryServiceOrLocalUserNames.ElementsAs(ctx, &exclUsers, false)
	if len(exclUsers) != 1 || exclUsers[0] != "alice" {
		t.Errorf("exclusion user names: got %v", exclUsers)
	}
}

func TestFlattenMobileAppVpp(t *testing.T) {
	state := &MobileAppVppModel{}
	v := &proclassic.MobileDeviceApplicationVpp{
		AssignVppDeviceBasedLicenses: new(true),
		VppAdminAccountID:            new(4),
	}
	flattenMobileAppVpp(v, state)
	if !state.AssignVppDeviceBasedLicenses.ValueBool() {
		t.Errorf("assign not flattened")
	}
	if state.VppAdminAccountID.ValueString() != "4" {
		t.Errorf("vpp_admin_account_id: got %q", state.VppAdminAccountID.ValueString())
	}
}

func TestFlattenMobileAppSelfService_Notification(t *testing.T) {
	state := &MobileAppSelfServiceModel{}
	ss := &proclassic.MobileDeviceApplicationSelfService{
		Notification: &proclassic.NotificationValue{Enabled: new(true)},
	}
	flattenMobileAppSelfService(ss, state)
	if !state.NotificationEnabled.ValueBool() {
		t.Errorf("notification_enabled not flattened")
	}
}
