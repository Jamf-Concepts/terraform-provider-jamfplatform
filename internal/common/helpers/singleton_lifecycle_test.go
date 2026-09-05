// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// The singleton import marker has to survive the protocol, not just a function
// call, so this drives the real framework server: ImportResourceState followed by
// ReadResource, with the private state carried between them exactly as Terraform
// carries it.
//
// A hand-built resource.ImportStateResponse cannot cover this. Its Private field
// is a *privatestate.ProviderData, that package is internal to
// terraform-plugin-framework, and nothing public constructs one — so a fixture
// leaves it nil, and SetKey on a nil receiver is an error rather than a write.
// Adding a nil guard to the helper to accommodate that would be guarding a
// condition production never reaches (fwserver always supplies an initialised
// value) while silently disabling the marker if it ever did.

// probeResource is the smallest resource that exercises the singleton import
// path: a state id, an identity, and a Read that records what
// helpers.IsSingletonImport told it.
type probeResource struct{}

// readOutcome records what the last ReadResource observed, so the test can assert
// on the helper's answer rather than only on the response.
type readOutcome struct {
	called   bool
	isImport bool
}

var lastRead readOutcome

func (probeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_singleton"
}

func (probeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true},
	}}
}

func (probeResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true},
	}}
}

func (probeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	helpers.ImportSingletonState(ctx, req, resp, "jamfplatform_pro_example_settings")
}

func (probeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	lastRead = readOutcome{called: true, isImport: helpers.IsSingletonImport(ctx, req, resp)}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), helpers.SingletonID)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), helpers.SingletonID)...)
	}
}

func (probeResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {}
func (probeResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {}
func (probeResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {}

type probeProvider struct{}

func (probeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jamfplatform"
}
func (probeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}
func (probeProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (probeProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return probeResource{} }}
}
func (probeProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

const probeTypeName = "jamfplatform_singleton"

var probeObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String}}

// TestSingletonImportMarker_SurvivesImportAndIsConsumedOnce covers the marker's
// whole life: written by the import, read by the Read that follows it, and gone
// from the private state that Read hands back, so the next refresh is a refresh.
//
// Both import forms are driven, because the marker is the mechanism that makes
// them behave alike — the flat-ID form is the one whose state is populated by
// ImportStatePassthroughWithIdentity and so cannot be detected by a null state.
func TestSingletonImportMarker_SurvivesImportAndIsConsumedOnce(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       string
		identity *tfprotov6.ResourceIdentityData
	}{
		{name: "flat id form", id: helpers.SingletonID},
		{name: "identity form", identity: identityData(t, helpers.SingletonID)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := providerserver.NewProtocol6(probeProvider{})()

			imported := mustImport(t, srv, tc.id, tc.identity)
			if len(imported.Private) == 0 {
				t.Fatal("the import wrote no private state, so the Read that follows cannot tell it from a refresh")
			}
			if !strings.Contains(string(imported.Private), "singleton_import") {
				t.Errorf("private state does not carry the import marker: %s", imported.Private)
			}

			lastRead = readOutcome{}
			afterImport := mustRead(t, srv, imported.State, imported.Private, imported.Identity)
			if !lastRead.called {
				t.Fatal("Read was not called")
			}
			if !lastRead.isImport {
				t.Error("the Read immediately following an import must report an import; the resource's own " +
					"\"nothing here to import\" diagnostic is unreachable otherwise")
			}
			if markerPresent(t, afterImport.Private) {
				t.Errorf("the marker must be consumed by the read it describes, or every later refresh looks "+
					"like an import: %s", afterImport.Private)
			}

			lastRead = readOutcome{}
			mustRead(t, srv, afterImport.NewState, afterImport.Private, afterImport.NewIdentity)
			if !lastRead.called {
				t.Fatal("second Read was not called")
			}
			if lastRead.isImport {
				t.Error("a refresh after the post-import read must NOT report an import")
			}
		})
	}
}

// TestSingletonImportMarker_NotWrittenWhenTheIdentifierIsRefused pins that a
// refused import leaves no marker behind. It cannot strand one for a later read —
// a failed import writes no state — but a marker written before the refusal would
// mean the helper wrote to a response it had already rejected.
func TestSingletonImportMarker_NotWrittenWhenTheIdentifierIsRefused(t *testing.T) {
	srv := providerserver.NewProtocol6(probeProvider{})()

	resp, err := srv.ImportResourceState(context.Background(), &tfprotov6.ImportResourceStateRequest{
		TypeName: probeTypeName,
		ID:       "not-the-singleton",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasError(resp.Diagnostics) {
		t.Fatal("a bogus identifier must be refused")
	}
	for _, ir := range resp.ImportedResources {
		if markerPresent(t, ir.Private) {
			t.Errorf("a refused import must write no marker, got: %s", ir.Private)
		}
	}
}

func mustImport(t *testing.T, srv tfprotov6.ProviderServer, id string, identity *tfprotov6.ResourceIdentityData) *tfprotov6.ImportedResource {
	t.Helper()

	resp, err := srv.ImportResourceState(context.Background(), &tfprotov6.ImportResourceStateRequest{
		TypeName: probeTypeName,
		ID:       id,
		Identity: identity,
	})
	if err != nil {
		t.Fatalf("ImportResourceState: %v", err)
	}
	if hasError(resp.Diagnostics) {
		t.Fatalf("import diagnostics: %s", renderDiags(resp.Diagnostics))
	}
	if len(resp.ImportedResources) != 1 {
		t.Fatalf("expected 1 imported resource, got %d", len(resp.ImportedResources))
	}
	return resp.ImportedResources[0]
}

func mustRead(t *testing.T, srv tfprotov6.ProviderServer, state *tfprotov6.DynamicValue, private []byte, identity *tfprotov6.ResourceIdentityData) *tfprotov6.ReadResourceResponse {
	t.Helper()

	resp, err := srv.ReadResource(context.Background(), &tfprotov6.ReadResourceRequest{
		TypeName:        probeTypeName,
		CurrentState:    state,
		Private:         private,
		CurrentIdentity: identity,
	})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if hasError(resp.Diagnostics) {
		t.Fatalf("read diagnostics: %s", renderDiags(resp.Diagnostics))
	}
	return resp
}

// markerPresent reports whether the private-state blob still carries the import
// marker. Private state is a JSON object keyed by provider-chosen names, and the
// framework removes a key rather than blanking it, so absence is the assertion.
func markerPresent(t *testing.T, private []byte) bool {
	t.Helper()
	if len(private) == 0 {
		return false
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(private, &keys); err != nil {
		t.Fatalf("private state is not a JSON object (%v): %s", err, private)
	}
	_, ok := keys["singleton_import"]
	return ok
}

func identityData(t *testing.T, id string) *tfprotov6.ResourceIdentityData {
	t.Helper()
	dv, err := tfprotov6.NewDynamicValue(probeObjType, tftypes.NewValue(probeObjType, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, id),
	}))
	if err != nil {
		t.Fatal(err)
	}
	return &tfprotov6.ResourceIdentityData{IdentityData: &dv}
}

func hasError(diags []*tfprotov6.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			return true
		}
	}
	return false
}

func renderDiags(diags []*tfprotov6.Diagnostic) string {
	var b strings.Builder
	for _, d := range diags {
		b.WriteString(d.Summary + ": " + d.Detail + "\n")
	}
	return b.String()
}
