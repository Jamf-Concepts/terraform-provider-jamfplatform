// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_prestage_enrollment

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

// unknownBool is a sentinel for buildConfig signalling an UNKNOWN Bool value.
type unknownBool struct{}

// nullBool is a sentinel for buildConfig signalling a NULL Bool value (the
// shape an Optional companion takes when the user omits it from config).
type nullBool struct{}

// buildConfig synthesises a *tfsdk.Config from a small flat attribute map for
// use by the cross-field validators that call req.Config.GetAttribute(...).
// All three validators read their companion via an absolute (path.Root) or
// re-rooted (ParentPath().AtName) path that resolves at the top level, so a
// flat config is sufficient.
func buildConfig(t *testing.T, attrs map[string]any) tfsdk.Config {
	t.Helper()
	typeMap := map[string]tftypes.Type{}
	valMap := map[string]tftypes.Value{}
	tfsdkAttrs := map[string]schema.Attribute{}

	for k, raw := range attrs {
		switch v := raw.(type) {
		case string:
			typeMap[k] = tftypes.String
			valMap[k] = tftypes.NewValue(tftypes.String, v)
			tfsdkAttrs[k] = schema.StringAttribute{Optional: true}
		case bool:
			typeMap[k] = tftypes.Bool
			valMap[k] = tftypes.NewValue(tftypes.Bool, v)
			tfsdkAttrs[k] = schema.BoolAttribute{Optional: true}
		case unknownBool:
			typeMap[k] = tftypes.Bool
			valMap[k] = tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)
			tfsdkAttrs[k] = schema.BoolAttribute{Optional: true}
		case nullBool:
			typeMap[k] = tftypes.Bool
			valMap[k] = tftypes.NewValue(tftypes.Bool, nil)
			tfsdkAttrs[k] = schema.BoolAttribute{Optional: true}
		case nil:
			// null String (used for the absent-companion case).
			typeMap[k] = tftypes.String
			valMap[k] = tftypes.NewValue(tftypes.String, nil)
			tfsdkAttrs[k] = schema.StringAttribute{Optional: true}
		default:
			t.Fatalf("buildConfig: unsupported type for %q: %T", k, raw)
		}
	}

	objType := tftypes.Object{AttributeTypes: typeMap}
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: tfsdkAttrs},
		Raw:    tftypes.NewValue(objType, valMap),
	}
}

func runStringValidator(t *testing.T, v validator.String, cfg tfsdk.Config, attrName string, value types.String) []string {
	t.Helper()
	req := validator.StringRequest{
		Path:        path.Root(attrName),
		ConfigValue: value,
		Config:      cfg,
	}
	var resp validator.StringResponse
	v.ValidateString(context.Background(), req, &resp)
	out := make([]string, 0, len(resp.Diagnostics))
	for _, d := range resp.Diagnostics {
		out = append(out, d.Detail())
	}
	return out
}

func runBoolValidator(t *testing.T, v validator.Bool, cfg tfsdk.Config, attrName string, value types.Bool) []string {
	t.Helper()
	req := validator.BoolRequest{
		Path:        path.Root(attrName),
		ConfigValue: value,
		Config:      cfg,
	}
	var resp validator.BoolResponse
	v.ValidateBool(context.Background(), req, &resp)
	out := make([]string, 0, len(resp.Diagnostics))
	for _, d := range resp.Diagnostics {
		out = append(out, d.Detail())
	}
	return out
}

func runInt64Validator(t *testing.T, v validator.Int64, cfg tfsdk.Config, attrName string, value types.Int64) []string {
	t.Helper()
	req := validator.Int64Request{
		Path:        path.Root(attrName),
		ConfigValue: value,
		Config:      cfg,
	}
	var resp validator.Int64Response
	v.ValidateInt64(context.Background(), req, &resp)
	out := make([]string, 0, len(resp.Diagnostics))
	for _, d := range resp.Diagnostics {
		out = append(out, d.Detail())
	}
	return out
}

// --- singleNameRequiresSingleDeviceName ------------------------------------

func TestSingleNameRequiresSingleDeviceName_FiresWhenMissing(t *testing.T) {
	// ParentPath() of assign_names_using is empty, so AtName re-roots at the
	// top level — single_device_name lives at root in the flat config.
	cfg := buildConfig(t, map[string]any{
		"single_device_name": "", // empty
	})
	out := runStringValidator(t, singleNameRequiresSingleDeviceName(),
		cfg, "assign_names_using", types.StringValue("Single Name"))
	if len(out) == 0 {
		t.Errorf(`"Single Name" + empty single_device_name should fire validator`)
	}
}

func TestSingleNameRequiresSingleDeviceName_NoErrorWhenPresent(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"single_device_name": "iPad",
	})
	out := runStringValidator(t, singleNameRequiresSingleDeviceName(),
		cfg, "assign_names_using", types.StringValue("Single Name"))
	if len(out) != 0 {
		t.Errorf(`"Single Name" + populated single_device_name should not fire, got %v`, out)
	}
}

