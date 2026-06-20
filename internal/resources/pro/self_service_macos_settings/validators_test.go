// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_macos_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The validators read four top-level attributes, checking IsNull/IsUnknown and the value,
// so a minimal four-attribute schema/object suffices.

var rootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"install_automatically":    tftypes.Bool,
	"install_location":         tftypes.String,
	"default_landing_page":     tftypes.String,
	"default_home_category_id": tftypes.Number,
}}

func minimalSchema() schema.Schema {
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"install_automatically":    schema.BoolAttribute{Optional: true, Computed: true},
		"install_location":         schema.StringAttribute{Optional: true, Computed: true},
		"default_landing_page":     schema.StringAttribute{Optional: true, Computed: true},
		"default_home_category_id": schema.Int64Attribute{Optional: true, Computed: true},
	}}
}

func boolNull() tftypes.Value      { return tftypes.NewValue(tftypes.Bool, nil) }
func boolUnknown() tftypes.Value   { return tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue) }
func boolOf(b bool) tftypes.Value  { return tftypes.NewValue(tftypes.Bool, b) }
func stringNull() tftypes.Value    { return tftypes.NewValue(tftypes.String, nil) }
func stringUnknown() tftypes.Value { return tftypes.NewValue(tftypes.String, tftypes.UnknownValue) }
func stringOf(s string) tftypes.Value {
	return tftypes.NewValue(tftypes.String, s)
}
func numberNull() tftypes.Value { return tftypes.NewValue(tftypes.Number, nil) }
func numberOf(i int64) tftypes.Value {
	return tftypes.NewValue(tftypes.Number, i)
}

func cfg(auto, location, landingPage, categoryID tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: minimalSchema(),
		Raw: tftypes.NewValue(rootObjType, map[string]tftypes.Value{
			"install_automatically":    auto,
			"install_location":         location,
			"default_landing_page":     landingPage,
			"default_home_category_id": categoryID,
		}),
	}
}

// runValidators runs both ConfigValidators and returns the error paths.
func runValidators(c tfsdk.Config) map[string]bool {
	var resp resource.ValidateConfigResponse
	req := resource.ValidateConfigRequest{Config: c}
	installLocationRequiredValidator{}.ValidateResource(context.Background(), req, &resp)
	categoryRequiresBrowseValidator{}.ValidateResource(context.Background(), req, &resp)
	paths := make(map[string]bool)
	for _, d := range resp.Diagnostics {
		if dwp, ok := d.(diag.DiagnosticWithPath); ok && d.Severity() == diag.SeverityError {
			paths[dwp.Path().String()] = true
		}
	}
	return paths
}

func TestValidator_AutoTrueWithLocationValid(t *testing.T) {
	c := cfg(boolOf(true), stringOf("/Applications"), stringNull(), numberNull())
	if paths := runValidators(c); len(paths) != 0 {
		t.Errorf("auto=true with non-empty location must pass; got errors at %v", paths)
	}
}

func TestValidator_AutoTrueBlankLocationErrors(t *testing.T) {
	c := cfg(boolOf(true), stringOf(""), stringNull(), numberNull())
	if !runValidators(c)["install_location"] {
		t.Error("auto=true with blank location must error on install_location")
	}
}

func TestValidator_AutoFalseBlankLocationValid(t *testing.T) {
	c := cfg(boolOf(false), stringOf(""), stringNull(), numberNull())
	if paths := runValidators(c); len(paths) != 0 {
		t.Errorf("auto=false with blank location must pass (wire-probed accepted); got errors at %v", paths)
	}
}

// defer-on-null: blank location with auto omitted (null = preserve current server value).
func TestValidator_BlankLocationAutoNull_Defers(t *testing.T) {
	c := cfg(boolNull(), stringOf(""), stringNull(), numberNull())
	if runValidators(c)["install_location"] {
		t.Error("blank location with auto null (preserve) must defer, not error")
	}
}

// defer-on-unknown: auto sourced from a variable / another resource is unknown.
func TestValidator_UnknownAuto_Defers(t *testing.T) {
	c := cfg(boolUnknown(), stringOf(""), stringNull(), numberNull())
	if paths := runValidators(c); len(paths) != 0 {
		t.Errorf("unknown auto must defer (no errors); got %v", paths)
	}
}

func TestValidator_CategoryWithBrowseValid(t *testing.T) {
	c := cfg(boolNull(), stringNull(), stringOf("BROWSE"), numberOf(42))
	if paths := runValidators(c); len(paths) != 0 {
		t.Errorf("category id under BROWSE must pass; got errors at %v", paths)
	}
}

func TestValidator_AllItemsUnderAnyPageValid(t *testing.T) {
	c := cfg(boolNull(), stringNull(), stringOf("HOME"), numberOf(-1))
	if paths := runValidators(c); len(paths) != 0 {
		t.Errorf("category id -1 (All Items) under HOME must pass; got errors at %v", paths)
	}
}

func TestValidator_CategoryWithoutBrowseErrors(t *testing.T) {
	c := cfg(boolNull(), stringNull(), stringOf("HOME"), numberOf(42))
	if !runValidators(c)["default_home_category_id"] {
		t.Error("category id != -1 under HOME must error on default_home_category_id")
	}
}

