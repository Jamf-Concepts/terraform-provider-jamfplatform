// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// twoNamesakesListBody is a collection holding two connections Jamf stores under
// the same name, which is possible because the stored name is not a unique key.
const twoNamesakesListBody = `{"totalCount":2,"results":[` + oidcSummaryBody + `,` +
	`{"id":"con_unittest0004","name":"` + unitConnectionName + `","type":"OIDC","region":"EU",` +
	`"domains":["tf-unit-two.example"],"enabledApplications":["ACCOUNT"],"easyConfig":false,` +
	`"syncUserProfileAttributesAtLogin":true,"ticketUrl":null,"tokenEndpointAuthMethod":"CLIENT_SECRET_POST"}]}`

// dataSourceConfig builds a data source configuration carrying one lookup key.
//
// Every other attribute is empty rather than the object itself being empty: a
// wholly empty object cannot be decoded into a Go struct.
func dataSourceConfig(ctx context.Context, t *testing.T, s datasourceschema.Schema, key, value string) tfsdk.Config {
	t.Helper()
	object := s.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	holder := tfsdk.State{Schema: s, Raw: tftypes.NewValue(object, values)}
	if diags := holder.SetAttribute(ctx, path.Root(key), value); diags.HasError() {
		t.Fatalf("setting %s: %v", key, diags)
	}
	return tfsdk.Config{Schema: s, Raw: holder.Raw}
}

// readConnectionDataSource runs the singular data source against a stub driven
// by handle, and returns the response.
func readConnectionDataSource(t *testing.T, key, value string, handle stubHandler) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()

	d := &ConnectionDataSource{client: newStubClient(t, handle)}

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: dataSourceConfig(ctx, t, schemaResp.Schema, key, value)}, &resp)
	return resp
}

// connectionStubHandler serves the collection from list and one connection from
// the single read, recording the calls it saw.
func connectionStubHandler(calls *[]string, list, single string) stubHandler {
	return func(w http.ResponseWriter, req *http.Request) {
		if calls != nil {
			*calls = append(*calls, req.Method+" "+req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/connections") {
			_, _ = w.Write([]byte(list))
			return
		}
		_, _ = w.Write([]byte(single))
	}
}

// TestDataSourceRead_ByIdentifierTakesBothCalls pins that a lookup by identifier
// still reads the collection, because the enabled products and the consent
// ticket appear nowhere else.
func TestDataSourceRead_ByIdentifierTakesBothCalls(t *testing.T) {
	ctx := context.Background()
	var calls []string

	resp := readConnectionDataSource(t, "id", unitConnectionID, connectionStubHandler(&calls, oneConnectionListBody, oidcConnectionBody))

	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	want := []string{"GET /sso/v1/connections", "GET /sso/v1/connections/" + unitConnectionID}
	if len(calls) != len(want) || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("read issued %v, want %v", calls, want)
	}

	var state ConnectionDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.Name.ValueString() != unitConnectionName {
		t.Errorf("name = %q, want the stored name reported as it is", state.Name.ValueString())
	}
	if state.ConnectionType.ValueString() != connectionTypeGenericOIDC {
		t.Errorf("connection_type = %q, want the renamed value", state.ConnectionType.ValueString())
	}
	if len(state.EnabledProductNames.Elements()) != 2 {
		t.Errorf("enabled_product_names = %s, want the products from the collection", state.EnabledProductNames)
	}
	if state.GenericOIDC == nil || state.GenericOIDC.JWKSURI.ValueString() != "idp.example/keys" {
		t.Errorf("generic_oidc = %+v, want the settings reported", state.GenericOIDC)
	}
	if state.Entra != nil || state.Okta != nil || state.GoogleWorkspace != nil {
		t.Error("only the block matching the connection's family may be reported")
	}
}

// TestDataSourceRead_ByNameResolvesTheIdentifier pins the name lookup, which
// resolves through the collection because nothing reads a connection by name.
func TestDataSourceRead_ByNameResolvesTheIdentifier(t *testing.T) {
	ctx := context.Background()
	var calls []string

	resp := readConnectionDataSource(t, "name", unitConnectionName, connectionStubHandler(&calls, oneConnectionListBody, oidcConnectionBody))

	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	if len(calls) != 2 || !strings.HasSuffix(calls[1], "/connections/"+unitConnectionID) {
		t.Errorf("read issued %v, want the name resolved to an identifier", calls)
	}

	var state ConnectionDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.ID.ValueString() != unitConnectionID {
		t.Errorf("id = %q, want the identifier the name resolved to", state.ID.ValueString())
	}
}

