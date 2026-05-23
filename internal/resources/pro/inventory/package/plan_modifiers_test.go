// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDecideResetForUnchangedString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		sourceEqual bool
		stateValue  types.String
		wantOK      bool
		wantValue   types.String
	}{
		{
			name:        "source changed → leave Unknown",
			sourceEqual: false,
			stateValue:  types.StringValue("c825103b7b1c"),
			wantOK:      false,
		},
		{
			name:        "source unchanged + state known → carry forward",
			sourceEqual: true,
			stateValue:  types.StringValue("c825103b7b1c"),
			wantOK:      true,
			wantValue:   types.StringValue("c825103b7b1c"),
		},
		{
			name:        "source unchanged + state null → leave Unknown",
			sourceEqual: true,
			stateValue:  types.StringNull(),
			wantOK:      false,
		},
		{
			name:        "source unchanged + state unknown → leave Unknown",
			sourceEqual: true,
			stateValue:  types.StringUnknown(),
			wantOK:      false,
		},
		{
			name:        "source changed + state null → leave Unknown",
			sourceEqual: false,
			stateValue:  types.StringNull(),
			wantOK:      false,
		},
		{
			name:        "source unchanged + empty-string state → carry forward (empty is a real value)",
			sourceEqual: true,
			stateValue:  types.StringValue(""),
			wantOK:      true,
			wantValue:   types.StringValue(""),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := decideResetForUnchangedString(tc.sourceEqual, tc.stateValue)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if !got.Equal(tc.wantValue) {
				t.Fatalf("value = %v, want %v", got, tc.wantValue)
			}
		})
	}
}

// testSchema mirrors the minimal shape the resetIfSourceChangedString
// modifier needs: a watched source attribute + a computed attribute the
// modifier is applied to.
func testSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"src":     schema.StringAttribute{Optional: true},
			"watched": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

// buildPlanState builds a (Plan, State) pair carrying the supplied
// `src` + `watched` values for each. Use empty string to mean null;
// pass a value to mean known.
func buildPlanState(t *testing.T, planSrc, planWatched, stateSrc, stateWatched string) (tfsdk.Plan, tfsdk.State) {
	t.Helper()

	s := testSchema()
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"src":     tftypes.String,
		"watched": tftypes.String,
	}}

	val := func(s string) tftypes.Value {
		if s == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	planRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"src":     val(planSrc),
		"watched": val(planWatched),
	})
	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"src":     val(stateSrc),
		"watched": val(stateWatched),
	})

	return tfsdk.Plan{Raw: planRaw, Schema: s}, tfsdk.State{Raw: stateRaw, Schema: s}
}

func TestResetIfSourceChangedString_PlanModifyString(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name        string
		planSrc     string // empty = null
		stateSrc    string
		stateWatch  string
		configValue types.String
		planValue   types.String
		stateValue  types.String
		stateIsNull bool
		planIsNull  bool
		wantPlan    types.String
	}{
		{
			name:       "source unchanged + state value known → carried forward",
			planSrc:    "https://example/v1.pkg",
			stateSrc:   "https://example/v1.pkg",
			stateWatch: "hash-v1",
			planValue:  types.StringUnknown(),
			stateValue: types.StringValue("hash-v1"),
			wantPlan:   types.StringValue("hash-v1"),
		},
		{
			name:       "source changed → plan stays Unknown",
			planSrc:    "https://example/v2.pkg",
			stateSrc:   "https://example/v1.pkg",
			stateWatch: "hash-v1",
			planValue:  types.StringUnknown(),
			stateValue: types.StringValue("hash-v1"),
			wantPlan:   types.StringUnknown(),
		},
		{
			name:        "config value supplied → modifier no-ops",
			planSrc:     "https://example/v1.pkg",
			stateSrc:    "https://example/v1.pkg",
			stateWatch:  "hash-v1",
			configValue: types.StringValue("user-supplied"),
			planValue:   types.StringValue("user-supplied"),
			stateValue:  types.StringValue("hash-v1"),
			wantPlan:    types.StringValue("user-supplied"),
		},
		{
			name:        "create (state null) → modifier no-ops",
			planSrc:     "https://example/v1.pkg",
			stateIsNull: true,
			planValue:   types.StringUnknown(),
			stateValue:  types.StringNull(),
			wantPlan:    types.StringUnknown(),
		},
		{
			name:       "destroy (plan null) → modifier no-ops",
			planIsNull: true,
			planValue:  types.StringNull(),
			stateValue: types.StringValue("hash-v1"),
			wantPlan:   types.StringNull(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plan, state := buildPlanState(t, tc.planSrc, "", tc.stateSrc, tc.stateWatch)
			if tc.stateIsNull {
				state.Raw = tftypes.NewValue(state.Raw.Type(), nil)
			}
			if tc.planIsNull {
				plan.Raw = tftypes.NewValue(plan.Raw.Type(), nil)
			}

			req := planmodifier.StringRequest{
				Path:        path.Root("watched"),
				Plan:        plan,
				State:       state,
				ConfigValue: tc.configValue,
				PlanValue:   tc.planValue,
				StateValue:  tc.stateValue,
			}
			resp := &planmodifier.StringResponse{PlanValue: tc.planValue}

			resetIfSourceChangedString(path.MatchRoot("src")).PlanModifyString(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.wantPlan) {
				t.Fatalf("plan value = %v, want %v", resp.PlanValue, tc.wantPlan)
			}
		})
	}
}
