// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_branding_macos

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildSelfServiceBrandingMacosInput_OmitsNulls(t *testing.T) {
	// All-null plan ⇒ every pointer nil (omitempty drops them all).
	plan := SelfServiceBrandingMacosResourceModel{
		ApplicationHeader:  types.StringNull(),
		SidebarHeading:     types.StringNull(),
		SidebarSubheading:  types.StringNull(),
		HomePageHeading:    types.StringNull(),
		HomePageSubheading: types.StringNull(),
		IconID:             types.Int64Null(),
		BannerImageID:      types.Int64Null(),
	}
	got := buildSelfServiceBrandingMacosInput(plan)
	if got.ApplicationName != nil || got.BrandingName != nil || got.BrandingNameSecondary != nil ||
		got.HomeHeading != nil || got.HomeSubheading != nil || got.IconID != nil || got.BrandingHeaderImageID != nil {
		t.Fatalf("expected all-nil pointers for all-null plan, got %+v", got)
	}
}

func TestBuildSelfServiceBrandingMacosInput_SetsValues(t *testing.T) {
	plan := SelfServiceBrandingMacosResourceModel{
		ApplicationHeader:  types.StringValue("App"),
		SidebarHeading:     types.StringValue("Side"),
		SidebarSubheading:  types.StringValue("Sub"),
		HomePageHeading:    types.StringValue("Home"),
		HomePageSubheading: types.StringValue("HomeSub"),
		IconID:             types.Int64Value(81),
		BannerImageID:      types.Int64Value(82),
	}
	got := buildSelfServiceBrandingMacosInput(plan)
	if got.ApplicationName == nil || *got.ApplicationName != "App" {
		t.Errorf("ApplicationName = %v, want App", got.ApplicationName)
	}
	if got.BrandingName == nil || *got.BrandingName != "Side" {
		t.Errorf("BrandingName = %v, want Side", got.BrandingName)
	}
	if got.IconID == nil || *got.IconID != 81 {
		t.Errorf("IconID = %v, want 81", got.IconID)
	}
	if got.BrandingHeaderImageID == nil || *got.BrandingHeaderImageID != 82 {
		t.Errorf("BrandingHeaderImageID = %v, want 82", got.BrandingHeaderImageID)
	}
}
