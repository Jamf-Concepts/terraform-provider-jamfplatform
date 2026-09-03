// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// resourceSchemaForTest returns the resource schema, which the cross-field
// validator needs in order to read its siblings.
func resourceSchemaForTest(t *testing.T) resourceschema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	(&ConnectionResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// validateConnection runs the cross-field validator against a configuration
// built by the given setters, and returns its diagnostics.
func validateConnection(t *testing.T, setters ...connectionSetter) validator.StringResponse {
	t.Helper()
	ctx := context.Background()
	s := resourceSchemaForTest(t)
	raw := connectionValue(ctx, t, s, setters...)

	var connectionType types.String
	config := tfsdk.Config{Schema: s, Raw: raw}
	if diags := config.GetAttribute(ctx, path.Root("connection_type"), &connectionType); diags.HasError() {
		t.Fatalf("reading connection_type: %v", diags)
	}

	var resp validator.StringResponse
	ConnectionSettings().ValidateString(ctx, validator.StringRequest{
		Path:        path.Root("connection_type"),
		Config:      config,
		ConfigValue: connectionType,
	}, &resp)
	return resp
}

// errorSummaries reduces a validator response to the paths and summaries it
// reported, which is what a practitioner sees.
func errorSummaries(resp validator.StringResponse) string {
	parts := make([]string, 0, len(resp.Diagnostics))
	for _, d := range resp.Diagnostics.Errors() {
		parts = append(parts, d.Summary()+": "+d.Detail())
	}
	return strings.Join(parts, "\n")
}

// setAttributeValue is a setter for an arbitrary attribute, so a test can state
// the one thing it is about.
func setAttributeValue(at path.Path, value any) connectionSetter {
	return func(ctx context.Context, t *testing.T, state *tfsdk.State) {
		t.Helper()
		setAttribute(ctx, t, state, at, value)
	}
}

// withEntraConfiguration sets the attributes a minimal Entra connection is
// configured with.
func withEntraConfiguration() connectionSetter {
	return func(ctx context.Context, t *testing.T, state *tfsdk.State) {
		t.Helper()
		setAttribute(ctx, t, state, path.Root("name"), "tf-unit-entra")
		setAttribute(ctx, t, state, path.Root("connection_type"), connectionTypeEntra)
		setAttribute(ctx, t, state, path.Root("hosting_region"), "US")
		setAttribute(ctx, t, state, path.Root("client_id"), "probe-client-id")
		setAttribute(ctx, t, state, path.Root("domains"), []string{"tf-unit.example"})
		setAttribute(ctx, t, state, path.Root("entra"), &EntraSettingsModel{
			Domain:       types.StringValue("contoso.example"),
			TenantDomain: types.StringValue("contoso.example"),
		})
	}
}

// withGoogleConfiguration sets the attributes a minimal Google Workspace
// connection is configured with.
func withGoogleConfiguration() connectionSetter {
	return func(ctx context.Context, t *testing.T, state *tfsdk.State) {
		t.Helper()
		setAttribute(ctx, t, state, path.Root("name"), "tf-unit-google")
		setAttribute(ctx, t, state, path.Root("connection_type"), connectionTypeGoogle)
		setAttribute(ctx, t, state, path.Root("hosting_region"), "US")
		setAttribute(ctx, t, state, path.Root("client_id"), "probe-client-id")
		setAttribute(ctx, t, state, path.Root("domains"), []string{"tf-unit.example"})
		setAttribute(ctx, t, state, path.Root("google_workspace"), &GoogleWorkspaceSettingsModel{
			Domain: types.StringValue("tf-unit.example"),
		})
	}
}

// TestConnectionSettings_ValidConfigurationsPass pins the negative side of every
// rule at once: a correct configuration of each family reports nothing. Without
// this, a rule that fires unconditionally would still pass every test that only
// asserts a refusal.
func TestConnectionSettings_ValidConfigurationsPass(t *testing.T) {
	for name, setters := range map[string][]connectionSetter{
		"generic_oidc": {withOIDCConfiguration()},
		"entra":        {withEntraConfiguration()},
		"google_workspace": {
			withGoogleConfiguration(),
			setAttributeValue(path.Root("scopes"), "openid email profile"),
		},
		"okta": {
			setAttributeValue(path.Root("name"), "tf-unit-okta"),
			setAttributeValue(path.Root("connection_type"), connectionTypeOkta),
			setAttributeValue(path.Root("hosting_region"), "US"),
			setAttributeValue(path.Root("client_id"), "probe-client-id"),
			setAttributeValue(path.Root("scopes"), "openid email profile"),
			setAttributeValue(path.Root("domains"), []string{"tf-unit.example"}),
			setAttributeValue(path.Root("okta"), &OktaSettingsModel{Domain: types.StringValue("example.okta.example")}),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if resp := validateConnection(t, setters...); resp.Diagnostics.HasError() {
				t.Errorf("a valid %s connection was refused:\n%s", name, errorSummaries(resp))
			}
		})
	}
}

