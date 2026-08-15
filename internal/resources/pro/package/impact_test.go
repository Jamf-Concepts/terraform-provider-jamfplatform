// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPackageIdentifyDependency(t *testing.T) {
	r := &PackageResource{}
	// A package has no `name`: the operator-facing label is its display name, and
	// that is what the alert must quote.
	id, name := r.identifyDependency(context.Background(), &PackageResourceModel{
		ID:          types.StringValue("42"),
		DisplayName: types.StringValue("Firefox 130"),
		FileName:    types.StringValue("firefox-130.pkg"),
	})
	if id != "42" {
		t.Fatalf("id = %q, want %q", id, "42")
	}
	if name != "Firefox 130" {
		t.Fatalf("name = %q, want the display name %q", name, "Firefox 130")
	}
}

func TestPackageIdentifyDependencyNilModel(t *testing.T) {
	r := &PackageResource{}
	// A destroy plan has no target model at all; the adapter must return nothing
	// rather than panic.
	id, name := r.identifyDependency(context.Background(), nil)
	if id != "" || name != "" {
		t.Fatalf("a nil model yields no identity, got id %q name %q", id, name)
	}
}
