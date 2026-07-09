// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobileconfig

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
)

func TestFunction_Metadata(t *testing.T) {
	resp := &function.MetadataResponse{}
	NewFunction().Metadata(context.Background(), function.MetadataRequest{}, resp)
	if resp.Name != "mobileconfig" {
		t.Fatalf("Name: got %q, want %q", resp.Name, "mobileconfig")
	}
}

func TestFunction_Definition(t *testing.T) {
	resp := &function.DefinitionResponse{}
	NewFunction().Definition(context.Background(), function.DefinitionRequest{}, resp)
	if got := len(resp.Definition.Parameters); got != 1 {
		t.Fatalf("Parameters: got %d, want 1", got)
	}
	if _, ok := resp.Definition.Parameters[0].(function.DynamicParameter); !ok {
		t.Fatalf("param 0: got %T, want function.DynamicParameter", resp.Definition.Parameters[0])
	}
	if _, ok := resp.Definition.Return.(function.StringReturn); !ok {
		t.Fatalf("return: got %T, want function.StringReturn", resp.Definition.Return)
	}
}

// runMobileconfig drives the function through its real framework seam: it builds
// a types.Dynamic argument the way Terraform would, calls Run, and returns the
// rendered string plus any function error. This exercises req.Arguments.Get, the
// TerraformDynamicToJSON decode, and the type-assert guards that the core unit
// tests (which feed Go maps directly) bypass.
//
// The argument is built with helpers.JSONToTerraformDynamic (the provider's own
// JSON→Dynamic helper) rather than a real HCL decode — a faithful stand-in that
// emits genuine ObjectValue/TupleValue/NumberValue. The end-to-end HCL decode
// path is covered by the acceptance test (function_acceptance_test.go), which
// invokes the function from real Terraform config.
func runMobileconfig(t *testing.T, arg any) (string, *function.FuncError) {
	t.Helper()
	dyn, err := helpers.JSONToTerraformDynamic(arg)
	if err != nil {
		t.Fatalf("build types.Dynamic argument: %v", err)
	}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringUnknown())}
	NewFunction().Run(context.Background(), function.RunRequest{
		Arguments: function.NewArgumentsData([]attr.Value{dyn}),
	}, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	out, ok := resp.Result.Value().(types.String)
	if !ok {
		t.Fatalf("result is not a types.String: %T", resp.Result.Value())
	}
	return out.ValueString(), nil
}

func TestFunction_Run_RendersProfile(t *testing.T) {
	out, ferr := runMobileconfig(t, map[string]any{
		"identifier": "com.example.dock",
		"payloads": []any{
			map[string]any{"PayloadType": "com.apple.dock", "tilesize": float64(48)},
		},
	})
	if ferr != nil {
		t.Fatalf("unexpected function error: %v", ferr)
	}
	if !strings.Contains(out, "<key>PayloadType</key>") || !strings.Contains(out, "com.apple.dock") {
		t.Fatalf("rendered output missing expected plist content:\n%s", out)
	}
	// Whole number must render as <integer>, proving the decode→normalize path.
	if !strings.Contains(out, "<integer>48</integer>") {
		t.Fatalf("tilesize did not render as <integer>48</integer>:\n%s", out)
	}
}

