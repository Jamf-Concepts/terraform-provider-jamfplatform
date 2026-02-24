// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandDeviceGroupCriteria(t *testing.T) {
	criteria := []DeviceGroupCriteriaModel{
		{
			Order:                 types.Int64Value(0),
			AttributeName:         types.StringValue("Device Name"),
			Operator:              types.StringValue("like"),
			AttributeValue:        types.StringValue("Mac"),
			JoinType:              types.StringValue("and"),
			HasOpeningParenthesis: types.BoolValue(true),
			HasClosingParenthesis: types.BoolValue(false),
		},
		{
			Order:                 types.Int64Value(1),
			AttributeName:         types.StringValue("OS Version"),
			Operator:              types.StringValue("greater than"),
			AttributeValue:        types.StringValue("14.0"),
			JoinType:              types.StringValue("or"),
			HasOpeningParenthesis: types.BoolNull(),
			HasClosingParenthesis: types.BoolValue(true),
		},
	}

	result := expandDeviceGroupCriteria(criteria)
	if len(result) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(result))
	}

	c0 := result[0]
	if c0.AttributeName != "Device Name" {
		t.Errorf("expected AttributeName 'Device Name', got %q", c0.AttributeName)
	}
	if c0.Operator != "LIKE" {
		t.Errorf("expected Operator 'LIKE', got %q", c0.Operator)
	}
	if c0.AttributeValue != "Mac" {
		t.Errorf("expected AttributeValue 'Mac', got %q", c0.AttributeValue)
	}
	if c0.JoinType != "AND" {
		t.Errorf("expected JoinType 'AND', got %q", c0.JoinType)
	}
	if c0.Order != 0 {
		t.Errorf("expected Order 0, got %d", c0.Order)
	}
	if !c0.HasOpeningParenthesis {
		t.Error("expected HasOpeningParenthesis true")
	}
	if c0.HasClosingParenthesis {
		t.Error("expected HasClosingParenthesis false")
	}

	c1 := result[1]
	if c1.Operator != "GREATER THAN" {
		t.Errorf("expected Operator 'GREATER THAN', got %q", c1.Operator)
	}
	if c1.JoinType != "OR" {
		t.Errorf("expected JoinType 'OR', got %q", c1.JoinType)
	}
	if c1.HasOpeningParenthesis {
		t.Error("expected HasOpeningParenthesis false for null input")
	}
	if !c1.HasClosingParenthesis {
		t.Error("expected HasClosingParenthesis true")
	}
}

func TestExpandDeviceGroupCriteria_Empty(t *testing.T) {
	result := expandDeviceGroupCriteria(nil)
	if result != nil {
		t.Errorf("expected nil for empty criteria, got %v", result)
	}
}

func TestExpandDeviceGroupCriteria_OrderDefaultsToIndex(t *testing.T) {
	criteria := []DeviceGroupCriteriaModel{
		{
			Order:          types.Int64Null(),
			AttributeName:  types.StringValue("Device Name"),
			Operator:       types.StringValue("like"),
			AttributeValue: types.StringValue("test"),
		},
		{
			Order:          types.Int64Null(),
			AttributeName:  types.StringValue("OS Version"),
			Operator:       types.StringValue("is"),
			AttributeValue: types.StringValue("15"),
		},
	}

	result := expandDeviceGroupCriteria(criteria)
	if len(result) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(result))
	}
	if result[0].Order != 0 {
		t.Errorf("expected Order 0 (index default), got %d", result[0].Order)
	}
	if result[1].Order != 1 {
		t.Errorf("expected Order 1 (index default), got %d", result[1].Order)
	}
}

func TestExpandDeviceGroupCriteria_SortsByOrder(t *testing.T) {
	criteria := []DeviceGroupCriteriaModel{
		{
			Order:          types.Int64Value(2),
			AttributeName:  types.StringValue("Second"),
			Operator:       types.StringValue("is"),
			AttributeValue: types.StringValue("b"),
		},
		{
			Order:          types.Int64Value(0),
			AttributeName:  types.StringValue("First"),
			Operator:       types.StringValue("is"),
			AttributeValue: types.StringValue("a"),
		},
	}

	result := expandDeviceGroupCriteria(criteria)
	if len(result) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(result))
	}
	if result[0].AttributeName != "First" {
		t.Errorf("expected first criterion to be 'First' after sorting, got %q", result[0].AttributeName)
	}
	if result[1].AttributeName != "Second" {
		t.Errorf("expected second criterion to be 'Second' after sorting, got %q", result[1].AttributeName)
	}
}

func TestExpandDeviceGroupCriteria_SkipsUnconfiguredAttributeName(t *testing.T) {
	criteria := []DeviceGroupCriteriaModel{
		{
			AttributeName:  types.StringNull(),
			Operator:       types.StringValue("is"),
			AttributeValue: types.StringValue("val"),
		},
		{
			AttributeName:  types.StringValue("Real Attribute"),
			Operator:       types.StringValue("is"),
			AttributeValue: types.StringValue("val"),
		},
	}

	result := expandDeviceGroupCriteria(criteria)
	if len(result) != 1 {
		t.Fatalf("expected 1 criterion (null attribute skipped), got %d", len(result))
	}
	if result[0].AttributeName != "Real Attribute" {
		t.Errorf("expected 'Real Attribute', got %q", result[0].AttributeName)
	}
}

func TestExpandDeviceGroupCriteria_NullOptionalFields(t *testing.T) {
	criteria := []DeviceGroupCriteriaModel{
		{
			AttributeName:         types.StringValue("Device Name"),
			Operator:              types.StringNull(),
			AttributeValue:        types.StringNull(),
			JoinType:              types.StringNull(),
			HasOpeningParenthesis: types.BoolNull(),
			HasClosingParenthesis: types.BoolNull(),
		},
	}

	result := expandDeviceGroupCriteria(criteria)
	if len(result) != 1 {
		t.Fatalf("expected 1 criterion, got %d", len(result))
	}
	if result[0].Operator != "" {
		t.Errorf("expected empty Operator for null, got %q", result[0].Operator)
	}
	if result[0].AttributeValue != "" {
		t.Errorf("expected empty AttributeValue for null, got %q", result[0].AttributeValue)
	}
	if result[0].JoinType != "" {
		t.Errorf("expected empty JoinType for null, got %q", result[0].JoinType)
	}
	if result[0].HasOpeningParenthesis {
		t.Error("expected false HasOpeningParenthesis for null")
	}
	if result[0].HasClosingParenthesis {
		t.Error("expected false HasClosingParenthesis for null")
	}
}