// TestConnectionSettings_RequiresTheMatchingBlock covers rule 2. Jamf answers a
// settings block disagreeing with the declared family with an internal failure
// naming nothing, so the provider owns this entirely.
func TestConnectionSettings_RequiresTheMatchingBlock(t *testing.T) {
	resp := validateConnection(t,
		setAttributeValue(path.Root("name"), "tf-unit-oidc"),
		setAttributeValue(path.Root("connection_type"), connectionTypeGenericOIDC),
		setAttributeValue(path.Root("hosting_region"), "US"),
		setAttributeValue(path.Root("client_id"), "probe-client-id"),
		setAttributeValue(path.Root("scopes"), "openid"),
		setAttributeValue(path.Root("domains"), []string{"tf-unit.example"}),
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a connection type with no settings block must be refused")
	}
	if !strings.Contains(errorSummaries(resp), "generic_oidc") {
		t.Errorf("the refusal does not name the block to add:\n%s", errorSummaries(resp))
	}
}

// TestConnectionSettings_RefusesTheWrongBlock covers rule 1. This is the mistake
// most worth catching: Jamf accepts the request and then fails opaquely, so
// without this the operator sees an internal failure indistinguishable from the
// fault currently affecting every write.
func TestConnectionSettings_RefusesTheWrongBlock(t *testing.T) {
	resp := validateConnection(t,
		withOIDCConfiguration(),
		setAttributeValue(path.Root("entra"), &EntraSettingsModel{
			Domain:       types.StringValue("contoso.example"),
			TenantDomain: types.StringValue("contoso.example"),
		}),
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a settings block that does not match the connection type must be refused")
	}
	summaries := errorSummaries(resp)
	if !strings.Contains(summaries, "entra") {
		t.Errorf("the refusal does not name the offending block:\n%s", summaries)
	}
}

// TestConnectionSettings_ScopesRules covers rules 3, 4 and 5, including the one
// that is a survey finding rather than a documented constraint: no Entra
// connection read carried any scopes, and the settings Jamf accepts for one have
// nowhere to put them.
func TestConnectionSettings_ScopesRules(t *testing.T) {
	t.Run("required for generic_oidc", func(t *testing.T) {
		resp := validateConnection(t,
			withOIDCConfiguration(),
			setAttributeValue(path.Root("scopes"), types.StringNull()),
		)
		if !strings.Contains(errorSummaries(resp), "Scopes are required") {
			t.Errorf("omitted scopes were not refused:\n%s", errorSummaries(resp))
		}
	})

	t.Run("refused for entra", func(t *testing.T) {
		resp := validateConnection(t,
			withEntraConfiguration(),
			setAttributeValue(path.Root("scopes"), "openid"),
		)
		if !strings.Contains(errorSummaries(resp), "not accepted for an Entra connection") {
			t.Errorf("scopes on an Entra connection were not refused:\n%s", errorSummaries(resp))
		}
	})
}

// TestConnectionSettings_ClientIDRules covers rules 6 to 9, including the one
// exception: a multi-tenant Entra application needs no client of its own.
func TestConnectionSettings_ClientIDRules(t *testing.T) {
	t.Run("required for generic_oidc", func(t *testing.T) {
		resp := validateConnection(t,
			withOIDCConfiguration(),
			setAttributeValue(path.Root("client_id"), types.StringNull()),
		)
		if !strings.Contains(errorSummaries(resp), "Client identifier is required") {
			t.Errorf("an omitted client identifier was not refused:\n%s", errorSummaries(resp))
		}
	})

	t.Run("required for entra without the common endpoint", func(t *testing.T) {
		resp := validateConnection(t,
			withEntraConfiguration(),
			setAttributeValue(path.Root("client_id"), types.StringNull()),
		)
		if !strings.Contains(errorSummaries(resp), "use_common_endpoint") {
			t.Errorf("the refusal does not name the alternative:\n%s", errorSummaries(resp))
		}
	})

	t.Run("not required for a multi-tenant entra application", func(t *testing.T) {
		resp := validateConnection(t,
			withEntraConfiguration(),
			setAttributeValue(path.Root("client_id"), types.StringNull()),
			setAttributeValue(path.Root("entra"), &EntraSettingsModel{
				Domain:            types.StringValue("contoso.example"),
				TenantDomain:      types.StringValue("contoso.example"),
				UseCommonEndpoint: types.BoolValue(true),
			}),
		)
		if resp.Diagnostics.HasError() {
			t.Errorf("a multi-tenant Entra application needs no client identifier:\n%s", errorSummaries(resp))
		}
	})
}

