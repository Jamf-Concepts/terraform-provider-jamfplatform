// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package automated_device_enrollment_public_key

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestAutomatedDeviceEnrollmentPublicKeyDataSource_Metadata(t *testing.T) {
	d := NewAutomatedDeviceEnrollmentPublicKeyDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*AutomatedDeviceEnrollmentPublicKeyDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_automated_device_enrollment_public_key" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_automated_device_enrollment_public_key", resp.TypeName)
	}
}

func TestAutomatedDeviceEnrollmentPublicKeyDataSource_Schema(t *testing.T) {
	d := NewAutomatedDeviceEnrollmentPublicKeyDataSource()
	var resp datasource.SchemaResponse
	d.(*AutomatedDeviceEnrollmentPublicKeyDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "public_key", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	idAttr := s.Attributes["id"]
	if idAttr.IsRequired() || idAttr.IsOptional() || !idAttr.IsComputed() {
		t.Errorf("id must be computed-only (singleton), got required=%v optional=%v computed=%v",
			idAttr.IsRequired(), idAttr.IsOptional(), idAttr.IsComputed())
	}

	pk := s.Attributes["public_key"]
	if !pk.IsComputed() {
		t.Errorf("public_key must be computed")
	}
	if pk.IsRequired() || pk.IsOptional() {
		t.Errorf("public_key must be computed-only, got required=%v optional=%v", pk.IsRequired(), pk.IsOptional())
	}
}