func TestSingleNameRequiresSingleDeviceName_NoErrorForOtherModes(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"single_device_name": "", // empty, but mode is not Single Name
	})
	out := runStringValidator(t, singleNameRequiresSingleDeviceName(),
		cfg, "assign_names_using", types.StringValue("Serial Numbers"))
	if len(out) != 0 {
		t.Errorf("non-Single-Name mode must not fire validator, got %v", out)
	}
}

// --- storageQuotaConflictsWithTemporarySession -----------------------------

func TestStorageQuotaConflictsWithTemporarySession_FiresWhenBothTrue(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"temporary_session_only": true,
	})
	out := runBoolValidator(t, storageQuotaConflictsWithTemporarySession(),
		cfg, "use_storage_quota_size", types.BoolValue(true))
	if len(out) == 0 {
		t.Errorf("both true should fire the mutual-exclusion validator")
	}
}

func TestStorageQuotaConflictsWithTemporarySession_NoErrorWhenTempFalse(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"temporary_session_only": false,
	})
	out := runBoolValidator(t, storageQuotaConflictsWithTemporarySession(),
		cfg, "use_storage_quota_size", types.BoolValue(true))
	if len(out) != 0 {
		t.Errorf("use_storage_quota_size=true + temp=false must not fire, got %v", out)
	}
}

func TestStorageQuotaConflictsWithTemporarySession_NoErrorWhenStorageFalse(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"temporary_session_only": true,
	})
	out := runBoolValidator(t, storageQuotaConflictsWithTemporarySession(),
		cfg, "use_storage_quota_size", types.BoolValue(false))
	if len(out) != 0 {
		t.Errorf("use_storage_quota_size=false must short-circuit, got %v", out)
	}
}

func TestStorageQuotaConflictsWithTemporarySession_DefersWhenTempUnknown(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"temporary_session_only": unknownBool{},
	})
	out := runBoolValidator(t, storageQuotaConflictsWithTemporarySession(),
		cfg, "use_storage_quota_size", types.BoolValue(true))
	if len(out) != 0 {
		t.Errorf("unknown companion must defer (no diagnostic), got %v", out)
	}
}

// --- temporarySessionTimeoutMinimum ----------------------------------------

func TestTemporarySessionTimeoutMinimum_FiresWhenBelowAndEnforced(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"enforce_temporary_session_timeout": true,
	})
	out := runInt64Validator(t, temporarySessionTimeoutMinimum(),
		cfg, "temporary_session_timeout", types.Int64Value(15))
	if len(out) == 0 {
		t.Errorf("value<30 + enforced should fire validator")
	}
}

func TestTemporarySessionTimeoutMinimum_NoErrorWhenNotEnforced(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"enforce_temporary_session_timeout": false,
	})
	out := runInt64Validator(t, temporarySessionTimeoutMinimum(),
		cfg, "temporary_session_timeout", types.Int64Value(15))
	if len(out) != 0 {
		t.Errorf("value<30 but not enforced must not fire, got %v", out)
	}
}

func TestTemporarySessionTimeoutMinimum_NoErrorWhenAtOrAboveMinimum(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"enforce_temporary_session_timeout": true,
	})
	out := runInt64Validator(t, temporarySessionTimeoutMinimum(),
		cfg, "temporary_session_timeout", types.Int64Value(30))
	if len(out) != 0 {
		t.Errorf("value>=30 must not fire even when enforced, got %v", out)
	}
}

func TestTemporarySessionTimeoutMinimum_DefersWhenEnforceUnknown(t *testing.T) {
	cfg := buildConfig(t, map[string]any{
		"enforce_temporary_session_timeout": unknownBool{},
	})
	out := runInt64Validator(t, temporarySessionTimeoutMinimum(),
		cfg, "temporary_session_timeout", types.Int64Value(15))
	if len(out) != 0 {
		t.Errorf("unknown enforce companion must defer, got %v", out)
	}
}

// runValidTimezone invokes validTimezone() against a single ConfigValue and
// returns the diagnostics. validTimezone reads only req.ConfigValue (no
// companion lookups), so no synthesised config is needed.
func runValidTimezone(v types.String) []string {
	resp := &validator.StringResponse{}
	validTimezone().ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("timezone"),
		ConfigValue: v,
	}, resp)
	out := []string{}
	for _, d := range resp.Diagnostics {
		out = append(out, d.Summary())
	}
	return out
}

func TestValidTimezone(t *testing.T) {
	tests := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"UTC accepted (absent from /v1/time-zones but create-valid)", types.StringValue("UTC"), false},
		{"region id", types.StringValue("America/Chicago"), false},
		{"Etc/UTC", types.StringValue("Etc/UTC"), false},
		{"GMT", types.StringValue("GMT"), false},
		{"typo rejected", types.StringValue("America/Chicagoo"), true},
		{"garbage rejected", types.StringValue("garbage"), true},
		{"Go-only Local alias rejected", types.StringValue("Local"), true},
		{"empty deferred to LengthAtLeast", types.StringValue(""), false},
		{"null skipped", types.StringNull(), false},
		{"unknown skipped", types.StringUnknown(), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runValidTimezone(tc.value)
			if tc.wantErr && len(got) == 0 {
				t.Fatalf("expected a validation error, got none")
			}
			if !tc.wantErr && len(got) != 0 {
				t.Fatalf("expected no error, got %v", got)
			}
		})
	}
}