// TestConnectionSettings_ClientSecretRefusedWithASignedAssertion covers rule 10,
// the one rule Jamf's own documentation is explicit about: with a signed
// assertion Jamf holds the key and there is no shared secret.
func TestConnectionSettings_ClientSecretRefusedWithASignedAssertion(t *testing.T) {
	resp := validateConnection(t,
		withOIDCConfiguration(),
		setAttributeValue(path.Root("auth_method"), authMethodPrivateKeyJWT),
		setAttributeValue(path.Root("client_secret"), "probe-client-secret"),
	)

	if !strings.Contains(errorSummaries(resp), "no shared secret") {
		t.Errorf("a client secret alongside a signed assertion was not refused:\n%s", errorSummaries(resp))
	}
}

// TestConnectionSettings_ConsoleAbsentControlsAreRefused covers rules 11 to 13 —
// the controls the Jamf Account console does not offer for a family. Each was
// checked against every readable connection of that family before being written
// as a refusal.
func TestConnectionSettings_ConsoleAbsentControlsAreRefused(t *testing.T) {
	t.Run("auth_method for google_workspace", func(t *testing.T) {
		resp := validateConnection(t,
			withGoogleConfiguration(),
			setAttributeValue(path.Root("auth_method"), authMethodClientSecret),
		)
		if !strings.Contains(errorSummaries(resp), "not a choice for this connection type") {
			t.Errorf("an authentication method on a Google connection was not refused:\n%s", errorSummaries(resp))
		}
	})

	t.Run("pkce for entra", func(t *testing.T) {
		resp := validateConnection(t,
			withEntraConfiguration(),
			setAttributeValue(path.Root("pkce"), pkceDisabled),
		)
		if !strings.Contains(errorSummaries(resp), "PKCE is not a choice") {
			t.Errorf("PKCE on an Entra connection was not refused:\n%s", errorSummaries(resp))
		}
	})

	t.Run("pkce for google_workspace", func(t *testing.T) {
		resp := validateConnection(t,
			withGoogleConfiguration(),
			setAttributeValue(path.Root("scopes"), "openid"),
			setAttributeValue(path.Root("pkce"), pkceDisabled),
		)
		if !strings.Contains(errorSummaries(resp), "PKCE is not a choice") {
			t.Errorf("PKCE on a Google connection was not refused:\n%s", errorSummaries(resp))
		}
	})
}

// TestConnectionSettings_EntraGroupOptionsNeedGroupMembership covers rule 14.
// Both options widen what is read from the directory, so neither means anything
// with group membership turned off — and Jamf stores both regardless, so the
// symptom would be a setting that quietly does nothing.
func TestConnectionSettings_EntraGroupOptionsNeedGroupMembership(t *testing.T) {
	t.Run("nested groups", func(t *testing.T) {
		resp := validateConnection(t,
			withEntraConfiguration(),
			setAttributeValue(path.Root("entra"), &EntraSettingsModel{
				Domain:              types.StringValue("contoso.example"),
				TenantDomain:        types.StringValue("contoso.example"),
				GetUserGroups:       types.BoolValue(false),
				IncludeNestedGroups: types.BoolValue(true),
			}),
		)
		if !strings.Contains(errorSummaries(resp), "Nested groups need group membership") {
			t.Errorf("nested groups without group membership were not refused:\n%s", errorSummaries(resp))
		}
	})

	t.Run("groups scope", func(t *testing.T) {
		resp := validateConnection(t,
			withEntraConfiguration(),
			setAttributeValue(path.Root("entra"), &EntraSettingsModel{
				Domain:       types.StringValue("contoso.example"),
				TenantDomain: types.StringValue("contoso.example"),
				GroupsScope:  types.StringValue("GROUP_READ_ALL"),
			}),
		)
		if !strings.Contains(errorSummaries(resp), "Group permission needs group membership") {
			t.Errorf("a group permission without group membership was not refused:\n%s", errorSummaries(resp))
		}
	})

	t.Run("both allowed with group membership on", func(t *testing.T) {
		resp := validateConnection(t,
			withEntraConfiguration(),
			setAttributeValue(path.Root("entra"), &EntraSettingsModel{
				Domain:              types.StringValue("contoso.example"),
				TenantDomain:        types.StringValue("contoso.example"),
				GetUserGroups:       types.BoolValue(true),
				IncludeNestedGroups: types.BoolValue(true),
				GroupsScope:         types.StringValue("GROUP_READ_ALL"),
			}),
		)
		if resp.Diagnostics.HasError() {
			t.Errorf("group options with group membership on must be accepted:\n%s", errorSummaries(resp))
		}
	})
}

