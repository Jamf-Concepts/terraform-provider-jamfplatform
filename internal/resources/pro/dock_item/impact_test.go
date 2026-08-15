// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dock_item

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDockItemIdentifyDependency(t *testing.T) {
	r := &DockItemResource{}
	// The alert names the object the operator recognises, so the pair must be the
	// dock item's id and the name shown in Jamf Pro.
	id, name := r.identifyDependency(context.Background(), &DockItemResourceModel{
		ID:   types.StringValue("42"),
		Name: types.StringValue("Self Service"),
	})
	if id != "42" {
		t.Fatalf("id = %q, want %q", id, "42")
	}
	if name != "Self Service" {
		t.Fatalf("name = %q, want %q", name, "Self Service")
	}
}

func TestDockItemIdentifyDependencyNilModel(t *testing.T) {
	r := &DockItemResource{}
	// A destroy plan has no target model at all; the adapter must return nothing
	// rather than panic.
	id, name := r.identifyDependency(context.Background(), nil)
	if id != "" || name != "" {
		t.Fatalf("a nil model yields no identity, got id %q name %q", id, name)
	}
}
