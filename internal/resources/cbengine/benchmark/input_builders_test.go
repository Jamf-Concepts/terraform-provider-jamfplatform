// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// setOfStrings constructs a non-null types.Set of string elements for test setup.
func setOfStrings(ids ...string) types.Set {
	vals := make([]attr.Value, len(ids))
	for i, id := range ids {
		vals[i] = types.StringValue(id)
	}
	out, _ := types.SetValue(types.StringType, vals)
	return out
}

// setOfOsVersions constructs a non-null types.Set of {os_type, os_version}
// objects (all MAC_OS) for test setup.
func setOfOsVersions(vs ...int64) types.Set {
	vals := make([]attr.Value, len(vs))
	for i, v := range vs {
		obj, _ := types.ObjectValue(osVersionObjectType.AttrTypes, map[string]attr.Value{
			"os_type":    types.StringValue("MAC_OS"),
			"os_version": types.Int64Value(v),
		})
		vals[i] = obj
	}
	out, _ := types.SetValue(osVersionObjectType, vals)
	return out
}

func TestBuildBenchmarkRequest_Full(t *testing.T) {
	data := &BenchmarkResourceModel{
		Title:              types.StringValue("My Benchmark"),
		Description:        types.StringValue("My Description"),
		SourceBaselineID:   types.StringValue("baseline-1"),
		SelectedOsVersions: setOfOsVersions(26, 15),
		Rules: []RuleModel{
			{
				ID:       types.StringValue("rule-1"),
				Enabled:  types.BoolValue(true),
				ODVValue: types.StringValue("custom-value"),
			},
			{
				ID:       types.StringValue("rule-2"),
				Enabled:  types.BoolValue(false),
				ODVValue: types.StringNull(),
			},
		},
		TargetDeviceGroups: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("group-1")}),
		EnforcementMode:    types.StringValue("audit"),
	}

	req := buildBenchmarkRequest(data)

	if req.Title != "My Benchmark" {
		t.Errorf("expected Title 'My Benchmark', got %q", req.Title)
	}
	if req.Description == nil || *req.Description != "My Description" {
		t.Errorf("expected Description 'My Description', got %v", req.Description)
	}
	if req.SourceBaselineID != "baseline-1" {
		t.Errorf("expected SourceBaselineID 'baseline-1', got %q", req.SourceBaselineID)
	}
	if req.EnforcementMode != "audit" {
		t.Errorf("expected EnforcementMode 'audit', got %q", req.EnforcementMode)
	}
	if len(req.Target.DeviceGroups) != 1 || req.Target.DeviceGroups[0] != "group-1" {
		t.Errorf("expected target device group ['group-1'], got %v", req.Target.DeviceGroups)
	}
	if req.SelectedOsVersions == nil {
		t.Fatal("expected SelectedOsVersions to be non-nil")
	}
	if len(*req.SelectedOsVersions) != 2 {
		t.Fatalf("expected 2 selected OS versions, got %d", len(*req.SelectedOsVersions))
	}
	for _, v := range *req.SelectedOsVersions {
		if v.OsType != "MAC_OS" {
			t.Errorf("expected osType MAC_OS, got %q", v.OsType)
		}
		if v.OsVersion != 26 && v.OsVersion != 15 {
			t.Errorf("unexpected osVersion %d", v.OsVersion)
		}
	}
	if len(req.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(req.Rules))
	}
	if req.Rules[0].ID != "rule-1" {
		t.Errorf("expected rule[0] ID 'rule-1', got %q", req.Rules[0].ID)
	}
	if req.Rules[0].Enabled != true {
		t.Errorf("expected rule[0] Enabled true, got %v", req.Rules[0].Enabled)
	}
	if req.Rules[0].ODV == nil {
		t.Fatal("expected rule[0] ODV to be non-nil")
	}
	if req.Rules[0].ODV.Value != "custom-value" {
		t.Errorf("expected rule[0] ODV value 'custom-value', got %q", req.Rules[0].ODV.Value)
	}
	if req.Rules[1].Enabled != false {
		t.Errorf("expected rule[1] Enabled false, got %v", req.Rules[1].Enabled)
	}
	if req.Rules[1].ODV != nil {
		t.Errorf("expected rule[1] ODV to be nil for null ODVValue, got %+v", req.Rules[1].ODV)
	}
}

