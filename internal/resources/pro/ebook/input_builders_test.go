// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ebook

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func idSet(ids ...string) types.Set {
	vals := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, types.StringValue(id))
	}
	return types.SetValueMust(types.StringType, vals)
}

func nullSet() types.Set { return types.SetNull(types.StringType) }

func TestBuildEbookGeneral(t *testing.T) {
	m := &EbookGeneralModel{
		Name:            types.StringValue("Field Guide"),
		Author:          types.StringValue("J. Doe"),
		URL:             types.StringValue("https://example.org/guide.pdf"),
		DeploymentType:  types.StringValue(deploymentTypeSelfService),
		DeployAsManaged: types.BoolValue(true),
		Free:            types.BoolValue(false),
		FileType:        types.StringValue("PDF"),
		Version:         types.StringValue("1.0"),
		CategoryID:      types.StringValue("7"),
		SiteID:          types.StringValue("3"),
	}
	g := buildEbookGeneral(m)
	if g.Name == nil || *g.Name != "Field Guide" {
		t.Errorf("name not mapped: %+v", g.Name)
	}
	if g.URL == nil || *g.URL != "https://example.org/guide.pdf" {
		t.Errorf("url not mapped")
	}
	if g.DeploymentType == nil || *g.DeploymentType != deploymentTypeSelfService {
		t.Errorf("deployment_type not mapped")
	}
	if g.FileType == nil || *g.FileType != "PDF" {
		t.Errorf("file_type not mapped")
	}
	if g.DeployAsManaged == nil || !*g.DeployAsManaged {
		t.Errorf("deploy_as_managed not mapped")
	}
	if g.Category == nil || g.Category.ID == nil || *g.Category.ID != 7 {
		t.Errorf("category id not mapped: %+v", g.Category)
	}
	if g.Site == nil || g.Site.ID == nil || *g.Site.ID != 3 {
		t.Errorf("site id not mapped: %+v", g.Site)
	}
}

func TestBuildEbookNotification(t *testing.T) {
	if n := buildEbookNotification(types.BoolNull(), types.StringNull()); n != nil {
		t.Errorf("expected nil notification when unconfigured, got %+v", n)
	}
	n := buildEbookNotification(types.BoolValue(true), types.StringValue("Self Service"))
	if n == nil || n.Enabled == nil || !*n.Enabled || n.Method == nil || *n.Method != "Self Service" {
		t.Fatalf("notification legs not assembled: %+v", n)
	}
}

func emptyEbookScope() *EbookScopeModel {
	return &EbookScopeModel{
		Targets: &EbookScopeTargetsModel{
			AllComputers:         types.BoolNull(),
			AllMobileDevices:     types.BoolNull(),
			AllJssUsers:          types.BoolNull(),
			ComputerIDs:          nullSet(),
			ComputerGroupIDs:     nullSet(),
			MobileDeviceIDs:      nullSet(),
			MobileDeviceGroupIDs: nullSet(),
			BuildingIDs:          nullSet(),
			DepartmentIDs:        nullSet(),
			UserIDs:              nullSet(),
			UserGroupIDs:         nullSet(),
			ClassIDs:             nullSet(),
		},
	}
}

func TestBuildEbookScope_Collapse(t *testing.T) {
	ctx := context.Background()
	s, diags := buildEbookScope(ctx, emptyEbookScope())
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if s != nil {
		t.Errorf("expected nil scope for empty model, got %+v", s)
	}
}