// TestDataSourceRead_AmbiguousNameIsReported pins that a name matching two
// connections is refused rather than resolved by picking one. The stored name is
// not a unique key, and silently choosing would give a configuration that reads a
// different connection from one run to the next.
func TestDataSourceRead_AmbiguousNameIsReported(t *testing.T) {
	var calls []string

	resp := readConnectionDataSource(t, "name", unitConnectionName, connectionStubHandler(&calls, twoNamesakesListBody, oidcConnectionBody))

	if !resp.Diagnostics.HasError() {
		t.Fatal("a name matching two connections must be reported, not resolved by picking one")
	}
	if len(calls) != 1 {
		t.Errorf("read issued %v, want no single read after an ambiguous name", calls)
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	for _, want := range []string{unitConnectionID, "con_unittest0004"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail %q does not name %s so the operator can pick", detail, want)
		}
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a failed lookup must write no state")
	}
}

// TestDataSourceRead_UnknownNameNamesTheAttribute pins the not-found path, and
// that it explains why a name a practitioner asked for may not be the name Jamf
// holds.
func TestDataSourceRead_UnknownNameNamesTheAttribute(t *testing.T) {
	resp := readConnectionDataSource(t, "name", "tf-unit-absent", connectionStubHandler(nil, oneConnectionListBody, oidcConnectionBody))

	if !resp.Diagnostics.HasError() {
		t.Fatal("a connection the organization does not hold must be reported, not returned empty")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "uniquified") {
		t.Errorf("detail %q does not explain why the name may differ", detail)
	}
	if !strings.Contains(detail, "jamfplatform_account_sso_connections") {
		t.Errorf("detail %q does not point at the plural data source", detail)
	}
}

// TestDataSourceRead_UnknownIdentifierNamesTheAttribute pins the identifier
// not-found path, which is settled by the collection alone and issues no second
// call.
func TestDataSourceRead_UnknownIdentifierNamesTheAttribute(t *testing.T) {
	var calls []string

	resp := readConnectionDataSource(t, "id", "con_unittestabsent", connectionStubHandler(&calls, oneConnectionListBody, oidcConnectionBody))

	if !resp.Diagnostics.HasError() {
		t.Fatal("an identifier the organization does not hold must be reported")
	}
	if len(calls) != 1 {
		t.Errorf("read issued %v, want the collection alone", calls)
	}
}

// TestDataSourceRead_GhostConnectionIsReported pins the second class of
// unmanageable connection on the read path. Returning it empty would read as
// "this connection has no settings", which is a different and wrong statement.
func TestDataSourceRead_GhostConnectionIsReported(t *testing.T) {
	resp := readConnectionDataSource(t, "id", "con_unittest0003", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/connections") {
			_, _ = w.Write([]byte(ghostOnlyListBody))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(notFoundBody))
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("a connection Jamf lists but cannot read must be reported")
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a failed lookup must write no state")
	}
	if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "con_unittest0003") {
		t.Errorf("detail %q does not name the identifier to raise", detail)
	}
}

// TestDataSourceRead_ConsentFlowConnectionIsReadable is the difference between
// the data source and the resource. The resource refuses such a connection
// because it could never apply; reading one takes no ownership of it, and is the
// only way to see it in Terraform at all.
func TestDataSourceRead_ConsentFlowConnectionIsReadable(t *testing.T) {
	ctx := context.Background()
	list := `{"totalCount":1,"results":[` + consentFlowSummaryBody + `]}`

	resp := readConnectionDataSource(t, "id", "con_unittest0002", connectionStubHandler(nil, list, consentFlowConnectionBody))

	if resp.Diagnostics.HasError() {
		t.Fatalf("a connection using Microsoft admin consent must be readable: %v", resp.Diagnostics)
	}

	var state ConnectionDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if !state.ConsentFlow.ValueBool() {
		t.Error("consent_flow must report the connection as using Microsoft admin consent")
	}
	if !state.ClientID.IsNull() {
		t.Errorf("client_id = %s, want nothing — such a connection has no client of its own", state.ClientID)
	}
	if state.Entra == nil || state.Entra.Domain.ValueString() != "contoso.example" {
		t.Errorf("entra = %+v, want the settings reported", state.Entra)
	}
	if state.Entra != nil && !state.Entra.BasicProfile.ValueBool() {
		t.Error("basic_profile must be reported, since it is always on and never a choice")
	}
}
