// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPrinterIdentifyDependency(t *testing.T) {
	r := &PrinterResource{}
	// The alert names the object the operator recognises, so the pair must be the
	// printer's id and the name shown in Jamf Pro.
	id, name := r.identifyDependency(context.Background(), &PrinterResourceModel{
		ID:   types.StringValue("42"),
		Name: types.StringValue("Third Floor Laser"),
	})
	if id != "42" {
		t.Fatalf("id = %q, want %q", id, "42")
	}
	if name != "Third Floor Laser" {
		t.Fatalf("name = %q, want %q", name, "Third Floor Laser")
	}
}

func TestPrinterIdentifyDependencyNilModel(t *testing.T) {
	r := &PrinterResource{}
	// A destroy plan has no target model at all; the adapter must return nothing
	// rather than panic.
	id, name := r.identifyDependency(context.Background(), nil)
	if id != "" || name != "" {
		t.Fatalf("a nil model yields no identity, got id %q name %q", id, name)
	}
}
