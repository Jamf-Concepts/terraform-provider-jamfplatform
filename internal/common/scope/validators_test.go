// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// scopeConfig models the synthetic resource shape used to drive the validator
// tests. all_computers is the bool under test; computers and computer_groups
// are the conflicting Set<String> siblings.
var scopeAttrTypes = map[string]tftypes.Type{
	"all_computers":   tftypes.Bool,
	"computers":       tftypes.Set{ElementType: tftypes.String},
	"computer_groups": tftypes.Set{ElementType: tftypes.String},
}

var scopeSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"all_computers":   schema.BoolAttribute{Optional: true},
		"computers":       schema.SetAttribute{ElementType: types.StringType, Optional: true},
		"computer_groups": schema.SetAttribute{ElementType: types.StringType, Optional: true},
	},
}

// stringSetTFValue builds a tftypes.Value for a Set<String>. Pass nil for a
// null Set; an empty (non-nil) slice for the zero-element Set.
func stringSetTFValue(values []string) tftypes.Value {
	if values == nil {
		return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil)
	}
	elems := make([]tftypes.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, tftypes.NewValue(tftypes.String, v))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, elems)
}

// buildConfig assembles a tfsdk.Config matching scopeSchema.
func buildConfig(allComputers tftypes.Value, computers, computerGroups []string) tfsdk.Config {
	return tfsdk.Config{
		Schema: scopeSchema,
		Raw: tftypes.NewValue(tftypes.Object{AttributeTypes: scopeAttrTypes}, map[string]tftypes.Value{
			"all_computers":   allComputers,
			"computers":       stringSetTFValue(computers),
			"computer_groups": stringSetTFValue(computerGroups),
		}),
	}
}

// runValidator drives the validator against a synthetic request shaped like
// what the framework would produce.
func runValidator(t *testing.T, configValue types.Bool, cfg tfsdk.Config) validator.BoolResponse {
	t.Helper()
	req := validator.BoolRequest{
		Path:           path.Root("all_computers"),
		PathExpression: path.MatchRoot("all_computers"),
		Config:         cfg,
		ConfigValue:    configValue,
	}
	var resp validator.BoolResponse
	v := AllFlagConflictsWith(
		path.MatchRoot("computers"),
		path.MatchRoot("computer_groups"),
	)
	v.ValidateBool(context.Background(), req, &resp)
	return resp
}

func TestAllFlagConflictsWith_FalseValue_NoViolation(t *testing.T) {
	resp := runValidator(t,
		types.BoolValue(false),
		buildConfig(tftypes.NewValue(tftypes.Bool, false), []string{"1"}, []string{"2"}),
	)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestAllFlagConflictsWith_NullValue_NoViolation(t *testing.T) {
	resp := runValidator(t,
		types.BoolNull(),
		buildConfig(tftypes.NewValue(tftypes.Bool, nil), []string{"1"}, nil),
	)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestAllFlagConflictsWith_UnknownValue_NoViolation(t *testing.T) {
	resp := runValidator(t,
		types.BoolUnknown(),
		buildConfig(tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue), []string{"1"}, nil),
	)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestAllFlagConflictsWith_TrueValue_NullConflicts_NoViolation(t *testing.T) {
	resp := runValidator(t,
		types.BoolValue(true),
		buildConfig(tftypes.NewValue(tftypes.Bool, true), nil, nil),
	)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestAllFlagConflictsWith_TrueValue_EmptyConflicts_NoViolation(t *testing.T) {
	resp := runValidator(t,
		types.BoolValue(true),
		buildConfig(tftypes.NewValue(tftypes.Bool, true), []string{}, []string{}),
	)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestAllFlagConflictsWith_TrueValue_OneConflictPopulated_OneError(t *testing.T) {
	resp := runValidator(t,
		types.BoolValue(true),
		buildConfig(tftypes.NewValue(tftypes.Bool, true), []string{"1"}, nil),
	)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected one attribute error")
	}
	if resp.Diagnostics.ErrorsCount() != 1 {
		t.Errorf("expected exactly 1 error, got %d", resp.Diagnostics.ErrorsCount())
	}
	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if attrDiag, ok := d.(interface{ Path() path.Path }); ok {
			if attrDiag.Path().Equal(path.Root("computers")) {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected error attached to computers path, got: %v", resp.Diagnostics)
	}
}

func TestAllFlagConflictsWith_TrueValue_MultipleConflictsPopulated_MultipleErrors(t *testing.T) {
	resp := runValidator(t,
		types.BoolValue(true),
		buildConfig(tftypes.NewValue(tftypes.Bool, true), []string{"1"}, []string{"2"}),
	)
	if resp.Diagnostics.ErrorsCount() != 2 {
		t.Errorf("expected exactly 2 errors, got %d: %v", resp.Diagnostics.ErrorsCount(), resp.Diagnostics)
	}
}

func TestAllFlagConflictsWith_Describer(t *testing.T) {
	v := AllFlagConflictsWith(path.MatchRoot("computers"))
	if v.Description(context.Background()) == "" {
		t.Error("expected non-empty Description")
	}
	if v.MarkdownDescription(context.Background()) == "" {
		t.Error("expected non-empty MarkdownDescription")
	}
}
