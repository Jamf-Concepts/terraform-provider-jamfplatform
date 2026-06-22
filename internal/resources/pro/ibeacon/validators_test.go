// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ibeacon

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestValidator_Descriptions confirms the ConfigValidator implements the
// Description / MarkdownDescription contract (framework requires both). The
// validator's value-based logic is exercised end-to-end by the acceptance
// tests in resource_acceptance_test.go (a plan-time check needs a live tfsdk
// Config to apply against; constructing one in unit tests with the right
// underlying tftypes value is brittle and adds no signal beyond the
// apply-time validateIbeaconPlan helper, which has full unit coverage in
// helpers_test.go). See user_group/ for the same pattern.
func TestValidator_Descriptions(t *testing.T) {
	v := includeAnyMajorMinorConfigValidator{}
	if v.Description(context.Background()) == "" {
		t.Errorf("Description must not be empty")
	}
	if v.MarkdownDescription(context.Background()) == "" {
		t.Errorf("MarkdownDescription must not be empty")
	}
}

// ibeaconConfig builds a tfsdk.Config from the real resource schema, filling
// every attribute null except those supplied in attrs (keyed by attribute
// name).
func ibeaconConfig(t *testing.T, attrs map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	r := NewIbeaconResource()
	var sr resource.SchemaResponse
	r.(*IbeaconResource).Schema(context.Background(), resource.SchemaRequest{}, &sr)
	if sr.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sr.Diagnostics)
	}
	objType := sr.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, typ := range objType.AttributeTypes {
		if v, ok := attrs[name]; ok {
			values[name] = v
			continue
		}
		values[name] = tftypes.NewValue(typ, nil)
	}
	return tfsdk.Config{Schema: sr.Schema, Raw: tftypes.NewValue(objType, values)}
}

// TestIncludeAnyMajorMinorConfigValidator_DefersOnUnknown is the §436 regression
// guard: with include_any_*_value = false (known) and the corresponding major /
// minor UNKNOWN (variable/for_each/resource-driven), the validator MUST defer,
// not treat unknown as missing and error. See STYLE_GUIDE "Config-time
// validators MUST defer on unknown values".
func TestIncludeAnyMajorMinorConfigValidator_DefersOnUnknown(t *testing.T) {
	unknownNum := tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)
	anyTrue := tftypes.NewValue(tftypes.Bool, true)
	// Each case keeps the OTHER axis in a valid state (include_any = true) so
	// only the axis under test — with its value unknown — is exercised.
	cases := []struct {
		name  string
		attrs map[string]tftypes.Value
	}{
		{"major unknown", map[string]tftypes.Value{"include_any_major_value": tftypes.NewValue(tftypes.Bool, false), "major": unknownNum, "include_any_minor_value": anyTrue}},
		{"minor unknown", map[string]tftypes.Value{"include_any_minor_value": tftypes.NewValue(tftypes.Bool, false), "minor": unknownNum, "include_any_major_value": anyTrue}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := resource.ValidateConfigRequest{Config: ibeaconConfig(t, tc.attrs)}
			var resp resource.ValidateConfigResponse
			includeAnyMajorMinorConfigValidator{}.ValidateResource(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("validator must defer on unknown axis value, got %v", resp.Diagnostics)
			}
		})
	}
}
