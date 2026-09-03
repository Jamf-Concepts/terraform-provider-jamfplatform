// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"context"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// urlishCharacters are the characters that cannot appear in a bare domain name
// and that show up when an address or an email has been pasted in place of one.
const urlishCharacters = "/:?#@ \t"

// bareHostValidator refuses a value that is plainly an address rather than a
// host name.
//
// Nothing else about the syntax is checked. Jamf is deliberately permissive with
// a domain — a reserved top-level domain such as `.example` is accepted — so a
// plan-time check strict enough to be useful would risk refusing a name Jamf
// would have taken.
type bareHostValidator struct{}

// BareHost returns a validator refusing anything that is not a bare host name.
func BareHost() validator.String {
	return bareHostValidator{}
}

// Description returns a plain-text description of the validator's behaviour.
func (v bareHostValidator) Description(_ context.Context) string {
	return "must be a bare host name, with no scheme, path, port or user part"
}

// MarkdownDescription returns a Markdown description of the validator's behaviour.
func (v bareHostValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString checks the configured host name.
func (v bareHostValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !helpers.IsConfiguredValue(req.ConfigValue) {
		return
	}
	appendBareHostDiagnostics(&resp.Diagnostics, req.Path, req.ConfigValue.ValueString())
}

// domainNameValidator adds the lower-case requirement to the bare-host check.
//
// Jamf lower-cases a domain name when it records the claim — wire-verified on the
// sibling SSO domain construct, which claims the very names this attribute
// references. A mixed-case entry here would therefore name a domain Jamf holds
// under a different spelling. Because the attribute is Required,
// STYLE_GUIDE §"Plan-modifier rewrites are NOT a valid option for Required
// attributes" rules out canonicalising the value, so strict acceptance plus a
// diagnostic naming the spelling to use is the only correct option.
type domainNameValidator struct{}

// DomainName returns a validator enforcing a lower-case, bare domain name.
func DomainName() validator.String {
	return domainNameValidator{}
}

// Description returns a plain-text description of the validator's behaviour.
func (v domainNameValidator) Description(_ context.Context) string {
	return "must be a bare domain name in lower case"
}

// MarkdownDescription returns a Markdown description of the validator's behaviour.
func (v domainNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString checks the configured domain name.
func (v domainNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !helpers.IsConfiguredValue(req.ConfigValue) {
		return
	}
	value := req.ConfigValue.ValueString()
	if appendBareHostDiagnostics(&resp.Diagnostics, req.Path, value) {
		return
	}
	if lowered := strings.ToLower(value); lowered != value {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Domain name must be lower case",
			"Jamf holds a claimed domain name in lower case, so a mixed-case value here names a domain your "+
				"organization does not hold under that spelling. Set it to \""+lowered+"\".",
		)
	}
}

// filterGroupNameValidator refuses a group name carrying the separator Jamf uses
// between them.
//
// The group filter is stored as a single comma-separated string, so a name
// containing a comma would be split into two on the way in and could never
// round-trip. Jamf validates nothing inside the filter, so the symptom would be
// a filter that quietly matches the wrong groups.
type filterGroupNameValidator struct{}

// FilterGroupName returns a validator refusing a group name holding a comma.
func FilterGroupName() validator.String {
	return filterGroupNameValidator{}
}

// Description returns a plain-text description of the validator's behaviour.
func (v filterGroupNameValidator) Description(_ context.Context) string {
	return "must not contain a comma"
}

// MarkdownDescription returns a Markdown description of the validator's behaviour.
func (v filterGroupNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString checks one configured group name.
func (v filterGroupNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !helpers.IsConfiguredValue(req.ConfigValue) {
		return
	}
	if strings.Contains(req.ConfigValue.ValueString(), ",") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Group name cannot contain a comma",
			"Jamf separates the filtered group names with commas, so a name holding one would be read as two "+
				"names and the filter would match the wrong groups. Configured value: \""+
				req.ConfigValue.ValueString()+"\".",
		)
	}
}

