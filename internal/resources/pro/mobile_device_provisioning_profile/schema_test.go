// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_provisioning_profile

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const wantTypeName = "jamfplatform_pro_mobile_device_provisioning_profile"

func TestProvisioningProfileResource_Metadata(t *testing.T) {
	r := NewProvisioningProfileResource()
	var resp resource.MetadataResponse
	r.(*ProvisioningProfileResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestProvisioningProfileResource_Schema(t *testing.T) {
	r := NewProvisioningProfileResource()
	var resp resource.SchemaResponse
	r.(*ProvisioningProfileResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "display_name", "profile_data", "uuid", "creation_date_utc", "creation_date_epoch", "expiration_date_utc", "expiration_date_epoch", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}
	if name := s.Attributes["name"]; !name.IsRequired() {
		t.Errorf("name must be required")
	}
	if pd := s.Attributes["profile_data"]; !pd.IsOptional() || pd.IsComputed() {
		t.Errorf("profile_data must be optional-only, got optional=%v computed=%v", pd.IsOptional(), pd.IsComputed())
	}
	// display_name is server-derived (forced == name) ⇒ computed-only.
	for _, c := range []string{"display_name", "uuid", "creation_date_utc", "creation_date_epoch", "expiration_date_utc", "expiration_date_epoch"} {
		a := s.Attributes[c]
		if !a.IsComputed() || a.IsOptional() || a.IsRequired() {
			t.Errorf("%q must be computed-only, got optional=%v computed=%v required=%v", c, a.IsOptional(), a.IsComputed(), a.IsRequired())
		}
	}
}

func TestProvisioningProfileDataSource_Metadata(t *testing.T) {
	d := NewProvisioningProfileDataSource()
	var resp datasource.MetadataResponse
	d.(*ProvisioningProfileDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestProvisioningProfileDataSource_Schema(t *testing.T) {
	d := NewProvisioningProfileDataSource()
	var resp datasource.SchemaResponse
	d.(*ProvisioningProfileDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "name", "uuid"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed selector, got optional=%v computed=%v", sel, a.IsOptional(), a.IsComputed())
		}
	}
}

func TestProvisioningProfileDataSource_ConfigValidators(t *testing.T) {
	d := NewProvisioningProfileDataSource().(*ProvisioningProfileDataSource)
	if got := d.ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestProvisioningProfileListResource_Metadata(t *testing.T) {
	r := NewProvisioningProfileListResource()
	var resp resource.MetadataResponse
	r.(*ProvisioningProfileListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected list type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestProvisioningProfileListResource_Schema(t *testing.T) {
	r := NewProvisioningProfileListResource()
	var resp list.ListResourceSchemaResponse
	r.(*ProvisioningProfileListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
