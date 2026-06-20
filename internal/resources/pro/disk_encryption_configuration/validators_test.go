// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// irkObjType is the tftypes shape of the institutional_recovery_key block.
var irkObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"key":                 tftypes.String,
	"certificate_type":    tftypes.String,
	"password":            tftypes.String,
	"password_wo_version": tftypes.Number,
	"data":                tftypes.String,
}}

// irkValidatorConfig builds a Config carrying only key_type and the
// institutional_recovery_key block — the two attributes the ConfigValidator
// reads via GetAttribute. keyType nil → null; irk is null, unknown, or present
// per the args.
func irkValidatorConfig(keyType *string, irkUnknown, irkPresent bool) tfsdk.Config {
	keyTypeVal := tftypes.NewValue(tftypes.String, nil)
	if keyType != nil {
		keyTypeVal = tftypes.NewValue(tftypes.String, *keyType)
	}

	var irkVal tftypes.Value
	switch {
	case irkUnknown:
		irkVal = tftypes.NewValue(irkObjType, tftypes.UnknownValue)
	case irkPresent:
		irkVal = tftypes.NewValue(irkObjType, map[string]tftypes.Value{
			"key":                 tftypes.NewValue(tftypes.String, nil),
			"certificate_type":    tftypes.NewValue(tftypes.String, "PKCS12"),
			"password":            tftypes.NewValue(tftypes.String, "pw"),
			"password_wo_version": tftypes.NewValue(tftypes.Number, 1),
			"data":                tftypes.NewValue(tftypes.String, "aGVsbG8="),
		})
	default:
		irkVal = tftypes.NewValue(irkObjType, nil)
	}

	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"key_type":                   tftypes.String,
		"institutional_recovery_key": irkObjType,
	}}

	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"key_type": schema.StringAttribute{Optional: true},
			"institutional_recovery_key": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"key":                 schema.StringAttribute{Computed: true},
					"certificate_type":    schema.StringAttribute{Required: true},
					"password":            schema.StringAttribute{Optional: true},
					"password_wo_version": schema.Int64Attribute{Optional: true},
					"data":                schema.StringAttribute{Required: true},
				},
			},
		}},
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"key_type":                   keyTypeVal,
			"institutional_recovery_key": irkVal,
		}),
	}
}

func runIRKValidator(cfg tfsdk.Config) []string {
	var resp resource.ValidateConfigResponse
	institutionalKeyTypeRequiresIRKConfigValidator{}.ValidateResource(
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

// TestIRK_DefersWhenBlockUnknown is the defer-on-unknown regression guard: with
// key_type = "Institutional" (known) and the institutional_recovery_key block
// UNKNOWN (e.g. variable-driven), the validator must DEFER, not treat unknown
// as a missing block and error. See STYLE_GUIDE "Config-time validators must
// defer on unknown values".
func TestIRK_DefersWhenBlockUnknown(t *testing.T) {
	kt := keyTypeInstitutional
	if out := runIRKValidator(irkValidatorConfig(&kt, true, false)); len(out) != 0 {
		t.Errorf("validator must defer when IRK block is unknown, got %v", out)
	}
}

// TestIRK_ErrorsWhenRequiredAndAbsent proves the invariant still fires when the
// block is genuinely absent (null) for a key_type that requires it.
func TestIRK_ErrorsWhenRequiredAndAbsent(t *testing.T) {
	kt := keyTypeInstitutional
	if out := runIRKValidator(irkValidatorConfig(&kt, false, false)); len(out) == 0 {
		t.Error("expected error when key_type=Institutional and no IRK block")
	}
}

// TestIRK_SilentWhenPresent proves no error when the block is supplied.
func TestIRK_SilentWhenPresent(t *testing.T) {
	kt := keyTypeIndividualInstitutional
	if out := runIRKValidator(irkValidatorConfig(&kt, false, true)); len(out) != 0 {
		t.Errorf("validator should not fire when IRK block present, got %v", out)
	}
}

// TestIRK_SilentWhenKeyTypeNotRequiringIRK proves no error for a key_type that
// does not need the block, and defers when key_type itself is null/unset.
func TestIRK_SilentWhenKeyTypeNotRequiringIRK(t *testing.T) {
	individual := "Individual"
	if out := runIRKValidator(irkValidatorConfig(&individual, false, false)); len(out) != 0 {
		t.Errorf("validator should not fire for key_type=Individual, got %v", out)
	}
	if out := runIRKValidator(irkValidatorConfig(nil, false, false)); len(out) != 0 {
		t.Errorf("validator should defer when key_type is unset, got %v", out)
	}
}
