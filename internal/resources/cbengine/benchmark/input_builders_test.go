// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildBenchmarkRequest_Full(t *testing.T) {
	data := &BenchmarkResourceModel{
		Title:            types.StringValue("My Benchmark"),
		Description:      types.StringValue("My Description"),
		SourceBaselineID: types.StringValue("baseline-1"),
		Sources: []SourceModel{
			{Branch: types.StringValue("main"), Revision: types.StringValue("rev-1")},
			{Branch: types.StringValue("release"), Revision: types.StringValue("rev-2")},
		},
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
		TargetDeviceGroup: types.StringValue("group-1"),
		EnforcementMode:   types.StringValue("audit"),
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
	if len(req.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(req.Sources))
	}
	if req.Sources[0].Branch != "main" {
		t.Errorf("expected source[0] branch 'main', got %q", req.Sources[0].Branch)
	}
	if req.Sources[1].Revision != "rev-2" {
		t.Errorf("expected source[1] revision 'rev-2', got %q", req.Sources[1].Revision)
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

func TestBuildBenchmarkRequest_EmptyRulesAndSources(t *testing.T) {
	data := &BenchmarkResourceModel{
		Title:             types.StringValue("Empty"),
		Description:       types.StringValue(""),
		SourceBaselineID:  types.StringValue("bl-1"),
		Sources:           nil,
		Rules:             nil,
		TargetDeviceGroup: types.StringValue("dg-1"),
		EnforcementMode:   types.StringValue("audit"),
	}

	req := buildBenchmarkRequest(data)

	if len(req.Sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(req.Sources))
	}
	if len(req.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(req.Rules))
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
		TargetDeviceGroup: types.StringValue("dg-1"),
		EnforcementMode:   types.StringValue("audit"),
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
		TargetDeviceGroup: types.StringValue("dg-1"),
		EnforcementMode:   types.StringValue("audit"),
	}

	req := buildBenchmarkRequest(data)

	if req.Rules[0].ODV != nil {
		t.Error("expected nil ODV for empty ODVValue — API rejects odv:{value:\"\"}")
	}
}
