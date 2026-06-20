// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ibeacon

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildIbeaconInput_ConcreteMajorMinor(t *testing.T) {
	plan := IbeaconResourceModel{
		Name:                 types.StringValue("Reception"),
		UUID:                 types.StringValue("759b0599-64e0-416a-8d31-d8e93482a4d7"),
		Major:                types.Int64Value(1),
		Minor:                types.Int64Value(2),
		IncludeAnyMajorValue: types.BoolValue(false),
		IncludeAnyMinorValue: types.BoolValue(false),
	}
	got := buildIbeaconInput(plan)

	if got.Name == nil || *got.Name != "Reception" {
		t.Errorf("expected Name Reception, got %v", got.Name)
	}
	if got.UUID == nil || *got.UUID != "759b0599-64e0-416a-8d31-d8e93482a4d7" {
		t.Errorf("expected UUID set, got %v", got.UUID)
	}
	if got.Major == nil || *got.Major != "1" {
		t.Errorf("expected Major \"1\", got %v", got.Major)
	}
	if got.Minor == nil || *got.Minor != "2" {
		t.Errorf("expected Minor \"2\", got %v", got.Minor)
	}
	if got.ID != nil {
		t.Errorf("expected nil ID on write payload, got %d", *got.ID)
	}
}

func TestBuildIbeaconInput_BothAxesAny(t *testing.T) {
	plan := IbeaconResourceModel{
		Name:                 types.StringValue("Match All"),
		UUID:                 types.StringValue("759b0599-64e0-416a-8d31-d8e93482a4d7"),
		Major:                types.Int64Null(),
		Minor:                types.Int64Null(),
		IncludeAnyMajorValue: types.BoolValue(true),
		IncludeAnyMinorValue: types.BoolValue(true),
	}
	got := buildIbeaconInput(plan)

	if got.Major == nil || *got.Major != "-1" {
		t.Errorf("expected Major sentinel \"-1\" when IncludeAnyMajor=true, got %v", got.Major)
	}
	if got.Minor == nil || *got.Minor != "-1" {
		t.Errorf("expected Minor sentinel \"-1\" when IncludeAnyMinor=true, got %v", got.Minor)
	}
}

func TestBuildIbeaconInput_AnyMajorConcreteMinor(t *testing.T) {
	// Independent axes: any major + specific minor is a valid wire shape.
	plan := IbeaconResourceModel{
		Name:                 types.StringValue("Mixed"),
		UUID:                 types.StringValue("759b0599-64e0-416a-8d31-d8e93482a4d7"),
		Major:                types.Int64Null(),
		Minor:                types.Int64Value(5),
		IncludeAnyMajorValue: types.BoolValue(true),
		IncludeAnyMinorValue: types.BoolValue(false),
	}
	got := buildIbeaconInput(plan)

	if got.Major == nil || *got.Major != "-1" {
		t.Errorf("expected Major sentinel \"-1\", got %v", got.Major)
	}
	if got.Minor == nil || *got.Minor != "5" {
		t.Errorf("expected Minor \"5\", got %v", got.Minor)
	}
}

func TestBuildIbeaconInput_ConcreteMajorAnyMinor(t *testing.T) {
	plan := IbeaconResourceModel{
		Name:                 types.StringValue("Mixed Reverse"),
		UUID:                 types.StringValue("759b0599-64e0-416a-8d31-d8e93482a4d7"),
		Major:                types.Int64Value(42),
		Minor:                types.Int64Null(),
		IncludeAnyMajorValue: types.BoolValue(false),
		IncludeAnyMinorValue: types.BoolValue(true),
	}
	got := buildIbeaconInput(plan)

	if got.Major == nil || *got.Major != "42" {
		t.Errorf("expected Major \"42\", got %v", got.Major)
	}
	if got.Minor == nil || *got.Minor != "-1" {
		t.Errorf("expected Minor sentinel \"-1\", got %v", got.Minor)
	}
}

func TestBuildIbeaconInput_IncludeAnyTrumpsConcreteValues(t *testing.T) {
	// If by some path the plan carries both include_any=true and a concrete
	// value on the same axis, the wire must reflect include_any. Validators
	// should reject this at plan time but defence-in-depth: builder respects
	// the bool. Major-only test; minor symmetry is exercised by the validator
	// truth table in helpers_test.
	plan := IbeaconResourceModel{
		Name:                 types.StringValue("Conflicted"),
		UUID:                 types.StringValue("759b0599-64e0-416a-8d31-d8e93482a4d7"),
		Major:                types.Int64Value(10),
		Minor:                types.Int64Value(20),
		IncludeAnyMajorValue: types.BoolValue(true),
		IncludeAnyMinorValue: types.BoolValue(false),
	}
	got := buildIbeaconInput(plan)

	if got.Major == nil || *got.Major != "-1" {
		t.Errorf("IncludeAnyMajor=true must win — expected Major=\"-1\", got %v", got.Major)
	}
	if got.Minor == nil || *got.Minor != "20" {
		t.Errorf("expected Minor=\"20\" when IncludeAnyMinor=false, got %v", got.Minor)
	}
}

func TestBuildIbeaconInput_NullMajorMinorOmitsFields(t *testing.T) {
	plan := IbeaconResourceModel{
		Name:                 types.StringValue("Half-Built"),
		UUID:                 types.StringValue("759b0599-64e0-416a-8d31-d8e93482a4d7"),
		Major:                types.Int64Null(),
		Minor:                types.Int64Unknown(),
		IncludeAnyMajorValue: types.BoolValue(false),
		IncludeAnyMinorValue: types.BoolValue(false),
	}
	got := buildIbeaconInput(plan)

	if got.Major != nil {
		t.Errorf("null Major must serialise to nil, got %v", *got.Major)
	}
	if got.Minor != nil {
		t.Errorf("unknown Minor must serialise to nil, got %v", *got.Minor)
	}
}

func TestBuildIbeaconInput_IncludeAnyNullTreatedAsFalse(t *testing.T) {
	plan := IbeaconResourceModel{
		Name:                 types.StringValue("Default"),
		UUID:                 types.StringValue("759b0599-64e0-416a-8d31-d8e93482a4d7"),
		Major:                types.Int64Value(0),
		Minor:                types.Int64Value(0),
		IncludeAnyMajorValue: types.BoolNull(),
		IncludeAnyMinorValue: types.BoolNull(),
	}
	got := buildIbeaconInput(plan)

	// Null IncludeAny* (e.g. plan-modifier hasn't applied yet) must NOT
	// trigger the sentinel — treated as false.
	if got.Major == nil || *got.Major != "0" {
		t.Errorf("expected Major=\"0\" when IncludeAnyMajor=null, got %v", got.Major)
	}
	if got.Minor == nil || *got.Minor != "0" {
		t.Errorf("expected Minor=\"0\" when IncludeAnyMinor=null, got %v", got.Minor)
	}
}
