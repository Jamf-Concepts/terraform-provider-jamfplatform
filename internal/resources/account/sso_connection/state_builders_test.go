// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/account"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// oidcConnectionRead is a generic OpenID Connect connection as Jamf returns it.
func oidcConnectionRead() *account.Connection {
	return &account.Connection{
		ID:                               unitConnectionID,
		Name:                             unitConnectionName,
		Type:                             new(account.ConnectionTypeOidc),
		Region:                           new(account.RegionUs),
		ClientID:                         new("probe-client-id"),
		Domains:                          []string{"tf-unit.example"},
		Scopes:                           new("openid email profile"),
		PkceAuthType:                     new(account.PkceAuthTypeDisabled),
		TokenEndpointAuthMethod:          new(account.TokenEndpointAuthMethodClientSecretPost),
		SendNonce:                        false,
		SyncUserProfileAttributesAtLogin: true,
		AliasLoginHintToIdp:              true,
		AttributeMap:                     new(`{"mapping_mode":"bind_all"}`),
		SessionInfo:                      &account.SessionInfo{},
		OidcOptions: &account.OidcOptions{
			IssuerURL:             new("idp.example"),
			AuthorizationEndpoint: new("idp.example/authorize"),
			TokenEndpoint:         new("idp.example/token"),
			JwksUri:               new("idp.example/keys"),
		},
	}
}

// oidcSummaryRead is the collection entry for the same connection.
func oidcSummaryRead() *account.ConnectionSummary {
	return &account.ConnectionSummary{
		ID:                  unitConnectionID,
		Name:                unitConnectionName,
		Type:                new(account.ConnectionTypeOidc),
		Region:              new(account.RegionUs),
		Domains:             []string{"tf-unit.example"},
		EnabledApplications: []string{account.ProductAccount, account.ProductPro},
	}
}

// TestAssignConnectionResourceModel_TranslatesTheRenamedVocabularies pins the
// read side of every rename. A rename applied on the way out and not undone on
// the way in reads back as Jamf's spelling and gives a difference on every plan.
func TestAssignConnectionResourceModel_TranslatesTheRenamedVocabularies(t *testing.T) {
	var state ConnectionResourceModel
	if diags := assignConnectionResourceModel(&state, oidcConnectionRead(), oidcSummaryRead(), true); diags.HasError() {
		t.Fatalf("assigning state: %v", diags)
	}

	if got := state.ConnectionType.ValueString(); got != connectionTypeGenericOIDC {
		t.Errorf("connection_type = %q, want the console's own name", got)
	}
	if got := state.AuthMethod.ValueString(); got != authMethodClientSecret {
		t.Errorf("auth_method = %q, want the console's own name", got)
	}
	if got := state.PKCE.ValueString(); got != pkceDisabled {
		t.Errorf("pkce = %q, want the console's own name", got)
	}
	if got := state.HostingRegion.ValueString(); got != account.RegionUs {
		t.Errorf("hosting_region = %q, want Jamf's own value kept", got)
	}
}

// TestAssignConnectionResourceModel_InvertsTheLoginHint is the other half of the
// pair that matters most in this package. Together with the input-builder test
// this rules out the failure neither test alone could: a pair that inverts twice
// looks exactly like one that never inverts.
func TestAssignConnectionResourceModel_InvertsTheLoginHint(t *testing.T) {
	for name, tc := range map[string]struct {
		aliasToIdp bool
		wantOmit   bool
	}{
		"forwarded": {true, false},
		"withheld":  {false, true},
	} {
		t.Run(name, func(t *testing.T) {
			read := oidcConnectionRead()
			read.AliasLoginHintToIdp = tc.aliasToIdp

			var state ConnectionResourceModel
			if diags := assignConnectionResourceModel(&state, read, oidcSummaryRead(), true); diags.HasError() {
				t.Fatalf("assigning state: %v", diags)
			}
			if got := state.OmitLoginHint.ValueBool(); got != tc.wantOmit {
				t.Errorf("omit_login_hint = %v, want %v — the two senses are opposites", got, tc.wantOmit)
			}
		})
	}
}