// appendBareHostDiagnostics reports a value that is an address rather than a
// host name, and says whether it did.
func appendBareHostDiagnostics(diags *diag.Diagnostics, at path.Path, value string) bool {
	if !strings.ContainsAny(value, urlishCharacters) {
		return false
	}
	diags.AddAttributeError(
		at,
		"Value is not a bare domain name",
		"This takes a bare domain name — no scheme, no path, no port, no user part and no whitespace. Set it "+
			"to `example.com` rather than `https://example.com/` or `user@example.com`. Configured value: \""+
			value+"\".",
	)
	return true
}

// connectionSettingsValidator owns every cross-field rule this construct has.
//
// It owns all of them because Jamf checks almost none. Only three top-level
// fields are ever named in a refusal; a bad vocabulary value is refused without
// naming the field it was on; and a settings block disagreeing with the declared
// provider family is an unattributable internal failure indistinguishable from
// any other. So a mistake the provider does not catch reaches an operator as an
// opaque failure mid-apply, and each rule below exists because Jamf will not
// report it.
//
// It hangs off `connection_type`, which is Required and therefore always present,
// per STYLE_GUIDE §Cross-field validation's requirement that a cross-field rule
// be an attribute-level validator. Every diagnostic is attached to the attribute
// the operator has to change rather than to `connection_type` itself. Sibling
// values are read one at a time with GetAttribute rather than by decoding the
// whole configuration, because decoding fails outright when any nested value is
// unknown and would take every rule down with it.
//
// No rule below refuses an unknown value — one derived from another resource in
// the same apply — and each has its own reason. A rule that requires a value
// fires only on a genuinely absent one: an unknown value is about to resolve to
// something, and refusing it would rule out the ordinary composition of
// registering an application with the provider and the Jamf connection that uses
// it in one apply. That covers a value such a rule takes as its exception too —
// an unknown `entra.use_common_endpoint` may yet be `true`, so rule 9 stays
// silent rather than demanding a `client_id` the connection may not need. A rule
// that refuses a value stays quiet as well, which is what `IsConfiguredValue`
// gives it: an unknown value can still resolve to nothing at all — a value set
// from a conditional is unknown at plan time and absent by the end of it — so a
// refusal here would turn away a configuration that applies cleanly, and one that
// does resolve to something is left to Jamf. And a rule whose condition is
// unknown reports nothing either way: an unknown `connection_type`, `auth_method`
// or `entra.get_user_groups` leaves every rule hanging off it unable to tell
// whether it applies.
//
// The fourteen rules:
//
//  1. exactly one settings block is set;
//  2. the block set is the one connection_type names;
//  3. scopes is required for generic_oidc;
//  4. scopes is required for okta;
//  5. scopes is refused for entra, which takes none;
//  6. client_id is required for generic_oidc;
//  7. client_id is required for okta;
//  8. client_id is required for google_workspace;
//  9. client_id is required for entra unless entra.use_common_endpoint is set;
//  10. client_secret is refused when auth_method is private_key_jwt;
//  11. auth_method is refused for google_workspace;
//  12. pkce is refused for entra;
//  13. pkce is refused for google_workspace;
//  14. entra.include_nested_groups and entra.groups_scope need
//     entra.get_user_groups.
//
// Rules 3 to 9 mirror what each settings shape declares as required. Rules 5, 11
// to 13 mirror controls the Jamf Account console does not offer for a family —
// each was checked against every readable connection of that family before being
// written as a refusal, and scopes was absent from every Entra connection read.
// Rule 10 follows the one thing Jamf's documentation is explicit about: with a
// signed assertion Jamf holds the key itself and there is no shared secret.
type connectionSettingsValidator struct{}

// ConnectionSettings returns the cross-field validator for an SSO connection.
func ConnectionSettings() validator.String {
	return connectionSettingsValidator{}
}

// Description returns a plain-text description of the validator's behaviour.
func (v connectionSettingsValidator) Description(_ context.Context) string {
	return "requires the settings block, credentials and options the chosen connection type accepts"
}

