// Copyright 2026 Jamf Software LLC.

package benchmark

import (
	"testing"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAssignBenchmarkModelFromResponse_Full(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	minVal := 1
	maxVal := 100

	model := &BenchmarkResourceModel{}
	bench := &client.CBEngineBenchmarkResponseV2{
		BenchmarkID:     "bench-1",
		TenantID:        "tenant-1",
		Title:           "Test Benchmark",
		Description:     "A test benchmark",
		EnforcementMode: "audit",
		Deleted:         false,
		UpdateAvailable: true,
		LastUpdatedAt:   ts,
		Sources: []client.CBEngineSourceV1{
			{Branch: "main", Revision: "abc123"},
		},
		Target: client.CBEngineTargetV2{
			DeviceGroups: []string{"group-1"},
		},
		Rules: []client.CBEngineRuleInfoV1{
			{
				ID:          "rule-1",
				SectionName: "Section A",
				Enabled:     true,
				Title:       "Rule Title",
				Description: "Rule Description",
				References:  []string{"ref-1", "ref-2"},
				ODV: &client.CBEngineOrganizationDefinedValueV1{
					Value:       "30",
					Hint:        "Enter a number",
					Placeholder: "30",
					Type:        "int",
					Validation: &client.CBEngineValidationConstraintsV1{
						Min:        &minVal,
						Max:        &maxVal,
						EnumValues: []string{"10", "20", "30"},
						Regex:      `^\d+$`,
					},
				},
				SupportedOS: []client.CBEngineOSInfoV1{
					{OSType: "macOS", OSVersion: 15, ManagementType: "SUPERVISED"},
				},
				OSSpecificDefaults: map[string]client.CBEngineOSSpecificRuleInfoV1{
					"macOS_15": {
						Title:       "macOS 15 Rule",
						Description: "macOS 15 specific",
						ODV: &client.CBEngineODVRecommendationV1{
							Value: "30",
							Hint:  "Recommended: 30",
						},
					},
				},
				RuleRelation: &client.CBEngineRuleRelationV1{
					DependsOn: []string{"rule-0"},
				},
			},
		},
	}

	assignBenchmarkModelFromResponse(model, bench)

	if model.ID.ValueString() != "bench-1" {
		t.Errorf("expected ID 'bench-1', got %q", model.ID.ValueString())
	}
	if model.Title.ValueString() != "Test Benchmark" {
		t.Errorf("expected Title 'Test Benchmark', got %q", model.Title.ValueString())
	}
	if model.Description.ValueString() != "A test benchmark" {
		t.Errorf("expected Description 'A test benchmark', got %q", model.Description.ValueString())
	}
	if model.TenantID.ValueString() != "tenant-1" {
		t.Errorf("expected TenantID 'tenant-1', got %q", model.TenantID.ValueString())
	}
	if model.EnforcementMode.ValueString() != "audit" {
		t.Errorf("expected EnforcementMode 'audit', got %q", model.EnforcementMode.ValueString())
	}
	if model.Deleted.ValueBool() != false {
		t.Errorf("expected Deleted false, got %v", model.Deleted.ValueBool())
	}
	if model.UpdateAvailable.ValueBool() != true {
		t.Errorf("expected UpdateAvailable true, got %v", model.UpdateAvailable.ValueBool())
	}
	if model.LastUpdatedAt.ValueString() != "2025-06-15T10:30:00Z" {
		t.Errorf("expected LastUpdatedAt '2025-06-15T10:30:00Z', got %q", model.LastUpdatedAt.ValueString())
	}
	if model.TargetDeviceGroup.ValueString() != "group-1" {
		t.Errorf("expected TargetDeviceGroup 'group-1', got %q", model.TargetDeviceGroup.ValueString())
	}
	if len(model.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(model.Sources))
	}
	if model.Sources[0].Branch.ValueString() != "main" {
		t.Errorf("expected source branch 'main', got %q", model.Sources[0].Branch.ValueString())
	}
	if model.Sources[0].Revision.ValueString() != "abc123" {
		t.Errorf("expected source revision 'abc123', got %q", model.Sources[0].Revision.ValueString())
	}

	if len(model.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(model.Rules))
	}
	rule := model.Rules[0]
	if rule.ID.ValueString() != "rule-1" {
		t.Errorf("expected rule ID 'rule-1', got %q", rule.ID.ValueString())
	}
	if rule.SectionName.ValueString() != "Section A" {
		t.Errorf("expected SectionName 'Section A', got %q", rule.SectionName.ValueString())
	}
	if rule.Enabled.ValueBool() != true {
		t.Errorf("expected Enabled true, got %v", rule.Enabled.ValueBool())
	}
	if rule.Title.ValueString() != "Rule Title" {
		t.Errorf("expected Title 'Rule Title', got %q", rule.Title.ValueString())
	}
	if rule.Description.ValueString() != "Rule Description" {
		t.Errorf("expected Description 'Rule Description', got %q", rule.Description.ValueString())
	}
	if rule.ODVValue.ValueString() != "30" {
		t.Errorf("expected ODVValue '30', got %q", rule.ODVValue.ValueString())
	}
	if rule.ODVHint.ValueString() != "Enter a number" {
		t.Errorf("expected ODVHint 'Enter a number', got %q", rule.ODVHint.ValueString())
	}
	if rule.ODVPlaceholder.ValueString() != "30" {
		t.Errorf("expected ODVPlaceholder '30', got %q", rule.ODVPlaceholder.ValueString())
	}
	if rule.ODVType.ValueString() != "int" {
		t.Errorf("expected ODVType 'int', got %q", rule.ODVType.ValueString())
	}
	if rule.ODVValidationMin.ValueInt64() != 1 {
		t.Errorf("expected ODVValidationMin 1, got %d", rule.ODVValidationMin.ValueInt64())
	}
	if rule.ODVValidationMax.ValueInt64() != 100 {
		t.Errorf("expected ODVValidationMax 100, got %d", rule.ODVValidationMax.ValueInt64())
	}
	if rule.ODVValidationRegex.ValueString() != `^\d+$` {
		t.Errorf("expected ODVValidationRegex '^\\d+$', got %q", rule.ODVValidationRegex.ValueString())
	}
}