func TestBuildBenchmarkRequest_EmptyRulesAndOmittedOsVersions(t *testing.T) {
	data := &BenchmarkResourceModel{
		Title:              types.StringValue("Empty"),
		Description:        types.StringValue(""),
		SourceBaselineID:   types.StringValue("bl-1"),
		SelectedOsVersions: types.SetNull(osVersionObjectType),
		Rules:              nil,
		TargetDeviceGroups: types.SetNull(types.StringType),
		EnforcementMode:    types.StringValue("audit"),
	}

	req := buildBenchmarkRequest(data)

	if req.SelectedOsVersions != nil {
		t.Errorf("expected nil SelectedOsVersions when attribute is null (server defaults to all), got %v", *req.SelectedOsVersions)
	}
	if len(req.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(req.Rules))
	}
}

func TestBuildSelectedOsVersionsRequest(t *testing.T) {
	// Unknown → nil (omit; server defaults to all available versions).
	unknown := &BenchmarkResourceModel{SelectedOsVersions: types.SetUnknown(osVersionObjectType)}
	if got := buildSelectedOsVersionsRequest(unknown); got != nil {
		t.Errorf("expected nil for unknown set, got %v", *got)
	}
	// Configured subset → pairs preserving os_type + os_version.
	set := &BenchmarkResourceModel{SelectedOsVersions: setOfOsVersions(26)}
	got := buildSelectedOsVersionsRequest(set)
	if got == nil || len(*got) != 1 {
		t.Fatalf("expected 1 selected OS version, got %v", got)
	}
	if (*got)[0].OsType != "MAC_OS" || (*got)[0].OsVersion != 26 {
		t.Errorf("expected {MAC_OS,26}, got %+v", (*got)[0])
	}
}

func TestBuildBenchmarkRequest_ODVNull(t *testing.T) {
	data := &BenchmarkResourceModel{
		Title:            types.StringValue("ODV Null"),
		SourceBaselineID: types.StringValue("bl-1"),
		Rules: []RuleModel{
			{
				ID:       types.StringValue("rule-u"),
				Enabled:  types.BoolValue(true),
				ODVValue: types.StringNull(),
			},
		},
		TargetDeviceGroups: types.SetNull(types.StringType),
		EnforcementMode:    types.StringValue("audit"),
	}

	req := buildBenchmarkRequest(data)

	if req.Rules[0].ODV != nil {
		t.Error("expected nil ODV for null ODVValue")
	}
}

func TestBuildBenchmarkRequest_ODVEmpty(t *testing.T) {
	data := &BenchmarkResourceModel{
		Title:            types.StringValue("ODV Empty"),
		SourceBaselineID: types.StringValue("bl-1"),
		Rules: []RuleModel{
			{
				ID:       types.StringValue("rule-u"),
				Enabled:  types.BoolValue(true),
				ODVValue: types.StringValue(""),
			},
		},
		TargetDeviceGroups: types.SetNull(types.StringType),
		EnforcementMode:    types.StringValue("audit"),
	}

	req := buildBenchmarkRequest(data)

	if req.Rules[0].ODV != nil {
		t.Error("expected nil ODV for empty ODVValue — API rejects odv:{value:\"\"}")
	}
}

func TestBuildBenchmarkRequest_PluralTargetDeviceGroups(t *testing.T) {
	data := &BenchmarkResourceModel{
		Title:              types.StringValue("Plural Scope"),
		SourceBaselineID:   types.StringValue("bl-1"),
		Rules:              nil,
		TargetDeviceGroups: setOfStrings("group-a", "group-b", "group-c"),
		EnforcementMode:    types.StringValue("audit"),
	}

	req := buildBenchmarkRequest(data)

	if len(req.Target.DeviceGroups) != 3 {
		t.Fatalf("expected 3 device groups, got %d (%v)", len(req.Target.DeviceGroups), req.Target.DeviceGroups)
	}
	seen := map[string]bool{}
	for _, g := range req.Target.DeviceGroups {
		seen[g] = true
	}
	for _, want := range []string{"group-a", "group-b", "group-c"} {
		if !seen[want] {
			t.Errorf("expected device group %q in payload, got %v", want, req.Target.DeviceGroups)
		}
	}
}

func TestBuildBenchmarkRequest_PluralWinsOverSingular(t *testing.T) {
	// ConflictsWith makes this impossible in practice, but the builder must still
	// prefer the plural path when both happen to be set — defensive behaviour for
	// older imported state or programmatic callers.
	data := &BenchmarkResourceModel{
		Title:              types.StringValue("Both"),
		SourceBaselineID:   types.StringValue("bl-1"),
		TargetDeviceGroups: setOfStrings("kept"),
		EnforcementMode:    types.StringValue("audit"),
	}

	req := buildBenchmarkRequest(data)

	if len(req.Target.DeviceGroups) != 1 || req.Target.DeviceGroups[0] != "kept" {
		t.Errorf("expected plural to win with ['kept'], got %v", req.Target.DeviceGroups)
	}
}
