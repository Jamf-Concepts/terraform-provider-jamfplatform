// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ldap_server

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ldapSchemaTypes returns the top-level, connection_settings, and account
// tftypes.Object shapes from the real resource schema, so the validator can be
// driven against a config matching the production schema.
func ldapSchemaTypes(t *testing.T) (sr resource.SchemaResponse, objType, connType, accountType tftypes.Object) {
	t.Helper()
	r := NewLdapServerResource()
	r.(*LdapServerResource).Schema(context.Background(), resource.SchemaRequest{}, &sr)
	if sr.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sr.Diagnostics)
	}
	objType = sr.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	connType = objType.AttributeTypes["connection_settings"].(tftypes.Object)
	accountType = connType.AttributeTypes["account"].(tftypes.Object)
	return sr, objType, connType, accountType
}

// ldapConfig builds a tfsdk.Config with the given authentication_type value and
// account block value, filling every other attribute null.
func ldapConfig(t *testing.T, authVal, account tftypes.Value) tfsdk.Config {
	t.Helper()
	sr, objType, connType, _ := ldapSchemaTypes(t)

	connVals := map[string]tftypes.Value{}
	for n, ty := range connType.AttributeTypes {
		connVals[n] = tftypes.NewValue(ty, nil)
	}
	connVals["authentication_type"] = authVal
	connVals["account"] = account

	topVals := map[string]tftypes.Value{}
	for n, ty := range objType.AttributeTypes {
		topVals[n] = tftypes.NewValue(ty, nil)
	}
	topVals["connection_settings"] = tftypes.NewValue(connType, connVals)
	return tfsdk.Config{Schema: sr.Schema, Raw: tftypes.NewValue(objType, topVals)}
}

// ldapAccount builds an account block value with the supplied
// distinguished_username and every other sub-attribute null.
func ldapAccount(t *testing.T, dn tftypes.Value) tftypes.Value {
	t.Helper()
	_, _, _, accountType := ldapSchemaTypes(t)
	vals := map[string]tftypes.Value{}
	for n, ty := range accountType.AttributeTypes {
		vals[n] = tftypes.NewValue(ty, nil)
	}
	vals["distinguished_username"] = dn
	return tftypes.NewValue(accountType, vals)
}

func runLdapAccountValidator(t *testing.T, cfg tfsdk.Config) resource.ValidateConfigResponse {
	t.Helper()
	var resp resource.ValidateConfigResponse
	accountAuthConfigValidator{}.ValidateResource(context.Background(), resource.ValidateConfigRequest{Config: cfg}, &resp)
	return resp
}

func ldapStr(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }

// TestAccountAuthValidator_DefersOnUnknownAccount is the §436 regression guard:
// with authentication_type known (non-"none") and the account block UNKNOWN
// (e.g. `account = var.x`), the validator MUST defer — a decode into the Go
// model would collapse the unknown block to a nil pointer and false-error
// "account required". See STYLE_GUIDE "Config-time validators MUST defer on
// unknown values".
func TestAccountAuthValidator_DefersOnUnknownAccount(t *testing.T) {
	_, _, _, accountType := ldapSchemaTypes(t)
	cfg := ldapConfig(t, ldapStr("simple"), tftypes.NewValue(accountType, tftypes.UnknownValue))
	if resp := runLdapAccountValidator(t, cfg); resp.Diagnostics.HasError() {
		t.Errorf("validator must defer when the account block is unknown, got %v", resp.Diagnostics)
	}
}

// TestAccountAuthValidator_DefersOnUnknownAuthType confirms an unknown
// authentication_type also defers.
func TestAccountAuthValidator_DefersOnUnknownAuthType(t *testing.T) {
	_, _, _, accountType := ldapSchemaTypes(t)
	cfg := ldapConfig(t, tftypes.NewValue(tftypes.String, tftypes.UnknownValue), tftypes.NewValue(accountType, nil))
	if resp := runLdapAccountValidator(t, cfg); resp.Diagnostics.HasError() {
		t.Errorf("validator must defer when authentication_type is unknown, got %v", resp.Diagnostics)
	}
}

// TestAccountAuthValidator_DefersOnUnknownDN confirms an unknown
// distinguished_username (account present) defers.
func TestAccountAuthValidator_DefersOnUnknownDN(t *testing.T) {
	cfg := ldapConfig(t, ldapStr("simple"), ldapAccount(t, tftypes.NewValue(tftypes.String, tftypes.UnknownValue)))
	if resp := runLdapAccountValidator(t, cfg); resp.Diagnostics.HasError() {
		t.Errorf("validator must defer when distinguished_username is unknown, got %v", resp.Diagnostics)
	}
}

func TestAccountAuthValidator_AccountRequiredForAuthenticated(t *testing.T) {
	_, _, _, accountType := ldapSchemaTypes(t)
	cfg := ldapConfig(t, ldapStr("simple"), tftypes.NewValue(accountType, nil))
	if resp := runLdapAccountValidator(t, cfg); !resp.Diagnostics.HasError() {
		t.Error("expected error: account required when authentication_type != none")
	}
}

func TestAccountAuthValidator_AccountForbiddenForNone(t *testing.T) {
	cfg := ldapConfig(t, ldapStr(authTypeNone), ldapAccount(t, ldapStr("CN=svc,DC=example,DC=com")))
	if resp := runLdapAccountValidator(t, cfg); !resp.Diagnostics.HasError() {
		t.Error("expected error: account forbidden when authentication_type = none")
	}
}

func TestAccountAuthValidator_DNRequiredForAuthenticated(t *testing.T) {
	cfg := ldapConfig(t, ldapStr("simple"), ldapAccount(t, tftypes.NewValue(tftypes.String, nil)))
	if resp := runLdapAccountValidator(t, cfg); !resp.Diagnostics.HasError() {
		t.Error("expected error: distinguished_username required when authentication_type != none")
	}
}

func TestAccountAuthValidator_HappyAnonymous(t *testing.T) {
	_, _, _, accountType := ldapSchemaTypes(t)
	cfg := ldapConfig(t, ldapStr(authTypeNone), tftypes.NewValue(accountType, nil))
	if resp := runLdapAccountValidator(t, cfg); resp.Diagnostics.HasError() {
		t.Errorf("anonymous bind with no account must pass, got %v", resp.Diagnostics)
	}
}

func TestAccountAuthValidator_HappyAuthenticated(t *testing.T) {
	cfg := ldapConfig(t, ldapStr("simple"), ldapAccount(t, ldapStr("CN=svc,DC=example,DC=com")))
	if resp := runLdapAccountValidator(t, cfg); resp.Diagnostics.HasError() {
		t.Errorf("authenticated bind with account+dn must pass, got %v", resp.Diagnostics)
	}
}