func TestAssignBenchmarkModelFromResponse_NilInputs(t *testing.T) {
	assignBenchmarkModelFromResponse(nil, &client.CBEngineBenchmarkResponseV2{})
	assignBenchmarkModelFromResponse(&BenchmarkResourceModel{}, nil)
}

func TestAssignBenchmarkDataSourceFromResponse(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	model := &BenchmarkDataSourceModel{}
	bench := &client.CBEngineBenchmarkResponseV2{
		BenchmarkID:     "bench-ds-1",
		TenantID:        "tenant-2",
		Title:           "DS Benchmark",
		Description:     "Data source test",
		EnforcementMode: "enforce",
		Deleted:         true,
		UpdateAvailable: false,
		LastUpdatedAt:   ts,
		Sources:         []client.CBEngineSourceV1{{Branch: "dev", Revision: "xyz"}},
		Target:          client.CBEngineTargetV2{DeviceGroups: []string{"dg-1"}},
		Rules:           nil,
	}

	assignBenchmarkDataSourceFromResponse(model, bench)

	if model.BenchmarkID.ValueString() != "bench-ds-1" {
		t.Errorf("expected BenchmarkID 'bench-ds-1', got %q", model.BenchmarkID.ValueString())
	}
	if model.TenantID.ValueString() != "tenant-2" {
		t.Errorf("expected TenantID 'tenant-2', got %q", model.TenantID.ValueString())
	}
	if model.Deleted.ValueBool() != true {
		t.Errorf("expected Deleted true, got %v", model.Deleted.ValueBool())
	}
	if model.EnforcementMode.ValueString() != "enforce" {
		t.Errorf("expected EnforcementMode 'enforce', got %q", model.EnforcementMode.ValueString())
	}
}

func TestBuildSourceModels(t *testing.T) {
	sources := []client.CBEngineSourceV1{
		{Branch: "main", Revision: "aaa"},
		{Branch: "release", Revision: "bbb"},
	}
	result := buildSourceModels(sources)
	if len(result) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(result))
	}
	if result[0].Branch.ValueString() != "main" {
		t.Errorf("expected branch 'main', got %q", result[0].Branch.ValueString())
	}
	if result[1].Revision.ValueString() != "bbb" {
		t.Errorf("expected revision 'bbb', got %q", result[1].Revision.ValueString())
	}
}

func TestBuildSourceModels_Empty(t *testing.T) {
	result := buildSourceModels(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 sources, got %d", len(result))
	}
}