func TestBuildEbookScope_DualTargetUnionAndClasses(t *testing.T) {
	ctx := context.Background()
	m := emptyEbookScope()
	m.Targets.AllComputers = types.BoolValue(false)
	m.Targets.ComputerIDs = idSet("11", "12")
	m.Targets.ComputerGroupIDs = idSet("5")
	m.Targets.MobileDeviceIDs = idSet("21")
	m.Targets.MobileDeviceGroupIDs = idSet("6", "7")
	m.Targets.BuildingIDs = idSet("3")
	m.Targets.UserIDs = idSet("31")
	m.Targets.ClassIDs = idSet("41", "42")
	m.Limitations = &EbookScopeLimitationsModel{
		NetworkSegmentIDs:                idSet("2"),
		DirectoryServiceOrLocalUserNames: idSetNames("alice"),
		DirectoryServiceUserGroupNames:   nullSet(),
	}
	m.Exclusions = &EbookScopeExclusionsModel{
		ComputerIDs:                      nullSet(),
		ComputerGroupIDs:                 nullSet(),
		MobileDeviceIDs:                  idSet("99"),
		MobileDeviceGroupIDs:             nullSet(),
		BuildingIDs:                      nullSet(),
		DepartmentIDs:                    nullSet(),
		UserIDs:                          nullSet(),
		UserGroupIDs:                     nullSet(),
		NetworkSegmentIDs:                nullSet(),
		DirectoryServiceOrLocalUserNames: idSetNames("bob"),
		DirectoryServiceUserGroupNames:   idSetNames("Admins"),
	}

	s, diags := buildEbookScope(ctx, m)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if s == nil {
		t.Fatalf("expected non-nil scope")
	}
	if s.Computers == nil || s.Computers.Computer == nil || len(*s.Computers.Computer) != 2 {
		t.Errorf("computers not mapped: %+v", s.Computers)
	}
	if s.MobileDevices == nil || s.MobileDevices.MobileDevice == nil || len(*s.MobileDevices.MobileDevice) != 1 {
		t.Errorf("mobile_devices not mapped: %+v", s.MobileDevices)
	}
	if s.MobileDeviceGroups == nil || s.MobileDeviceGroups.MobileDeviceGroup == nil || len(*s.MobileDeviceGroups.MobileDeviceGroup) != 2 {
		t.Errorf("mobile_device_groups not mapped")
	}
	if s.Classes == nil || s.Classes.Class == nil || len(*s.Classes.Class) != 2 {
		t.Errorf("classes not mapped: %+v", s.Classes)
	}
	if s.JssUsers == nil || s.JssUsers.User == nil || len(*s.JssUsers.User) != 1 {
		t.Errorf("jss_users (user_ids) not mapped")
	}
	if s.Limitations == nil || s.Limitations.Users == nil || s.Limitations.Users.User == nil || len(*s.Limitations.Users.User) != 1 {
		t.Errorf("limitations users (names) not mapped")
	}
	if s.Exclusions == nil || s.Exclusions.MobileDevices == nil {
		t.Errorf("exclusions mobile_devices not mapped")
	}
	if s.Exclusions.UserGroups == nil || s.Exclusions.UserGroups.UserGroup == nil || len(*s.Exclusions.UserGroups.UserGroup) != 1 {
		t.Errorf("exclusions user_groups (names) not mapped")
	}
}

func TestBuildEbookInput_IconStampedIntoBothBlocks(t *testing.T) {
	ctx := context.Background()
	plan := EbookResourceModel{
		General: &EbookGeneralModel{
			Name: types.StringValue("Field Guide"),
			URL:  types.StringValue("https://example.org/guide.pdf"),
		},
		SelfService: &EbookSelfServiceModel{
			IconID: types.StringValue("805"),
			Categories: []EbookSelfServiceCategoryModel{
				{ID: types.StringValue("3"), DisplayIn: types.BoolValue(true), FeatureIn: types.BoolValue(false)},
			},
		},
	}
	out, diags := buildEbookInput(ctx, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if out.SelfService == nil || out.SelfService.SelfServiceIcon == nil || out.SelfService.SelfServiceIcon.ID == nil || *out.SelfService.SelfServiceIcon.ID != 805 {
		t.Errorf("self_service icon id not mapped: %+v", out.SelfService)
	}
	// The icon must also be stamped into <general> (it round-trips under both).
	if out.General == nil || out.General.SelfServiceIcon == nil || out.General.SelfServiceIcon.ID == nil || *out.General.SelfServiceIcon.ID != 805 {
		t.Errorf("general icon id not stamped: %+v", out.General)
	}
	if out.SelfService.SelfServiceCategories == nil || out.SelfService.SelfServiceCategories.Category == nil || len(*out.SelfService.SelfServiceCategories.Category) != 1 {
		t.Fatalf("categories not mapped")
	}
	c := (*out.SelfService.SelfServiceCategories.Category)[0]
	if c.ID == nil || *c.ID != 3 || c.DisplayIn == nil || !*c.DisplayIn {
		t.Errorf("category fields not mapped: %+v", c)
	}
}

// idSetNames is an alias for idSet used where the elements are names rather than
// numeric IDs, for readability in the scope tests.
func idSetNames(names ...string) types.Set { return idSet(names...) }
