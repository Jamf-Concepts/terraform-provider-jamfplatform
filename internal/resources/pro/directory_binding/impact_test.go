// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package directory_binding

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDirectoryBindingIdentifyDependency(t *testing.T) {
	r := &DirectoryBindingResource{}
	// The alert names the object the operator recognises, so the pair must be the
	// binding's id and the name shown in Jamf Pro.
	id, name := r.identifyDependency(context.Background(), &DirectoryBindingResourceModel{
		ID:   types.StringValue("42"),
		Name: types.StringValue("Campus AD"),
	})
	if id != "42" {
		t.Fatalf("id = %q, want %q", id, "42")
	}
	if name != "Campus AD" {
		t.Fatalf("name = %q, want %q", name, "Campus AD")
	}
}

func TestDirectoryBindingIdentifyDependencyNilModel(t *testing.T) {
	r := &DirectoryBindingResource{}
	// A destroy plan has no target model at all; the adapter must return nothing
	// rather than panic.
	id, name := r.identifyDependency(context.Background(), nil)
	if id != "" || name != "" {
		t.Fatalf("a nil model yields no identity, got id %q name %q", id, name)
	}
}
