// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/account"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// stubHandler is one recorded exchange the stub server should serve.
type stubHandler func(w http.ResponseWriter, r *http.Request)

// newStubClient returns a Jamf Account client pointed at a stub server driven by
// handle.
//
// The seam is the HTTP boundary rather than an injected interface, matching the
// sibling SSO domain package: the CRUD methods hold a concrete *account.Client,
// and an interface introduced only for a test would be a bigger change than the
// behaviour it pins. The stub is local rather than testhelpers.NewMockClient
// because testhelpers reaches the provider package under the acceptance build
// tag, and this package is one the provider registers — importing it from an
// in-package test makes that a cycle.
func newStubClient(t *testing.T, handle stubHandler) *account.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "test-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		}
		handle(w, r)
	}))
	t.Cleanup(server.Close)
	return account.New(jamfplatform.NewClient(server.URL, "test-id", "test-secret", jamfplatform.WithRetryPolicy(0, 0, 0)))
}

// Fixtures below carry deliberately low-entropy, role-named values. A
// realistic-looking client secret or identifier in a checked-in fixture trips the
// repository's secret-scanning gate, which is the only workflow that runs on this
// base branch, and an exemption file keys its fingerprints to the introducing
// commit and breaks on the next rebase.
const (
	unitConnectionID   = "con_unittest0001"
	unitConnectionName = "tf-unit-oidc"
)

// oidcConnectionBody is the single-connection read for a generic OpenID Connect
// connection: the per-provider settings and nothing about products.
const oidcConnectionBody = `{
	"id": "` + unitConnectionID + `",
	"name": "` + unitConnectionName + `",
	"type": "OIDC",
	"region": "US",
	"clientId": "probe-client-id",
	"consentFlow": false,
	"easyConfig": false,
	"domains": ["tf-unit.example"],
	"scopes": "openid email profile",
	"pkceAuthType": "DISABLED",
	"tokenEndpointAuthMethod": "CLIENT_SECRET_POST",
	"sendNonce": false,
	"syncUserProfileAttributesAtLogin": true,
	"aliasLoginHintToIdp": true,
	"customUsernameClaimName": null,
	"usernameDomain": null,
	"attributeMap": "{\"mapping_mode\":\"bind_all\"}",
	"groupNameFilter": "{\"op\":\"OR\",\"groups\":\"\"}",
	"sessionInfo": {"maxSessionTimeInMinutes": null, "maxInactivityTimeInMinutes": null},
	"oidcOptions": {
		"issuerUrl": "idp.example",
		"authorizationEndpoint": "idp.example/authorize",
		"tokenEndpoint": "idp.example/token",
		"jwksUri": "idp.example/keys",
		"userInfoEndpoint": null
	},
	"azureOptions": null,
	"oktaOptions": null,
	"googleOptions": null
}`

// consentFlowConnectionBody is a connection built through Microsoft's
// admin-consent flow: it reads back cleanly, has no client of its own, and
// cannot be written again.
const consentFlowConnectionBody = `{
	"id": "con_unittest0002",
	"name": "tf-unit-consent",
	"type": "WAAD",
	"region": "US",
	"clientId": null,
	"consentFlow": true,
	"easyConfig": false,
	"domains": ["tf-unit.example"],
	"pkceAuthType": "DISABLED",
	"tokenEndpointAuthMethod": "CLIENT_SECRET_POST",
	"sendNonce": false,
	"syncUserProfileAttributesAtLogin": true,
	"aliasLoginHintToIdp": true,
	"attributeMap": "{\"mapping_mode\":\"basic_profile\"}",
	"sessionInfo": {"maxSessionTimeInMinutes": null, "maxInactivityTimeInMinutes": null},
	"azureOptions": {
		"domain": "contoso.example",
		"tenantDomain": "contoso.example",
		"basicProfile": true,
		"identityApi": "MICROSOFT_IDENTITY_PLATFORM_V2",
		"maxGroups": 250,
		"setEmailsVerified": true,
		"enableUsersApi": false,
		"useCommonEndpoint": false,
		"useWsfed": false,
		"extOptions": {"extendedProfile": false, "groups": false, "nestedGroups": false}
	}
}`

