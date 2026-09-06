// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ebook

import (
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestFlattenEbookGeneral_ReportsDrift pins the wire-authoritative read: an
// echoed value that differs from state must land in state so `terraform plan`
// reports the change. Every field asserted here round-trips through the classic
// /ebooks GET (Jamf Pro 11.31.1, wire-probed 2026-09-06). See issue #387.
func TestFlattenEbookGeneral_ReportsDrift(t *testing.T) {
	t.Parallel()
	state := &EbookGeneralModel{
		Author:         types.StringValue("state author"),
		DeploymentType: types.StringValue("Make Available in Self Service"),
		Version:        types.StringValue("1.0"),
		Free:           types.BoolValue(false),
		CategoryID:     types.StringValue("11"),
		SiteID:         types.StringValue("-1"),
	}
	flattenEbookGeneral(&proclassic.EbookGeneral{
		Name:           new("guide"),
		Author:         new("wire author"),
		DeploymentType: new("Install Automatically/Prompt Users to Install"),
		Version:        new("2.0"),
		Free:           new(true),
		Category:       &proclassic.CategoryObject{ID: new(653), Name: new("Operations")},
		Site:           &proclassic.SiteObject{ID: new(1), Name: new("AGATA")},
	}, state)

	for _, tc := range []struct{ name, want, got string }{
		{"author", "wire author", state.Author.ValueString()},
		{"deployment_type", "Install Automatically/Prompt Users to Install", state.DeploymentType.ValueString()},
		{"version", "2.0", state.Version.ValueString()},
		{"category_id", "653", state.CategoryID.ValueString()},
		{"site_id", "1", state.SiteID.ValueString()},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: wire value must win, want %q got %q", tc.name, tc.want, tc.got)
		}
	}
	if !state.Free.ValueBool() {
		t.Error("free: wire value must win, got false")
	}
}

// TestFlattenEbook_StickyFieldsIgnoreDrift pins the other half of the #387
// split for this resource: file_type (server canonicalises the casing),
// deploy_as_managed (the write does not persist) and the four self_service
// notification_* fields (never echoed) keep the value already in state. The
// empty EbookSelfService passed in is the shape a real GET returns: the whole
// <notification> family absent.
func TestFlattenEbook_StickyFieldsIgnoreDrift(t *testing.T) {
	t.Parallel()
	general := &EbookGeneralModel{
		FileType:        types.StringValue("ePub"),
		DeployAsManaged: types.BoolValue(false),
	}
	flattenEbookGeneral(&proclassic.EbookGeneral{
		Name:            new("guide"),
		FileType:        new("EPUB"),
		DeployAsManaged: new(true),
	}, general)
	if got := general.FileType.ValueString(); got != "ePub" {
		t.Errorf("file_type: sticky read must keep the caller's casing, got %q", got)
	}
	if general.DeployAsManaged.ValueBool() {
		t.Error("deploy_as_managed: sticky read must keep false")
	}

	ss := &EbookSelfServiceModel{
		NotificationEnabled: types.BoolValue(true),
		NotificationMethod:  types.StringValue("Self Service"),
		NotificationSubject: types.StringValue("state subject"),
		NotificationMessage: types.StringValue("state message"),
	}
	flattenEbookSelfService(&proclassic.EbookSelfService{}, ss)
	for _, tc := range []struct{ name, want, got string }{
		{"notification_method", "Self Service", ss.NotificationMethod.ValueString()},
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
