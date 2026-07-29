// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_extension_attribute

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// scriptPlan renders a minimal SCRIPT-EA plan model with the given enabled value.
func scriptPlan(enabled types.Bool) ComputerExtensionAttributeResourceModel {
	return ComputerExtensionAttributeResourceModel{
		Name:      types.StringValue("zz-script"),
		DataType:  types.StringValue("STRING"),
		InputType: types.StringValue(inputTypeScript),
		Script:    types.StringValue("#!/bin/sh\necho 1"),
		Enabled:   enabled,
	}
}

// TestBuildComputerExtensionAttributeInput_ManageExistingData pins the wire law
// documented on manageExistingDataFor. It guards two live 400s:
//
//   - create: "[INVALID_CONTENT] manageExistingData: This field should be blank
//     for first time CEA creation."
//   - update of an ENABLED SCRIPT EA (issue #302): "[INVALID_CONTENT]
//     manageExistingData: This field should be blank if the input type is not
//     'SCRIPT' and enabled value is not false"
//
// and the third case, an update that disables the EA, where Jamf Pro *requires*
// the field.
func TestBuildComputerExtensionAttributeInput_ManageExistingData(t *testing.T) {
	tests := []struct {
		name     string
		enabled  types.Bool
		config   types.String
		isCreate bool
		want     *string // nil = must be omitted
	}{
		{"create enabled omits", types.BoolValue(true), types.StringNull(), true, nil},
		{"create disabled omits", types.BoolValue(false), types.StringNull(), true, nil},
		{"create disabled ignores explicit value", types.BoolValue(false), types.StringValue("DELETE"), true, nil},
		{"update enabled omits", types.BoolValue(true), types.StringNull(), false, nil},
		{"update enabled omits even when configured", types.BoolValue(true), types.StringValue("DELETE"), false, nil},
		{"update null enabled treated as enabled", types.BoolNull(), types.StringNull(), false, nil},
		{"update unknown enabled treated as enabled", types.BoolUnknown(), types.StringNull(), false, nil},
		{"update disabled defaults to RETAIN", types.BoolValue(false), types.StringNull(), false, new(manageExistingDataDefault)},
		{"update disabled honours DELETE", types.BoolValue(false), types.StringValue("DELETE"), false, new("DELETE")},
		{"update disabled honours RETAIN", types.BoolValue(false), types.StringValue("RETAIN"), false, new("RETAIN")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ea, diags := buildComputerExtensionAttributeInput(context.Background(), scriptPlan(tc.enabled), tc.config, tc.isCreate)
			if diags.HasError() {
				t.Fatalf("diags: %v", diags)
			}
			switch {
			case tc.want == nil && ea.ManageExistingData != nil:
				t.Fatalf("manageExistingData must be omitted, got %q", *ea.ManageExistingData)
			case tc.want != nil && ea.ManageExistingData == nil:
				t.Fatalf("manageExistingData must be %q, got nil", *tc.want)
			case tc.want != nil && *ea.ManageExistingData != *tc.want:
				t.Fatalf("manageExistingData = %q, want %q", *ea.ManageExistingData, *tc.want)
			}
		})
	}
}

// TestBuildComputerExtensionAttributeInput_ManageExistingDataNonScript proves the
// field never reaches the wire for a non-SCRIPT input type, on create or update.
func TestBuildComputerExtensionAttributeInput_ManageExistingDataNonScript(t *testing.T) {
	for _, inputType := range []string{inputTypeText, inputTypePopup, inputTypeLDAP} {
		for _, isCreate := range []bool{true, false} {
			plan := ComputerExtensionAttributeResourceModel{
				Name:      types.StringValue("zz-other"),
				DataType:  types.StringValue("STRING"),
				InputType: types.StringValue(inputType),
				Enabled:   types.BoolValue(true),
			}
			ea, diags := buildComputerExtensionAttributeInput(context.Background(), plan, types.StringValue("RETAIN"), isCreate)
			if diags.HasError() {
				t.Fatalf("%s create=%v diags: %v", inputType, isCreate, diags)
			}
			if ea.ManageExistingData != nil {
				t.Fatalf("%s create=%v must omit manageExistingData, got %q", inputType, isCreate, *ea.ManageExistingData)
			}
		}
	}
}