// oidcSummaryBody is the collection entry for the same connection: the products
// and the consent ticket, and nothing about the provider.
const oidcSummaryBody = `{
	"id": "` + unitConnectionID + `",
	"name": "` + unitConnectionName + `",
	"type": "OIDC",
	"region": "US",
	"domains": ["tf-unit.example"],
	"enabledApplications": ["ACCOUNT", "PRO"],
	"easyConfig": false,
	"syncUserProfileAttributesAtLogin": true,
	"ticketUrl": null,
	"tokenEndpointAuthMethod": "CLIENT_SECRET_POST"
}`

// consentFlowSummaryBody is the collection entry for the admin-consent
// connection. Note it carries no consent flag — which is exactly why the list
// resource has to read each connection individually to find one.
const consentFlowSummaryBody = `{
	"id": "con_unittest0002",
	"name": "tf-unit-consent",
	"type": "WAAD",
	"region": "US",
	"domains": ["tf-unit.example"],
	"enabledApplications": ["ACCOUNT"],
	"easyConfig": false,
	"syncUserProfileAttributesAtLogin": true,
	"ticketUrl": null,
	"tokenEndpointAuthMethod": "CLIENT_SECRET_POST"
}`

// ghostSummaryBody is the collection entry for a connection Jamf lists but
// cannot read on its own identifier.
const ghostSummaryBody = `{
	"id": "con_unittest0003",
	"name": "tf-unit-ghost",
	"type": "OIDC",
	"region": "AU",
	"domains": [],
	"enabledApplications": [],
	"easyConfig": false,
	"syncUserProfileAttributesAtLogin": true,
	"ticketUrl": null,
	"tokenEndpointAuthMethod": "CLIENT_SECRET_POST"
}`

// Collection envelopes assembled from the entries above.
const (
	oneConnectionListBody   = `{"totalCount":1,"results":[` + oidcSummaryBody + `]}`
	emptyConnectionListBody = `{"totalCount":0,"results":[]}`
	ghostOnlyListBody       = `{"totalCount":1,"results":[` + ghostSummaryBody + `]}`
	mixedConnectionListBody = `{"totalCount":3,"results":[` + oidcSummaryBody + `,` + consentFlowSummaryBody + `,` + ghostSummaryBody + `]}`
)

// upstreamErrorBody is the refusal every connection write currently answers with.
const upstreamErrorBody = `{"errors":[{"code":"UPSTREAM_ERROR","field":null,"description":"The request could not be completed"}],"httpStatus":500,"traceId":"unit-trace-0001"}`

// notFoundBody is Jamf's not-found refusal on a single connection.
const notFoundBody = `{"errors":[{"code":"NOT_FOUND","field":null,"description":"No configured connections found for ID"}],"httpStatus":404}`

// resourceUnderTest wires a stub client into the resource and returns it with
// its schemas, so each test states only the exchange it cares about.
func resourceUnderTest(t *testing.T, handle stubHandler) (*ConnectionResource, resourceschema.Schema, *tfsdk.ResourceIdentity) {
	t.Helper()
	ctx := context.Background()
	r := &ConnectionResource{client: newStubClient(t, handle)}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}
	var identityResp resource.IdentitySchemaResponse
	r.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)

	return r, schemaResp.Schema, &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema}
}

// connectionSetter mutates one attribute of a resource object under
// construction.
type connectionSetter func(ctx context.Context, t *testing.T, state *tfsdk.State)