func TestBuildRuleModel_NoODV(t *testing.T) {
	rule := client.CBEngineRuleInfoV1{
		ID:          "rule-no-odv",
		SectionName: "Section B",
		Enabled:     false,
		Title:       "No ODV Rule",
		Description: "Rule without ODV",
	}
	result := buildRuleModel(rule)

	if result.ID.ValueString() != "rule-no-odv" {
		t.Errorf("expected ID 'rule-no-odv', got %q", result.ID.ValueString())
	}
	if !result.ODVValue.IsNull() {
		t.Error("expected null ODVValue when ODV is nil")
	}
	if !result.ODVHint.IsNull() {
		t.Error("expected null ODVHint when ODV is nil")
	}
	if !result.ODVValidationMin.IsNull() {
		t.Error("expected null ODVValidationMin when ODV is nil")
	}
	if !result.ODVValidationMax.IsNull() {
		t.Error("expected null ODVValidationMax when ODV is nil")
	}
	if !result.ODVValidationRegex.IsNull() {
		t.Error("expected null ODVValidationRegex when ODV is nil")
	}
}

func TestBuildRuleModel_ODVWithoutValidation(t *testing.T) {
	rule := client.CBEngineRuleInfoV1{
		ID:      "rule-odv-no-val",
		Enabled: true,
		ODV: &client.CBEngineOrganizationDefinedValueV1{
			Value: "custom",
			Type:  "string",
		},
	}
	result := buildRuleModel(rule)

	if result.ODVValue.ValueString() != "custom" {
		t.Errorf("expected ODVValue 'custom', got %q", result.ODVValue.ValueString())
	}
	if result.ODVType.ValueString() != "string" {
		t.Errorf("expected ODVType 'string', got %q", result.ODVType.ValueString())
	}
	if !result.ODVValidationMin.IsNull() {
		t.Error("expected null ODVValidationMin when validation is nil")
	}
	if !result.ODVValidationMax.IsNull() {
		t.Error("expected null ODVValidationMax when validation is nil")
	}
}

func TestBuildTargetDeviceGroup(t *testing.T) {
	result := buildTargetDeviceGroup([]string{"group-a", "group-b"})
	if result.ValueString() != "group-a" {
		t.Errorf("expected 'group-a', got %q", result.ValueString())
	}
}

func TestBuildTargetDeviceGroup_Empty(t *testing.T) {
	result := buildTargetDeviceGroup(nil)
	if !result.IsNull() {
		t.Error("expected null for empty device groups")
	}
}

func TestBuildStringList(t *testing.T) {
	result := buildStringList([]string{"a", "b", "c"})
	if result.IsNull() {
		t.Fatal("expected non-null list")
	}
	elems := result.Elements()
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
}

func TestBuildStringList_Empty(t *testing.T) {
	result := buildStringList(nil)
	if !result.IsNull() {
		t.Error("expected null for empty string list")
	}
}

func TestBuildSupportedOSList(t *testing.T) {
	osInfo := []client.CBEngineOSInfoV1{
		{OSType: "macOS", OSVersion: 14, ManagementType: "SUPERVISED"},
		{OSType: "iOS", OSVersion: 17, ManagementType: "UNSUPERVISED"},
	}
	result := buildSupportedOSList(osInfo)
	if result.IsNull() {
		t.Fatal("expected non-null list")
	}
	elems := result.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}
}

func TestBuildSupportedOSList_Empty(t *testing.T) {
	result := buildSupportedOSList(nil)
	if !result.IsNull() {
		t.Error("expected null for empty supported OS list")
	}
}

func TestBuildOSSpecificDefaultsMap(t *testing.T) {
	defaults := map[string]client.CBEngineOSSpecificRuleInfoV1{
		"macOS_15": {
			Title:       "macOS 15",
			Description: "For macOS 15",
			ODV: &client.CBEngineODVRecommendationV1{
				Value: "recommended",
				Hint:  "Use recommended",
			},
		},
	}
	result := buildOSSpecificDefaultsMap(defaults)
	if result.IsNull() {
		t.Fatal("expected non-null map")
	}
	elems := result.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
}

func TestBuildOSSpecificDefaultsMap_NilODV(t *testing.T) {
	defaults := map[string]client.CBEngineOSSpecificRuleInfoV1{
		"macOS_14": {
			Title:       "macOS 14",
			Description: "No ODV",
			ODV:         nil,
		},
	}
	result := buildOSSpecificDefaultsMap(defaults)
	if result.IsNull() {
		t.Fatal("expected non-null map even with nil ODV")
	}
}

func TestBuildOSSpecificDefaultsMap_Empty(t *testing.T) {
	result := buildOSSpecificDefaultsMap(nil)
	if !result.IsNull() {
		t.Error("expected null for empty defaults map")
	}
}

func TestOdvStringValue_NilODV(t *testing.T) {
	result := odvStringValue(nil, func(o *client.CBEngineOrganizationDefinedValueV1) string { return o.Value })
	if !result.IsNull() {
		t.Error("expected null when ODV is nil")
	}
}

