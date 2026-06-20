// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact_alert_notification_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The validator reads the four toggles as top-level bools, checking IsNull/IsUnknown and
// the value, so a minimal four-bool schema/object suffices.

var rootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"deployable_objects_alert_enabled":             tftypes.Bool,
	"deployable_objects_confirmation_code_enabled": tftypes.Bool,
	"scopeable_objects_alert_enabled":              tftypes.Bool,
	"scopeable_objects_confirmation_code_enabled":  tftypes.Bool,
}}

func minimalSchema() schema.Schema {
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"deployable_objects_alert_enabled":             schema.BoolAttribute{Optional: true, Computed: true},
		"deployable_objects_confirmation_code_enabled": schema.BoolAttribute{Optional: true, Computed: true},
		"scopeable_objects_alert_enabled":              schema.BoolAttribute{Optional: true, Computed: true},
		"scopeable_objects_confirmation_code_enabled":  schema.BoolAttribute{Optional: true, Computed: true},
	}}
}

// boolVal returns null (nil), unknown, or a concrete bool depending on the marker.
func boolNull() tftypes.Value    { return tftypes.NewValue(tftypes.Bool, nil) }
func boolUnknown() tftypes.Value { return tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue) }
func boolOf(b bool) tftypes.Value {
	return tftypes.NewValue(tftypes.Bool, b)
}

func cfg(depAlert, depConf, scopeAlert, scopeConf tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: minimalSchema(),
		Raw: tftypes.NewValue(rootObjType, map[string]tftypes.Value{
			"deployable_objects_alert_enabled":             depAlert,
			"deployable_objects_confirmation_code_enabled": depConf,
			"scopeable_objects_alert_enabled":              scopeAlert,
			"scopeable_objects_confirmation_code_enabled":  scopeConf,
		}),
	}
}

func runValidator(c tfsdk.Config) map[string]bool {
	var resp resource.ValidateConfigResponse
	confirmationCodeRequiresAlertValidator{}.ValidateResource(
		context.Background(),
		resource.ValidateConfigRequest{Config: c},
		&resp,
	)
	paths := make(map[string]bool)
	for _, d := range resp.Diagnostics {
		if dwp, ok := d.(diag.DiagnosticWithPath); ok && d.Severity() == diag.SeverityError {
			paths[dwp.Path().String()] = true
		}
	}
	return paths
}

func errCount(c tfsdk.Config) int {
	var resp resource.ValidateConfigResponse
	confirmationCodeRequiresAlertValidator{}.ValidateResource(
		context.Background(),
		resource.ValidateConfigRequest{Config: c},
		&resp,
	)
	n := 0
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError {
			n++
		}
	}
	return n
}

func TestValidator_BothPairsValid(t *testing.T) {
	c := cfg(boolOf(true), boolOf(true), boolOf(true), boolOf(true))
	if n := errCount(c); n != 0 {
		t.Errorf("alert=true + confcode=true on both pairs must pass; got %d errors", n)
	}
}

func TestValidator_AllFalseValid(t *testing.T) {
	c := cfg(boolOf(false), boolOf(false), boolOf(false), boolOf(false))
	if n := errCount(c); n != 0 {
		t.Errorf("all false must pass; got %d errors", n)
	}
}

func TestValidator_DeployableConfWithoutAlert(t *testing.T) {
	c := cfg(boolOf(false), boolOf(true), boolOf(false), boolOf(false))
	if !runValidator(c)["deployable_objects_confirmation_code_enabled"] {
		t.Error("deployable confcode=true with alert=false must error on deployable_objects_confirmation_code_enabled")
	}
}

func TestValidator_ScopeableConfWithoutAlert(t *testing.T) {
	c := cfg(boolOf(false), boolOf(false), boolOf(false), boolOf(true))
	if !runValidator(c)["scopeable_objects_confirmation_code_enabled"] {
		t.Error("scopeable confcode=true with alert=false must error on scopeable_objects_confirmation_code_enabled")
	}
}

func TestValidator_BothPairsViolate(t *testing.T) {
	c := cfg(boolOf(false), boolOf(true), boolOf(false), boolOf(true))
	if n := errCount(c); n != 2 {
		t.Errorf("both pairs violating must produce 2 errors; got %d", n)
	}
}

// defer-on-null: confcode declared true, alert omitted (null = preserve current). The
// validator cannot see the preserved server value, so it must defer (the server 400 is
// the backstop) rather than false-error.
func TestValidator_ConfTrueAlertNull_Defers(t *testing.T) {
	c := cfg(boolNull(), boolOf(true), boolOf(false), boolOf(false))
	if runValidator(c)["deployable_objects_confirmation_code_enabled"] {
		t.Error("confcode=true with alert null (preserve) must defer, not error")
	}
}

// defer-on-unknown: a toggle sourced from a variable / another resource is unknown.
func TestValidator_UnknownAlert_Defers(t *testing.T) {
	c := cfg(boolUnknown(), boolOf(true), boolOf(false), boolOf(false))
	if n := errCount(c); n != 0 {
		t.Errorf("unknown alert must defer (no errors); got %d", n)
	}
}

func TestValidator_UnknownConf_Defers(t *testing.T) {
	c := cfg(boolOf(false), boolUnknown(), boolOf(false), boolOf(false))
	if n := errCount(c); n != 0 {
		t.Errorf("unknown confcode must defer (no errors); got %d", n)
	}
}