// --- multiUserRequiresPreventActivationLock --------------------------------

func TestMultiUserRequiresPreventActivationLock_SatisfiedWhenBothTrue(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"prevent_activation_lock": true})
	out := runBoolValidator(t, multiUserRequiresPreventActivationLock(),
		cfg, "multi_user", types.BoolValue(true))
	if len(out) != 0 {
		t.Errorf("multi_user=true + prevent_activation_lock=true must not fire, got %v", out)
	}
}

func TestMultiUserRequiresPreventActivationLock_FiresWhenCompanionFalse(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"prevent_activation_lock": false})
	out := runBoolValidator(t, multiUserRequiresPreventActivationLock(),
		cfg, "multi_user", types.BoolValue(true))
	if len(out) == 0 {
		t.Errorf("multi_user=true + prevent_activation_lock=false must fire")
	}
}

func TestMultiUserRequiresPreventActivationLock_FiresWhenCompanionNull(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"prevent_activation_lock": nullBool{}})
	out := runBoolValidator(t, multiUserRequiresPreventActivationLock(),
		cfg, "multi_user", types.BoolValue(true))
	if len(out) == 0 {
		t.Errorf("multi_user=true + prevent_activation_lock omitted (null) must fire")
	}
}

func TestMultiUserRequiresPreventActivationLock_DefersWhenCompanionUnknown(t *testing.T) {
	// STYLE_GUIDE §Config-time validators MUST defer on unknown: a companion
	// sourced from a variable/for_each is Unknown at validate/plan time.
	cfg := buildConfig(t, map[string]any{"prevent_activation_lock": unknownBool{}})
	out := runBoolValidator(t, multiUserRequiresPreventActivationLock(),
		cfg, "multi_user", types.BoolValue(true))
	if len(out) != 0 {
		t.Errorf("unknown companion must defer (no diagnostic), got %v", out)
	}
}

func TestMultiUserRequiresPreventActivationLock_ShortCircuitsWhenMultiUserFalse(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"prevent_activation_lock": false})
	out := runBoolValidator(t, multiUserRequiresPreventActivationLock(),
		cfg, "multi_user", types.BoolValue(false))
	if len(out) != 0 {
		t.Errorf("multi_user=false must short-circuit, got %v", out)
	}
}

func TestMultiUserRequiresPreventActivationLock_DefersWhenMultiUserUnknown(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"prevent_activation_lock": false})
	out := runBoolValidator(t, multiUserRequiresPreventActivationLock(),
		cfg, "multi_user", types.BoolUnknown())
	if len(out) != 0 {
		t.Errorf("multi_user unknown must defer, got %v", out)
	}
}

// --- multiUserRequiresSupervised -------------------------------------------

func TestMultiUserRequiresSupervised_SatisfiedWhenBothTrue(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"supervised": true})
	out := runBoolValidator(t, multiUserRequiresSupervised(),
		cfg, "multi_user", types.BoolValue(true))
	if len(out) != 0 {
		t.Errorf("multi_user=true + supervised=true must not fire, got %v", out)
	}
}

func TestMultiUserRequiresSupervised_FiresWhenCompanionFalse(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"supervised": false})
	out := runBoolValidator(t, multiUserRequiresSupervised(),
		cfg, "multi_user", types.BoolValue(true))
	if len(out) == 0 {
		t.Errorf("multi_user=true + supervised=false must fire")
	}
}

func TestMultiUserRequiresSupervised_FiresWhenCompanionNull(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"supervised": nullBool{}})
	out := runBoolValidator(t, multiUserRequiresSupervised(),
		cfg, "multi_user", types.BoolValue(true))
	if len(out) == 0 {
		t.Errorf("multi_user=true + supervised omitted (null) must fire")
	}
}

func TestMultiUserRequiresSupervised_DefersWhenCompanionUnknown(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"supervised": unknownBool{}})
	out := runBoolValidator(t, multiUserRequiresSupervised(),
		cfg, "multi_user", types.BoolValue(true))
	if len(out) != 0 {
		t.Errorf("unknown companion must defer (no diagnostic), got %v", out)
	}
}

func TestMultiUserRequiresSupervised_ShortCircuitsWhenMultiUserFalse(t *testing.T) {
	cfg := buildConfig(t, map[string]any{"supervised": false})
	out := runBoolValidator(t, multiUserRequiresSupervised(),
		cfg, "multi_user", types.BoolValue(false))
	if len(out) != 0 {
		t.Errorf("multi_user=false must short-circuit, got %v", out)
	}
}