// TestOmitLoginHintRoundTrips is the direct statement of the same thing, and the
// one a reader can check at a glance: whatever goes in has to come back.
func TestOmitLoginHintRoundTrips(t *testing.T) {
	for _, omit := range []bool{true, false} {
		wire := omitLoginHintToWire(types.BoolValue(omit))
		if got := omitLoginHintFromWire(wire).ValueBool(); got != omit {
			t.Errorf("omit_login_hint %v round-tripped to %v through %v", omit, got, wire)
		}
		if wire == omit {
			t.Errorf("omit_login_hint %v was sent as %v — the two senses must be opposites", omit, wire)
		}
	}
}

// TestAssignConnectionResourceModel_KeepsTheConfiguredName pins the reason `name`
// and `display_name` are two attributes. Eighteen of the twenty-two connections
// read carry a suffix Jamf added, and overwriting the configured value with the
// stored one would give every such connection a difference on every plan.
func TestAssignConnectionResourceModel_KeepsTheConfiguredName(t *testing.T) {
	read := oidcConnectionRead()
	read.Name = unitConnectionName + "-uniquified"

	state := ConnectionResourceModel{Name: types.StringValue(unitConnectionName)}
	if diags := assignConnectionResourceModel(&state, read, oidcSummaryRead(), false); diags.HasError() {
		t.Fatalf("assigning state: %v", diags)
	}

	if got := state.Name.ValueString(); got != unitConnectionName {
		t.Errorf("name = %q, want the configured value untouched", got)
	}
	if got := state.DisplayName.ValueString(); got != unitConnectionName+"-uniquified" {
		t.Errorf("display_name = %q, want the stored value", got)
	}
}

// TestAssignConnectionResourceModel_AdoptionTakesTheStoredName covers the one
// case where the stored name is the right thing to take: an import has no
// configured value to protect.
func TestAssignConnectionResourceModel_AdoptionTakesTheStoredName(t *testing.T) {
	read := oidcConnectionRead()
	read.Name = unitConnectionName + "-uniquified"

	var state ConnectionResourceModel
	if diags := assignConnectionResourceModel(&state, read, oidcSummaryRead(), true); diags.HasError() {
		t.Fatalf("assigning state: %v", diags)
	}

	if got := state.Name.ValueString(); got != unitConnectionName+"-uniquified" {
		t.Errorf("name = %q, want the stored value adopted on an import", got)
	}
}

// TestAssignConnectionResourceModel_GatesBlocksOnTheTargetModel pins
// STYLE_GUIDE §`SingleNestedAttribute` blocks: a block absent from the model
// being written stays absent, however much Jamf returned for it. Populating one
// the plan said was empty breaks the framework's consistency contract, and it
// fails only under a real apply.
func TestAssignConnectionResourceModel_GatesBlocksOnTheTargetModel(t *testing.T) {
	read := oidcConnectionRead()
	read.GroupNameFilter = new(`{"op":"OR","groups":"jamf"}`)

	var state ConnectionResourceModel
	if diags := assignConnectionResourceModel(&state, read, oidcSummaryRead(), false); diags.HasError() {
		t.Fatalf("assigning state: %v", diags)
	}

	if state.GenericOIDC != nil {
		t.Error("a settings block the model did not carry must stay absent on a refresh")
	}
	if state.GroupNameFilter != nil {
		t.Error("a group filter the model did not carry must stay absent on a refresh")
	}

	state = ConnectionResourceModel{
		Name:            types.StringValue(unitConnectionName),
		GenericOIDC:     &GenericOIDCSettingsModel{},
		GroupNameFilter: &GroupNameFilterModel{},
	}
	if diags := assignConnectionResourceModel(&state, read, oidcSummaryRead(), false); diags.HasError() {
		t.Fatalf("assigning state: %v", diags)
	}
	if state.GenericOIDC == nil || state.GenericOIDC.IssuerURL.ValueString() != "idp.example" {
		t.Errorf("generic_oidc = %+v, want a block the model carried refreshed", state.GenericOIDC)
	}
	if state.GroupNameFilter == nil || len(state.GroupNameFilter.Groups.Elements()) != 1 {
		t.Errorf("group_name_filter = %+v, want a filter the model carried refreshed", state.GroupNameFilter)
	}
}