// connectionValue builds a resource object with every attribute empty, then
// applies each setter.
//
// Assembling it through the framework's own attribute writer rather than by hand
// keeps the nested settings blocks honest: a renamed attribute inside one shows up
// as a failure here instead of as a silently-dropped value.
func connectionValue(ctx context.Context, t *testing.T, s resourceschema.Schema, setters ...connectionSetter) tftypes.Value {
	t.Helper()
	object := s.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(object, values)}
	for _, set := range setters {
		set(ctx, t, &state)
	}
	return state.Raw
}

// withOIDCConfiguration sets the attributes a minimal generic OpenID Connect
// connection is configured with.
func withOIDCConfiguration() connectionSetter {
	return func(ctx context.Context, t *testing.T, state *tfsdk.State) {
		t.Helper()
		setAttribute(ctx, t, state, path.Root("name"), unitConnectionName)
		setAttribute(ctx, t, state, path.Root("connection_type"), connectionTypeGenericOIDC)
		setAttribute(ctx, t, state, path.Root("hosting_region"), account.RegionUs)
		setAttribute(ctx, t, state, path.Root("scopes"), "openid email profile")
		setAttribute(ctx, t, state, path.Root("client_id"), "probe-client-id")
		setAttribute(ctx, t, state, path.Root("domains"), []string{"tf-unit.example"})
		setAttribute(ctx, t, state, path.Root("generic_oidc"), &GenericOIDCSettingsModel{
			IssuerURL:             types.StringValue("idp.example"),
			AuthorizationEndpoint: types.StringValue("idp.example/authorize"),
			TokenEndpoint:         types.StringValue("idp.example/token"),
			JWKSURI:               types.StringValue("idp.example/keys"),
			UserInfoEndpoint:      types.StringNull(),
		})
	}
}

// withID records the identifier a refresh or a destroy runs against.
func withID(id string) connectionSetter {
	return func(ctx context.Context, t *testing.T, state *tfsdk.State) {
		t.Helper()
		setAttribute(ctx, t, state, path.Root("id"), id)
	}
}

// withConsentFlow records a connection already in state as using Microsoft
// admin consent, which is the state an earlier provider version could have
// written.
func withConsentFlow() connectionSetter {
	return func(ctx context.Context, t *testing.T, state *tfsdk.State) {
		t.Helper()
		setAttribute(ctx, t, state, path.Root("consent_flow"), true)
		setAttribute(ctx, t, state, path.Root("display_name"), "tf-unit-consent")
	}
}

// withGroupNameFilter declares the group filter, so the read has a block to
// refresh into.
func withGroupNameFilter() connectionSetter {
	return func(ctx context.Context, t *testing.T, state *tfsdk.State) {
		t.Helper()
		groups, diags := types.SetValue(types.StringType, nil)
		if diags.HasError() {
			t.Fatalf("building an empty group set: %v", diags)
		}
		setAttribute(ctx, t, state, path.Root("group_name_filter"), &GroupNameFilterModel{
			Operator: types.StringValue(filterOperatorOr),
			Groups:   groups,
		})
	}
}

// withClientSecret records the write-only client secret, which reaches Create
// and Update through the configuration rather than the plan.
func withClientSecret(secret string) connectionSetter {
	return func(ctx context.Context, t *testing.T, state *tfsdk.State) {
		t.Helper()
		setAttribute(ctx, t, state, path.Root("client_secret"), secret)
	}
}

// setAttribute writes one attribute, failing the test on a diagnostic rather
// than leaving a half-built object behind.
func setAttribute(ctx context.Context, t *testing.T, state *tfsdk.State, at path.Path, value any) {
	t.Helper()
	if diags := state.SetAttribute(ctx, at, value); diags.HasError() {
		t.Fatalf("setting %s: %v", at, diags)
	}
}

