// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDescribeBlueprintBlocks(t *testing.T) {
	named := "Passcode"
	empty := ""

	got := describeBlueprintBlocks([]blueprints.BlueprintStep{
		{
			Name: &named,
			Components: []blueprints.Component{
				{Identifier: "com.jamf.ddm.passcode-settings"},
				{Identifier: "com.jamf.ddm.math-settings"},
			},
		},
		{Name: nil, Components: []blueprints.Component{{Identifier: "com.jamf.ddm.safari-settings"}}},
		{Name: &empty, Components: nil},
	})

	want := strings.Join([]string{
		"  1. Passcode — com.jamf.ddm.passcode-settings, com.jamf.ddm.math-settings",
		"  2. (unnamed) — com.jamf.ddm.safari-settings",
		"  3. (unnamed) — no components",
		"",
	}, "\n")

	if got != want {
		t.Errorf("describeBlueprintBlocks() =\n%q\nwant\n%q", got, want)
	}
}

func TestDescribeBlueprintBlocks_NoSteps(t *testing.T) {
	if got := describeBlueprintBlocks(nil); got != "" {
		t.Errorf("describeBlueprintBlocks(nil) = %q, want empty", got)
	}
}

func TestSetNestedValue_SimpleString(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "name", "test")
	if obj["name"] != "test" {
		t.Errorf("expected 'test', got %v", obj["name"])
	}
}

func TestSetNestedValue_BoolTrue(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "enabled", "true")
	if obj["enabled"] != true {
		t.Errorf("expected true, got %v", obj["enabled"])
	}
}

func TestSetNestedValue_BoolFalse(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "disabled", "false")
	if obj["disabled"] != false {
		t.Errorf("expected false, got %v", obj["disabled"])
	}
}

func TestSetNestedValue_Integer(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "count", "42")
	if obj["count"] != 42 {
		t.Errorf("expected 42, got %v", obj["count"])
	}
}

func TestSetNestedValue_EmptyValue(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "field", "")
	if obj["field"] != nil {
		t.Errorf("expected nil, got %v", obj["field"])
	}
}

func TestSetNestedValue_NestedKey(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "parent_child", "value")
	parent, ok := obj["parent"].(map[string]any)
	if !ok {
		t.Fatal("expected nested map at 'parent'")
	}
	if parent["child"] != "value" {
		t.Errorf("expected 'value' at parent.child, got %v", parent["child"])
	}
}

func TestSetNestedValue_DeepNested(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "a_b_c", "deep")
	a, ok := obj["a"].(map[string]any)
	if !ok {
		t.Fatal("expected nested map at 'a'")
	}
	b, ok := a["b"].(map[string]any)
	if !ok {
		t.Fatal("expected nested map at 'a.b'")
	}
	if b["c"] != "deep" {
		t.Errorf("expected 'deep' at a.b.c, got %v", b["c"])
	}
}

func TestSetNestedValue_JSONArray(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "items", `[1,2,3]`)
	arr, ok := obj["items"].([]any)
	if !ok {
		t.Fatalf("expected array, got %T", obj["items"])
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr))
	}
}

func TestSetNestedValue_JSONObject(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "config", `{"key":"val"}`)
	m, ok := obj["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", obj["config"])
	}
	if m["key"] != "val" {
		t.Errorf("expected 'val', got %v", m["key"])
	}
}

func TestSetNestedValue_OverwritesExistingNonMap(t *testing.T) {
	obj := map[string]any{"parent": "not-a-map"}
	setNestedValue(obj, "parent_child", "value")
	parent, ok := obj["parent"].(map[string]any)
	if !ok {
		t.Fatal("expected parent to be overwritten to a map")
	}
	if parent["child"] != "value" {
		t.Errorf("expected 'value', got %v", parent["child"])
	}
}