// TestFunction_Run_HeterogeneousMultiPayloadProfile covers the complex
// real-world case the generic function exists for: one profile mixing three
// payload types of entirely different shapes (a rules array, an array of
// notification dicts, and a hand-built MCX Forced envelope with 4-deep
// nesting). Because the three payload objects have different key sets, the
// payloads list decodes as a cty tuple, not a uniform list — exactly the case
// that forces DynamicParameter over a typed parameter, so it belongs at the
// Run seam rather than only in core tests.
func TestFunction_Run_HeterogeneousMultiPayloadProfile(t *testing.T) {
	out, ferr := runMobileconfig(t, map[string]any{
		"display_name":       "Privileges",
		"identifier":         "com.example.privileges",
		"scope":              "System",
		"removal_disallowed": true,
		"payloads": []any{
			map[string]any{
				"PayloadType": "com.apple.servicemanagement",
				"Rules": []any{
					map[string]any{
						"RuleType":  "TeamIdentifier",
						"RuleValue": "7R5ZEU67FQ",
					},
				},
			},
			map[string]any{
				"PayloadType": "com.apple.notificationsettings",
				"NotificationSettings": []any{
					map[string]any{
						"AlertType":            float64(1),
						"BundleIdentifier":     "corp.sap.privileges",
						"NotificationsEnabled": true,
					},
				},
			},
			map[string]any{
				"PayloadType": "com.apple.ManagedClient.preferences",
				"PayloadContent": map[string]any{
					"corp.sap.privileges": map[string]any{
						"Forced": []any{
							map[string]any{
								"mcx_preference_settings": map[string]any{
									"DockToggleTimeout":     float64(10),
									"RequireAuthentication": true,
								},
							},
						},
					},
				},
			},
		},
	})
	if ferr != nil {
		t.Fatalf("unexpected function error: %v", ferr)
	}

	parsed, _, err := plisthelpers.ParsePlist([]byte(out))
	if err != nil {
		t.Fatalf("output is not valid plist: %v", err)
	}
	payloads, ok := parsed["PayloadContent"].([]any)
	if !ok || len(payloads) != 3 {
		t.Fatalf("expected 3 payloads, got %T len %d", parsed["PayloadContent"], len(payloads))
	}

	// Each payload keeps its own shape and PayloadType.
	for i, want := range []string{
		"com.apple.servicemanagement",
		"com.apple.notificationsettings",
		"com.apple.ManagedClient.preferences",
	} {
		p := payloads[i].(map[string]any)
		if p["PayloadType"] != want {
			t.Fatalf("payload %d: PayloadType %v, want %s", i, p["PayloadType"], want)
		}
	}

	// The 4-deep MCX Forced envelope survives the Dynamic decode intact.
	mcx := payloads[2].(map[string]any)
	domain, ok := mcx["PayloadContent"].(map[string]any)["corp.sap.privileges"].(map[string]any)
	if !ok {
		t.Fatalf("MCX PayloadContent missing domain dict: %#v", mcx["PayloadContent"])
	}
	settings, ok := domain["Forced"].([]any)[0].(map[string]any)["mcx_preference_settings"].(map[string]any)
	if !ok {
		t.Fatalf("MCX Forced envelope malformed: %#v", domain["Forced"])
	}
	// plist round-trip note: the value was rendered as <integer>10</integer>
	// (ParsePlist decodes plist integers as uint64).
	if settings["DockToggleTimeout"] != uint64(10) {
		t.Fatalf("DockToggleTimeout: got %#v, want uint64(10) (whole number must stay <integer> at depth)", settings["DockToggleTimeout"])
	}
	if settings["RequireAuthentication"] != true {
		t.Fatalf("RequireAuthentication: got %#v, want true", settings["RequireAuthentication"])
	}

	// The rules array inside payload 0 also survives.
	rules := payloads[0].(map[string]any)["Rules"].([]any)
	if rules[0].(map[string]any)["RuleValue"] != "7R5ZEU67FQ" {
		t.Fatalf("Rules[0].RuleValue: got %#v", rules[0])
	}
}

func TestFunction_Run_NonObjectArgumentErrors(t *testing.T) {
	// A string where the function expects an object must surface a function
	// ARGUMENT error attributed to argument 0 — not a panic, and not a generic
	// function error. Asserting the argument index ties this to the seam guard
	// (function.go NewArgumentFuncError(0, …)) so a downgrade to a plain
	// NewFuncError, or the guard being removed, is caught here rather than
	// masked by a deeper Assemble/profileFromObject error.
	_, ferr := runMobileconfig(t, "not-an-object")
	if ferr == nil {
		t.Fatal("expected a function error for a non-object argument, got nil")
	}
	if ferr.FunctionArgument == nil || *ferr.FunctionArgument != 0 {
		t.Fatalf("expected an argument error at index 0, got FunctionArgument=%v (%q)", ferr.FunctionArgument, ferr.Text)
	}
}

func TestFunction_Run_MissingPayloadsErrors(t *testing.T) {
	// A well-formed object with no payloads is a decoded-input error, also
	// attributed to argument 0 (profileFromObject via NewArgumentFuncError(0)).
	_, ferr := runMobileconfig(t, map[string]any{"identifier": "com.example.x"})
	if ferr == nil {
		t.Fatal("expected a function error when payloads is absent, got nil")
	}
	if ferr.FunctionArgument == nil || *ferr.FunctionArgument != 0 {
		t.Fatalf("expected an argument error at index 0, got FunctionArgument=%v (%q)", ferr.FunctionArgument, ferr.Text)
	}
}

func TestProfileFromObject_ParsesMetadataAndPayloads(t *testing.T) {
	p, err := profileFromObject(map[string]any{
		"display_name": "Example",
		"identifier":   "com.example.dock",
		"payloads":     []any{map[string]any{"PayloadType": "com.apple.dock"}},
	})
	if err != nil {
		t.Fatalf("profileFromObject: %v", err)
	}
	if p.DisplayName != "Example" || p.Identifier != "com.example.dock" {
		t.Fatalf("metadata not parsed: got display_name=%q identifier=%q", p.DisplayName, p.Identifier)
	}
	if len(p.Payloads) != 1 {
		t.Fatalf("payloads: got %d, want 1", len(p.Payloads))
	}
}

func TestProfileFromObject_RequiresPayloads(t *testing.T) {
	if _, err := profileFromObject(map[string]any{"display_name": "x"}); err == nil {
		t.Fatal("expected error when payloads is missing")
	}
}
