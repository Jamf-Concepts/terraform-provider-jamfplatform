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
