// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestFlattenMobileApp_ReportsDrift pins the wire-authoritative read: an echoed
// value that differs from state must land in state so `terraform plan` reports
// the change. Every field asserted here round-trips through the classic
// /mobiledeviceapplications GET on both the POST and the PUT path (Jamf Pro
// 11.31.1, wire-probed 2026-09-06). See issue #387.
func TestFlattenMobileApp_ReportsDrift(t *testing.T) {
	t.Parallel()
	general := &MobileAppGeneralModel{
		DeploymentType:                   types.StringValue("Make Available in Self Service"),
		ExternalURL:                      types.StringValue("https://state.invalid/"),
		ItunesStoreURL:                   types.StringValue("https://apps.apple.com/app/id1"),
		ItunesCountryRegion:              types.StringValue("GB"),
		ItunesSyncTime:                   types.Int64Value(9),
		IsFree:                           types.BoolValue(false),
		KeepAppUpdatedOnDevices:          types.BoolValue(false),
		RemoveAppWhenMDMProfileIsRemoved: types.BoolValue(false),
		CategoryID:                       types.StringValue("11"),
		SiteID:                           types.StringValue("-1"),
	}
	flattenMobileAppGeneral(&proclassic.MobileDeviceApplicationGeneral{
		Name:                             new("app"),
		DeploymentType:                   new("Install Automatically/Prompt Users to Install"),
		ExternalURL:                      new("https://wire.invalid/"),
		ItunesStoreURL:                   new("https://apps.apple.com/app/id2"),
		ItunesCountryRegion:              new("US"),
		ItunesSyncTime:                   new(5),
		Free:                             new(true),
		KeepAppUpdatedOnDevices:          new(true),
		RemoveAppWhenMDMProfileIsRemoved: new(true),
		Category:                         &proclassic.CategoryObject{ID: new(653), Name: new("Operations")},
		Site:                             &proclassic.SiteObject{ID: new(1), Name: new("AGATA")},
	}, general)

	for _, tc := range []struct{ name, want, got string }{
		{"deployment_type", "Install Automatically/Prompt Users to Install", general.DeploymentType.ValueString()},
		{"external_url", "https://wire.invalid/", general.ExternalURL.ValueString()},
		{"itunes_store_url", "https://apps.apple.com/app/id2", general.ItunesStoreURL.ValueString()},
		{"itunes_country_region", "US", general.ItunesCountryRegion.ValueString()},
		{"category_id", "653", general.CategoryID.ValueString()},
		{"site_id", "1", general.SiteID.ValueString()},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: wire value must win, want %q got %q", tc.name, tc.want, tc.got)
		}
	}
	if general.ItunesSyncTime.ValueInt64() != 5 {
		t.Errorf("itunes_sync_time: wire value must win, got %d", general.ItunesSyncTime.ValueInt64())
	}
	for _, tc := range []struct {
		name string
		got  types.Bool
	}{
		{"is_free", general.IsFree},
		{"keep_app_updated_on_devices", general.KeepAppUpdatedOnDevices},
		{"remove_app_when_mdm_profile_is_removed", general.RemoveAppWhenMDMProfileIsRemoved},
	} {
		if !tc.got.ValueBool() {
			t.Errorf("%s: wire value must win, got false", tc.name)
		}
	}

	vpp := &MobileAppVppModel{
		AssignVppDeviceBasedLicenses: types.BoolValue(false),
		VppAdminAccountID:            types.StringValue("-1"),
	}
	flattenMobileAppVpp(&proclassic.MobileDeviceApplicationVpp{
		AssignVppDeviceBasedLicenses: new(true),
		VppAdminAccountID:            new(4),
	}, vpp)
	if !vpp.AssignVppDeviceBasedLicenses.ValueBool() {
		t.Error("vpp.assign_vpp_device_based_licenses: wire value must win, got false")
	}
	if got := vpp.VppAdminAccountID.ValueString(); got != "4" {
		t.Errorf("vpp.vpp_admin_account_id: wire value must win, got %q", got)
	}
}

// TestFlattenMobileApp_StickyFieldsIgnoreDrift pins the other half of the #387
// split: host_externally (the write does not persist while external_url is
// set), after_install_button_text (echoed on create, absent from every GET
// after a PUT) and the three self_service notification_* fields (never echoed)
// keep the value already in state.
func TestFlattenMobileApp_StickyFieldsIgnoreDrift(t *testing.T) {
	t.Parallel()
	general := &MobileAppGeneralModel{HostExternally: types.BoolValue(false)}
	flattenMobileAppGeneral(&proclassic.MobileDeviceApplicationGeneral{
		Name:           new("app"),
		HostExternally: new(true),
	}, general)
	if general.HostExternally.ValueBool() {
		t.Error("host_externally: sticky read must keep false")
	}

	ss := &MobileAppSelfServiceModel{
		AfterInstallButtonText: types.StringValue("state after"),
		NotificationEnabled:    types.BoolValue(true),
		NotificationSubject:    types.StringValue("state subject"),
		NotificationMessage:    types.StringValue("state message"),
	}
	// The wire shape a GET returns after any PUT: after_install_button_text and
	// the whole <notification> family absent.
	flattenMobileAppSelfService(&proclassic.MobileDeviceApplicationSelfService{}, ss)
	for _, tc := range []struct{ name, want, got string }{
		{"after_install_button_text", "state after", ss.AfterInstallButtonText.ValueString()},
		{"notification_subject", "state subject", ss.NotificationSubject.ValueString()},
		{"notification_message", "state message", ss.NotificationMessage.ValueString()},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: sticky read must keep %q, got %q", tc.name, tc.want, tc.got)
		}
	}
	if !ss.NotificationEnabled.ValueBool() {
		t.Error("notification_enabled: sticky read must keep true")
	}
}