func TestOdvStringValue_WithODV(t *testing.T) {
	odv := &client.CBEngineOrganizationDefinedValueV1{Value: "test-val", Hint: "test-hint"}
	result := odvStringValue(odv, func(o *client.CBEngineOrganizationDefinedValueV1) string { return o.Hint })
	if result.ValueString() != "test-hint" {
		t.Errorf("expected 'test-hint', got %q", result.ValueString())
	}
}

func TestBuildODVValidationEnumValues(t *testing.T) {
	odv := &client.CBEngineOrganizationDefinedValueV1{
		Validation: &client.CBEngineValidationConstraintsV1{
			EnumValues: []string{"low", "medium", "high"},
		},
	}
	result := buildODVValidationEnumValues(odv)
	if result.IsNull() {
		t.Fatal("expected non-null enum values list")
	}
	elems := result.Elements()
	if len(elems) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(elems))
	}
}

func TestBuildODVValidationEnumValues_Empty(t *testing.T) {
	odv := &client.CBEngineOrganizationDefinedValueV1{
		Validation: &client.CBEngineValidationConstraintsV1{},
	}
	result := buildODVValidationEnumValues(odv)
	if !result.IsNull() {
		t.Error("expected null for empty enum values")
	}
}

func TestBuildODVValidationEnumValues_NilODV(t *testing.T) {
	result := buildODVValidationEnumValues(nil)
	if !result.IsNull() {
		t.Error("expected null for nil ODV")
	}
}

func TestBuildDependsOnList(t *testing.T) {
	relation := &client.CBEngineRuleRelationV1{
		DependsOn: []string{"dep-1", "dep-2"},
	}
	result := buildDependsOnList(relation)
	if result.IsNull() {
		t.Fatal("expected non-null depends_on list")
	}
	elems := result.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 depends_on entries, got %d", len(elems))
	}
}

func TestBuildDependsOnList_NilRelation(t *testing.T) {
	result := buildDependsOnList(nil)
	if !result.IsNull() {
		t.Error("expected null for nil relation")
	}
}

func TestBuildDependsOnList_EmptyDependsOn(t *testing.T) {
	relation := &client.CBEngineRuleRelationV1{DependsOn: nil}
	result := buildDependsOnList(relation)
	if !result.IsNull() {
		t.Error("expected null for empty depends_on")
	}
}

func TestBuildODVValidationRegex_NilValidation(t *testing.T) {
	result := buildODVValidationRegex(nil)
	if !result.IsNull() {
		t.Error("expected null for nil ODV")
	}

	odv := &client.CBEngineOrganizationDefinedValueV1{}
	result = buildODVValidationRegex(odv)
	if !result.IsNull() {
		t.Error("expected null for nil validation")
	}
}

func TestBuildODVValidationRegex_WithRegex(t *testing.T) {
	odv := &client.CBEngineOrganizationDefinedValueV1{
		Validation: &client.CBEngineValidationConstraintsV1{
			Regex: `^[a-z]+$`,
		},
	}
	result := buildODVValidationRegex(odv)
	if result.ValueString() != `^[a-z]+$` {
		t.Errorf("expected regex '^[a-z]+$', got %q", result.ValueString())
	}
}

func TestBuildODVValidationMin_NilChain(t *testing.T) {
	if !buildODVValidationMin(nil).IsNull() {
		t.Error("expected null for nil ODV")
	}
	if !buildODVValidationMin(&client.CBEngineOrganizationDefinedValueV1{}).IsNull() {
		t.Error("expected null for nil validation")
	}
	if !buildODVValidationMin(&client.CBEngineOrganizationDefinedValueV1{
		Validation: &client.CBEngineValidationConstraintsV1{},
	}).IsNull() {
		t.Error("expected null for nil min")
	}
}

func TestBuildODVValidationMax_NilChain(t *testing.T) {
	if !buildODVValidationMax(nil).IsNull() {
		t.Error("expected null for nil ODV")
	}
	if !buildODVValidationMax(&client.CBEngineOrganizationDefinedValueV1{}).IsNull() {
		t.Error("expected null for nil validation")
	}
	maxVal := 50
	result := buildODVValidationMax(&client.CBEngineOrganizationDefinedValueV1{
		Validation: &client.CBEngineValidationConstraintsV1{Max: &maxVal},
	})
	if result.ValueInt64() != 50 {
		t.Errorf("expected max 50, got %d", result.ValueInt64())
	}
}

// suppress unused import warning for types package
var _ = types.StringNull