func TestFlattenJSON_SimpleValues(t *testing.T) {
	obj := map[string]any{
		"name":    "test",
		"enabled": true,
		"count":   float64(42),
	}
	result := make(map[string]string)
	flattenJSON(obj, "", result)

	if result["name"] != "test" {
		t.Errorf("expected 'test', got %q", result["name"])
	}
	if result["enabled"] != "true" {
		t.Errorf("expected 'true', got %q", result["enabled"])
	}
	if result["count"] != "42" {
		t.Errorf("expected '42', got %q", result["count"])
	}
}

func TestFlattenJSON_BoolFalse(t *testing.T) {
	obj := map[string]any{"disabled": false}
	result := make(map[string]string)
	flattenJSON(obj, "", result)
	if result["disabled"] != "false" {
		t.Errorf("expected 'false', got %q", result["disabled"])
	}
}

func TestFlattenJSON_NilValue(t *testing.T) {
	obj := map[string]any{"field": nil}
	result := make(map[string]string)
	flattenJSON(obj, "", result)
	if result["field"] != "" {
		t.Errorf("expected empty string for nil, got %q", result["field"])
	}
}

func TestFlattenJSON_NestedObject(t *testing.T) {
	obj := map[string]any{
		"parent": map[string]any{
			"child": "value",
		},
	}
	result := make(map[string]string)
	flattenJSON(obj, "", result)
	if result["parent_child"] != "value" {
		t.Errorf("expected 'value' at 'parent_child', got %q", result["parent_child"])
	}
}

func TestFlattenJSON_WithPrefix(t *testing.T) {
	obj := map[string]any{"key": "val"}
	result := make(map[string]string)
	flattenJSON(obj, "prefix", result)
	if result["prefix_key"] != "val" {
		t.Errorf("expected 'val' at 'prefix_key', got %q", result["prefix_key"])
	}
}

func TestFlattenJSON_IntValue(t *testing.T) {
	obj := map[string]any{"num": 10}
	result := make(map[string]string)
	flattenJSON(obj, "", result)
	if result["num"] != "10" {
		t.Errorf("expected '10', got %q", result["num"])
	}
}

func TestFlattenJSON_FloatDecimal(t *testing.T) {
	obj := map[string]any{"ratio": float64(3.14)}
	result := make(map[string]string)
	flattenJSON(obj, "", result)
	if result["ratio"] != "3.14" {
		t.Errorf("expected '3.14', got %q", result["ratio"])
	}
}

func TestFlattenJSON_ArrayValue(t *testing.T) {
	obj := map[string]any{
		"items": []any{"a", "b"},
	}
	result := make(map[string]string)
	flattenJSON(obj, "", result)
	if result["items"] != `["a","b"]` {
		t.Errorf("expected JSON array, got %q", result["items"])
	}
}

func TestSetNestedAndFlattenRoundtrip(t *testing.T) {
	obj := make(map[string]any)
	setNestedValue(obj, "parent_child", "value")
	setNestedValue(obj, "parent_other", "42")
	setNestedValue(obj, "top", "true")

	result := make(map[string]string)
	flattenJSON(obj, "", result)

	if result["parent_child"] != "value" {
		t.Errorf("expected 'value', got %q", result["parent_child"])
	}
	if result["parent_other"] != "42" {
		t.Errorf("expected '42', got %q", result["parent_other"])
	}
	if result["top"] != "true" {
		t.Errorf("expected 'true', got %q", result["top"])
	}
}

func TestDesiredDeployedValue_Configured(t *testing.T) {
	if desiredDeployedValue(types.BoolValue(true)) != true {
		t.Error("expected true for configured true")
	}
	if desiredDeployedValue(types.BoolValue(false)) != false {
		t.Error("expected false for configured false")
	}
}

func TestDesiredDeployedValue_NullDefaults(t *testing.T) {
	if desiredDeployedValue(types.BoolNull()) != true {
		t.Error("expected true as default for null value")
	}
}

func TestDesiredDeployedValue_UnknownDefaults(t *testing.T) {
	if desiredDeployedValue(types.BoolUnknown()) != true {
		t.Error("expected true as default for unknown value")
	}
}
