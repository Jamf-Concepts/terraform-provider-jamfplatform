// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_domain

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// argumentlessConfig builds the configuration the plural data source arrives
// with: every attribute null, because it takes no arguments at all. The object
// itself cannot be null — a wholly null object does not decode into a Go struct.
func argumentlessConfig(ctx context.Context, s datasourceschema.Schema) tfsdk.Config {
	object := s.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	return tfsdk.Config{Schema: s, Raw: tftypes.NewValue(object, values)}
}

// readDomainsDataSource runs the plural data source against a stub serving body
// from the collection endpoint, and returns the state it produced.
func readDomainsDataSource(t *testing.T, body string) (datasource.ReadResponse, DomainsDataSourceModel) {
	t.Helper()
	ctx := context.Background()

	d := &DomainsDataSource{client: newStubClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})}

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: argumentlessConfig(ctx, schemaResp.Schema)}, &resp)
	if resp.Diagnostics.HasError() {
		return resp, DomainsDataSourceModel{}
	}

	var state DomainsDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	return resp, state
}

// TestPluralDataSourceRead_ReportsEveryDomainIncludingSharedOnes pins what the
// plural read is for. Unlike the list resource, which excludes a shared domain
// because nothing can manage one, this data source reports the organization's
// whole holding — so a shared domain has to survive the read and be identifiable
// as shared.
func TestPluralDataSourceRead_ReportsEveryDomainIncludingSharedOnes(t *testing.T) {
	_, state := readDomainsDataSource(t, ownedAndSharedDomainsBody)

	if len(state.SSODomains) != 2 {
		t.Fatalf("sso_domains holds %d entries, want both the shared and the owned domain", len(state.SSODomains))
	}
	if got := state.SSODomains[0].Domain.ValueString(); got != "tf-unit-shared.example" {
		t.Errorf("first entry = %q, want Jamf's own order preserved", got)
	}
	if !state.SSODomains[0].Shared.ValueBool() {
		t.Error("shared must be true for a domain another organization owns")
	}
	if state.SSODomains[1].Shared.ValueBool() {
		t.Error("shared must be false for a domain this organization claimed")
	}
	if state.ID.ValueString() != pluralDataSourceID {
		t.Errorf("id = %q, want the fixed identifier %q", state.ID.ValueString(), pluralDataSourceID)
	}
}

// TestPluralDataSourceRead_EmptyCollectionIsAnEmptyList pins that an
// organization holding nothing yields an empty collection rather than a null one:
// a null list makes `length(data...)` fail for a tenant that is simply empty.
func TestPluralDataSourceRead_EmptyCollectionIsAnEmptyList(t *testing.T) {
	resp, state := readDomainsDataSource(t, emptyDomainListBody)

	if resp.Diagnostics.HasError() {
		t.Fatalf("an organization holding no domains is not an error: %v", resp.Diagnostics)
	}
	if state.SSODomains == nil {
		t.Error("sso_domains must be an empty list, not null")
	}
	if len(state.SSODomains) != 0 {
		t.Errorf("sso_domains holds %d entries, want none", len(state.SSODomains))
	}
	if state.ID.ValueString() != pluralDataSourceID {
		t.Errorf("id = %q, want the fixed identifier even when the collection is empty", state.ID.ValueString())
	}
}

// TestPluralDataSourceRead_FailedCollectionReadIsReported pins the failure path:
// the read has one call and nothing to fall back on, so a refused collection read
// must be an error rather than an empty holding.
func TestPluralDataSourceRead_FailedCollectionReadIsReported(t *testing.T) {
	ctx := context.Background()

	d := &DomainsDataSource{client: newStubClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":[{"code":"BAD_PERMISSIONS","field":null,"description":"Forbidden"}],"httpStatus":403}`))
	})}

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: argumentlessConfig(ctx, schemaResp.Schema)}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a refused collection read must not be reported as an organization holding no domains")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Unable to list Jamf Account SSO domains" {
		t.Errorf("summary = %q", got)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a failed read must write no state")
	}
}
