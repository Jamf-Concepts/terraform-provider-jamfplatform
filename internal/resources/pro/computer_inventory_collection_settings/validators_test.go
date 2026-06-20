// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_inventory_collection_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// validatorObjType is the tftypes shape of the minimal config the validator reads:
// the gated sub-option plus its parent toggle.
var validatorObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"collect_local_user_accounts":  tftypes.Bool,
	"include_home_directory_sizes": tftypes.Bool,
}}

// buildValidatorConfig synthesises a tfsdk.Config carrying the parent toggle and the
// gated sub-option, each as a known bool, unknown, or null (nil pointer = null).
func buildValidatorConfig(parent, child tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: rschema.Schema{Attributes: map[string]rschema.Attribute{
			"collect_local_user_accounts":  rschema.BoolAttribute{Optional: true, Computed: true},
			"include_home_directory_sizes": rschema.BoolAttribute{Optional: true, Computed: true},
		}},
		Raw: tftypes.NewValue(validatorObjType, map[string]tftypes.Value{
			"collect_local_user_accounts":  parent,
			"include_home_directory_sizes": child,
		}),
	}
}

// run executes requiresAccountCollection against a child ConfigValue and a parent value
// in the config, returning the diagnostic count.
func run(t *testing.T, childConfigValue types.Bool, parent tftypes.Value, child tftypes.Value) int {
	t.Helper()
	req := validator.BoolRequest{
		Path:        path.Root("include_home_directory_sizes"),
		ConfigValue: childConfigValue,
		Config:      buildValidatorConfig(parent, child),
	}
	var resp validator.BoolResponse
	requiresAccountCollection{}.ValidateBool(context.Background(), req, &resp)
	return resp.Diagnostics.ErrorsCount()
}

func TestRequiresAccountCollection_Descriptions(t *testing.T) {
	v := requiresAccountCollection{}
	if v.Description(context.Background()) == "" || v.MarkdownDescription(context.Background()) == "" {
		t.Error("Description / MarkdownDescription must not be empty")
	}
}

func TestRequiresAccountCollection_FiresWhenChildTrueParentFalse(t *testing.T) {
	got := run(t,
		types.BoolValue(true),
		tftypes.NewValue(tftypes.Bool, false),
		tftypes.NewValue(tftypes.Bool, true),
	)
	if got != 1 {
		t.Errorf("expected 1 error when sub-option true and parent false, got %d", got)
	}
}

func TestRequiresAccountCollection_SilentWhenParentTrue(t *testing.T) {
	got := run(t,
		types.BoolValue(true),
		tftypes.NewValue(tftypes.Bool, true),
		tftypes.NewValue(tftypes.Bool, true),
	)
	if got != 0 {
		t.Errorf("expected no error when parent true, got %d", got)
	}
}

func TestRequiresAccountCollection_SilentWhenChildFalse(t *testing.T) {
	got := run(t,
		types.BoolValue(false),
		tftypes.NewValue(tftypes.Bool, false),
		tftypes.NewValue(tftypes.Bool, false),
	)
	if got != 0 {
		t.Errorf("expected no error when sub-option false, got %d", got)
	}
}

func TestRequiresAccountCollection_SilentWhenChildNullOrUnknown(t *testing.T) {
	if got := run(t, types.BoolNull(), tftypes.NewValue(tftypes.Bool, false), tftypes.NewValue(tftypes.Bool, nil)); got != 0 {
		t.Errorf("null sub-option: expected no error, got %d", got)
	}
	if got := run(t, types.BoolUnknown(), tftypes.NewValue(tftypes.Bool, false), tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)); got != 0 {
		t.Errorf("unknown sub-option: expected no error, got %d", got)
	}
}

// TestRequiresAccountCollection_SilentWhenParentUnknownOrNull is the defer-on-unknown
// regression test: the parent sourced from a variable/resource (unknown) or omitted
// (null) must not produce a false positive, even with the sub-option set true.
func TestRequiresAccountCollection_SilentWhenParentUnknownOrNull(t *testing.T) {
	if got := run(t, types.BoolValue(true), tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue), tftypes.NewValue(tftypes.Bool, true)); got != 0 {
		t.Errorf("unknown parent: expected no error (defer), got %d", got)
	}
	if got := run(t, types.BoolValue(true), tftypes.NewValue(tftypes.Bool, nil), tftypes.NewValue(tftypes.Bool, true)); got != 0 {
		t.Errorf("null parent: expected no error (defer), got %d", got)
	}
}
