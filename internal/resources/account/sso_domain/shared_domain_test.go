// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_domain

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The Jamf Account domain collection returns the domains an organization has
// claimed and the domains other organizations have shared with it, in one
// undifferentiated list — probed 2026-09-02 in the test organization, which holds
// a shared entry carrying a foreign accountId next to its own claims. A shared
// domain is assignable to an SSO connection and refused every change and
// withdrawal, so it can never be a managed jamfplatform_account_sso_domain.
// list_resource_test.go pins the filter on the list stream; these tests pin the
// managed lifecycle, where a match on the name alone would otherwise let
// `terraform import` adopt one and leave an entry no destroy can remove.
//
// The collection fixtures are list_resource_test.go's: ownedAndSharedDomainsBody
// holds the shared domain named below alongside claimBody's owned one.
const (
	sharedFixtureDomain    = "tf-unit-shared.example"
	sharedFixtureAccountID = "001ZYXWVUTSRQPONM"
	ownedFixtureDomain     = "tf-unit.example"
)

// TestRead_SharedDomainIsRefused pins the import path. `terraform import` writes
// the domain name into state and refreshes, so this request shape is the one an
// import of a shared domain arrives as — and an ordinary refresh of a domain
// shared in after Terraform adopted it is the same shape again.
func TestRead_SharedDomainIsRefused(t *testing.T) {
	ctx := context.Background()

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(ownedAndSharedDomainsBody))
	})

	prior := domainRawValue(ctx, s, sharedFixtureDomain, true)
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: prior}, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: prior}}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a shared domain must not be adopted as a managed resource")
	}
	err := resp.Diagnostics.Errors()[0]
	if err.Summary() != "Domain is shared, not owned" {
		t.Errorf("summary = %q", err.Summary())
	}
	if !strings.Contains(err.Detail(), sharedFixtureAccountID) {
		t.Errorf("detail %q does not name the owning account", err.Detail())
	}
	if !strings.Contains(err.Detail(), "jamfplatform_account_sso_domain` data source") {
		t.Errorf("detail %q does not point at the data source", err.Detail())
	}
	if !strings.Contains(err.Detail(), "terraform state rm") {
		t.Errorf("detail %q does not say how to drop an entry already in state", err.Detail())
	}
}

// TestRead_SharedDomainIsRefusedOnTheIdentityOnlyPath covers the refresh
// Terraform performs when it holds an identity and no state, which is what an
// identity imported from a `terraform query` result produces.
func TestRead_SharedDomainIsRefusedOnTheIdentityOnlyPath(t *testing.T) {
	ctx := context.Background()

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(ownedAndSharedDomainsBody))
	})

	identityType := identity.Schema.Type().TerraformType(ctx).(tftypes.Object)
	requestIdentity := &tfsdk.ResourceIdentity{
		Schema: identity.Schema,
		Raw: tftypes.NewValue(identityType, map[string]tftypes.Value{
			"domain": tftypes.NewValue(tftypes.String, sharedFixtureDomain),
		}),
	}
	stateType := s.Type().TerraformType(ctx)

	resp := resource.ReadResponse{
		State:    tfsdk.State{Schema: s, Raw: tftypes.NewValue(stateType, nil)},
		Identity: identity,
	}
	r.Read(ctx, resource.ReadRequest{
		State:    tfsdk.State{Schema: s, Raw: tftypes.NewValue(stateType, nil)},
		Identity: requestIdentity,
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("an identity-only refresh of a shared domain must be refused")
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a refused shared domain must write no state")
	}
}

// TestRead_OwnedDomainStillHydrates pins that the ownership gate did not break
// the ordinary path: the owned claim in the same mixed collection reads fine.
func TestRead_OwnedDomainStillHydrates(t *testing.T) {
	ctx := context.Background()

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(ownedAndSharedDomainsBody))
	})

	prior := domainRawValue(ctx, s, ownedFixtureDomain, true)
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: prior}, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: prior}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	var state DomainResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.ID.ValueString() != "26917" {
		t.Errorf("id = %q, want the owned claim's identifier", state.ID.ValueString())
	}
	if state.Shared.ValueBool() {
		t.Error("shared must be false for the owned claim")
	}
}

// TestUpdate_SharedDomainIsRefused covers the refresh Update performs. Read
// refuses first in a normal apply, but a plan applied with the refresh skipped
// reaches Update directly, and that path writes state too.
func TestUpdate_SharedDomainIsRefused(t *testing.T) {
	ctx := context.Background()

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(ownedAndSharedDomainsBody))
	})

	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Update(ctx, resource.UpdateRequest{
		Plan: tfsdk.Plan{Schema: s, Raw: domainRawValue(ctx, s, sharedFixtureDomain, false)},
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("an update against a shared domain must be refused")
	}
	if summary := resp.Diagnostics.Errors()[0].Summary(); summary != "Domain is shared, not owned" {
		t.Errorf("summary = %q", summary)
	}
}

