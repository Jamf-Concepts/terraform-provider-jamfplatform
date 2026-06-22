// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var testElementType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":  types.StringType,
		"value": types.StringType,
	},
}

// element builds a known collection element with the given name and value.
func element(t *testing.T, name, value types.String) attr.Value {
	t.Helper()
	obj, diags := types.ObjectValue(testElementType.AttrTypes, map[string]attr.Value{
		"name":  name,
		"value": value,
	})
	if diags.HasError() {
		t.Fatalf("building test object: %v", diags)
	}
	return obj
}

func runSet(t *testing.T, value types.Set) validator.SetResponse {
	t.Helper()
	req := validator.SetRequest{Path: path.Root("test_set"), ConfigValue: value}
	var resp validator.SetResponse
	UniqueStringFieldSet("name").ValidateSet(context.Background(), req, &resp)
	return resp
}

func runList(t *testing.T, value types.List) validator.ListResponse {
	t.Helper()
	req := validator.ListRequest{Path: path.Root("test_list"), ConfigValue: value}
	var resp validator.ListResponse
	UniqueStringFieldList("name").ValidateList(context.Background(), req, &resp)
	return resp
}

func setOf(t *testing.T, elems ...attr.Value) types.Set {
	t.Helper()
	set, diags := types.SetValue(testElementType, elems)
	if diags.HasError() {
		t.Fatalf("building test set: %v", diags)
	}
	return set
}

func listOf(t *testing.T, elems ...attr.Value) types.List {
	t.Helper()
	list, diags := types.ListValue(testElementType, elems)
	if diags.HasError() {
		t.Fatalf("building test list: %v", diags)
	}
	return list
}

func TestUniqueStringFieldSet_DuplicateNamesError(t *testing.T) {
	resp := runSet(t, setOf(t,
		element(t, types.StringValue("EA One"), types.StringValue("a")),
		element(t, types.StringValue("EA Two"), types.StringValue("b")),
		element(t, types.StringValue("EA One"), types.StringValue("c")),
	))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected duplicate-name diagnostic, got none")
	}
	if got := resp.Diagnostics.ErrorsCount(); got != 1 {
		t.Errorf("expected 1 error diagnostic, got %d", got)
	}
}

func TestUniqueStringFieldSet_UniqueNamesPass(t *testing.T) {
	resp := runSet(t, setOf(t,
		element(t, types.StringValue("EA One"), types.StringValue("a")),
		element(t, types.StringValue("EA Two"), types.StringValue("a")),
	))
	if resp.Diagnostics.HasError() {
		t.Fatalf("unique names rejected: %v", resp.Diagnostics)
	}
}

func TestUniqueStringFieldSet_DefersOnUnknowns(t *testing.T) {
	cases := map[string]types.Set{
		"null set":    types.SetNull(testElementType),
		"unknown set": types.SetUnknown(testElementType),
		"unknown name elements": setOf(t,
			element(t, types.StringUnknown(), types.StringValue("a")),
			element(t, types.StringUnknown(), types.StringValue("b")),
		),
		"unknown name beside known": setOf(t,
			element(t, types.StringUnknown(), types.StringValue("a")),
			element(t, types.StringValue("EA One"), types.StringValue("b")),
		),
	}
	for name, set := range cases {
		t.Run(name, func(t *testing.T) {
			if resp := runSet(t, set); resp.Diagnostics.HasError() {
				t.Errorf("expected defer (no diagnostics), got: %v", resp.Diagnostics)
			}
		})
	}
}

func TestUniqueStringFieldList_DuplicateNamesError(t *testing.T) {
	resp := runList(t, listOf(t,
		element(t, types.StringValue("Welcome"), types.StringValue("a")),
		element(t, types.StringValue("EULA"), types.StringValue("b")),
		element(t, types.StringValue("Welcome"), types.StringValue("c")),
	))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected duplicate-name diagnostic, got none")
	}
	// One diagnostic only — for the offending second occurrence.
	if got := resp.Diagnostics.ErrorsCount(); got != 1 {
		t.Errorf("expected 1 error diagnostic, got %d", got)
	}
}

func TestUniqueStringFieldList_UniqueNamesPass(t *testing.T) {
	resp := runList(t, listOf(t,
		element(t, types.StringValue("Welcome"), types.StringValue("a")),
		element(t, types.StringValue("EULA"), types.StringValue("b")),
	))
	if resp.Diagnostics.HasError() {
		t.Fatalf("unique list rejected: %v", resp.Diagnostics)
	}
}

func TestUniqueStringFieldList_DefersOnNullAndUnknown(t *testing.T) {
	cases := map[string]types.List{
		"null list":    types.ListNull(testElementType),
		"unknown list": types.ListUnknown(testElementType),
		"unknown name elements": listOf(t,
			element(t, types.StringUnknown(), types.StringValue("a")),
			element(t, types.StringUnknown(), types.StringValue("b")),
		),
	}
	for name, list := range cases {
		t.Run(name, func(t *testing.T) {
			if resp := runList(t, list); resp.Diagnostics.HasError() {
				t.Errorf("expected defer (no diagnostics), got: %v", resp.Diagnostics)
			}
		})
	}
}
