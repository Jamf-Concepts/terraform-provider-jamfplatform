// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package network_segment

import (
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildNetworkSegmentInput_FullPlan(t *testing.T) {
	plan := NetworkSegmentResourceModel{
		Name:                types.StringValue("HQ"),
		StartingAddress:     types.StringValue("10.0.0.0"),
		EndingAddress:       types.StringValue("10.0.0.255"),
		Building:            types.StringValue("HQ Building"),
		Department:          types.StringValue("IT"),
		OverrideBuildings:   types.BoolValue(true),
		OverrideDepartments: types.BoolValue(false),
	}
	got := buildNetworkSegmentInput(plan)

	if got.Name == nil || *got.Name != "HQ" {
		t.Errorf("expected Name HQ, got %v", got.Name)
	}
	if got.StartingAddress == nil || *got.StartingAddress != "10.0.0.0" {
		t.Errorf("expected StartingAddress 10.0.0.0, got %v", got.StartingAddress)
	}
	if got.EndingAddress == nil || *got.EndingAddress != "10.0.0.255" {
		t.Errorf("expected EndingAddress 10.0.0.255, got %v", got.EndingAddress)
	}
	if got.Building == nil || *got.Building != "HQ Building" {
		t.Errorf("expected Building HQ Building, got %v", got.Building)
	}
	if got.Department == nil || *got.Department != "IT" {
		t.Errorf("expected Department IT, got %v", got.Department)
	}
	if got.OverrideBuildings == nil || *got.OverrideBuildings != true {
		t.Errorf("expected OverrideBuildings=true, got %v", got.OverrideBuildings)
	}
	if got.OverrideDepartments == nil || *got.OverrideDepartments != false {
		t.Errorf("expected OverrideDepartments=false, got %v", got.OverrideDepartments)
	}
	if got.ID != nil {
		t.Errorf("expected nil ID on write payload, got %d", *got.ID)
	}
}

// TestBuildNetworkSegmentInput_NullBuildingDepartmentEmitEmpty pins the
// always-emit contract for the two name references: null must reach the wire
// as an empty element (which clears) and unknown must stay off the wire
// (server-owned).
func TestBuildNetworkSegmentInput_NullBuildingDepartmentEmitEmpty(t *testing.T) {
	plan := NetworkSegmentResourceModel{
		Name:            types.StringValue("Bare"),
		StartingAddress: types.StringValue("10.0.0.0"),
		EndingAddress:   types.StringValue("10.0.0.255"),
		Building:        types.StringNull(),
		Department:      types.StringUnknown(),
	}
	got := buildNetworkSegmentInput(plan)

	if got.Building == nil || *got.Building != "" {
		t.Errorf("null Building must serialise to an empty element, got %v", got.Building)
	}
	if got.Department != nil {
		t.Errorf("unknown Department must serialise to nil, got %q", *got.Department)
	}
}

// TestBuildNetworkSegmentInput_NullOverridesEmitFalse pins the always-emit
// contract for the two override flags: a null flag must reach the wire as an
// explicit false so a flag the user removed is turned off, not retained.
func TestBuildNetworkSegmentInput_NullOverridesEmitFalse(t *testing.T) {
	plan := NetworkSegmentResourceModel{
		Name:                types.StringValue("Branch"),
		StartingAddress:     types.StringValue("192.168.1.0"),
		EndingAddress:       types.StringValue("192.168.1.255"),
		OverrideBuildings:   types.BoolNull(),
		OverrideDepartments: types.BoolUnknown(),
	}
	got := buildNetworkSegmentInput(plan)

	if got.OverrideBuildings == nil || *got.OverrideBuildings {
		t.Errorf("null OverrideBuildings must serialise to an explicit false, got %v", got.OverrideBuildings)
	}
	if got.OverrideDepartments != nil {
		t.Errorf("unknown OverrideDepartments must serialise to nil, got %v", *got.OverrideDepartments)
	}
}

func TestOptionalBoolPointer(t *testing.T) {
	cases := []struct {
		name string
		in   types.Bool
		want *bool
	}{
		{"null", types.BoolNull(), nil},
		{"unknown", types.BoolUnknown(), nil},
		{"true", types.BoolValue(true), new(true)},
		{"false", types.BoolValue(false), new(false)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := helpers.OptionalBoolPointer(c.in)
			switch {
			case c.want == nil && got != nil:
				t.Errorf("expected nil, got %v", *got)
			case c.want != nil && got == nil:
				t.Errorf("expected %v, got nil", *c.want)
			case c.want != nil && got != nil && *c.want != *got:
				t.Errorf("expected %v, got %v", *c.want, *got)
			}
		})
	}
}
