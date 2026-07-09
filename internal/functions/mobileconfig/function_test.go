// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobileconfig

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/function"
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
