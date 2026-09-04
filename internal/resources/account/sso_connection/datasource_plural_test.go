// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// readConnectionsDataSource runs the plural data source against a stub serving
// body from the collection, and returns the response.
func readConnectionsDataSource(t *testing.T, body string) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()

	d := &ConnectionsDataSource{client: newStubClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})}

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}

	object := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(object, values)},
	}, &resp)
	return resp
}

// TestPluralDataSourceRead_ReportsEveryConnection pins the plural read, and the
// one thing that separates it from the list resource: nothing is left out. A
// connection this provider could not manage is still part of the organization's
// configuration, and this is where it can be seen.
func TestPluralDataSourceRead_ReportsEveryConnection(t *testing.T) {
	ctx := context.Background()

	resp := readConnectionsDataSource(t, mixedConnectionListBody)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var state ConnectionsDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if len(state.SSOConnections) != 3 {
		t.Fatalf("reported %d connections, want all three including the two that cannot be managed", len(state.SSOConnections))
	}
	if state.ID.ValueString() != pluralDataSourceID {
		t.Errorf("id = %q, want the fixed identifier", state.ID.ValueString())
	}

	first := state.SSOConnections[0]
	if first.ID.ValueString() != unitConnectionID || first.Name.ValueString() != unitConnectionName {
		t.Errorf("first entry = %+v, want Jamf's own order preserved", first)
	}
	if first.ConnectionType.ValueString() != connectionTypeGenericOIDC {
		t.Errorf("connection_type = %q, want the renamed value", first.ConnectionType.ValueString())
	}
	if first.AuthMethod.ValueString() != authMethodClientSecret {
		t.Errorf("auth_method = %q, want the renamed value", first.AuthMethod.ValueString())
	}
	if len(first.EnabledProductNames.Elements()) != 2 {
		t.Errorf("enabled_product_names = %s, want the products the collection reports", first.EnabledProductNames)
	}
	if !first.TicketURL.IsNull() {
		t.Errorf("ticket_url = %s, want nothing for a connection with no outstanding consent", first.TicketURL)
	}
}

// TestPluralDataSourceRead_EmptyOrganizationIsNotAnError pins that an
// organization holding no connections reports an empty collection rather than
// failing — and reports it as a known empty set, not as nothing.
func TestPluralDataSourceRead_EmptyOrganizationIsNotAnError(t *testing.T) {
	ctx := context.Background()

	resp := readConnectionsDataSource(t, emptyConnectionListBody)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var state ConnectionsDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.SSOConnections == nil || len(state.SSOConnections) != 0 {
		t.Errorf("sso_connections = %v, want a known empty collection", state.SSOConnections)
	}
}

// TestPluralDataSourceRead_BareArrayCollectionIsAccepted pins the shape Jamf
// Account is known to answer either way. The account collections shipped broken
// for five days over exactly this, so it is worth an assertion rather than trust
// in the declared envelope.
func TestPluralDataSourceRead_BareArrayCollectionIsAccepted(t *testing.T) {
	ctx := context.Background()

	resp := readConnectionsDataSource(t, `[`+oidcSummaryBody+`]`)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var state ConnectionsDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if len(state.SSOConnections) != 1 {
		t.Errorf("reported %d connections from a bare collection, want one", len(state.SSOConnections))
	}
}

// TestPluralDataSourceRead_FailureIsReported pins that a collection that cannot
// be read fails rather than reporting an organization with no connections, which
// is the inference an empty result would invite.
func TestPluralDataSourceRead_FailureIsReported(t *testing.T) {
	ctx := context.Background()

	d := &ConnectionsDataSource{client: newStubClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":[{"code":"BAD_PERMISSIONS","field":null,"description":"Forbidden"}],"httpStatus":403}`))
	})}

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	object := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(object, values)},
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a collection that cannot be read must be reported, not returned empty")
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a failed read must write no state")
	}
}
