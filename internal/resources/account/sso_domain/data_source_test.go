// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_domain

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// allocationBody is the assignment record a claimed domain answers with: the
// connections it is in use by, and whether Jamf ID sign-in is still allowed.
const allocationBody = `{
	"domain": "tf-unit.example",
	"jamfIdEnabled": true,
	"connections": [
		{
			"assignedConnection": "con_ABCDEFGHIJKLMNOP",
			"assignedConnectionOrgId": "org_ABCDEFGHIJKLMNOP",
			"authZeroRegion": "US"
		}
	]
}`

// configWithDomain builds a data source configuration carrying just the lookup
// key, which is the only argument either domain data source takes.
//
// Every other attribute is null rather than the object itself being null: a
// wholly null object cannot be decoded into a Go struct.
func configWithDomain(ctx context.Context, s datasourceschema.Schema, domain string) tfsdk.Config {
	object := s.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		if name == "domain" {
			values[name] = tftypes.NewValue(attributeType, domain)
			continue
		}
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	return tfsdk.Config{Schema: s, Raw: tftypes.NewValue(object, values)}
}

// readDomainDataSource runs the singular data source against a stub driven by
// handle, and returns the response.
func readDomainDataSource(t *testing.T, domain string, handle stubHandler) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()

	d := &DomainDataSource{client: newStubClient(t, handle)}

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{Config: configWithDomain(ctx, schemaResp.Schema, domain)}, &resp)
	return resp
}

// TestDataSourceRead_MatchesTheCollectionThenReadsTheAssignments pins both calls
// the read takes, and that they happen in that order for a reason: the assignment
// lookup is keyed on the domain name as Jamf stores it, which the collection scan
// is what supplies.
func TestDataSourceRead_MatchesTheCollectionThenReadsTheAssignments(t *testing.T) {
	ctx := context.Background()
	var calls []string

	resp := readDomainDataSource(t, "tf-unit.example", func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(req.URL.Path, "/sso/v1/domains/allocation/") {
			_, _ = w.Write([]byte(allocationBody))
			return
		}
		_, _ = w.Write([]byte(domainListBody))
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	want := []string{"GET /sso/v1/domains", "GET /sso/v1/domains/allocation/tf-unit.example"}
	if len(calls) != len(want) || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("read issued %v, want %v", calls, want)
	}

	var state DomainDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.ID.ValueString() != "26917" {
		t.Errorf("id = %q, want the identifier from the collection", state.ID.ValueString())
	}
	if state.VerificationTXTRecord.ValueString() != "jamf-site-verification=verification-key-claim" {
		t.Errorf("verification_txt_record = %q", state.VerificationTXTRecord.ValueString())
	}
	if !state.JamfIDEnabled.ValueBool() {
		t.Error("jamf_id_enabled must come from the assignment record, not be left null")
	}
	if state.AssignedConnections.IsNull() || len(state.AssignedConnections.Elements()) != 1 {
		t.Errorf("assigned_connections = %s, want the one connection the domain is assigned to", state.AssignedConnections)
	}
}

// TestDataSourceRead_MatchesTheStoredCaseOfTheDomain pins the case-insensitive
// match. Jamf lower-cases a domain when it stores it, so a lookup typed in mixed
// case has to find the claim rather than report it missing.
func TestDataSourceRead_MatchesTheStoredCaseOfTheDomain(t *testing.T) {
	resp := readDomainDataSource(t, "TF-Unit.example", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(req.URL.Path, "/sso/v1/domains/allocation/") {
			_, _ = w.Write([]byte(allocationBody))
			return
		}
		_, _ = w.Write([]byte(domainListBody))
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("a mixed-case lookup must find the stored claim: %v", resp.Diagnostics)
	}

	var state DomainDataSourceModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.Domain.ValueString() != "tf-unit.example" {
		t.Errorf("domain = %q, want Jamf's stored spelling", state.Domain.ValueString())
	}
}

// TestDataSourceRead_UnclaimedDomainNamesTheAttribute pins the not-found path.
// The collection carries no not-found status to branch on, so absence from the
// scan is the only signal — and the diagnostic has to land on `domain`, the one
// value the practitioner can change, and never leave state populated.
func TestDataSourceRead_UnclaimedDomainNamesTheAttribute(t *testing.T) {
	var calls []string

	resp := readDomainDataSource(t, "tf-unit-absent.example", func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(domainListBody))
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("a domain the organization has not claimed must be reported, not returned empty")
	}
	if len(calls) != 1 {
		t.Errorf("read issued %v, want the collection scan alone with no assignment lookup", calls)
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Unable to find Jamf Account SSO domain" {
		t.Errorf("summary = %q", got)
	}
	if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "jamfplatform_account_sso_domains") {
		t.Errorf("detail %q does not point at the plural data source", detail)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a failed lookup must write no state")
	}
}

// TestDataSourceRead_AssignmentFailureIsReported pins the second call's failure
// separately from the first. The claim exists, so the honest outcome is an error
// naming the domain rather than state carrying a null assignment list, which
// would read as "assigned to nothing".
func TestDataSourceRead_AssignmentFailureIsReported(t *testing.T) {
	resp := readDomainDataSource(t, "tf-unit.example", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(req.URL.Path, "/sso/v1/domains/allocation/") {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errors":[{"code":"BAD_PERMISSIONS","field":null,"description":"Forbidden"}],"httpStatus":403}`))
			return
		}
		_, _ = w.Write([]byte(domainListBody))
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("an assignment read that failed must not be reported as a domain assigned to nothing")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Unable to read Jamf Account SSO domain assignments" {
		t.Errorf("summary = %q", got)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a failed assignment read must write no state")
	}
}
