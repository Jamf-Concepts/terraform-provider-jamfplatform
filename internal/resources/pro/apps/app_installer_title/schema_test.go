// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer_title

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

const wantTypeName = "jamfplatform_pro_app_installer_title"

func TestAppInstallerTitleDataSource_Metadata(t *testing.T) {
	d := NewAppInstallerTitleDataSource()
	var resp datasource.MetadataResponse
	d.(*AppInstallerTitleDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestAppInstallerTitleDataSource_Schema(t *testing.T) {
	d := NewAppInstallerTitleDataSource()
	var resp datasource.SchemaResponse
	d.(*AppInstallerTitleDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, name := range []string{
		"id", "title_name", "publisher", "bundle_id", "version", "short_version",
		"architecture", "minimum_os_version", "language", "availability_date",
		"icon_url", "size_in_bytes", "installer_package_hash",
		"installer_package_hash_type", "launch_daemon_included",
		"notification_available", "package_signing_identity", "suppress_auto_update",
		"original_media_sources",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if id := s.Attributes["id"]; !id.IsRequired() {
		t.Errorf("id must be required for lookup")
	}
}

func TestAssignTitleDataSource(t *testing.T) {
	got := AssignTitleDataSource(&pro.AppInstallerTitle{
		ID:                       "027",
		TitleName:                "Adobe Lightroom Classic",
		Publisher:                "Adobe",
		BundleID:                 "com.adobe.LightroomClassicCC7",
		Version:                  "14.2",
		ShortVersion:             "14.2",
		Architecture:             "universal",
		MinimumOsVersion:         "12.0",
		Language:                 "en",
		AvailabilityDate:         "2026-01-01",
		IconURL:                  "https://example/icon.png",
		SizeInBytes:              123456,
		InstallerPackageHash:     "abc",
		InstallerPackageHashType: "SHA_256",
		LaunchDaemonIncluded:     true,
		NotificationAvailable:    true,
		PackageSigningIdentity:   "Developer ID Installer: Adobe",
		SuppressAutoUpdate:       true,
		OriginalMediaSources: []pro.OriginalMediaSource{
			{Hash: "h1", HashType: "SHA_256", URL: "https://example/media"},
		},
	})

	if got.ID.ValueString() != "027" {
		t.Errorf("id: got %q", got.ID.ValueString())
	}
	if got.SizeInBytes.ValueInt64() != 123456 {
		t.Errorf("size_in_bytes: got %d", got.SizeInBytes.ValueInt64())
	}
	if !got.LaunchDaemonIncluded.ValueBool() {
		t.Errorf("launch_daemon_included should be true")
	}
	if len(got.OriginalMediaSources) != 1 || got.OriginalMediaSources[0].Hash.ValueString() != "h1" {
		t.Errorf("original_media_sources not mapped: %+v", got.OriginalMediaSources)
	}
}

func TestAssignTitleDataSource_NilSafe(t *testing.T) {
	got := AssignTitleDataSource(nil)
	if !got.ID.IsNull() {
		t.Errorf("nil title should produce null id")
	}
}