// TestCreate_SendsTheSettingsAndReadsBackTheProducts pins the two calls a create
// takes and why the second is not optional: the create response carries the
// per-provider settings and no products, so the enabled products would be
// unknown at the end of the apply without the collection read.
func TestCreate_SendsTheSettingsAndReadsBackTheProducts(t *testing.T) {
	ctx := context.Background()
	var calls []string
	var body map[string]any

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodPost {
			_ = json.NewDecoder(req.Body).Decode(&body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(oidcConnectionBody))
			return
		}
		_, _ = w.Write([]byte(oneConnectionListBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withClientSecret("probe-client-secret"))
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Create(ctx, resource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: s, Raw: raw},
		Config: tfsdk.Config{Schema: s, Raw: raw},
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", resp.Diagnostics)
	}
	want := []string{"POST /sso/v1/connections", "GET /sso/v1/connections"}
	if len(calls) != len(want) || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("create issued %v, want %v", calls, want)
	}

	if got := body["connectionType"]; got != account.ConnectionTypeOidc {
		t.Errorf("connectionType = %v, want the renamed value translated back", got)
	}
	settings, ok := body["connection"].(map[string]any)
	if !ok {
		t.Fatalf("connection = %v, want the settings object", body["connection"])
	}
	if got := settings["clientSecret"]; got != "probe-client-secret" {
		t.Errorf("clientSecret = %v, want the write-only value read from the configuration", got)
	}
	if got := settings["issuerUrl"]; got != "idp.example" {
		t.Errorf("issuerUrl = %v, want the value from the settings block", got)
	}

	var state ConnectionResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.ID.ValueString() != unitConnectionID {
		t.Errorf("id = %q", state.ID.ValueString())
	}
	if len(state.EnabledProductNames.Elements()) != 2 {
		t.Errorf("enabled_product_names = %s, want the products from the collection read", state.EnabledProductNames)
	}
	if !resp.State.Raw.IsFullyKnown() {
		t.Errorf("state must be wholly known after create, got %s", resp.State.Raw)
	}
}

// TestCreate_UpstreamFailureBlamesJamf pins the diagnostic for the refusal every
// write currently takes. It is proven to happen before Jamf inspects the request,
// so a diagnostic that sent an operator back to their configuration would send
// them somewhere the fix cannot be.
func TestCreate_UpstreamFailureBlamesJamf(t *testing.T) {
	ctx := context.Background()

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(upstreamErrorBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration())
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Create(ctx, resource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: s, Raw: raw},
		Config: tfsdk.Config{Schema: s, Raw: raw},
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a refused create must be reported as an error")
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a refused create must write no state")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "known fault on Jamf's side") {
		t.Errorf("detail %q does not say the fault is Jamf's", detail)
	}
	if !strings.Contains(detail, "unit-trace-0001") {
		t.Errorf("detail %q does not name the trace identifier to quote", detail)
	}
}

// TestRead_TakesBothCalls pins that a refresh reads the connection and the
// collection, because neither alone is a whole connection.
func TestRead_TakesBothCalls(t *testing.T) {
	ctx := context.Background()
	var calls []string

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/connections") {
			_, _ = w.Write([]byte(oneConnectionListBody))
			return
		}
		_, _ = w.Write([]byte(oidcConnectionBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID(unitConnectionID))
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	want := []string{"GET /sso/v1/connections/" + unitConnectionID, "GET /sso/v1/connections"}
	if len(calls) != len(want) || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("read issued %v, want %v", calls, want)
	}

	var state ConnectionResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.DisplayName.ValueString() != unitConnectionName {
		t.Errorf("display_name = %q, want the stored name", state.DisplayName.ValueString())
	}
	if state.Name.ValueString() != unitConnectionName {
		t.Errorf("name = %q, want the configured name left alone", state.Name.ValueString())
	}
	if state.GenericOIDC == nil || state.GenericOIDC.IssuerURL.ValueString() != "idp.example" {
		t.Errorf("generic_oidc = %+v, want the settings block refreshed", state.GenericOIDC)
	}
	if state.GroupNameFilter != nil {
		t.Errorf("group_name_filter = %+v, want a block the configuration never declared left alone", state.GroupNameFilter)
	}
}