// TestAssignConnectionResourceModel_LeavesTheProductAssignmentsAlone pins the
// configuration-authoritative collections. Nothing Jamf returns echoes the
// tenants, so adopting anything would mean inventing it — and overwriting a
// configured value with a guess is worse than leaving it.
func TestAssignConnectionResourceModel_LeavesTheProductAssignmentsAlone(t *testing.T) {
	state := ConnectionResourceModel{
		Name: types.StringValue(unitConnectionName),
		EnabledProducts: []EnabledProductModel{{
			Product: types.StringValue(account.ProductPro),
			Tenants: mustStringSet("tenant-one"),
		}},
		EnabledEnvironments: []EnabledEnvironmentModel{{
			Product:      types.StringValue(account.ProductProtect),
			Environments: mustStringSet("environment-one"),
		}},
	}

	if diags := assignConnectionResourceModel(&state, oidcConnectionRead(), oidcSummaryRead(), false); diags.HasError() {
		t.Fatalf("assigning state: %v", diags)
	}

	if len(state.EnabledProducts) != 1 || state.EnabledProducts[0].Product.ValueString() != account.ProductPro {
		t.Errorf("enabled_products = %+v, want the configured value untouched", state.EnabledProducts)
	}
	if len(state.EnabledEnvironments) != 1 {
		t.Errorf("enabled_environments = %+v, want the configured value untouched", state.EnabledEnvironments)
	}
	if len(state.EnabledProductNames.Elements()) != 2 {
		t.Errorf("enabled_product_names = %s, want the products the collection reports", state.EnabledProductNames)
	}
}

// TestAssignConnectionResourceModel_WithoutACollectionEntry pins the partial
// case. A connection readable on its own identifier but missing from the
// collection leaves the two attributes only the collection supplies empty rather
// than wrong, which is better than failing the refresh over the lesser half.
func TestAssignConnectionResourceModel_WithoutACollectionEntry(t *testing.T) {
	var state ConnectionResourceModel
	if diags := assignConnectionResourceModel(&state, oidcConnectionRead(), nil, true); diags.HasError() {
		t.Fatalf("assigning state: %v", diags)
	}

	if state.EnabledProductNames.IsNull() || len(state.EnabledProductNames.Elements()) != 0 {
		t.Errorf("enabled_product_names = %s, want a known empty set", state.EnabledProductNames)
	}
	if !state.TicketURL.IsNull() {
		t.Errorf("ticket_url = %s, want nothing", state.TicketURL)
	}
	if state.ID.ValueString() != unitConnectionID {
		t.Error("the rest of the connection must still be adopted")
	}
}

// TestCollectionDerivedValues_FallsBackToTheGoogleSettings pins the consent
// ticket's second source. It appears on the collection entry for every family and
// again inside a Google connection's own settings, so a Google connection can
// supply it even when the collection entry is unavailable.
func TestCollectionDerivedValues_FallsBackToTheGoogleSettings(t *testing.T) {
	_, ticket := collectionDerivedValues(nil, &account.GoogleOptions{
		TicketURL: new("consent/request/one"),
	})
	if ticket.ValueString() != "consent/request/one" {
		t.Errorf("ticket_url = %s, want the value from the Google settings", ticket)
	}

	summary := oidcSummaryRead()
	summary.TicketURL = new("consent/request/two")
	_, ticket = collectionDerivedValues(summary, &account.GoogleOptions{
		TicketURL: new("consent/request/one"),
	})
	if ticket.ValueString() != "consent/request/two" {
		t.Errorf("ticket_url = %s, want the collection entry to win", ticket)
	}
}

