// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package managed_software_updates

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// --- buildVersionCustomOnlyValidator ---

// buildVersionConfigSchema is a minimal stand-in carrying just the two
// attributes the validator reads. It mirrors validatorConfigSchema in
// schema_test.go, which covers the specific_version rule.
var buildVersionConfigSchema = schema.Schema{Attributes: map[string]schema.Attribute{
	"version_type":  schema.StringAttribute{Optional: true},
	"build_version": schema.StringAttribute{Optional: true},
}}

var buildVersionObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"version_type":  tftypes.String,
	"build_version": tftypes.String,
}}

func runBuildVersionValidator(versionType, buildVersion tftypes.Value) *action.ValidateConfigResponse {
	cfg := tfsdk.Config{
		Schema: buildVersionConfigSchema,
		Raw: tftypes.NewValue(buildVersionObjType, map[string]tftypes.Value{
			"version_type":  versionType,
			"build_version": buildVersion,
		}),
	}
	var resp action.ValidateConfigResponse
	buildVersionCustomOnlyValidator{}.ValidateAction(
		context.Background(),
		action.ValidateConfigRequest{Config: cfg},
		&resp,
	)
	return &resp
}

// TestBuildVersionCustomOnlyValidator covers the rule wire-probed 2026-08-03:
// Jamf Pro returns 400 INVALID_BUILD_VERSION unless version_type is
// CUSTOM_VERSION — including for SPECIFIC_VERSION, which the field's own name
// suggests it pairs with.
//
// The unknown_* cases are the STYLE_GUIDE-mandated deferral guard: config
// validation runs with unknown values for anything sourced from a variable,
// for_each, count, or another resource, so erroring on unknown would make the
// action unusable from a module. Acceptance tests use literal HCL and cannot
// catch this.
func TestBuildVersionCustomOnlyValidator(t *testing.T) {
	cases := []struct {
		name         string
		versionType  tftypes.Value
		buildVersion tftypes.Value
		wantErr      bool
	}{
		{"custom_with_build", strVal("CUSTOM_VERSION"), strVal("21F79"), false},
		{"custom_without_build", strVal("CUSTOM_VERSION"), strNull(), false},
		{"specific_with_build", strVal("SPECIFIC_VERSION"), strVal("21F79"), true},
		{"latest_any_with_build", strVal("LATEST_ANY"), strVal("21F79"), true},
		{"latest_major_with_build", strVal("LATEST_MAJOR"), strVal("21F79"), true},
		{"latest_any_without_build", strVal("LATEST_ANY"), strNull(), false},
		{"unknown_build_version_defers", strVal("LATEST_ANY"), strUnknown(), false},
		{"unknown_version_type_defers", strUnknown(), strVal("21F79"), false},
		{"null_version_type_defers", strNull(), strVal("21F79"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := runBuildVersionValidator(c.versionType, c.buildVersion)
			if got := resp.Diagnostics.HasError(); got != c.wantErr {
				t.Errorf("hasError = %v, want %v (diags: %v)", got, c.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestBuildVersionCustomOnlyValidator_NamesTheOffendingValue checks the
// diagnostic tells the user which version_type tripped the rule, since
// "CUSTOM_VERSION only" is otherwise easy to misread as "SPECIFIC_VERSION too".
func TestBuildVersionCustomOnlyValidator_NamesTheOffendingValue(t *testing.T) {
	resp := runBuildVersionValidator(strVal("SPECIFIC_VERSION"), strVal("21F79"))
	var found bool
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Detail(), "SPECIFIC_VERSION") {
			found = true
		}
	}
	if !found {
		t.Errorf("no diagnostic named the offending version_type: %v", resp.Diagnostics)
	}
}

// --- force_install_local_date_time ---

// TestForceInstallLocalDateTimePattern pins the shape Jamf Pro parses. Probed
// 2026-08-03: an unparseable value is rejected with
// INVALID_FORCE_INSTALL_LOCAL_DATE_TIME ("Please provide a valid local date/time
// in YYYY-mm-DDTHH:MM:SS format") and that check runs before the target group is
// resolved. The pattern is deliberately shape-only — calendar validity is left
// to the server.
func TestForceInstallLocalDateTimePattern(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"2026-12-25T21:09:31", true},
		{"2026-01-01T00:00:00", true},
		{"tomorrow", false},
		{"2026-12-25", false},
		{"2026-12-25T21:09", false},
		{"2026-12-25T21:09:31Z", false},
		{"2026-12-25 21:09:31", false},
		{"2026-12-25T21:09:31.000", false},
		{"", false},
	}
	for _, c := range cases {
		if got := forceInstallLocalDateTimePattern.MatchString(c.in); got != c.valid {
			t.Errorf("MatchString(%q) = %v, want %v", c.in, got, c.valid)
		}
	}
}
