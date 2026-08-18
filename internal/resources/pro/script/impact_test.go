// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package script

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestScriptIdentifyDependency(t *testing.T) {
	r := &ScriptResource{}
	// The alert names the object the operator recognises, so the pair must be the
	// script's id and the name shown in Jamf Pro.
	id, name := r.identifyDependency(context.Background(), &ScriptResourceModel{
		ID:   types.StringValue("42"),
		Name: types.StringValue("Reset Dock"),
	})
	if id != "42" {
		t.Fatalf("id = %q, want %q", id, "42")
	}
	if name != "Reset Dock" {
		t.Fatalf("name = %q, want %q", name, "Reset Dock")
	}
}

func TestScriptIdentifyDependencyNilModel(t *testing.T) {
	r := &ScriptResource{}
	// A destroy plan has no target model at all; the adapter must return nothing
	// rather than panic.
	id, name := r.identifyDependency(context.Background(), nil)
	if id != "" || name != "" {
		t.Fatalf("a nil model yields no identity, got id %q name %q", id, name)
	}
}