// TestParseGroupNameFilter pins the read side of the filter document, including
// the two things it must never do: collapse an operator with no groups into
// nothing, and read a document it does not understand back as no filter — which
// would let the next apply clear a live filter.
func TestParseGroupNameFilter(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		for _, raw := range []*string{nil, new(""), new("   ")} {
			filter, diags := parseGroupNameFilter(raw)
			if diags.HasError() {
				t.Fatalf("parsing %v: %v", raw, diags)
			}
			if filter != nil {
				t.Errorf("parsed %v into %+v, want nothing", raw, filter)
			}
		}
	})

	t.Run("an operator with no groups", func(t *testing.T) {
		filter, diags := parseGroupNameFilter(new(`{"op":"OR","groups":""}`))
		if diags.HasError() {
			t.Fatalf("parsing: %v", diags)
		}
		if filter == nil {
			t.Fatal("an operator with no groups is a real filter and must not collapse to nothing")
		}
		if got := filter.Operator.ValueString(); got != filterOperatorOr {
			t.Errorf("operator = %q, want the renamed value", got)
		}
		if filter.Groups.IsNull() || len(filter.Groups.Elements()) != 0 {
			t.Errorf("groups = %s, want a known empty set", filter.Groups)
		}
	})

	t.Run("groups split and trimmed", func(t *testing.T) {
		filter, diags := parseGroupNameFilter(new(`{"op":"AND","groups":"apple, mango ,zebra"}`))
		if diags.HasError() {
			t.Fatalf("parsing: %v", diags)
		}
		if got := filter.Operator.ValueString(); got != filterOperatorAnd {
			t.Errorf("operator = %q", got)
		}
		if len(filter.Groups.Elements()) != 3 {
			t.Errorf("groups = %s, want three members", filter.Groups)
		}
	})

	t.Run("unparseable is reported", func(t *testing.T) {
		_, diags := parseGroupNameFilter(new("not json"))
		if !diags.HasError() {
			t.Fatal("a filter this provider cannot read must be reported, not read back as no filter")
		}
	})

	t.Run("an unknown operator is reported", func(t *testing.T) {
		_, diags := parseGroupNameFilter(new(`{"op":"XOR","groups":"jamf"}`))
		if !diags.HasError() {
			t.Fatal("an operator this provider does not recognise must be reported")
		}
	})
}

// TestPreferEquivalentJSON pins the claim-mapping reconcile. Jamf re-serialises
// the document and `jsonencode` sorts keys, so the two forms differ constantly
// while meaning the same — and a difference that is only formatting must not
// churn state.
func TestPreferEquivalentJSON(t *testing.T) {
	planned := types.StringValue(`{"mapping_mode":"use_map","userinfo_scope":"profile"}`)
	reordered := `{"userinfo_scope":"profile","mapping_mode":"use_map"}`

	if got := preferEquivalentJSON(planned, new(reordered)); got.ValueString() != planned.ValueString() {
		t.Errorf("preferEquivalentJSON = %s, want the planned bytes kept", got)
	}

	changed := `{"mapping_mode":"bind_all"}`
	if got := preferEquivalentJSON(planned, new(changed)); got.ValueString() != changed {
		t.Errorf("preferEquivalentJSON = %s, want a genuine change adopted", got)
	}

	if got := preferEquivalentJSON(types.StringNull(), new(changed)); got.ValueString() != changed {
		t.Errorf("preferEquivalentJSON = %s, want Jamf's value where nothing was planned", got)
	}
	if got := preferEquivalentJSON(planned, nil); !got.IsNull() {
		t.Errorf("preferEquivalentJSON = %s, want nothing where Jamf holds nothing", got)
	}
}

