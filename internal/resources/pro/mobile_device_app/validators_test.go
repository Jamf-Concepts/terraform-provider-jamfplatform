// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

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

// afterInstallGateObjType is the tftypes shape of the minimal config the
// validator reads: the gated label under self_service and its toggle under
// general. The two live in different top-level blocks, which is what the
// absolute path in makeAvailableAfterInstallPath exists for.
var afterInstallGateObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"general": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"make_available_after_install": tftypes.Bool,
	}},
	"self_service": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"after_install_button_text": tftypes.String,
	}},
}}

// buildAfterInstallConfig synthesises a tfsdk.Config carrying the toggle and the
// label, each known, unknown or null.
func buildAfterInstallConfig(gate, label tftypes.Value) tfsdk.Config {
	generalType := afterInstallGateObjType.AttributeTypes["general"]
	selfServiceType := afterInstallGateObjType.AttributeTypes["self_service"]
	return tfsdk.Config{
		Schema: rschema.Schema{Attributes: map[string]rschema.Attribute{
			"general": rschema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]rschema.Attribute{
					"make_available_after_install": rschema.BoolAttribute{Optional: true, Computed: true},
				},
			},
			"self_service": rschema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]rschema.Attribute{
					"after_install_button_text": rschema.StringAttribute{Optional: true, Computed: true},
				},
			},
		}},
		Raw: tftypes.NewValue(afterInstallGateObjType, map[string]tftypes.Value{
			"general": tftypes.NewValue(generalType, map[string]tftypes.Value{
				"make_available_after_install": gate,
			}),
			"self_service": tftypes.NewValue(selfServiceType, map[string]tftypes.Value{
				"after_install_button_text": label,
			}),
		}),
	}
}

// runAfterInstall executes the validator and returns the error count.
func runAfterInstall(t *testing.T, labelConfigValue types.String, gate, label tftypes.Value) int {
	t.Helper()
	req := validator.StringRequest{
		Path:        path.Root("self_service").AtName("after_install_button_text"),
		ConfigValue: labelConfigValue,
		Config:      buildAfterInstallConfig(gate, label),
	}
	var resp validator.StringResponse
	requiresMakeAvailableAfterInstall{}.ValidateString(context.Background(), req, &resp)
	return resp.Diagnostics.ErrorsCount()
}

func TestRequiresMakeAvailableAfterInstall_Descriptions(t *testing.T) {
	t.Parallel()
	v := requiresMakeAvailableAfterInstall{}
	if v.Description(context.Background()) == "" || v.MarkdownDescription(context.Background()) == "" {
		t.Error("Description / MarkdownDescription must not be empty")
	}
}

// TestRequiresMakeAvailableAfterInstall_FiresWhenGateFalse is the case the
// validator exists for: a label Jamf Pro will silently discard.
func TestRequiresMakeAvailableAfterInstall_FiresWhenGateFalse(t *testing.T) {
	t.Parallel()
	got := runAfterInstall(t,
		types.StringValue("Open"),
		tftypes.NewValue(tftypes.Bool, false),
		tftypes.NewValue(tftypes.String, "Open"),
	)
	if got != 1 {
		t.Errorf("expected 1 error when the label is set and the toggle is false, got %d", got)
	}
}

func TestRequiresMakeAvailableAfterInstall_SilentWhenGateTrue(t *testing.T) {
	t.Parallel()
	got := runAfterInstall(t,
		types.StringValue("Open"),
		tftypes.NewValue(tftypes.Bool, true),
		tftypes.NewValue(tftypes.String, "Open"),
	)
	if got != 0 {
		t.Errorf("expected no error when the toggle is true, got %d", got)
	}
}

// TestRequiresMakeAvailableAfterInstall_SilentWhenGateUnknown is the case
// acceptance tests cannot reach: a toggle sourced from a variable, count,
// for_each or another resource is Unknown at config-validation time, and
// erroring there would make the resource unusable from any reusable module.
// STYLE_GUIDE.md §"Config-time validators MUST defer on unknown values".
func TestRequiresMakeAvailableAfterInstall_SilentWhenGateUnknown(t *testing.T) {
	t.Parallel()
	got := runAfterInstall(t,
		types.StringValue("Open"),
		tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		tftypes.NewValue(tftypes.String, "Open"),
	)
	if got != 0 {
		t.Errorf("expected no error when the toggle is unknown, got %d", got)
	}
}

// TestRequiresMakeAvailableAfterInstall_SilentWhenGateNull covers an omitted
// Optional+Computed toggle: it resolves to the server's own default, which this
// validator has no business second-guessing at config time.
func TestRequiresMakeAvailableAfterInstall_SilentWhenGateNull(t *testing.T) {
	t.Parallel()
	got := runAfterInstall(t,
		types.StringValue("Open"),
		tftypes.NewValue(tftypes.Bool, nil),
		tftypes.NewValue(tftypes.String, "Open"),
	)
	if got != 0 {
		t.Errorf("expected no error when the toggle is null, got %d", got)
	}
}

// TestRequiresMakeAvailableAfterInstall_SilentWhenLabelNotSet covers both
// no-op cases for the validated attribute itself: an unset label has nothing to
// discard, and an unknown one is not resolvable yet.
func TestRequiresMakeAvailableAfterInstall_SilentWhenLabelNotSet(t *testing.T) {
	t.Parallel()
	for name, labelConfigValue := range map[string]types.String{
		"null":    types.StringNull(),
		"unknown": types.StringUnknown(),
	} {
		if got := runAfterInstall(t, labelConfigValue,
			tftypes.NewValue(tftypes.Bool, false),
			tftypes.NewValue(tftypes.String, nil),
		); got != 0 {
			t.Errorf("label %s: expected no error, got %d", name, got)
		}
	}
}
