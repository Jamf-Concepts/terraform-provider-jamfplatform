// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_enrollment_profile

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildInput_GeneralAlwaysEmitsDescription(t *testing.T) {
	// Merge semantics: a null Optional string is emitted as empty (to clear), not omitted.
	plan := EnrollmentProfileResourceModel{
		Name:        types.StringValue("Configurator"),
		Description: types.StringNull(),
	}
	in := buildEnrollmentProfileInput(plan)
	if in.General.Name == nil || *in.General.Name != "Configurator" {
		t.Fatalf("name not carried: %v", in.General.Name)
	}
	if in.General.Description == nil || *in.General.Description != "" {
		t.Errorf("null description must emit empty string (clear), got %v", in.General.Description)
	}
	if in.General.Site != nil {
		t.Errorf("null site_id must omit site, got %+v", in.General.Site)
	}
}

func TestBuildInput_SiteFromID(t *testing.T) {
	plan := EnrollmentProfileResourceModel{Name: types.StringValue("x"), SiteID: types.StringValue("70")}
	in := buildEnrollmentProfileInput(plan)
	if in.General.Site == nil || in.General.Site.ID == nil || *in.General.Site.ID != 70 {
		t.Errorf("site id not carried, got %+v", in.General.Site)
	}
}

func TestBuildInput_OmitsBlocksWhenNil(t *testing.T) {
	plan := EnrollmentProfileResourceModel{Name: types.StringValue("x")}
	in := buildEnrollmentProfileInput(plan)
	if in.Location != nil {
		t.Errorf("nil location model must omit block, got %+v", in.Location)
	}
	if in.Purchasing != nil {
		t.Errorf("nil purchasing model must omit block, got %+v", in.Purchasing)
	}
}

func TestBuildInput_LocationAlwaysEmitsAndMirrorsLegacy(t *testing.T) {
	plan := EnrollmentProfileResourceModel{
		Name: types.StringValue("x"),
		Location: &LocationModel{
			Username:    types.StringValue("alice"),
			RealName:    types.StringValue("Alice A"),
			PhoneNumber: types.StringValue("555"),
			Room:        types.StringNull(), // cleared
		},
	}
	in := buildEnrollmentProfileInput(plan)
	if in.Location == nil {
		t.Fatal("expected location block")
	}
	if in.Location.Username == nil || *in.Location.Username != "alice" {
		t.Error("username not carried")
	}
	// real_name mirrored to both real_name and realname
	if in.Location.RealName == nil || *in.Location.RealName != "Alice A" || in.Location.Realname == nil || *in.Location.Realname != "Alice A" {
		t.Error("real_name must mirror to both real_name and realname")
	}
	// phone_number mirrored to both phone_number and phone
	if in.Location.PhoneNumber == nil || *in.Location.PhoneNumber != "555" || in.Location.Phone == nil || *in.Location.Phone != "555" {
		t.Error("phone_number must mirror to both phone_number and phone")
	}
	// null room emitted as empty (clear)
	if in.Location.Room == nil || *in.Location.Room != "" {
		t.Errorf("null room must emit empty, got %v", in.Location.Room)
	}
}

func TestBuildInput_PurchasingBoolsOmitWhenUnknown(t *testing.T) {
	plan := EnrollmentProfileResourceModel{
		Name: types.StringValue("x"),
		Purchasing: &PurchasingModel{
			IsPurchased: types.BoolUnknown(),
			IsLeased:    types.BoolNull(),
			Vendor:      types.StringValue("Acme"),
		},
	}
	in := buildEnrollmentProfileInput(plan)
	if in.Purchasing == nil {
		t.Fatal("expected purchasing block")
	}
	if in.Purchasing.IsPurchased != nil || in.Purchasing.IsLeased != nil {
		t.Error("unknown/null bools must omit (let server default/retain)")
	}
	if in.Purchasing.Vendor == nil || *in.Purchasing.Vendor != "Acme" {
		t.Error("vendor not carried")
	}
}
