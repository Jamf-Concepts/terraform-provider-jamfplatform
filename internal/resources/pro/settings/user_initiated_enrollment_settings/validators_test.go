// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_initiated_enrollment_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// certObjType is the tftypes shape of the mdm_signing_certificate object used
// in the validator config fixtures.
var certObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"keystore_file":                tftypes.String,
	"keystore_file_name":           tftypes.String,
	"keystore_password":            tftypes.String,
	"keystore_password_wo_version": tftypes.Number,
	"subject":                      tftypes.String,
	"serial_number":                tftypes.String,
}}

// buildValidatorConfig synthesises a tfsdk.Config carrying
// signing_mdm_profile_enabled and an (optionally present) mdm_signing_certificate
// block, matching the attributes the ConfigValidator reads.
func buildValidatorConfig(t *testing.T, enabled *bool, certPresent bool) tfsdk.Config {
	t.Helper()

	certSchema := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"keystore_file":                schema.StringAttribute{Optional: true},
			"keystore_file_name":           schema.StringAttribute{Optional: true, Computed: true},
			"keystore_password":            schema.StringAttribute{Optional: true},
			"keystore_password_wo_version": schema.Int64Attribute{Optional: true},
			"subject":                      schema.StringAttribute{Computed: true},
			"serial_number":                schema.StringAttribute{Computed: true},
		},
	}

	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"signing_mdm_profile_enabled": tftypes.Bool,
		"mdm_signing_certificate":     certObjType,
	}}

	var enabledVal tftypes.Value
	if enabled == nil {
		enabledVal = tftypes.NewValue(tftypes.Bool, nil)
	} else {
		enabledVal = tftypes.NewValue(tftypes.Bool, *enabled)
	}

	certVal := tftypes.NewValue(certObjType, nil)
	if certPresent {
		certVal = tftypes.NewValue(certObjType, map[string]tftypes.Value{
			"keystore_file":                tftypes.NewValue(tftypes.String, "aGVsbG8="),
			"keystore_file_name":           tftypes.NewValue(tftypes.String, "k.p12"),
			"keystore_password":            tftypes.NewValue(tftypes.String, "pw"),
			"keystore_password_wo_version": tftypes.NewValue(tftypes.Number, 1),
			"subject":                      tftypes.NewValue(tftypes.String, nil),
			"serial_number":                tftypes.NewValue(tftypes.String, nil),
		})
	}

	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"signing_mdm_profile_enabled": schema.BoolAttribute{Optional: true},
			"mdm_signing_certificate":     certSchema,
		}},
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"signing_mdm_profile_enabled": enabledVal,
			"mdm_signing_certificate":     certVal,
		}),
	}
}

func runCertValidator(t *testing.T, cfg tfsdk.Config) []string {
	t.Helper()
	var resp resource.ValidateConfigResponse
	mdmSigningCertificateRequiredValidator{}.ValidateResource(
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

// TestCertInvariant_FiresWhenEnabledNoCert proves the invariant trips when
// signing_mdm_profile_enabled = true and no cert block is present.
func TestCertInvariant_FiresWhenEnabledNoCert(t *testing.T) {
	enabled := true
	out := runCertValidator(t, buildValidatorConfig(t, &enabled, false))
	if len(out) == 0 {
		t.Error("expected validator to fire when enabled=true and no mdm_signing_certificate block")
	}
}

// TestCertInvariant_SilentWhenEnabledWithCert proves no error when the block is
// present.
func TestCertInvariant_SilentWhenEnabledWithCert(t *testing.T) {
	enabled := true
	out := runCertValidator(t, buildValidatorConfig(t, &enabled, true))
	if len(out) != 0 {
		t.Errorf("validator should not fire when cert block present, got %v", out)
	}
}

// TestCertInvariant_SilentWhenDisabled proves no error when the toggle is off.
func TestCertInvariant_SilentWhenDisabled(t *testing.T) {
	enabled := false
	out := runCertValidator(t, buildValidatorConfig(t, &enabled, false))
	if len(out) != 0 {
		t.Errorf("validator should not fire when disabled, got %v", out)
	}
}

// TestCertInvariant_SilentWhenUnset proves no error when the toggle is unset.
func TestCertInvariant_SilentWhenUnset(t *testing.T) {
	out := runCertValidator(t, buildValidatorConfig(t, nil, false))
	if len(out) != 0 {
		t.Errorf("validator should not fire when toggle unset, got %v", out)
	}
}
