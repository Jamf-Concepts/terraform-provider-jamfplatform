// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestJSONToTerraformDynamic_String(t *testing.T) {
	d, err := JSONToTerraformDynamic("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestJSONToTerraformDynamic_Bool(t *testing.T) {
	d, err := JSONToTerraformDynamic(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestJSONToTerraformDynamic_Number(t *testing.T) {
	d, err := JSONToTerraformDynamic(float64(42))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != float64(42) {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestJSONToTerraformDynamic_Null(t *testing.T) {
	d, err := JSONToTerraformDynamic(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestJSONToTerraformDynamic_Array(t *testing.T) {
	input := []any{"a", "b", "c"}
	d, err := JSONToTerraformDynamic(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr))
	}
}

func TestJSONToTerraformDynamic_EmptyArray(t *testing.T) {
	input := []any{}
	d, err := JSONToTerraformDynamic(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(arr) != 0 {
		t.Errorf("expected 0 elements, got %d", len(arr))
	}
}

func TestJSONToTerraformDynamic_Object(t *testing.T) {
	input := map[string]any{
		"name": "test",
		"age":  float64(30),
	}
	d, err := JSONToTerraformDynamic(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if obj["name"] != "test" {
		t.Errorf("expected name 'test', got %v", obj["name"])
	}
	if obj["age"] != float64(30) {
		t.Errorf("expected age 30, got %v", obj["age"])
	}
}

func TestJSONToTerraformDynamic_NestedObject(t *testing.T) {
	input := map[string]any{
		"outer": map[string]any{
			"inner": "value",
		},
	}
	d, err := JSONToTerraformDynamic(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj := result.(map[string]any)
	outer := obj["outer"].(map[string]any)
	if outer["inner"] != "value" {
		t.Errorf("expected inner 'value', got %v", outer["inner"])
	}
}

func TestJSONToTerraformDynamic_MixedTypeObject(t *testing.T) {
	input := map[string]any{
		"boolVal":   true,
		"stringVal": "hello",
		"numVal":    float64(42),
	}
	d, err := JSONToTerraformDynamic(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj := result.(map[string]any)
	if obj["boolVal"] != true {
		t.Errorf("expected boolVal true, got %v", obj["boolVal"])
	}
	if obj["stringVal"] != "hello" {
		t.Errorf("expected stringVal 'hello', got %v", obj["stringVal"])
	}
	if obj["numVal"] != float64(42) {
		t.Errorf("expected numVal 42, got %v", obj["numVal"])
	}
}

func TestRoundTrip_ComplexPayload(t *testing.T) {
	jsonStr := `{
		"allowSafariHistoryClearing": false,
		"allowSafariPrivateBrowsing": false,
		"SSID_STR": "CorporateWiFi",
		"EAPClientConfiguration": {
			"AcceptEAPTypes": [13, 25],
			"UserName": "user@corp.com",
			"TLSAllowTrustExceptions": true
		}
	}`

	var input map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &input); err != nil {
		t.Fatalf("failed to parse test JSON: %v", err)
	}

	d, err := JSONToTerraformDynamic(input)
	if err != nil {
		t.Fatalf("JSONToTerraformDynamic error: %v", err)
	}

	result, err := TerraformDynamicToJSON(d)
	if err != nil {
		t.Fatalf("TerraformDynamicToJSON error: %v", err)
	}

	resultJSON, _ := json.Marshal(result)
	inputJSON, _ := json.Marshal(input)

	var resultParsed, inputParsed any
	json.Unmarshal(resultJSON, &resultParsed)
	json.Unmarshal(inputJSON, &inputParsed)

	resultNorm, _ := json.Marshal(resultParsed)
	inputNorm, _ := json.Marshal(inputParsed)

	if string(resultNorm) != string(inputNorm) {
		t.Errorf("round-trip mismatch:\ninput:  %s\nresult: %s", inputNorm, resultNorm)
	}
}

func TestTerraformDynamicToJSON_NullDynamic(t *testing.T) {
	result, err := TerraformDynamicToJSON(types.DynamicNull())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for null dynamic, got %v", result)
	}
}

func TestTerraformDynamicToJSON_UnknownDynamic(t *testing.T) {
	result, err := TerraformDynamicToJSON(types.DynamicUnknown())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for unknown dynamic, got %v", result)
	}
}
