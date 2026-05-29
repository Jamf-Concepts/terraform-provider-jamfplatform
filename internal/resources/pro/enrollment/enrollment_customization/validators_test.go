// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enrollment_customization

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

// accessGroupConfig builds a tfsdk.Config carrying the two sibling attributes
// the accessGroupNameValidator reads (enrollment_access discriminator +
// access_group_name validated value), so GetAttribute on the parent path
// resolves enrollment_access. Pass tftypes.UnknownValue / nil for the
// access_group_name value to exercise unknown / null states.
func accessGroupConfig(enrollmentAccess, accessGroupName tftypes.Value) tfsdk.Config {
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"enrollment_access": tftypes.String,
		"access_group_name": tftypes.String,
	}}
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"enrollment_access": schema.StringAttribute{Optional: true},
			"access_group_name": schema.StringAttribute{Optional: true},
		}},
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"enrollment_access": enrollmentAccess,
			"access_group_name": accessGroupName,
		}),
	}
}

func runAccessGroupNameValidator(cfg tfsdk.Config, configValue types.String) []string {
	req := validator.StringRequest{
		Path:        path.Root("access_group_name"),
		Config:      cfg,
		ConfigValue: configValue,
	}
	var resp validator.StringResponse
	AccessGroupNameValidator().ValidateString(context.Background(), req, &resp)
	out := make([]string, 0, len(resp.Diagnostics))
	for _, d := range resp.Diagnostics {
		out = append(out, d.Summary())
	}
	return out
}

// TestAccessGroupName_DefersWhenValueUnknown is the defer-on-unknown regression
// guard: enrollment_access = "specific_group" (known) + access_group_name
// UNKNOWN (variable/for_each-driven) must DEFER, not treat unknown as missing
// and error. See STYLE_GUIDE "Config-time validators must defer on unknown
// values".
func TestAccessGroupName_DefersWhenValueUnknown(t *testing.T) {
	cfg := accessGroupConfig(
		tftypes.NewValue(tftypes.String, enrollmentAccessSpecificGroup),
		tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	)
	if out := runAccessGroupNameValidator(cfg, types.StringUnknown()); len(out) != 0 {
		t.Errorf("specific_group + unknown access_group_name must defer, got %v", out)
	}
}

// TestAccessGroupName_ErrorsWhenNull proves the requirement still fires when
// access_group_name is genuinely absent on a specific_group pane.
func TestAccessGroupName_ErrorsWhenNull(t *testing.T) {
	cfg := accessGroupConfig(
		tftypes.NewValue(tftypes.String, enrollmentAccessSpecificGroup),
		tftypes.NewValue(tftypes.String, nil),
	)
	if out := runAccessGroupNameValidator(cfg, types.StringNull()); len(out) == 0 {
		t.Error("expected error for specific_group with null access_group_name")
	}
}

// TestAccessGroupName_ErrorsWhenEmpty proves an empty-string access_group_name
// still trips the requirement.
func TestAccessGroupName_ErrorsWhenEmpty(t *testing.T) {
	cfg := accessGroupConfig(
		tftypes.NewValue(tftypes.String, enrollmentAccessSpecificGroup),
		tftypes.NewValue(tftypes.String, ""),
	)
	if out := runAccessGroupNameValidator(cfg, types.StringValue("")); len(out) == 0 {
		t.Error("expected error for specific_group with empty access_group_name")
	}
}

// TestAccessGroupName_PassesWhenPresent proves a populated access_group_name on
// a specific_group pane passes.
func TestAccessGroupName_PassesWhenPresent(t *testing.T) {
	cfg := accessGroupConfig(
		tftypes.NewValue(tftypes.String, enrollmentAccessSpecificGroup),
		tftypes.NewValue(tftypes.String, "GroupX"),
	)
	if out := runAccessGroupNameValidator(cfg, types.StringValue("GroupX")); len(out) != 0 {
		t.Errorf("specific_group + populated access_group_name should pass, got %v", out)
	}
}

// TestAccessGroupName_PassesForAnyIdpUser proves the validator is a no-op when
// enrollment_access != "specific_group", regardless of access_group_name.
func TestAccessGroupName_PassesForAnyIdpUser(t *testing.T) {
	cfg := accessGroupConfig(
		tftypes.NewValue(tftypes.String, enrollmentAccessAnyIdpUser),
		tftypes.NewValue(tftypes.String, nil),
	)
	if out := runAccessGroupNameValidator(cfg, types.StringNull()); len(out) != 0 {
		t.Errorf("any_idp_user must not require access_group_name, got %v", out)
	}
}

// TestAccessGroupName_DefersWhenDiscriminatorUnknown proves the validator
// defers when enrollment_access itself is unknown.
func TestAccessGroupName_DefersWhenDiscriminatorUnknown(t *testing.T) {
	cfg := accessGroupConfig(
		tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		tftypes.NewValue(tftypes.String, nil),
	)
	if out := runAccessGroupNameValidator(cfg, types.StringNull()); len(out) != 0 {
		t.Errorf("unknown enrollment_access must defer, got %v", out)
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