// MarkdownDescription returns a Markdown description of the validator's behaviour.
func (v connectionSettingsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// settingsBlockNames pairs each connection type with the settings block it
// requires, in declaration order so a diagnostic listing them is stable.
var settingsBlockNames = []struct {
	connectionType string
	block          string
}{
	{connectionTypeGenericOIDC, "generic_oidc"},
	{connectionTypeEntra, "entra"},
	{connectionTypeOkta, "okta"},
	{connectionTypeGoogle, "google_workspace"},
}

// ValidateString applies every cross-field rule.
func (v connectionSettingsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !helpers.IsConfiguredValue(req.ConfigValue) {
		return
	}
	connectionType := req.ConfigValue.ValueString()

	validateSettingsBlocks(ctx, req, resp, connectionType)
	validateScopes(ctx, req, resp, connectionType)
	validateClientCredentials(ctx, req, resp, connectionType)
	validateProviderOptions(ctx, req, resp, connectionType)
	validateEntraGroupOptions(ctx, req, resp, connectionType)
}

// validateSettingsBlocks applies rules 1 and 2.
func validateSettingsBlocks(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse, connectionType string) {
	for _, pairing := range settingsBlockNames {
		block, ok := configObject(ctx, req, resp, path.Root(pairing.block))
		if !ok {
			continue
		}
		wanted := pairing.connectionType == connectionType
		switch {
		case wanted && block.IsNull():
			resp.Diagnostics.AddAttributeError(
				path.Root(pairing.block),
				"Settings block required for this connection type",
				"`connection_type` is \""+connectionType+"\", so the `"+pairing.block+"` block has to be set. "+
					"It carries the settings only that kind of provider takes. Jamf Account refuses a connection whose "+
					"settings do not match its type without saying which part was wrong, which is why this is "+
					"reported here.",
			)
		case !wanted && !block.IsNull() && !block.IsUnknown():
			resp.Diagnostics.AddAttributeError(
				path.Root(pairing.block),
				"Settings block does not match this connection type",
				"`connection_type` is \""+connectionType+"\", so the `"+pairing.block+"` block cannot be set — "+
					"exactly one settings block may be present and it has to be the one the connection type "+
					"names. Remove this block, or change `connection_type` to \""+pairing.connectionType+"\".",
			)
		}
	}
}

// validateScopes applies rules 3, 4 and 5.
func validateScopes(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse, connectionType string) {
	scopes, ok := configString(ctx, req, resp, path.Root("scopes"))
	if !ok {
		return
	}

	switch connectionType {
	case connectionTypeGenericOIDC, connectionTypeOkta:
		if scopes.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("scopes"),
				"Scopes are required for this connection type",
				"A \""+connectionType+"\" connection has to say which scopes Jamf asks your provider for. "+
					"`openid` is required; the Jamf Account console's default is `openid email profile`, and a "+
					"`groups` scope is needed if you want group memberships passed through.",
			)
		}
	case connectionTypeEntra:
		if helpers.IsConfiguredValue(scopes) {
			resp.Diagnostics.AddAttributeError(
				path.Root("scopes"),
				"Scopes are not accepted for an Entra connection",
				"An Entra connection takes no scopes — no Entra connection read carried any, and the settings "+
					"Jamf Account accepts for one have no place to put them. Remove `scopes`, and control what is read "+
					"from the directory with the `entra` block's profile and group options instead.",
			)
		}
	}
}

