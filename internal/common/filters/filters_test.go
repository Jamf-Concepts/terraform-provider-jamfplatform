// Copyright 2026 Jamf Software LLC.

package filters

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildRSQLExpression_EmptyFilters(t *testing.T) {
	result := BuildRSQLExpression(nil, nil)
	if result != "" {
		t.Errorf("expected empty string for nil filters, got %q", result)
	}

	result = BuildRSQLExpression([]FilterModel{}, nil)
	if result != "" {
		t.Errorf("expected empty string for empty filters, got %q", result)
	}
}

func TestBuildRSQLExpression_SingleClause(t *testing.T) {
	filters := []FilterModel{
		{
			Selector: types.StringValue("name"),
			Operator: types.StringValue("=="),
			Argument: types.StringValue("test"),
		},
	}

	result := BuildRSQLExpression(filters, nil)
	expected := "name==test"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildRSQLExpression_DefaultOperator(t *testing.T) {
	filters := []FilterModel{
		{
			Selector: types.StringValue("name"),
			Argument: types.StringValue("test"),
		},
	}

	result := BuildRSQLExpression(filters, nil)
	expected := "name==test"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildRSQLExpression_MultipleClauses_DefaultAnd(t *testing.T) {
	filters := []FilterModel{
		{
			Selector: types.StringValue("deviceType"),
			Argument: types.StringValue("COMPUTER"),
		},
		{
			Selector: types.StringValue("groupType"),
			Argument: types.StringValue("STATIC"),
		},
	}

	result := BuildRSQLExpression(filters, nil)
	expected := "deviceType==COMPUTER and groupType==STATIC"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildRSQLExpression_OrJoin(t *testing.T) {
	filters := []FilterModel{
		{
			Selector: types.StringValue("name"),
			Argument: types.StringValue("a"),
		},
		{
			Selector: types.StringValue("name"),
			Argument: types.StringValue("b"),
			JoinWith: types.StringValue("or"),
		},
	}

	result := BuildRSQLExpression(filters, nil)
	expected := "name==a or name==b"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildRSQLExpression_Parentheses(t *testing.T) {
	filters := []FilterModel{
		{
			Selector:              types.StringValue("name"),
			Argument:              types.StringValue("a"),
			HasOpeningParenthesis: types.BoolValue(true),
		},
		{
			Selector:              types.StringValue("name"),
			Argument:              types.StringValue("b"),
			JoinWith:              types.StringValue("or"),
			HasClosingParenthesis: types.BoolValue(true),
		},
		{
			Selector: types.StringValue("type"),
			Argument: types.StringValue("c"),
		},
	}

	result := BuildRSQLExpression(filters, nil)
	expected := "(name==a or name==b) and type==c"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildRSQLExpression_SelectorValidator_FilterOut(t *testing.T) {
	filters := []FilterModel{
		{
			Selector: types.StringValue("allowed"),
			Argument: types.StringValue("yes"),
		},
		{
			Selector: types.StringValue("denied"),
			Argument: types.StringValue("no"),
		},
	}

	validator := AllowList([]string{"allowed"})
	result := BuildRSQLExpression(filters, validator)
	expected := "allowed==yes"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildRSQLExpression_SkipsNullValues(t *testing.T) {
	filters := []FilterModel{
		{
			Selector: types.StringNull(),
			Argument: types.StringValue("test"),
		},
		{
			Selector: types.StringValue("name"),
			Argument: types.StringNull(),
		},
	}

	result := BuildRSQLExpression(filters, nil)
	if result != "" {
		t.Errorf("expected empty string for null selector/argument, got %q", result)
	}
}

func TestFormatArgument_SimpleValue(t *testing.T) {
	result := FormatArgument("hello")
	if result != "hello" {
		t.Errorf("expected %q, got %q", "hello", result)
	}
}

func TestFormatArgument_ValueWithSpaces(t *testing.T) {
	result := FormatArgument("hello world")
	expected := `"hello world"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatArgument_AlreadyQuoted(t *testing.T) {
	result := FormatArgument(`"already quoted"`)
	expected := `"already quoted"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatArgument_ListArgument(t *testing.T) {
	result := FormatArgument("(a,b,c)")
	expected := "(a,b,c)"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatArgument_Empty(t *testing.T) {
	result := FormatArgument("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFormatArgument_EscapeQuotes(t *testing.T) {
	result := FormatArgument(`has"quote`)
	expected := `has\"quote`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestClause_DefaultOperator(t *testing.T) {
	result := Clause("name", "", "value")
	expected := "name==value"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestClause_CustomOperator(t *testing.T) {
	result := Clause("count", ">=", "5")
	expected := "count>=5"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestAllowList_EmptyReturnsNil(t *testing.T) {
	validator := AllowList(nil)
	if validator != nil {
		t.Error("expected nil validator for empty selector list")
	}
}

func TestAllowList_Permits(t *testing.T) {
	validator := AllowList([]string{"name", "type"})
	if !validator("name") {
		t.Error("expected 'name' to be allowed")
	}
	if !validator("type") {
		t.Error("expected 'type' to be allowed")
	}
	if validator("other") {
		t.Error("expected 'other' to be rejected")
	}
}

func TestSelectorDescription(t *testing.T) {
	result := SelectorDescription([]string{"name", "type"})
	if result != "RSQL selector. Valid values are `name`, `type`." {
		t.Errorf("unexpected description: %s", result)
	}
}