// TestPreferEquivalentScopes pins the scope reconcile, which exists because the
// order of OAuth scopes carries no meaning and Jamf's copy of them was never seen
// written back. A re-ordering costs nothing to absorb; a genuine change is
// adopted and shows up as one.
func TestPreferEquivalentScopes(t *testing.T) {
	planned := types.StringValue("openid email profile")

	if got := preferEquivalentScopes(planned, new("profile openid email")); got.ValueString() != planned.ValueString() {
		t.Errorf("preferEquivalentScopes = %s, want the planned value kept", got)
	}
	if got := preferEquivalentScopes(planned, new("openid email profile groups")); got.ValueString() != "openid email profile groups" {
		t.Errorf("preferEquivalentScopes = %s, want an added scope adopted", got)
	}
	if got := preferEquivalentScopes(planned, new("openid email")); got.ValueString() != "openid email" {
		t.Errorf("preferEquivalentScopes = %s, want a removed scope adopted", got)
	}
	if got := preferEquivalentScopes(planned, nil); !got.IsNull() {
		t.Errorf("preferEquivalentScopes = %s, want nothing where Jamf holds nothing", got)
	}
}

// TestBuildEntraStateModel_FlattensTheNestedGroupOptions pins the one place the
// read and write shapes disagree about nesting: three of the Entra options sit a
// level deeper in Jamf's copy than in the settings it accepts.
func TestBuildEntraStateModel_FlattensTheNestedGroupOptions(t *testing.T) {
	block := buildEntraStateModel(&account.EntraOptions{
		Domain:       new("contoso.example"),
		TenantDomain: new("contoso.example"),
		BasicProfile: new(true),
		MaxGroups:    new(250),
		IdentityApi:  new(account.EntraIdentityApiMicrosoftIdentityPlatformV2),
		ExtOptions: &account.EntraExtendedOptions{
			ExtendedProfile: new(true),
			Groups:          new(true),
			NestedGroups:    new(false),
		},
	})

	if block == nil {
		t.Fatal("the Entra block must be built from Jamf's own settings")
	}
	if !block.ExtendedProfile.ValueBool() {
		t.Error("extended_profile must come from the nested options")
	}
	if !block.GetUserGroups.ValueBool() {
		t.Error("get_user_groups must come from the nested options")
	}
	if block.IncludeNestedGroups.IsNull() || block.IncludeNestedGroups.ValueBool() {
		t.Errorf("include_nested_groups = %s, want the nested value", block.IncludeNestedGroups)
	}
	if !block.BasicProfile.ValueBool() {
		t.Error("basic_profile must be reported, since it is always on and never a choice")
	}
}

// TestBuildEntraStateModel_WithoutTheNestedOptions pins the absent-inner-object
// case, which leaves the three deeper options empty rather than guessing at
// false.
func TestBuildEntraStateModel_WithoutTheNestedOptions(t *testing.T) {
	block := buildEntraStateModel(&account.EntraOptions{Domain: new("contoso.example")})
	if block == nil {
		t.Fatal("the Entra block must be built")
	}
	for name, value := range map[string]types.Bool{
		"extended_profile":      block.ExtendedProfile,
		"get_user_groups":       block.GetUserGroups,
		"include_nested_groups": block.IncludeNestedGroups,
	} {
		if !value.IsNull() {
			t.Errorf("%s = %s, want nothing where Jamf sent no nested options", name, value)
		}
	}
}

// TestBuildGoogleStateModel_MapsTheGroupSwitches pins the Google read mapping,
// which is a trap: Jamf's copy calls the group switch extGroups and the directory
// read extGroupsExtended, so the shorter name is not the simpler option.
func TestBuildGoogleStateModel_MapsTheGroupSwitches(t *testing.T) {
	block := buildGoogleStateModel(&account.GoogleOptions{
		Domain:            new("tf-unit.example"),
		ExtGroups:         new(true),
		ExtGroupsExtended: new(false),
		EnableUsersApi:    new(true),
	})

	if block == nil {
		t.Fatal("the Google block must be built from Jamf's own settings")
	}
	if !block.GetUserGroups.ValueBool() {
		t.Error("get_user_groups must come from the group switch")
	}
	if block.ExtendedGroups.ValueBool() {
		t.Error("extended_groups must come from the directory read, not from the group switch")
	}
	if !block.EnableUsersAPI.ValueBool() {
		t.Error("enable_users_api must be mapped")
	}
}

