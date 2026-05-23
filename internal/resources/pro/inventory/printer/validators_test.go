// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Constructing a tfsdk.Config to drive the resource.ConfigValidator at unit
// scope is brittle (requires hand-rolling a tftypes.Object that matches the
// generated schema). The value-based logic is exercised end-to-end by the
// acceptance tests; the unit suite here covers the API contract (Description
// + MarkdownDescription non-empty) and the noLiteralSentinelValidator, which
// runs against a single types.String and is straightforward to drive.
// See ibeacon/validators_test.go for the same precedent.
func TestUseGenericPPDConfigValidator_Descriptions(t *testing.T) {
	v := useGenericPPDConfigValidator{}
	if v.Description(context.Background()) == "" {
		t.Errorf("Description must not be empty")
	}
	if v.MarkdownDescription(context.Background()) == "" {
		t.Errorf("MarkdownDescription must not be empty")
	}
}

func TestNoLiteralSentinelValidator_Descriptions(t *testing.T) {
	v := noLiteralSentinelValidator{}
	if v.Description(context.Background()) == "" {
		t.Errorf("Description must not be empty")
	}
	if v.MarkdownDescription(context.Background()) == "" {
		t.Errorf("MarkdownDescription must not be empty")
	}
}

func TestNoLiteralSentinelValidator_RejectsSentinel(t *testing.T) {
	v := noLiteralSentinelValidator{}
	req := validator.StringRequest{
		Path:        path.Root("category"),
		ConfigValue: types.StringValue(categoryUnassignedSentinel),
	}
	var resp validator.StringResponse
	v.ValidateString(context.Background(), req, &resp)
	if !resp.Diagnostics.HasError() {
		t.Errorf("expected error on literal sentinel, got none")
	}
}

func TestNoLiteralSentinelValidator_AllowsRealCategory(t *testing.T) {
	v := noLiteralSentinelValidator{}
	req := validator.StringRequest{
		Path:        path.Root("category"),
		ConfigValue: types.StringValue("Printers"),
	}
	var resp validator.StringResponse
	v.ValidateString(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error on real category, got %v", resp.Diagnostics)
	}
}

func TestNoLiteralSentinelValidator_AllowsNullAndUnknown(t *testing.T) {
	v := noLiteralSentinelValidator{}
	for _, cv := range []types.String{types.StringNull(), types.StringUnknown()} {
		req := validator.StringRequest{Path: path.Root("category"), ConfigValue: cv}
		var resp validator.StringResponse
		v.ValidateString(context.Background(), req, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("expected no error on null/unknown, got %v", resp.Diagnostics)
		}
	}
}

func TestIsSet(t *testing.T) {
	cases := []struct {
		name string
		in   types.String
		want bool
	}{
		{"value", types.StringValue("x"), true},
		{"empty", types.StringValue(""), false},
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isSet(c.in); got != c.want {
				t.Errorf("isSet(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