// TestRead_DeclaredGroupFilterKeepsAnEmptyGroupList pins the distinction the
// block exists to preserve. Jamf's own copy of "no filtering" is an operator with
// an empty group list, which is a different value from the field being absent, so
// reading one back must not collapse it to nothing.
func TestRead_DeclaredGroupFilterKeepsAnEmptyGroupList(t *testing.T) {
	ctx := context.Background()

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/connections") {
			_, _ = w.Write([]byte(oneConnectionListBody))
			return
		}
		_, _ = w.Write([]byte(oidcConnectionBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID(unitConnectionID), withGroupNameFilter())
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var state ConnectionResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.GroupNameFilter == nil {
		t.Fatal("a declared group filter must be refreshed, not dropped")
	}
	if got := state.GroupNameFilter.Operator.ValueString(); got != filterOperatorOr {
		t.Errorf("operator = %q, want the renamed value", got)
	}
	if state.GroupNameFilter.Groups.IsNull() || len(state.GroupNameFilter.Groups.Elements()) != 0 {
		t.Errorf("groups = %s, want a known empty set rather than nothing", state.GroupNameFilter.Groups)
	}
}

// TestRead_ConfiguredNameSurvivesAUniquifiedStoredName is the point of splitting
// the name in two. Jamf may store a suffixed form of the configured name, and
// overwriting the configured value with it would give such a connection a
// difference on every plan for ever.
func TestRead_ConfiguredNameSurvivesAUniquifiedStoredName(t *testing.T) {
	ctx := context.Background()
	suffixed := strings.Replace(oidcConnectionBody, `"name": "`+unitConnectionName+`"`, `"name": "`+unitConnectionName+`-uniquified"`, 1)
	suffixedList := `{"totalCount":1,"results":[` +
		strings.Replace(oidcSummaryBody, `"name": "`+unitConnectionName+`"`, `"name": "`+unitConnectionName+`-uniquified"`, 1) + `]}`

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/connections") {
			_, _ = w.Write([]byte(suffixedList))
			return
		}
		_, _ = w.Write([]byte(suffixed))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID(unitConnectionID))
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var state ConnectionResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.Name.ValueString() != unitConnectionName {
		t.Errorf("name = %q, want the configured name untouched", state.Name.ValueString())
	}
	if state.DisplayName.ValueString() != unitConnectionName+"-uniquified" {
		t.Errorf("display_name = %q, want the stored name", state.DisplayName.ValueString())
	}
}

// TestRead_ImportAdoptsTheStoredName covers the one case where the stored name is
// the right thing to take: an import leaves no configured name to protect.
// Detection keys on a Required attribute being absent, because the raw state is
// not empty after an import.
func TestRead_ImportAdoptsTheStoredName(t *testing.T) {
	ctx := context.Background()

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/connections") {
			_, _ = w.Write([]byte(oneConnectionListBody))
			return
		}
		_, _ = w.Write([]byte(oidcConnectionBody))
	})

	raw := connectionValue(ctx, t, s, withID(unitConnectionID))
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var state ConnectionResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("reading back the state: %v", diags)
	}
	if state.Name.ValueString() != unitConnectionName {
		t.Errorf("name = %q, want the stored name adopted on import", state.Name.ValueString())
	}
	if state.GenericOIDC == nil {
		t.Error("an import must fill the settings block, which no plan said anything about")
	}
	if state.GroupNameFilter == nil {
		t.Error("an import must fill the group filter, which no plan said anything about")
	}
}

// TestRead_RemovedConnectionIsRemovedFromState pins ordinary drift recovery: a
// connection gone from Jamf's own collection as well as from the single read is
// genuinely gone.
func TestRead_RemovedConnectionIsRemovedFromState(t *testing.T) {
	ctx := context.Background()

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/connections") {
			_, _ = w.Write([]byte(emptyConnectionListBody))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(notFoundBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID(unitConnectionID))
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: raw}, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a removed connection must be dropped from state so the next plan makes it again")
	}
}