// validateClientCredentials applies rules 6 to 10.
func validateClientCredentials(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse, connectionType string) {
	clientID, ok := configString(ctx, req, resp, path.Root("client_id"))
	if ok && clientID.IsNull() {
		switch connectionType {
		case connectionTypeGenericOIDC, connectionTypeOkta, connectionTypeGoogle:
			resp.Diagnostics.AddAttributeError(
				path.Root("client_id"),
				"Client identifier is required for this connection type",
				"A \""+connectionType+"\" connection signs in with an application you registered with your "+
					"provider, so `client_id` has to be set to that application's identifier.",
			)
		case connectionTypeEntra:
			common, commonOK := configBool(ctx, req, resp, path.Root("entra").AtName("use_common_endpoint"))
			if commonOK && !common.IsUnknown() && !common.ValueBool() {
				resp.Diagnostics.AddAttributeError(
					path.Root("client_id"),
					"Client identifier is required for this connection type",
					"An Entra connection needs `client_id` unless it is a multi-tenant application using "+
						"`entra.use_common_endpoint`. Set `client_id` to your application registration's "+
						"identifier, or set `entra.use_common_endpoint` to `true`.",
				)
			}
		}
	}

	authMethod, authOK := configString(ctx, req, resp, path.Root("auth_method"))
	if !authOK {
		return
	}
	if !helpers.IsConfiguredValue(authMethod) || authMethod.ValueString() != authMethodPrivateKeyJWT {
		return
	}
	secret, secretOK := configString(ctx, req, resp, path.Root("client_secret"))
	if secretOK && helpers.IsConfiguredValue(secret) {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_secret"),
			"Client secret is not used with a signed assertion",
			"`auth_method` is \""+authMethodPrivateKeyJWT+"\", where Jamf proves itself with a key it holds "+
				"itself and there is no shared secret. Remove `client_secret`, or set `auth_method` to \""+
				authMethodClientSecret+"\".",
		)
	}
}

// validateProviderOptions applies rules 11, 12 and 13.
func validateProviderOptions(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse, connectionType string) {
	if connectionType == connectionTypeGoogle {
		if authMethod, ok := configString(ctx, req, resp, path.Root("auth_method")); ok && helpers.IsConfiguredValue(authMethod) {
			resp.Diagnostics.AddAttributeError(
				path.Root("auth_method"),
				"Authentication method is not a choice for this connection type",
				"The Jamf Account console offers no authentication method for a Google Workspace connection — it "+
					"always uses a client secret. Remove `auth_method`.",
			)
		}
	}

	if connectionType != connectionTypeEntra && connectionType != connectionTypeGoogle {
		return
	}
	pkce, ok := configString(ctx, req, resp, path.Root("pkce"))
	if ok && helpers.IsConfiguredValue(pkce) {
		resp.Diagnostics.AddAttributeError(
			path.Root("pkce"),
			"PKCE is not a choice for this connection type",
			"The Jamf Account console offers a PKCE setting only for a generic OpenID Connect or an Okta "+
				"connection. Remove `pkce`.",
		)
	}
}

// validateEntraGroupOptions applies rule 14.
func validateEntraGroupOptions(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse, connectionType string) {
	if connectionType != connectionTypeEntra {
		return
	}
	groups, ok := configBool(ctx, req, resp, path.Root("entra").AtName("get_user_groups"))
	if !ok || groups.IsUnknown() || groups.ValueBool() {
		return
	}

	if nested, nestedOK := configBool(ctx, req, resp, path.Root("entra").AtName("include_nested_groups")); nestedOK &&
		helpers.IsConfiguredValue(nested) && nested.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("entra").AtName("include_nested_groups"),
			"Nested groups need group membership turned on",
			"`entra.include_nested_groups` widens the groups Jamf reads, so it means nothing unless "+
				"`entra.get_user_groups` is `true`. Set `entra.get_user_groups` to `true`, or remove this.",
		)
	}

	if scope, scopeOK := configString(ctx, req, resp, path.Root("entra").AtName("groups_scope")); scopeOK &&
		helpers.IsConfiguredValue(scope) {
		resp.Diagnostics.AddAttributeError(
			path.Root("entra").AtName("groups_scope"),
			"Group permission needs group membership turned on",
			"`entra.groups_scope` names the Microsoft Graph permission groups are read with, so it means "+
				"nothing unless `entra.get_user_groups` is `true`. Set `entra.get_user_groups` to `true`, or "+
				"remove this.",
		)
	}
}

