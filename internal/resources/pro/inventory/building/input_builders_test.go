// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package building

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildBuildingInput_AllNullsBecomeNilPointers(t *testing.T) {
	plan := BuildingResourceModel{
		Name:           types.StringValue("HQ"),
		City:           types.StringNull(),
		Country:        types.StringNull(),
		StateProvince:  types.StringNull(),
		StreetAddress1: types.StringNull(),
		StreetAddress2: types.StringNull(),
		ZipPostalCode:  types.StringNull(),
	}

	got := buildBuildingInput(plan)

	if got.Name != "HQ" {
		t.Errorf("expected Name %q, got %q", "HQ", got.Name)
	}
	if got.City != nil {
		t.Errorf("expected City nil for null input, got %v", *got.City)
	}
	if got.Country != nil {
		t.Errorf("expected Country nil for null input, got %v", *got.Country)
	}
	if got.StateProvince != nil {
		t.Errorf("expected StateProvince nil for null input, got %v", *got.StateProvince)
	}
	if got.StreetAddress1 != nil {
		t.Errorf("expected StreetAddress1 nil for null input, got %v", *got.StreetAddress1)
	}
	if got.StreetAddress2 != nil {
		t.Errorf("expected StreetAddress2 nil for null input, got %v", *got.StreetAddress2)
	}
	if got.ZipPostalCode != nil {
		t.Errorf("expected ZipPostalCode nil for null input, got %v", *got.ZipPostalCode)
	}
}

func TestBuildBuildingInput_UnknownsBecomeNilPointers(t *testing.T) {
	plan := BuildingResourceModel{
		Name:           types.StringValue("Branch"),
		City:           types.StringUnknown(),
		Country:        types.StringUnknown(),
		StateProvince:  types.StringUnknown(),
		StreetAddress1: types.StringUnknown(),
		StreetAddress2: types.StringUnknown(),
		ZipPostalCode:  types.StringUnknown(),
	}

	got := buildBuildingInput(plan)

	if got.City != nil || got.Country != nil || got.StateProvince != nil ||
		got.StreetAddress1 != nil || got.StreetAddress2 != nil || got.ZipPostalCode != nil {
		t.Errorf("expected all optional pointers nil for Unknown input, got %+v", got)
	}
}

func TestBuildBuildingInput_AllFieldsPopulated(t *testing.T) {
	plan := BuildingResourceModel{
		Name:           types.StringValue("Campus"),
		City:           types.StringValue("Minneapolis"),
		Country:        types.StringValue("USA"),
		StateProvince:  types.StringValue("MN"),
		StreetAddress1: types.StringValue("100 Washington Ave S"),
		StreetAddress2: types.StringValue("Suite 1100"),
		ZipPostalCode:  types.StringValue("55401"),
	}

	got := buildBuildingInput(plan)

	cases := []struct {
		field    string
		got      *string
		expected string
	}{
		{"City", got.City, "Minneapolis"},
		{"Country", got.Country, "USA"},
		{"StateProvince", got.StateProvince, "MN"},
		{"StreetAddress1", got.StreetAddress1, "100 Washington Ave S"},
		{"StreetAddress2", got.StreetAddress2, "Suite 1100"},
		{"ZipPostalCode", got.ZipPostalCode, "55401"},
	}
	for _, c := range cases {
		if c.got == nil {
			t.Errorf("%s: expected non-nil pointer, got nil", c.field)
			continue
		}
		if *c.got != c.expected {
			t.Errorf("%s: expected %q, got %q", c.field, c.expected, *c.got)
		}
	}
}
