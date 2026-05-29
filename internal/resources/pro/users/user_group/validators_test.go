// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var critObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"priority":                tftypes.Number,
	"name":                    tftypes.String,
	"search_type":             tftypes.String,
	"value":                   tftypes.String,
	"and_or":                  tftypes.String,
	"has_opening_parenthesis": tftypes.Bool,
	"has_closing_parenthesis": tftypes.Bool,
}}

var (
	criteriaListType = tftypes.List{ElementType: critObjType}
	membersSetType   = tftypes.Set{ElementType: tftypes.String}
)

// criteriaMode selects the tftypes value for the criteria attribute.
type criteriaMode int

const (
	critNull criteriaMode = iota
	critUnknown
	critEmpty
	critOne
)

func criteriaValue(m criteriaMode) tftypes.Value {
	switch m {
	case critUnknown:
		return tftypes.NewValue(criteriaListType, tftypes.UnknownValue)
	case critEmpty:
		return tftypes.NewValue(criteriaListType, []tftypes.Value{})
	case critOne:
		return tftypes.NewValue(criteriaListType, []tftypes.Value{
			tftypes.NewValue(critObjType, map[string]tftypes.Value{
				"priority":                tftypes.NewValue(tftypes.Number, 0),
				"name":                    tftypes.NewValue(tftypes.String, "Username"),
				"search_type":             tftypes.NewValue(tftypes.String, "is"),
				"value":                   tftypes.NewValue(tftypes.String, "jappleseed"),
				"and_or":                  tftypes.NewValue(tftypes.String, "and"),
				"has_opening_parenthesis": tftypes.NewValue(tftypes.Bool, false),
				"has_closing_parenthesis": tftypes.NewValue(tftypes.Bool, false),
			}),
		})
	default:
		return tftypes.NewValue(criteriaListType, nil)
	}
}

// ugConfig builds a Config with only the three attributes the validator reads.
func ugConfig(groupType *string, crit criteriaMode, membersSet bool) tfsdk.Config {
	gtVal := tftypes.NewValue(tftypes.String, nil)
	if groupType != nil {
		gtVal = tftypes.NewValue(tftypes.String, *groupType)
	}
	membersVal := tftypes.NewValue(membersSetType, nil)
	if membersSet {
		membersVal = tftypes.NewValue(membersSetType, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "123"),
		})
	}

	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"group_type": tftypes.String,
		"criteria":   criteriaListType,
		"members":    membersSetType,
	}}

	critAttrs := map[string]schema.Attribute{
		"priority":                schema.Int64Attribute{Optional: true, Computed: true},
		"name":                    schema.StringAttribute{Required: true},
		"search_type":             schema.StringAttribute{Required: true},
		"value":                   schema.StringAttribute{Required: true},
		"and_or":                  schema.StringAttribute{Optional: true, Computed: true},
		"has_opening_parenthesis": schema.BoolAttribute{Optional: true, Computed: true},
		"has_closing_parenthesis": schema.BoolAttribute{Optional: true, Computed: true},
	}

	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"group_type": schema.StringAttribute{Optional: true},
			"members":    schema.SetAttribute{Optional: true, ElementType: types.StringType},
			"criteria": schema.ListNestedAttribute{
				Optional:     true,
				NestedObject: schema.NestedAttributeObject{Attributes: critAttrs},
			},
		}},
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"group_type": gtVal,
			"criteria":   criteriaValue(crit),
			"members":    membersVal,
		}),
	}
}

func runSmartStaticValidator(cfg tfsdk.Config) []string {
	var resp resource.ValidateConfigResponse
	smartStaticConfigValidator{}.ValidateResource(
		context.Background(),
		resource.ValidateConfigRequest{Config: cfg},
		&resp,
	)
	out := make([]string, 0, len(resp.Diagnostics))
	for _, d := range resp.Diagnostics {
		out = append(out, d.Summary())
	}
	return out
}

// TestSmartStatic_DefersWhenCriteriaUnknown is the defer-on-unknown regression
// guard: smart group + unknown criteria (e.g. variable-driven) must DEFER, not
// treat unknown as an empty/missing list and error. See STYLE_GUIDE
// "Config-time validators must defer on unknown values".
func TestSmartStatic_DefersWhenCriteriaUnknown(t *testing.T) {
	if out := runSmartStaticValidator(ugConfig(new("smart"), critUnknown, false)); len(out) != 0 {
		t.Errorf("smart + unknown criteria must defer, got %v", out)
	}
}

// TestSmartStatic_ErrorsWhenSmartCriteriaEmpty proves the requirement still
// fires for a known-empty/absent criteria on a smart group.
func TestSmartStatic_ErrorsWhenSmartCriteriaEmpty(t *testing.T) {
	if out := runSmartStaticValidator(ugConfig(new("smart"), critEmpty, false)); len(out) == 0 {
		t.Error("expected error for smart group with empty criteria")
	}
	if out := runSmartStaticValidator(ugConfig(new("smart"), critNull, false)); len(out) == 0 {
		t.Error("expected error for smart group with no criteria")
	}
}

// TestSmartStatic_SmartWithCriteriaOK proves a smart group with criteria passes.
func TestSmartStatic_SmartWithCriteriaOK(t *testing.T) {
	if out := runSmartStaticValidator(ugConfig(new("smart"), critOne, false)); len(out) != 0 {
		t.Errorf("smart + one criterion should pass, got %v", out)
	}
}

// TestSmartStatic_StaticForbidsKnownCriteria proves a static group errors on a
// known non-empty criteria but DEFERS on unknown criteria.
func TestSmartStatic_StaticForbidsKnownCriteria(t *testing.T) {
	if out := runSmartStaticValidator(ugConfig(new("static"), critOne, true)); len(out) == 0 {
		t.Error("expected error for static group with criteria set")
	}
	if out := runSmartStaticValidator(ugConfig(new("static"), critUnknown, false)); len(out) != 0 {
		t.Errorf("static + unknown criteria must defer, got %v", out)
	}
}

// TestSmartStatic_DefersWhenGroupTypeUnknownOrUnset proves the validator defers
// when the discriminator is not yet known.
func TestSmartStatic_DefersWhenGroupTypeUnknownOrUnset(t *testing.T) {
	if out := runSmartStaticValidator(ugConfig(nil, critNull, false)); len(out) != 0 {
		t.Errorf("unset group_type must defer, got %v", out)
	}
}