// TestConnectionSettings_UnknownConnectionTypeReportsNothing pins the limitation
// every validator shares: a value coming from a variable or another resource is
// unknown at plan time, and nothing can be checked against it. Reporting a
// refusal there would refuse a configuration that is very likely correct.
func TestConnectionSettings_UnknownConnectionTypeReportsNothing(t *testing.T) {
	ctx := context.Background()
	s := resourceSchemaForTest(t)

	var resp validator.StringResponse
	ConnectionSettings().ValidateString(ctx, validator.StringRequest{
		Path:        path.Root("connection_type"),
		Config:      tfsdk.Config{Schema: s, Raw: connectionValue(ctx, t, s)},
		ConfigValue: types.StringUnknown(),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("an unknown connection type must report nothing:\n%s", errorSummaries(resp))
	}
}

// TestBareHost pins the address check, which exists because Jamf refuses a value
// carrying a scheme or a path without naming either the value or the part that
// offends.
func TestBareHost(t *testing.T) {
	for value, wantError := range map[string]bool{
		"example.com":         false,
		"example.okta.com":    false,
		"tf-unit.example":     false,
		"https://example.com": true,
		"example.com/path":    true,
		"user@example.com":    true,
		"example.com:443":     true,
		"example .com":        true,
	} {
		var resp validator.StringResponse
		BareHost().ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("domain"),
			ConfigValue: types.StringValue(value),
		}, &resp)
		if got := resp.Diagnostics.HasError(); got != wantError {
			t.Errorf("BareHost(%q) reported an error = %v, want %v", value, got, wantError)
		}
	}
}

// TestDomainName pins the lower-case requirement on top of the address check.
// Jamf holds a claimed domain in lower case, so a mixed-case entry names a domain
// the organization does not hold under that spelling — and the diagnostic has to
// say which spelling to use.
func TestDomainName(t *testing.T) {
	var resp validator.StringResponse
	DomainName().ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("domains"),
		ConfigValue: types.StringValue("TF-Unit.Example"),
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a mixed-case domain name must be refused")
	}
	if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "tf-unit.example") {
		t.Errorf("detail %q does not name the spelling to use", detail)
	}
}

// TestFilterGroupName pins the comma check. The filter is stored as one
// comma-separated string, so a name holding a comma would be split in two on the
// way in and match the wrong groups, with nothing reporting it.
func TestFilterGroupName(t *testing.T) {
	for value, wantError := range map[string]bool{
		"jamf-admins":  false,
		"jamf admins":  false,
		"jamf,admins":  true,
		"a,b,c":        true,
		"jamf-admins,": true,
	} {
		var resp validator.StringResponse
		FilterGroupName().ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("group_name_filter").AtName("groups"),
			ConfigValue: types.StringValue(value),
		}, &resp)
		if got := resp.Diagnostics.HasError(); got != wantError {
			t.Errorf("FilterGroupName(%q) reported an error = %v, want %v", value, got, wantError)
		}
	}
}

