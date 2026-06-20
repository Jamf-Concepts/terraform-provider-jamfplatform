// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_identity_provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// --- tftypes object shapes --------------------------------------------------
//
// The validator reads path.Root("provider_name"), path.Root("google"), and
// path.Root("entra_id") as top-level scalars/objects. The google/entra_id
// objects need only a single dummy inner attribute so the schema and Raw shapes
// match; the validator only checks IsNull/IsUnknown on the outer object.

var (
	googleInnerType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"domain_name": tftypes.String,
	}}
	entraIDInnerType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"tenant_id": tftypes.String,
	}}
	providerRootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"provider_name": tftypes.String,
		"google":        googleInnerType,
		"entra_id":      entraIDInnerType,
	}}
)

// minimalProviderSchema is the schema subset the validator reads from Config.
func minimalProviderSchema() schema.Schema {
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"provider_name": schema.StringAttribute{Optional: true},
		"google": schema.SingleNestedAttribute{
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"domain_name": schema.StringAttribute{Optional: true},
			},
		},
		"entra_id": schema.SingleNestedAttribute{
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"tenant_id": schema.StringAttribute{Optional: true},
			},
		},
	}}
}

// providerNameVal constructs a tftypes string value.
func providerNameVal(v string) tftypes.Value {
	if v == "" {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, v)
}

// providerNameUnknown returns an unknown string value.
func providerNameUnknown() tftypes.Value {
	return tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
}

// googleVal builds a non-null google object value.
func googleVal(present bool) tftypes.Value {
	if !present {
		return tftypes.NewValue(googleInnerType, nil)
	}
	return tftypes.NewValue(googleInnerType, map[string]tftypes.Value{
		"domain_name": tftypes.NewValue(tftypes.String, "example.com"),
	})
}

// googleUnknown returns an unknown google object (before config resolution).
func googleUnknown() tftypes.Value {
	return tftypes.NewValue(googleInnerType, tftypes.UnknownValue)
}

// entraIDVal builds a non-null entra_id object value.
func entraIDVal(present bool) tftypes.Value {
	if !present {
		return tftypes.NewValue(entraIDInnerType, nil)
	}
	return tftypes.NewValue(entraIDInnerType, map[string]tftypes.Value{
		"tenant_id": tftypes.NewValue(tftypes.String, "d5749c84-5cc5-4691-a187-4545c02ff915"),
	})
}

// entraIDUnknown returns an unknown entra_id object.
func entraIDUnknown() tftypes.Value {
	return tftypes.NewValue(entraIDInnerType, tftypes.UnknownValue)
}

// buildProviderConfig assembles a minimal tfsdk.Config for the validator.
func buildProviderConfig(provName, google, entraID tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: minimalProviderSchema(),
		Raw: tftypes.NewValue(providerRootObjType, map[string]tftypes.Value{
			"provider_name": provName,
			"google":        google,
			"entra_id":      entraID,
		}),
	}
}

// runProviderBlockValidator invokes the validator and returns diagnostic summaries
// plus a map of attribute paths that had errors.
func runProviderBlockValidator(cfg tfsdk.Config) ([]string, map[string]bool) {
	var resp resource.ValidateConfigResponse
	providerBlockConfigValidator{}.ValidateResource(
		context.Background(),
		resource.ValidateConfigRequest{Config: cfg},
		&resp,
	)
	summaries := make([]string, 0, len(resp.Diagnostics))
	paths := make(map[string]bool)
	for _, d := range resp.Diagnostics {
		summaries = append(summaries, d.Summary())
		if dwp, ok := d.(diag.DiagnosticWithPath); ok {
			paths[dwp.Path().String()] = true
		}
	}
	return summaries, paths
}

// TestProviderBlockValidator_GooglePresent_EntraIDAbsent verifies no error when
// provider_name=GOOGLE, google block present, entra_id block absent.
func TestProviderBlockValidator_GooglePresent_EntraIDAbsent(t *testing.T) {
	cfg := buildProviderConfig(
		providerNameVal(providerGoogle),
		googleVal(true),
		entraIDVal(false),
	)
	summaries, _ := runProviderBlockValidator(cfg)
	if len(summaries) != 0 {
		t.Errorf("GOOGLE + google-present + entra_id-absent must pass; got %v", summaries)
	}
}

// TestProviderBlockValidator_GoogleAbsent_Errors verifies an error is added to
// the google path when provider_name=GOOGLE but google block is absent.
func TestProviderBlockValidator_GoogleAbsent_Errors(t *testing.T) {
	cfg := buildProviderConfig(
		providerNameVal(providerGoogle),
		googleVal(false),
		entraIDVal(false),
	)
	_, paths := runProviderBlockValidator(cfg)
	if !paths["google"] {
		t.Errorf("expected error on 'google' path; got paths=%v", paths)
	}
}

