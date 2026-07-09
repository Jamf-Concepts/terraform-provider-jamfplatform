// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mcx_forced_payload

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
	if resp.Name != "mcx_forced_payload" {
		t.Fatalf("Name: got %q, want %q", resp.Name, "mcx_forced_payload")
	}
}

func TestFunction_Definition(t *testing.T) {
	resp := &function.DefinitionResponse{}
	NewFunction().Definition(context.Background(), function.DefinitionRequest{}, resp)

	if got := len(resp.Definition.Parameters); got != 2 {
		t.Fatalf("Parameters: got %d, want 2", got)
	}
	if _, ok := resp.Definition.Parameters[0].(function.StringParameter); !ok {
		t.Fatalf("param 0: got %T, want function.StringParameter", resp.Definition.Parameters[0])
	}
	if _, ok := resp.Definition.Parameters[1].(function.DynamicParameter); !ok {
		t.Fatalf("param 1: got %T, want function.DynamicParameter", resp.Definition.Parameters[1])
	}
	if _, ok := resp.Definition.Return.(function.StringReturn); !ok {
		t.Fatalf("return: got %T, want function.StringReturn", resp.Definition.Return)
	}
}

// runMCX drives the function through its real framework seam: it builds the
// string + types.Dynamic arguments the way Terraform would, calls Run, and
// returns the rendered string plus any function error. This exercises
// req.Arguments.Get, the TerraformDynamicToJSON decode, and the type-assert
// guard that the core unit tests (which call renderMCXForcedPayload directly)
// bypass.
//
// The prefs argument is built with helpers.JSONToTerraformDynamic (the
// provider's own JSON→Dynamic helper) rather than a real HCL decode — a faithful
// stand-in. The end-to-end HCL decode path is covered by the acceptance test
// (function_acceptance_test.go), which invokes the function from real Terraform.
func runMCX(t *testing.T, domain string, prefs any) (string, *function.FuncError) {
	t.Helper()
	dyn, err := helpers.JSONToTerraformDynamic(prefs)
	if err != nil {
		t.Fatalf("build types.Dynamic argument: %v", err)
	}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringUnknown())}
	NewFunction().Run(context.Background(), function.RunRequest{
		Arguments: function.NewArgumentsData([]attr.Value{types.StringValue(domain), dyn}),
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

func TestFunction_Run_RendersEnvelope(t *testing.T) {
	out, ferr := runMCX(t, "com.example.app", map[string]any{
		"AdminBase":         "https://admin.example.com",
		"RotateWithinHours": float64(24),
	})
	if ferr != nil {
		t.Fatalf("unexpected function error: %v", ferr)
	}
	if !strings.Contains(out, "com.apple.ManagedClient.preferences") || !strings.Contains(out, "com.example.app") {
		t.Fatalf("rendered output missing expected envelope content:\n%s", out)
	}
	if !strings.Contains(out, "<integer>24</integer>") {
		t.Fatalf("RotateWithinHours did not render as <integer>24</integer>:\n%s", out)
	}
}

func TestFunction_Run_NonObjectPreferencesErrors(t *testing.T) {
	// preferences given as a string (not an object) must surface a function
	// ARGUMENT error attributed to argument 1 (the preferences arg) — not a
	// panic and not a generic error. Asserting the index ties this to the seam
	// guard (function.go NewArgumentFuncError(1, …)); a valid domain here means
	// the only thing that can fail is the prefs decode, isolating the seam.
	_, ferr := runMCX(t, "com.example.app", "not-an-object")
	if ferr == nil {
		t.Fatal("expected a function error for non-object preferences, got nil")
	}
	if ferr.FunctionArgument == nil || *ferr.FunctionArgument != 1 {
		t.Fatalf("expected an argument error at index 1, got FunctionArgument=%v (%q)", ferr.FunctionArgument, ferr.Text)
	}
}

func TestFunction_Run_EmptyDomainErrors(t *testing.T) {
	// An empty domain is a render-level error (not argument-attributed): it must
	// surface as a generic function error with no FunctionArgument set.
	_, ferr := runMCX(t, "", map[string]any{"AdminBase": "x"})
	if ferr == nil {
		t.Fatal("expected a function error for an empty preference_domain, got nil")
	}
	if ferr.FunctionArgument != nil {
		t.Fatalf("expected a non-argument function error, got FunctionArgument=%d (%q)", *ferr.FunctionArgument, ferr.Text)
	}
}