// TestAttributeMap pins the two strengths of check the claim mapping gets, and
// the reasoning is the point: Jamf validates nothing inside it, so the provider
// is the only thing that will ever report a mistake — but the recognised modes are
// a survey of one organization's connections rather than a declared set, so
// refusing an unfamiliar one would refuse a configuration Jamf may well accept.
func TestAttributeMap(t *testing.T) {
	cases := map[string]struct {
		value     string
		wantError bool
		wantWarn  bool
	}{
		"a recognised mode": {`{"mapping_mode":"bind_all"}`, false, false},
		"a full mapping": {
			`{"mapping_mode":"use_map","userinfo_scope":"profile","attributes":{"name":"claim"}}`,
			false, false,
		},
		"unparseable":        {`not json`, true, false},
		"a JSON array":       {`["bind_all"]`, true, false},
		"a JSON string":      {`"bind_all"`, true, false},
		"no mode":            {`{"userinfo_scope":"profile"}`, false, true},
		"an unfamiliar mode": {`{"mapping_mode":"something_new"}`, false, true},
		"a non-string mode":  {`{"mapping_mode":42}`, false, true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var resp validator.StringResponse
			AttributeMap().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("attribute_map"),
				ConfigValue: types.StringValue(tc.value),
			}, &resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantError {
				t.Errorf("reported an error = %v, want %v (%v)", got, tc.wantError, resp.Diagnostics)
			}
			if got := len(resp.Diagnostics.Warnings()) > 0; got != tc.wantWarn {
				t.Errorf("reported a warning = %v, want %v (%v)", got, tc.wantWarn, resp.Diagnostics)
			}
		})
	}
}

// TestAttributeMap_IgnoresAnAbsentValue pins that every validator here skips a
// value the configuration does not set, so an optional attribute is not made
// required by the check on it.
func TestAttributeMap_IgnoresAnAbsentValue(t *testing.T) {
	for name, value := range map[string]types.String{
		"absent":  types.StringNull(),
		"unknown": types.StringUnknown(),
	} {
		var resp validator.StringResponse
		AttributeMap().ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("attribute_map"),
			ConfigValue: value,
		}, &resp)
		if len(resp.Diagnostics) != 0 {
			t.Errorf("an %s claim mapping produced %v", name, resp.Diagnostics)
		}
	}
}

// TestValidatorDescriptionsAreStated pins that every validator says what it
// wants, since the framework renders these in the published documentation.
func TestValidatorDescriptionsAreStated(t *testing.T) {
	ctx := context.Background()
	for name, v := range map[string]validator.String{
		"BareHost":           BareHost(),
		"DomainName":         DomainName(),
		"FilterGroupName":    FilterGroupName(),
		"AttributeMap":       AttributeMap(),
		"ConnectionSettings": ConnectionSettings(),
	} {
		if v.Description(ctx) == "" {
			t.Errorf("%s has no description", name)
		}
		if v.MarkdownDescription(ctx) == "" {
			t.Errorf("%s has no Markdown description", name)
		}
	}
}

// TestNameAllowedPattern covers the character set Jamf accepts in a connection
// name. It is the one rule from the 2026-09-02 probing that needed new code: the
// SDK's value-typed aliasLoginHintToIdp already made the other undocumented
// requirement unreachable through this provider.
func TestNameAllowedPattern(t *testing.T) {
	for _, accepted := range []string{"tfProbeOidc", "Corp", "OIDC2", "a", "A1b2C3"} {
		if !nameAllowedPattern.MatchString(accepted) {
			t.Errorf("%q must be accepted; letters and digits are what Jamf takes", accepted)
		}
	}
	// A hyphen is the case that cost a day of probing: Jamf refuses it with an
	// unattributed 500. A space and an underscore are the same class. The empty
	// string is covered by LengthBetween, but must not match either.
	for _, refused := range []string{"tf-probe-oidc", "Corp OIDC", "corp_oidc", "corp.oidc", "", "tfProbe!"} {
		if nameAllowedPattern.MatchString(refused) {
			t.Errorf("%q must be refused before the plan is applied", refused)
		}
	}
}

// TestNameAllowedPatternRejectsTheStoredForm is the subtle half: the suffix Jamf
// appends to the stored name contains a hyphen, so the stored form fails the very
// rule the sent form must satisfy. Anything that validates a value read back from
// Jamf against this pattern would reject Jamf's own data.
func TestNameAllowedPatternRejectsTheStoredForm(t *testing.T) {
	if nameAllowedPattern.MatchString("tfProbeOidc-jqxld7tl4m454ed7s35647nmje5bmq") {
		t.Fatal("the stored suffixed form is expected to fail this pattern; it constrains what is sent, not what is read")
	}
}
