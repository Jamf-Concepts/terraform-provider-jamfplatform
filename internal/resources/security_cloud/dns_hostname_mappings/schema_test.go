// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_hostname_mappings

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	commonvalidators "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/validators"
)

func TestHostnameMappingsResource_Metadata(t *testing.T) {
	r := NewHostnameMappingsResource()
	var resp resource.MetadataResponse
	r.(*HostnameMappingsResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if want := "jamfplatform_security_cloud_dns_hostname_mappings"; resp.TypeName != want {
		t.Errorf("expected type name %q, got %q", want, resp.TypeName)
	}
}

// resourceSchema builds the resource schema for the tests below.
func resourceSchema(t *testing.T) rschema.Schema {
	t.Helper()
	r := NewHostnameMappingsResource()
	var resp resource.SchemaResponse
	r.(*HostnameMappingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestHostnameMappingsResource_Schema(t *testing.T) {
	s := resourceSchema(t)

	for _, name := range []string{"id", "mappings", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["mappings"].IsRequired() {
		t.Error("mappings must be Required — the resource owns the whole collection")
	}
	if s.Attributes["id"].IsRequired() || s.Attributes["id"].IsOptional() {
		t.Error("id must be Computed-only")
	}
}

// TestHostnameMappingsResource_MappingsIsASet pins the wire truth that mapping order
// is not preserved: sending z, a, m reads back m, a, z. A ListNestedAttribute here
// would diff on every refresh.
func TestHostnameMappingsResource_MappingsIsASet(t *testing.T) {
	s := resourceSchema(t)
	if _, ok := s.Attributes["mappings"].(rschema.SetNestedAttribute); !ok {
		t.Fatalf("mappings must be a SetNestedAttribute, got %T", s.Attributes["mappings"])
	}
}

// TestHostnameMappingsResource_AddressesAreSets pins the second reason: the server
// dedupes addresses within a mapping, so a list would diff forever against a
// configuration containing a duplicate.
func TestHostnameMappingsResource_AddressesAreSets(t *testing.T) {
	mappings := resourceSchema(t).Attributes["mappings"].(rschema.SetNestedAttribute)
	for _, name := range []string{"ipv4_addresses", "ipv6_addresses"} {
		attribute, ok := mappings.NestedObject.Attributes[name]
		if !ok {
			t.Fatalf("missing nested attribute %q", name)
		}
		if _, isSet := attribute.(rschema.SetAttribute); !isSet {
			t.Errorf("%s must be a SetAttribute, got %T", name, attribute)
		}
		if !attribute.IsOptional() {
			t.Errorf("%s must be Optional — either list may be omitted", name)
		}
		if attribute.IsComputed() {
			t.Errorf("%s must not be Computed; an empty list is collapsed to null in the state builder instead", name)
		}
	}
}

// TestHostnameMappingsResource_BooleansAreRequiredWithoutDefaults guards a regression
// that only a real plan exposes.
//
// These two started as Optional with booldefault.StaticBool(false), matching the
// wire's own default. That produces a permanent diff, because an attribute default
// inside a SetNestedAttribute overrides a value the configuration set explicitly:
// `connect_to_ztna = true` planned as `false`, verified against a live tenant with a
// single mapping on 2026-08-29. Unit tests pass either way, which is exactly why this
// one pins the shape rather than the behaviour.
func TestHostnameMappingsResource_BooleansAreRequiredWithoutDefaults(t *testing.T) {
	mappings := resourceSchema(t).Attributes["mappings"].(rschema.SetNestedAttribute)
	for _, name := range []string{"connect_to_ztna", "connect_to_secure_dns"} {
		attribute, ok := mappings.NestedObject.Attributes[name].(rschema.BoolAttribute)
		if !ok {
			t.Fatalf("%s must be a BoolAttribute", name)
		}
		if !attribute.Required {
			t.Errorf("%s must be Required", name)
		}
		if attribute.Computed {
			t.Errorf("%s must not be Computed", name)
		}
		if attribute.Default != nil {
			t.Errorf("%s must carry no default — a default inside a SetNestedAttribute overrides explicit config", name)
		}
	}

	ztna := mappings.NestedObject.Attributes["connect_to_ztna"].GetMarkdownDescription()
	if !strings.Contains(ztna, "admin UI's add") {
		t.Errorf("connect_to_ztna description must warn that the admin UI pre-selects it, got: %s", ztna)
	}
}

// TestHostnameMappingsResource_DescribesWholeCollectionOwnership is the one thing a
// reader can get materially wrong: this resource replaces the tenant's entire mapping
// set, so a mapping added elsewhere disappears on the next apply.
func TestHostnameMappingsResource_DescribesWholeCollectionOwnership(t *testing.T) {
	s := resourceSchema(t)
	if got := s.MarkdownDescription; !strings.Contains(got, "**entire** set") {
		t.Errorf("resource description must say it owns the **entire** set, got: %s", got)
	}

	mappings := s.Attributes["mappings"].GetMarkdownDescription()
	if !strings.Contains(mappings, "destroy the resource") {
		t.Errorf("mappings description must point at destroy rather than an empty collection, got: %s", mappings)
	}
}

// describable is what the wiring assertions below match on: every framework
// validator renders a description, and a description survives the rename of an
// unexported validator type that a type assertion would not.
type describable interface {
	Description(context.Context) string
}

// validatorDescriptions renders each validator in a schema slice.
func validatorDescriptions[T describable](ctx context.Context, validators []T) []string {
	out := make([]string, 0, len(validators))
	for _, v := range validators {
		out = append(out, v.Description(ctx))
	}
	return out
}

// requireValidators asserts that every expected validator is wired to an attribute.
//
// Presence rather than an exact slice, so a validator added later — the sibling
// agent's trailing-dot rule on `hostname` is the immediate case — does not fail a
// test that has nothing to say about it.
func requireValidators(t *testing.T, attribute string, got, want []string) {
	t.Helper()
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("%s is missing a validator described as %q; wired: %v", attribute, w, got)
		}
	}
}

// TestHostnameMappingsResource_ValidatorsAreWired pins each schema Validators slice.
//
// Deleting one of these entries compiles and leaves the whole package suite green,
// because the validators' own unit tests call them directly rather than through the
// schema — so the unwired validator keeps passing its tests while no plan ever runs
// it. Every entry here is load-bearing on a failure the server reports badly or not
// at all: a duplicate host name is a 500 naming nothing, a mapping with no address is
// blamed on aRecords whichever list was omitted, and an address in the wrong family
// is a bare "Invalid field value.".
func TestHostnameMappingsResource_ValidatorsAreWired(t *testing.T) {
	ctx := context.Background()
	mappings := resourceSchema(t).Attributes["mappings"].(rschema.SetNestedAttribute)

	requireValidators(t, "mappings", validatorDescriptions(ctx, mappings.Validators), []string{
		setvalidator.SizeBetween(minMappings, maxMappings).Description(ctx),
		commonvalidators.UniqueStringFieldSet("hostname").Description(ctx),
		EachMappingHasAnAddress().Description(ctx),
	})

	hostname, ok := mappings.NestedObject.Attributes["hostname"].(rschema.StringAttribute)
	if !ok {
		t.Fatalf("hostname must be a StringAttribute, got %T", mappings.NestedObject.Attributes["hostname"])
	}
	requireValidators(t, "hostname", validatorDescriptions(ctx, hostname.Validators), []string{
		commonvalidators.DNSHostname().Description(ctx),
	})

	for name, element := range map[string]validator.String{
		"ipv4_addresses": commonvalidators.IPv4Address(),
		"ipv6_addresses": IPv6Address(),
	} {
		addresses, ok := mappings.NestedObject.Attributes[name].(rschema.SetAttribute)
		if !ok {
			t.Fatalf("%s must be a SetAttribute, got %T", name, mappings.NestedObject.Attributes[name])
		}
		requireValidators(t, name, validatorDescriptions(ctx, addresses.Validators), []string{
			setvalidator.SizeBetween(minAddresses, maxAddresses).Description(ctx),
			setvalidator.ValueStringsAre(element).Description(ctx),
		})
	}
}

// TestHostnameMappingsResource_IdentitySchema pins the singleton import contract.
func TestHostnameMappingsResource_IdentitySchema(t *testing.T) {
	r := NewHostnameMappingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*HostnameMappingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	attribute, ok := resp.IdentitySchema.Attributes["id"]
	if !ok {
		t.Fatal("identity schema missing id")
	}
	if !attribute.IsRequiredForImport() {
		t.Error("identity id must be RequiredForImport")
	}
}

// TestHostnameMappingsResource_ImportStateRejectsOtherIDs makes the singleton import
// identifier load-bearing. The endpoint takes no identifier, so a mis-typed import
// would otherwise succeed against whatever the tenant holds and hide the mistake.
func TestHostnameMappingsResource_ImportStateRejectsOtherIDs(t *testing.T) {
	for _, id := range []string{"", "1", "hostname-mappings", "SINGLETON"} {
		t.Run(id, func(t *testing.T) {
			r := NewHostnameMappingsResource().(*HostnameMappingsResource)
			var resp resource.ImportStateResponse
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("ImportState(%q) must be rejected", id)
			}
		})
	}
}

func TestHostnameMappingsDataSource_Metadata(t *testing.T) {
	d := NewHostnameMappingsDataSource()
	var resp datasource.MetadataResponse
	d.(*HostnameMappingsDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if want := "jamfplatform_security_cloud_dns_hostname_mappings"; resp.TypeName != want {
		t.Errorf("expected type name %q, got %q", want, resp.TypeName)
	}
}

// TestHostnameMappingsDataSource_ReadOnlyAndListShaped pins two rules at once: the
// data source takes no arguments, and its Computed nested collections are lists per
// STYLE_GUIDE §Sets vs Lists — a mistake that otherwise surfaces only under an
// acceptance apply.
func TestHostnameMappingsDataSource_ReadOnlyAndListShaped(t *testing.T) {
	d := NewHostnameMappingsDataSource()
	var resp datasource.SchemaResponse
	d.(*HostnameMappingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	for name, attribute := range resp.Schema.Attributes {
		if name == "timeouts" {
			continue
		}
		if attribute.IsRequired() || attribute.IsOptional() {
			t.Errorf("%s must be computed-only; the endpoint takes no arguments", name)
		}
	}

	mappings, ok := resp.Schema.Attributes["mappings"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("mappings must be a ListNestedAttribute, got %T", resp.Schema.Attributes["mappings"])
	}
	for _, name := range []string{"ipv4_addresses", "ipv6_addresses"} {
		if _, isList := mappings.NestedObject.Attributes[name].(dsschema.ListAttribute); !isList {
			t.Errorf("%s must be a ListAttribute, got %T", name, mappings.NestedObject.Attributes[name])
		}
	}
}

// TestSingletonIDIsStable catches drift in the shared constant this package's import
// contract and error messages are both written against.
func TestSingletonIDIsStable(t *testing.T) {
	if helpers.SingletonID != "singleton" {
		t.Fatalf("helpers.SingletonID = %q, want \"singleton\"", helpers.SingletonID)
	}
}