// TestRead_GhostConnectionStaysInState is the counterpart, and the one that must
// not be got wrong. A connection Jamf's collection still lists exists, so a
// not-found on its own identifier is a disagreement inside Jamf and not a
// withdrawal — dropping it would plan a create of something already there.
func TestRead_GhostConnectionStaysInState(t *testing.T) {
	ctx := context.Background()

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/connections") {
			_, _ = w.Write([]byte(ghostOnlyListBody))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(notFoundBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID("con_unittest0003"))
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: raw}, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a connection Jamf lists but cannot read must be reported")
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("a connection Jamf lists but cannot read must stay in state — it exists")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "con_unittest0003") {
		t.Errorf("detail %q does not name the identifier to raise", detail)
	}
	if !strings.Contains(detail, "duplicate") {
		t.Errorf("detail %q does not say why it was kept", detail)
	}
}

// TestRead_ConsentFlowConnectionIsRefused pins the import guard. Such a
// connection reads back cleanly and looks ordinary, so the refusal has to land
// where it is first seen rather than where an apply would first fail.
func TestRead_ConsentFlowConnectionIsRefused(t *testing.T) {
	ctx := context.Background()
	var calls []string

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(consentFlowConnectionBody))
	})

	raw := connectionValue(ctx, t, s, withID("con_unittest0002"))
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: raw}, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a connection using Microsoft admin consent must be refused, not imported")
	}
	if len(calls) != 1 {
		t.Errorf("read issued %v, want the refusal before the collection read", calls)
	}
	if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "terraform state rm") {
		t.Errorf("detail %q does not name the way out", detail)
	}
}

// TestUpdate_SendsEverythingAndReadsBackTheProducts pins the replacement
// semantics at the call level: one write carrying the whole settings, then the
// collection read the write response cannot supply.
func TestUpdate_SendsEverythingAndReadsBackTheProducts(t *testing.T) {
	ctx := context.Background()
	var calls []string
	var body map[string]any

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodPut {
			_ = json.NewDecoder(req.Body).Decode(&body)
			_, _ = w.Write([]byte(oidcConnectionBody))
			return
		}
		_, _ = w.Write([]byte(oneConnectionListBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID(unitConnectionID))
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Update(ctx, resource.UpdateRequest{
		Plan:   tfsdk.Plan{Schema: s, Raw: raw},
		State:  tfsdk.State{Schema: s, Raw: raw},
		Config: tfsdk.Config{Schema: s, Raw: raw},
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", resp.Diagnostics)
	}
	want := []string{"PUT /sso/v1/connections/" + unitConnectionID, "GET /sso/v1/connections"}
	if len(calls) != len(want) || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("update issued %v, want %v", calls, want)
	}
	settings, ok := body["connection"].(map[string]any)
	if !ok {
		t.Fatalf("connection = %v, want the whole settings object", body["connection"])
	}
	for _, field := range []string{"issuerUrl", "authorizationEndpoint", "tokenEndpoint", "jwksUri", "name", "region"} {
		if _, present := settings[field]; !present {
			t.Errorf("update omitted %q — a replacement has to carry the whole settings", field)
		}
	}
}

// TestUpdate_OmittedClientSecretIsNotSent pins the one documented exception to
// the replacement: leaving the secret out keeps the stored one, so the field has
// to be absent from the body rather than present and empty.
func TestUpdate_OmittedClientSecretIsNotSent(t *testing.T) {
	ctx := context.Background()
	var body map[string]any

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodPut {
			_ = json.NewDecoder(req.Body).Decode(&body)
			_, _ = w.Write([]byte(oidcConnectionBody))
			return
		}
		_, _ = w.Write([]byte(oneConnectionListBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID(unitConnectionID))
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Update(ctx, resource.UpdateRequest{
		Plan:   tfsdk.Plan{Schema: s, Raw: raw},
		State:  tfsdk.State{Schema: s, Raw: raw},
		Config: tfsdk.Config{Schema: s, Raw: raw},
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", resp.Diagnostics)
	}
	settings, ok := body["connection"].(map[string]any)
	if !ok {
		t.Fatalf("connection = %v", body["connection"])
	}
	if _, present := settings["clientSecret"]; present {
		t.Error("an omitted client secret must be absent from the body, not present and empty")
	}
}

// TestUpdate_ConsentFlowConnectionIsRefusedBeforeAnyRequest pins that the
// refusal is keyed on state rather than on how Jamf answers. Every write is
// currently refused identically whatever the reason, so a wire-keyed check could
// not tell this apart from the fault affecting every connection.
func TestUpdate_ConsentFlowConnectionIsRefusedBeforeAnyRequest(t *testing.T) {
	ctx := context.Background()
	var calls []string

	r, s, identity := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(oidcConnectionBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID("con_unittest0002"), withConsentFlow())
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}, Identity: identity}
	r.Update(ctx, resource.UpdateRequest{
		Plan:   tfsdk.Plan{Schema: s, Raw: raw},
		State:  tfsdk.State{Schema: s, Raw: raw},
		Config: tfsdk.Config{Schema: s, Raw: raw},
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("changing a connection using Microsoft admin consent must be refused")
	}
	if len(calls) != 0 {
		t.Errorf("update issued %v, want nothing sent", calls)
	}
}

// TestDelete_TreatsAnAlreadyRemovedConnectionAsSuccess pins the not-found branch
// STYLE_GUIDE §Delete semantics requires of every delete: the transport neither
// retries nor swallows a not-found, so a connection already gone has to be
// recognised here.
func TestDelete_TreatsAnAlreadyRemovedConnectionAsSuccess(t *testing.T) {
	ctx := context.Background()
	var calls []string

	r, s, _ := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(notFoundBody))
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID(unitConnectionID))
	resp := resource.DeleteResponse{State: tfsdk.State{Schema: s, Raw: raw}}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("an already-removed connection must not fail the destroy: %v", resp.Diagnostics)
	}
	if len(calls) != 1 || calls[0] != "DELETE /sso/v1/connections/"+unitConnectionID {
		t.Errorf("delete issued %v, want one call keyed on the identifier", calls)
	}
}

