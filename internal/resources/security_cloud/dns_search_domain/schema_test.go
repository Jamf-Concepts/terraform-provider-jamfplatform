// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_search_domain

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	commonvalidators "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/validators"
)

func TestSearchDomainResource_Metadata(t *testing.T) {
	r := NewSearchDomainResource()
	var resp resource.MetadataResponse
	r.(*SearchDomainResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if want := "jamfplatform_security_cloud_dns_search_domain"; resp.TypeName != want {
		t.Errorf("expected type name %q, got %q", want, resp.TypeName)
	}
}

func TestSearchDomainResource_Schema(t *testing.T) {
	r := NewSearchDomainResource()
	var resp resource.SchemaResponse
	r.(*SearchDomainResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "domain_name", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["domain_name"].IsRequired() {
		t.Error("domain_name must be Required — the endpoint refuses a null or empty suffix")
	}
	if !s.Attributes["id"].IsComputed() {
		t.Error("id must be Computed — users cannot pick a singleton's ID")
	}
	if s.Attributes["id"].IsRequired() || s.Attributes["id"].IsOptional() {
		t.Error("id must be Computed-only")
	}
}

// TestSearchDomainResource_DomainNameValidatorIsWired reads the schema's own
// Validators slice, which nothing else in this suite does: with the assertions above
// alone, deleting the validator from the schema left every test passing while every
// malformed name went to the server for an opaque 400 naming no field.
//
// The match is on the validator's own description rather than its type, which is
// unexported, so the assertion says "this behaviour is wired in" rather than pinning
// an internal name.
//
// Exactly one validator is the second half of the assertion. The shared DNS host
// name validator already bounds the empty and over-long cases, so a length validator
// beside it produces two diagnostics for one typo — which is what this package
// shipped before.
func TestSearchDomainResource_DomainNameValidatorIsWired(t *testing.T) {
	ctx := context.Background()
	r := NewSearchDomainResource()
	var resp resource.SchemaResponse
	r.(*SearchDomainResource).Schema(ctx, resource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["domain_name"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("domain_name is %T, want schema.StringAttribute", resp.Schema.Attributes["domain_name"])
	}
	if len(attr.Validators) != 1 {
		t.Fatalf("domain_name carries %d validators, want exactly 1: the DNS host name validator already "+
			"bounds length, so a second one duplicates every diagnostic", len(attr.Validators))
	}
	if got, want := attr.Validators[0].Description(ctx), commonvalidators.DNSHostname().Description(ctx); got != want {
		t.Errorf("domain_name validator description = %q, want the shared DNS host name validator's %q", got, want)
	}
}

// TestSearchDomainResource_IdentitySchema pins the singleton import contract.
func TestSearchDomainResource_IdentitySchema(t *testing.T) {
	r := NewSearchDomainResource()
	var resp resource.IdentitySchemaResponse
	r.(*SearchDomainResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	attr, ok := resp.IdentitySchema.Attributes["id"]
	if !ok {
		t.Fatal("identity schema missing id")
	}
	if !attr.IsRequiredForImport() {
		t.Error("identity id must be RequiredForImport")
	}
}

// TestSearchDomainResource_ImportStateRejectsOtherIDs is the check that makes the
// singleton import identifier load-bearing. The endpoint takes no identifier, so a
// mis-typed import would otherwise succeed against whatever the tenant has and hide
// the mistake.
func TestSearchDomainResource_ImportStateRejectsOtherIDs(t *testing.T) {
	for _, id := range []string{"", "1", "search-domain", "SINGLETON"} {
		t.Run(id, func(t *testing.T) {
			r := NewSearchDomainResource().(*SearchDomainResource)
			var resp resource.ImportStateResponse
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("ImportState(%q) must be rejected", id)
			}
		})
	}
}

func TestSearchDomainDataSource_Metadata(t *testing.T) {
	d := NewSearchDomainDataSource()
	var resp datasource.MetadataResponse
	d.(*SearchDomainDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if want := "jamfplatform_security_cloud_dns_search_domain"; resp.TypeName != want {
		t.Errorf("expected type name %q, got %q", want, resp.TypeName)
	}
}

// TestSearchDomainDataSource_TakesNoArguments pins that the data source has nothing
// to select by: there is one search domain per tenant.
func TestSearchDomainDataSource_TakesNoArguments(t *testing.T) {
	d := NewSearchDomainDataSource()
	var resp datasource.SchemaResponse
	d.(*SearchDomainDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for name, attr := range resp.Schema.Attributes {
		if name == "timeouts" {
			continue
		}
		if attr.IsRequired() || attr.IsOptional() {
			t.Errorf("%s must be computed-only; the endpoint takes no arguments", name)
		}
	}
}

// TestSearchDomainDescriptionsSayItIsSingular is the one thing a reader can get
// materially wrong from the admin UI, which renders the saved value as a one-row
// table with its input box still present and so looks like a list.
func TestSearchDomainDescriptionsSayItIsSingular(t *testing.T) {
	r := NewSearchDomainResource()
	var resp resource.SchemaResponse
	r.(*SearchDomainResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if got := resp.Schema.MarkdownDescription; !strings.Contains(got, "one search domain per tenant") {
		t.Errorf("resource description must say there is one per tenant, got: %s", got)
	}
	if got := resp.Schema.Attributes["domain_name"].GetMarkdownDescription(); !strings.Contains(got, "not a list") {
		t.Errorf("domain_name description must say it is not a list, got: %s", got)
	}
}

// TestSingletonIDIsStable catches drift in the shared constant this package's import
// contract and error message are both written against.
func TestSingletonIDIsStable(t *testing.T) {
	if helpers.SingletonID != "singleton" {
		t.Fatalf("helpers.SingletonID = %q, want \"singleton\"", helpers.SingletonID)
	}
}
