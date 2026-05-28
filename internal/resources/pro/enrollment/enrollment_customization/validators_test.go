// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enrollment_customization

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// textPaneObjectType is the framework attr.Type for a single text pane,
// matching the schema declaration. Used to fabricate a ListValue in the
// uniqueness test below.
var textPaneObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":                   types.StringType,
		"display_name":         types.StringType,
		"rank":                 types.Int64Type,
		"title":                types.StringType,
		"body":                 types.StringType,
		"subtext":              types.StringType,
		"previous_button_text": types.StringType,
		"next_button_text":     types.StringType,
	},
}

func paneObject(t *testing.T, displayName string) types.Object {
	t.Helper()
	obj, diags := types.ObjectValue(textPaneObjectType.AttrTypes, map[string]attr.Value{
		"id":                   types.StringNull(),
		"display_name":         types.StringValue(displayName),
		"rank":                 types.Int64Value(0),
		"title":                types.StringValue("t"),
		"body":                 types.StringValue("b"),
		"subtext":              types.StringNull(),
		"previous_button_text": types.StringValue("prev"),
		"next_button_text":     types.StringValue("next"),
	})
	if diags.HasError() {
		t.Fatalf("constructing pane object: %v", diags)
	}
	return obj
}

func TestUniqueDisplayNameValidator_FlagsDuplicates(t *testing.T) {
	listVal, diags := types.ListValue(textPaneObjectType, []attr.Value{
		paneObject(t, "Welcome"),
		paneObject(t, "EULA"),
		paneObject(t, "Welcome"),
	})
	if diags.HasError() {
		t.Fatalf("constructing list: %v", diags)
	}
	req := validator.ListRequest{
		Path:        path.Root("text_panes"),
		ConfigValue: listVal,
	}
	var resp validator.ListResponse
	UniqueDisplayNameValidator().ValidateList(context.Background(), req, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected duplicate display_name diagnostic, got none")
	}
	// One diagnostic only — for the offending second occurrence.
	if got := resp.Diagnostics.ErrorsCount(); got != 1 {
		t.Errorf("expected 1 error diagnostic, got %d", got)
	}
}

func TestUniqueDisplayNameValidator_PassesUniqueLists(t *testing.T) {
	listVal, diags := types.ListValue(textPaneObjectType, []attr.Value{
		paneObject(t, "Welcome"),
		paneObject(t, "EULA"),
	})
	if diags.HasError() {
		t.Fatalf("constructing list: %v", diags)
	}
	req := validator.ListRequest{Path: path.Root("text_panes"), ConfigValue: listVal}
	var resp validator.ListResponse
	UniqueDisplayNameValidator().ValidateList(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unique list rejected: %v", resp.Diagnostics)
	}
}

func TestUniqueDisplayNameValidator_NullListNoop(t *testing.T) {
	req := validator.ListRequest{
		Path:        path.Root("text_panes"),
		ConfigValue: types.ListNull(textPaneObjectType),
	}
	var resp validator.ListResponse
	UniqueDisplayNameValidator().ValidateList(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("null list should not produce diagnostics: %v", resp.Diagnostics)
	}
}