// TestProviderBlockValidator_EntraIDPresentWithGoogle_Errors verifies an error
// is added to the entra_id path when provider_name=GOOGLE but entra_id block is
// also present.
func TestProviderBlockValidator_EntraIDPresentWithGoogle_Errors(t *testing.T) {
	cfg := buildProviderConfig(
		providerNameVal(providerGoogle),
		googleVal(true),
		entraIDVal(true),
	)
	_, paths := runProviderBlockValidator(cfg)
	if !paths["entra_id"] {
		t.Errorf("expected error on 'entra_id' path (entra_id forbidden); got paths=%v", paths)
	}
	// google block is present — no error on google.
	if paths["google"] {
		t.Errorf("unexpected error on 'google' path; got paths=%v", paths)
	}
}

// TestProviderBlockValidator_EntraIDPresent_GoogleAbsent verifies no error when
// provider_name=ENTRA_ID, entra_id block present, google block absent.
func TestProviderBlockValidator_EntraIDPresent_GoogleAbsent(t *testing.T) {
	cfg := buildProviderConfig(
		providerNameVal(providerEntraID),
		googleVal(false),
		entraIDVal(true),
	)
	summaries, _ := runProviderBlockValidator(cfg)
	if len(summaries) != 0 {
		t.Errorf("ENTRA_ID + entra_id-present + google-absent must pass; got %v", summaries)
	}
}

// TestProviderBlockValidator_EntraIDAbsent_Errors verifies an error is added to
// the entra_id path when provider_name=ENTRA_ID but entra_id block is absent.
func TestProviderBlockValidator_EntraIDAbsent_Errors(t *testing.T) {
	cfg := buildProviderConfig(
		providerNameVal(providerEntraID),
		googleVal(false),
		entraIDVal(false),
	)
	_, paths := runProviderBlockValidator(cfg)
	if !paths["entra_id"] {
		t.Errorf("expected error on 'entra_id' path; got paths=%v", paths)
	}
}

// TestProviderBlockValidator_GooglePresentWithEntraID_Errors verifies an error
// is added to the google path when provider_name=ENTRA_ID but google block is
// also present.
func TestProviderBlockValidator_GooglePresentWithEntraID_Errors(t *testing.T) {
	cfg := buildProviderConfig(
		providerNameVal(providerEntraID),
		googleVal(true),
		entraIDVal(true),
	)
	_, paths := runProviderBlockValidator(cfg)
	if !paths["google"] {
		t.Errorf("expected error on 'google' path (google forbidden); got paths=%v", paths)
	}
	// entra_id block is present — no error on entra_id for the require check.
	if paths["entra_id"] {
		t.Errorf("unexpected error on 'entra_id' path; got paths=%v", paths)
	}
}

// TestProviderBlockValidator_ProviderNameUnknown_Defers verifies the validator
// defers (no error) when provider_name is unknown (e.g. driven by a variable).
func TestProviderBlockValidator_ProviderNameUnknown_Defers(t *testing.T) {
	cfg := buildProviderConfig(
		providerNameUnknown(),
		googleVal(false),
		entraIDVal(false),
	)
	summaries, _ := runProviderBlockValidator(cfg)
	if len(summaries) != 0 {
		t.Errorf("unknown provider_name must defer; got %v", summaries)
	}
}

// TestProviderBlockValidator_MatchingBlockUnknown_Defers verifies the
// requireBlock check defers (no error) when the matching block is unknown rather
// than null. An unknown object means the value is not yet resolved and the block
// may still be present — the validator must not false-error.
func TestProviderBlockValidator_MatchingBlockUnknown_Defers(t *testing.T) {
	// GOOGLE, google block unknown, entra_id absent — must not error.
	cfg := buildProviderConfig(
		providerNameVal(providerGoogle),
		googleUnknown(),
		entraIDVal(false),
	)
	summaries, _ := runProviderBlockValidator(cfg)
	if len(summaries) != 0 {
		t.Errorf("GOOGLE + google-unknown + entra_id-absent must defer; got %v", summaries)
	}

	// ENTRA_ID, entra_id block unknown, google absent — must not error.
	cfg2 := buildProviderConfig(
		providerNameVal(providerEntraID),
		googleVal(false),
		entraIDUnknown(),
	)
	summaries2, _ := runProviderBlockValidator(cfg2)
	if len(summaries2) != 0 {
		t.Errorf("ENTRA_ID + entra_id-unknown + google-absent must defer; got %v", summaries2)
	}
}

// TestProviderBlockValidator_Descriptions verifies the validator descriptions are
// non-empty.
func TestProviderBlockValidator_Descriptions(t *testing.T) {
	v := providerBlockConfigValidator{}
	if v.Description(context.Background()) == "" {
		t.Error("Description must not be empty")
	}
	if v.MarkdownDescription(context.Background()) == "" {
		t.Error("MarkdownDescription must not be empty")
	}
}