// TestDelete_ConsentFlowConnectionIsAllowed pins the asymmetry with Read and
// Update. Removal works on such a connection, and refusing it would leave an
// operator who adopted one under an earlier provider version unable to get rid
// of it except by editing state by hand.
func TestDelete_ConsentFlowConnectionIsAllowed(t *testing.T) {
	ctx := context.Background()
	var calls []string

	r, s, _ := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration(), withID("con_unittest0002"), withConsentFlow())
	resp := resource.DeleteResponse{State: tfsdk.State{Schema: s, Raw: raw}}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("destroying a connection using Microsoft admin consent must be allowed: %v", resp.Diagnostics)
	}
	if len(calls) != 1 || calls[0] != "DELETE /sso/v1/connections/con_unittest0002" {
		t.Errorf("delete issued %v, want the withdrawal to go through", calls)
	}
}

// TestDelete_WithoutAnIdentifierSaysWhatToDo covers the state a partly-applied
// import can leave: no identifier recorded, and removing a connection needs one.
func TestDelete_WithoutAnIdentifierSaysWhatToDo(t *testing.T) {
	ctx := context.Background()
	var calls []string

	r, s, _ := resourceUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})

	raw := connectionValue(ctx, t, s, withOIDCConfiguration())
	resp := resource.DeleteResponse{State: tfsdk.State{Schema: s, Raw: raw}}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a destroy with no identifier recorded must be reported, not silently skipped")
	}
	if len(calls) != 0 {
		t.Errorf("delete must issue nothing without an identifier, issued %v", calls)
	}
	if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "-refresh-only") {
		t.Errorf("detail %q does not name the remedy", detail)
	}
}