// configString reads one sibling string out of the configuration, reporting
// ok=false when the read produced a diagnostic so the caller skips its rule
// rather than acting on a zero value.
func configString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse, at path.Path) (types.String, bool) {
	var value types.String
	diags := req.Config.GetAttribute(ctx, at, &value)
	resp.Diagnostics.Append(diags...)
	return value, !diags.HasError()
}

// configBool reads one sibling boolean out of the configuration.
func configBool(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse, at path.Path) (types.Bool, bool) {
	var value types.Bool
	diags := req.Config.GetAttribute(ctx, at, &value)
	resp.Diagnostics.Append(diags...)
	return value, !diags.HasError()
}

// configObject reads one sibling block out of the configuration, so its presence
// can be checked without decoding what is inside it.
func configObject(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse, at path.Path) (types.Object, bool) {
	var value types.Object
	diags := req.Config.GetAttribute(ctx, at, &value)
	resp.Diagnostics.Append(diags...)
	return value, !diags.HasError()
}

// attributeMapValidator checks the claim-mapping document.
//
// Jamf serves no schema for it and documents no vocabulary, and its own
// validation layer never looks inside — an unrecognised mode is stored and then
// quietly ignored, which is the worst failure this construct has, because nothing
// reports it and sign-in simply maps the wrong details. So the shape is checked
// here, and the strength of each check matches how well the value behind it is
// known.
//
// Not being a JSON object is an error: it is the one thing the value certainly
// has to be, since every readable connection carried an object and nothing else
// could carry a mode. An unrecognised or missing mode is a warning: the three
// known modes are a survey of one organization's connections rather than a
// declared set, so refusing a fourth would refuse a configuration Jamf may well
// accept — the same reasoning the Apple profile and vendor schema validators in
// this provider use for a name they do not recognise.
type attributeMapValidator struct{}

// AttributeMap returns the validator for the claim-mapping document.
func AttributeMap() validator.String {
	return attributeMapValidator{}
}

// Description returns a plain-text description of the validator's behaviour.
func (v attributeMapValidator) Description(_ context.Context) string {
	return "must be a JSON object, and should carry a recognised " + mappingModeKey
}

// MarkdownDescription returns a Markdown description of the validator's behaviour.
func (v attributeMapValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString checks the configured claim-mapping document.
func (v attributeMapValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !helpers.IsConfiguredValue(req.ConfigValue) {
		return
	}

	decoded, err := decodeJSONObject(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Claim mapping is not a JSON object",
			"`attribute_map` has to be a JSON object of mapping settings. Author it with `jsonencode({ ... })`. "+
				"Jamf Account stores whatever it is given here without checking it and then ignores what it cannot "+
				"read, so a malformed value would take effect as no mapping at all. Reported while parsing: "+
				err.Error(),
		)
		return
	}

	mode, present := decoded[mappingModeKey]
	if !present {
		resp.Diagnostics.AddAttributeWarning(
			req.Path,
			"Claim mapping names no mode",
			"Every connection read carried a `"+mappingModeKey+"` in its claim mapping, one of "+
				markdownValueList(mappingModeValues())+". This value names none, which Jamf Account will store and "+
				"then ignore. There is no published schema for this, so it is possible a mapping without a mode is "+
				"meaningful and this warning is wrong — hence a warning rather than a refusal.",
		)
		return
	}

	text, isString := mode.(string)
	if !isString || !slices.Contains(mappingModeValues(), text) {
		resp.Diagnostics.AddAttributeWarning(
			req.Path,
			"Claim mapping names an unrecognised mode",
			"`"+mappingModeKey+"` is not one of "+markdownValueList(mappingModeValues())+", which are the modes "+
				"observed across every readable connection. There is no published schema for this, and Jamf Account "+
				"validates "+
				"nothing inside it, so a mode it does not recognise is stored and then ignored rather than "+
				"refused. This is a warning rather than an error because the known modes are an observation "+
				"and not a declared set.",
		)
	}
}

// mappingModeValues returns the claim-mapping modes this provider recognises.
func mappingModeValues() []string {
	return []string{mappingModeBindAll, mappingModeBasicProfile, mappingModeUseMap}
}
