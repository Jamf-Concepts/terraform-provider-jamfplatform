// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package licensed_software

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildInput_OmitDoesNotEmitWrapper(t *testing.T) {
	// Opt-out: a nil (omitted) list must NOT emit its wrapper, so the classic
	// merge PUT retains the server's copy.
	plan := LicensedSoftwareResourceModel{
		Name:     types.StringValue("Unmanaged"),
		Platform: types.StringValue("Any"),
		// SoftwareDefinitions and Licenses left nil (omitted).
	}
	out, diags := buildLicensedSoftwareInput(plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if out.SoftwareDefinitions != nil {
		t.Error("omitted software_definitions must NOT emit a wrapper (retain on PUT)")
	}
	if out.Licenses != nil {
		t.Error("omitted licenses must NOT emit a wrapper (retain on PUT)")
	}
}

func TestBuildInput_EmptyListEmitsClearingWrapper(t *testing.T) {
	// Opt-out: an explicit [] must emit an empty wrapper so the server clears.
	plan := LicensedSoftwareResourceModel{
		Name:                types.StringValue("Cleared"),
		SoftwareDefinitions: []LicensedSoftwareDefinitionModel{},
		Licenses:            []LicensedSoftwareLicenseModel{},
	}
	out, diags := buildLicensedSoftwareInput(plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if out.SoftwareDefinitions == nil || out.SoftwareDefinitions.Definition != nil {
		t.Error("[] software_definitions must emit a wrapper with a nil definition slice (clear)")
	}
	if out.Licenses == nil || out.Licenses.License != nil {
		t.Error("[] licenses must emit a wrapper with a nil license slice (clear)")
	}
}

func TestBuildInput_GeneralAndSite(t *testing.T) {
	plan := LicensedSoftwareResourceModel{
		Name:                             types.StringValue("Acme"),
		Publisher:                        types.StringValue("Acme Corp"),
		Platform:                         types.StringValue("Mac"),
		SendEmailOnViolation:             types.BoolValue(true),
		RemoveTitlesFromInventoryReports: types.BoolValue(false),
		SiteID:                           types.StringValue("5"),
	}
	out, _ := buildLicensedSoftwareInput(plan)
	if out.General == nil {
		t.Fatal("general nil")
	}
	if out.General.Name == nil || *out.General.Name != "Acme" {
		t.Errorf("name = %v", out.General.Name)
	}
	if out.General.Platform == nil || *out.General.Platform != "Mac" {
		t.Errorf("platform = %v", out.General.Platform)
	}
	if out.General.Site == nil || out.General.Site.ID == nil || *out.General.Site.ID != 5 {
		t.Errorf("site id = %v", out.General.Site)
	}
}

func TestBuildPurchasing_LicenseTermExpandsToBoolPair(t *testing.T) {
	cases := []struct {
		term          string
		wantPerpetual bool
		wantAnnual    bool
	}{
		{"perpetual", true, false},
		{"annual", false, true},
	}
	for _, c := range cases {
		t.Run(c.term, func(t *testing.T) {
			p := &LicensedSoftwarePurchasingModel{
				LicenseTerm:    types.StringValue(c.term),
				PoDate:         types.StringValue("2026-01-01"),
				LifeExpectancy: types.Int64Value(2),
			}
			out := buildPurchasing(p)
			if out.IsPerpetual == nil || *out.IsPerpetual != c.wantPerpetual {
				t.Errorf("is_perpetual = %v, want %v", out.IsPerpetual, c.wantPerpetual)
			}
			if out.IsAnnual == nil || *out.IsAnnual != c.wantAnnual {
				t.Errorf("is_annual = %v, want %v", out.IsAnnual, c.wantAnnual)
			}
			// Server-derived echoes must never be sent.
			if out.PoDateEpoch != nil || out.PoDateUtc != nil || out.LicenseExpiresEpoch != nil || out.LicenseExpiresUtc != nil {
				t.Error("server-derived epoch/utc fields must not be sent on write")
			}
			if out.LifeExpectancy == nil || *out.LifeExpectancy != 2 {
				t.Errorf("life_expectancy = %v, want 2", out.LifeExpectancy)
			}
		})
	}
}

func TestBuildInput_DefinitionsAndLicensesPositional(t *testing.T) {
	plan := LicensedSoftwareResourceModel{
		Name: types.StringValue("WithLists"),
		SoftwareDefinitions: []LicensedSoftwareDefinitionModel{
			{Name: types.StringValue("AppA"), Version: types.StringValue("1"), CompareType: types.StringValue("is")},
			{Name: types.StringValue("AppB"), CompareType: types.StringValue("like")},
		},
		Licenses: []LicensedSoftwareLicenseModel{
			{SerialNumber1: types.StringValue("S1"), LicenseType: types.StringValue("Standard"), LicenseCount: types.Int64Value(0)},
		},
	}
	out, _ := buildLicensedSoftwareInput(plan)
	if out.SoftwareDefinitions.Definition == nil || len(*out.SoftwareDefinitions.Definition) != 2 {
		t.Fatalf("want 2 definitions")
	}
	if d := (*out.SoftwareDefinitions.Definition)[0]; d.Name == nil || *d.Name != "AppA" {
		t.Errorf("definition[0].name = %v", d.Name)
	}
	if out.Licenses.License == nil || len(*out.Licenses.License) != 1 {
		t.Fatalf("want 1 license")
	}
	if l := (*out.Licenses.License)[0]; l.LicenseCount == nil || *l.LicenseCount != 0 {
		t.Errorf("license[0].license_count = %v, want 0", l.LicenseCount)
	}
}