// TestBuildStateModels_WithoutSettings pins that a block Jamf did not return is
// left absent rather than built empty, which is what keeps only the family's own
// block populated.
func TestBuildStateModels_WithoutSettings(t *testing.T) {
	if buildOIDCStateModel(nil) != nil {
		t.Error("the generic block must be absent when Jamf returned none")
	}
	if buildEntraStateModel(nil) != nil {
		t.Error("the Entra block must be absent when Jamf returned none")
	}
	if buildOktaStateModel(nil) != nil {
		t.Error("the Okta block must be absent when Jamf returned none")
	}
	if buildGoogleStateModel(nil) != nil {
		t.Error("the Google block must be absent when Jamf returned none")
	}
}

// TestSessionMinutes pins the two session limits, including the absent-object
// case: Jamf reports the object with explicit absences, and no object at all is
// still no limits rather than zero.
func TestSessionMinutes(t *testing.T) {
	duration, inactivity := sessionMinutes(nil)
	if !duration.IsNull() || !inactivity.IsNull() {
		t.Errorf("no session object gave %s/%s, want nothing", duration, inactivity)
	}

	duration, inactivity = sessionMinutes(&account.SessionInfo{})
	if !duration.IsNull() || !inactivity.IsNull() {
		t.Errorf("explicit absences gave %s/%s, want nothing", duration, inactivity)
	}

	duration, inactivity = sessionMinutes(&account.SessionInfo{
		MaxSessionTimeInMinutes:    new(480),
		MaxInactivityTimeInMinutes: new(30),
	})
	if duration.ValueInt64() != 480 || inactivity.ValueInt64() != 30 {
		t.Errorf("session limits gave %s/%s, want 480/30", duration, inactivity)
	}
}

// TestRenameFromWire_CarriesThroughAnUnknownValue pins the read-side fallback. A
// value Jamf adds before this provider names it is reported as Jamf spelled it
// rather than read back as nothing, which would look like the connection had no
// such setting.
func TestRenameFromWire_CarriesThroughAnUnknownValue(t *testing.T) {
	if got := renameFromWire(new("SOMETHING_NEW"), connectionTypeFromWire); got.ValueString() != "SOMETHING_NEW" {
		t.Errorf("renameFromWire = %s, want Jamf's own spelling carried through", got)
	}
	if got := renameFromWire(nil, connectionTypeFromWire); !got.IsNull() {
		t.Errorf("renameFromWire = %s, want nothing", got)
	}
	if got := renameFromWire(new(""), connectionTypeFromWire); !got.IsNull() {
		t.Errorf("renameFromWire = %s, want nothing for an empty value", got)
	}
}

// TestBuildConnectionsResultModel pins the plural result, whose shape is Jamf's
// rather than this provider's: the collection read returns no per-provider
// settings, so the subset is what can be reported without an extra read per
// connection.
func TestBuildConnectionsResultModel(t *testing.T) {
	result, diags := buildConnectionsResultModel(*oidcSummaryRead())
	if diags.HasError() {
		t.Fatalf("building the result: %v", diags)
	}

	if result.ID.ValueString() != unitConnectionID {
		t.Errorf("id = %q", result.ID.ValueString())
	}
	if result.ConnectionType.ValueString() != connectionTypeGenericOIDC {
		t.Errorf("connection_type = %q, want the renamed value", result.ConnectionType.ValueString())
	}
	if len(result.Domains.Elements()) != 1 {
		t.Errorf("domains = %s, want the one domain", result.Domains)
	}
	if len(result.EnabledProductNames.Elements()) != 2 {
		t.Errorf("enabled_product_names = %s, want both products", result.EnabledProductNames)
	}
}
