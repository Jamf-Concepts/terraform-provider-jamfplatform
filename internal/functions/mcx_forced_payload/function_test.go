// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mcx_forced_payload

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/function"
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
