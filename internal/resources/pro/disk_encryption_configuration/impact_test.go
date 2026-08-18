// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDiskEncryptionConfigurationIdentifyDependency(t *testing.T) {
	r := &DiskEncryptionConfigurationResource{}
	// The alert names the object the operator recognises, so the pair must be the
	// configuration's id and the name shown in Jamf Pro.
	id, name := r.identifyDependency(context.Background(), &DiskEncryptionConfigurationResourceModel{
		ID:   types.StringValue("42"),
		Name: types.StringValue("FileVault Individual"),
	})
	if id != "42" {
		t.Fatalf("id = %q, want %q", id, "42")
	}
	if name != "FileVault Individual" {
		t.Fatalf("name = %q, want %q", name, "FileVault Individual")
	}
}

func TestDiskEncryptionConfigurationIdentifyDependencyNilModel(t *testing.T) {
	r := &DiskEncryptionConfigurationResource{}
	// A destroy plan has no target model at all; the adapter must return nothing
	// rather than panic.
	id, name := r.identifyDependency(context.Background(), nil)
	if id != "" || name != "" {
		t.Fatalf("a nil model yields no identity, got id %q name %q", id, name)
	}
}