// sharedDomainState builds a destroy-time state object for a domain state records
// as shared, which is what a provider version without the Read gate could have
// written.
func sharedDomainState(ctx context.Context, t *testing.T, s resourceschema.Schema, domain, id string, shared bool) tftypes.Value {
	t.Helper()
	object := s.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		switch name {
		case "domain":
			values[name] = tftypes.NewValue(attributeType, domain)
		case "id":
			values[name] = tftypes.NewValue(attributeType, id)
		case "shared":
			values[name] = tftypes.NewValue(attributeType, shared)
		default:
			values[name] = tftypes.NewValue(attributeType, nil)
		}
	}
	return tftypes.NewValue(object, values)
}

// TestDelete_SharedDomainIsRefusedBeforeAnyRequest pins that no
// cross-organization withdrawal is ever issued. The check keys on the `shared`
// value in state rather than on a wire code, because the refusal Jamf answers a
// cross-organization delete with was never probed — probing it would have meant a
// destructive request against another organization's domain.
func TestDelete_SharedDomainIsRefusedBeforeAnyRequest(t *testing.T) {
	ctx := context.Background()
	var calls []string

	r, s, _ := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})

	state := sharedDomainState(ctx, t, s, sharedFixtureDomain, "26919", true)
	resp := resource.DeleteResponse{State: tfsdk.State{Schema: s, Raw: state}}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: state}}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("withdrawing a shared domain must be refused, not attempted")
	}
	if len(calls) != 0 {
		t.Errorf("delete must issue nothing for a shared domain, issued %v", calls)
	}
	err := resp.Diagnostics.Errors()[0]
	if err.Summary() != "Shared Jamf Account SSO domain cannot be withdrawn" {
		t.Errorf("summary = %q", err.Summary())
	}
	if !strings.Contains(err.Detail(), "terraform state rm") {
		t.Errorf("detail %q does not name the remedy", err.Detail())
	}
}

// TestDelete_OwnedDomainStillWithdraws pins that the ownership gate did not
// break the ordinary destroy.
func TestDelete_OwnedDomainStillWithdraws(t *testing.T) {
	ctx := context.Background()
	var calls []string

	r, s, _ := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})

	state := sharedDomainState(ctx, t, s, ownedFixtureDomain, "26917", false)
	resp := resource.DeleteResponse{State: tfsdk.State{Schema: s, Raw: state}}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: state}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", resp.Diagnostics)
	}
	if len(calls) != 1 || calls[0] != "DELETE /sso/v1/domains/26917" {
		t.Errorf("delete issued %v, want one withdrawal keyed on the identifier", calls)
	}
}

// TestList_LimitCountsKeptResultsNotScannedDomains pins the interaction between
// the ownership filter and req.Limit, which the two-entry fixtures cannot reach.
// The limit is clamped against the number of domains returned, which is an upper
// bound on the number kept, and the loop counts kept entries — so a limit of five
// against ten domains of which six are shared has to yield all four owned ones
// rather than stopping early on the shared prefix.
func TestList_LimitCountsKeptResultsNotScannedDomains(t *testing.T) {
	entries := make([]string, 0, 10)
	for i := range 6 {
		entries = append(entries, sharedDomainCollectionEntry(
			fmt.Sprintf("90%02d", i), fmt.Sprintf("shared-%d.example", i), sharedFixtureAccountID, true))
	}
	for i := range 4 {
		entries = append(entries, sharedDomainCollectionEntry(
			fmt.Sprintf("10%02d", i), fmt.Sprintf("owned-%d.example", i), "001ABCDEFGHIJKLMNO", false))
	}
	body := fmt.Sprintf(`{"totalCount":%d,"results":[%s]}`, len(entries), strings.Join(entries, ","))

	got := displayNames(listDomains(t, body, 5))
	if len(got) != 4 {
		t.Fatalf("streamed %d results (%v), want all 4 owned domains", len(got), got)
	}
	for _, name := range got {
		if strings.HasPrefix(name, "shared-") {
			t.Errorf("streamed a shared domain: %q", name)
		}
	}
}

// sharedDomainCollectionEntry renders one collection element, varying only the
// fields the ownership gate turns on. The two-entry fixtures are consts; this
// exists for the case that needs ten of them.
func sharedDomainCollectionEntry(id, domain, accountID string, shared bool) string {
	return fmt.Sprintf(`{
		"id": %q,
		"createdByName": null,
		"accountId": %q,
		"domain": %q,
		"verificationKey": "verification-key-%s",
		"domainStatus": "VERIFIED",
		"createdDate": "2026-09-02T12:33:32.658Z",
		"lastModifiedDate": "2026-09-02T12:33:32.658Z",
		"lastVerificationDate": "2026-09-02T12:35:00.000Z",
		"verificationExpirationDate": "2026-09-16T12:33:32.658Z",
		"sharedDomain": %t,
		"verifiedTldId": null
	}`, id, accountID, domain, id, shared)
}
