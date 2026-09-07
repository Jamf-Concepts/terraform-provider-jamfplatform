// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package planmodifiers

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

			got, ok := DecideResetForUnchangedString(tc.sourceEqual, tc.stateValue)
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

// testSchema mirrors the minimal shape the ResetIfSourceChangedString
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

			ResetIfSourceChangedString(path.MatchRoot("src")).PlanModifyString(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.wantPlan) {
				t.Fatalf("plan value = %v, want %v", resp.PlanValue, tc.wantPlan)
			}
		})
	}
}

// mirrorSchema and buildMirrorPlanState build a two-attribute resource whose
// `mirror` string is derived from a `flag` bool — the shape of
// jamfplatform_pro_mobile_device_app's general.deployment_type over
// general.deploy_automatically. The bool source is the case the pre-existing
// string-only reader could not express.
func mirrorSchema() schema.Schema {
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"flag":   schema.BoolAttribute{Optional: true, Computed: true},
		"mirror": schema.StringAttribute{Computed: true},
	}}
}

func buildMirrorPlanState(configFlag, stateFlag *bool, planMirror, stateMirror types.String) (tfsdk.Config, tfsdk.Plan, tfsdk.State) {
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"flag":   tftypes.Bool,
		"mirror": tftypes.String,
	}}
	boolVal := func(b *bool) tftypes.Value {
		if b == nil {
			return tftypes.NewValue(tftypes.Bool, nil)
		}
		return tftypes.NewValue(tftypes.Bool, *b)
	}
	strVal := func(v types.String) tftypes.Value {
		switch {
		case v.IsUnknown():
			return tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		case v.IsNull():
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, v.ValueString())
	}
	sch := mirrorSchema()
	return tfsdk.Config{Schema: sch, Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"flag": boolVal(configFlag), "mirror": tftypes.NewValue(tftypes.String, nil),
		})},
		tfsdk.Plan{Schema: sch, Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"flag": boolVal(configFlag), "mirror": strVal(planMirror),
		})},
		tfsdk.State{Schema: sch, Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"flag": boolVal(stateFlag), "mirror": strVal(stateMirror),
		})}
}

// TestMirrorOfString_PlanModifyString exercises the modifier end to end over a
// BOOL source, which is the whole reason it exists: general.deployment_type
// mirrors a bool, and the prior string-only source reader could not read it.
func TestMirrorOfString_PlanModifyString(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	yes, no := true, false

	for _, tc := range []struct {
		name                  string
		configFlag, stateFlag *bool
		stateValue            types.String
		stateIsNull           bool
		planIsNull            bool
		want                  types.String
	}{
		{
			name:       "source unchanged → prior value carried forward, so plan -refresh=false is clean",
			configFlag: &no, stateFlag: &no,
			stateValue: types.StringValue("Make Available in Self Service"),
			want:       types.StringValue("Make Available in Self Service"),
		},
		{
			name:       "source moved → Unknown, so the apply can return the new mirrored value",
			configFlag: &yes, stateFlag: &no,
			stateValue: types.StringValue("Make Available in Self Service"),
			want:       types.StringUnknown(),
		},
		{
			// The case that made `plan -refresh=false` dirty forever: a config
			// that never touches the source, on a resource whose mirror has no
			// value. Null must be carried, not resolved to Unknown.
			name:       "source unchanged and prior value null → null carried forward",
			configFlag: &no, stateFlag: &no,
			stateValue: types.StringNull(),
			want:       types.StringNull(),
		},
		{
			name:       "source absent from config counts as unchanged",
			configFlag: nil, stateFlag: nil,
			stateValue: types.StringValue("carried"),
			want:       types.StringValue("carried"),
		},
		{
			// An unconfigured Optional+Computed source keeps its prior value,
			// so the mirror keeps its own. Reading the PLAN here would see the
			// source Unknown and wrongly call it a change.
			name:       "source absent from config while state holds a value → still unchanged",
			configFlag: nil, stateFlag: &yes,
			stateValue: types.StringValue("carried"),
			want:       types.StringValue("carried"),
		},
		{
			name:       "create (state null) → left Unknown",
			configFlag: &no, stateFlag: &no, stateIsNull: true,
			stateValue: types.StringNull(),
			want:       types.StringUnknown(),
		},
		{
			name:       "destroy (plan null) → left Unknown",
			configFlag: &no, stateFlag: &no, planIsNull: true,
			stateValue: types.StringValue("whatever"),
			want:       types.StringUnknown(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config, plan, state := buildMirrorPlanState(tc.configFlag, tc.stateFlag, types.StringUnknown(), tc.stateValue)
			if tc.stateIsNull {
				state.Raw = tftypes.NewValue(state.Raw.Type(), nil)
			}
			if tc.planIsNull {
				plan.Raw = tftypes.NewValue(plan.Raw.Type(), nil)
			}

			req := planmodifier.StringRequest{
				Path:       path.Root("mirror"),
				Config:     config,
				Plan:       plan,
				State:      state,
				StateValue: tc.stateValue,
				PlanValue:  types.StringUnknown(),
			}
			resp := &planmodifier.StringResponse{PlanValue: types.StringUnknown()}

			MirrorOfString(BoolSource(path.MatchRoot("flag"))).PlanModifyString(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.want) {
				t.Errorf("plan value = %#v, want %#v", resp.PlanValue, tc.want)
			}
		})
	}
}

// TestMirrorOfString_DescriptionsAreSet keeps the modifier self-describing: the
// framework surfaces Description in plan output on some paths, and an empty one
// is a papercut no other test would catch.
func TestMirrorOfString_DescriptionsAreSet(t *testing.T) {
	t.Parallel()

	m := MirrorOfString(BoolSource(path.MatchRoot("flag")))
	if m.Description(context.Background()) == "" {
		t.Error("Description must not be empty")
	}
	if m.MarkdownDescription(context.Background()) == "" {
		t.Error("MarkdownDescription must not be empty")
	}
}