// Negative sentinels other than -1 are still category selections — they require BROWSE too.
func TestValidator_NegativeSentinelWithoutBrowseErrors(t *testing.T) {
	c := cfg(boolNull(), stringNull(), stringOf("HISTORY"), numberOf(-2))
	if !runValidators(c)["default_home_category_id"] {
		t.Error("category id -2 under HISTORY must error on default_home_category_id")
	}
}

// defer-on-null: category id declared, landing page omitted (server may already be BROWSE).
func TestValidator_CategoryLandingPageNull_Defers(t *testing.T) {
	c := cfg(boolNull(), stringNull(), stringNull(), numberOf(42))
	if runValidators(c)["default_home_category_id"] {
		t.Error("category id with landing page null (preserve) must defer, not error")
	}
}

func TestValidator_CategoryLandingPageUnknown_Defers(t *testing.T) {
	c := cfg(boolNull(), stringNull(), stringUnknown(), numberOf(42))
	if paths := runValidators(c); len(paths) != 0 {
		t.Errorf("unknown landing page must defer (no errors); got %v", paths)
	}
}

// Both validators violate at once: each must report independently (local-diags guard — the
// first validator's error must not suppress the second).
func TestValidator_BothViolateBothReport(t *testing.T) {
	c := cfg(boolOf(true), stringOf(""), stringOf("HOME"), numberOf(42))
	paths := runValidators(c)
	if !paths["install_location"] || !paths["default_home_category_id"] {
		t.Errorf("both validators must report independently; got %v", paths)
	}
}

// --- samlRequiresLoginValidator (reads login_method + authentication_type only) ---

var loginObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"login_method":        tftypes.String,
	"authentication_type": tftypes.String,
}}

func loginCfg(loginMethod, authType tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"login_method":        schema.StringAttribute{Optional: true, Computed: true},
			"authentication_type": schema.StringAttribute{Optional: true, Computed: true},
		}},
		Raw: tftypes.NewValue(loginObjType, map[string]tftypes.Value{
			"login_method":        loginMethod,
			"authentication_type": authType,
		}),
	}
}

func runSamlValidator(c tfsdk.Config) bool {
	var resp resource.ValidateConfigResponse
	samlRequiresLoginValidator{}.ValidateResource(
		context.Background(),
		resource.ValidateConfigRequest{Config: c},
		&resp,
	)
	return resp.Diagnostics.HasError()
}

func TestValidator_SamlWithNotRequiredErrors(t *testing.T) {
	if !runSamlValidator(loginCfg(stringOf("NotRequired"), stringOf("Saml"))) {
		t.Error("Saml with NotRequired (both declared) must error")
	}
}

func TestValidator_SamlWithRequiredValid(t *testing.T) {
	if runSamlValidator(loginCfg(stringOf("Required"), stringOf("Saml"))) {
		t.Error("Saml with Required must pass")
	}
}

func TestValidator_BasicWithNotRequiredValid(t *testing.T) {
	if runSamlValidator(loginCfg(stringOf("NotRequired"), stringOf("Basic"))) {
		t.Error("Basic with NotRequired must pass")
	}
}

// defer-on-null: Saml declared, login_method omitted (server value may be login-enabled).
func TestValidator_SamlLoginMethodNull_Defers(t *testing.T) {
	if runSamlValidator(loginCfg(stringNull(), stringOf("Saml"))) {
		t.Error("Saml with login_method null (preserve) must defer, not error")
	}
}

func TestValidator_SamlLoginMethodUnknown_Defers(t *testing.T) {
	if runSamlValidator(loginCfg(stringUnknown(), stringOf("Saml"))) {
		t.Error("Saml with login_method unknown must defer, not error")
	}
}

// --- predictAuthTypeUnknown (ModifyPlan decision) ---

func TestPredictAuthTypeUnknown_TransitionWithCarriedSaml(t *testing.T) {
	if !predictAuthTypeUnknown(types.StringValue("NotRequired"), types.StringValue("Saml"), types.StringNull()) {
		t.Error("NotRequired + carried Saml (config null) must predict Unknown")
	}
}

func TestPredictAuthTypeUnknown_DeclaredSamlNotPredicted(t *testing.T) {
	// User-declared Saml is the ConfigValidator's job; ModifyPlan must not mask it.
	if predictAuthTypeUnknown(types.StringValue("NotRequired"), types.StringValue("Saml"), types.StringValue("Saml")) {
		t.Error("declared Saml must not be re-marked Unknown")
	}
}

func TestPredictAuthTypeUnknown_BasicNotPredicted(t *testing.T) {
	if predictAuthTypeUnknown(types.StringValue("NotRequired"), types.StringValue("Basic"), types.StringNull()) {
		t.Error("carried Basic survives NotRequired — no prediction")
	}
}

func TestPredictAuthTypeUnknown_LoginEnabledNotPredicted(t *testing.T) {
	if predictAuthTypeUnknown(types.StringValue("Required"), types.StringValue("Saml"), types.StringNull()) {
		t.Error("Saml under Required is valid — no prediction")
	}
}

func TestPredictAuthTypeUnknown_UnknownsNotPredicted(t *testing.T) {
	if predictAuthTypeUnknown(types.StringUnknown(), types.StringValue("Saml"), types.StringNull()) {
		t.Error("unknown login_method must not predict")
	}
	if predictAuthTypeUnknown(types.StringValue("NotRequired"), types.StringUnknown(), types.StringNull()) {
		t.Error("already-unknown authentication_type must not predict")
	}
}
