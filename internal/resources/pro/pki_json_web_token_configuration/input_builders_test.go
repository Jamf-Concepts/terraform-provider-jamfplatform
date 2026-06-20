// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_json_web_token_configuration

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildJSONWebTokenConfigurationInput_FieldMapping verifies the attributes
// project into the correct wire fields, including the key threaded on create.
func TestBuildJSONWebTokenConfigurationInput_FieldMapping(t *testing.T) {
	plan := JSONWebTokenConfigurationResourceModel{
		Name:        types.StringValue("token config"),
		TokenExpiry: types.Int64Value(30),
		Enabled:     types.BoolValue(true),
	}
	in := buildJSONWebTokenConfigurationInput(plan, new("c2VjcmV0LWtleQ=="))

	if in.Name == nil || *in.Name != "token config" {
		t.Errorf("name not mapped")
	}
	if in.EncryptionKey == nil || *in.EncryptionKey != "c2VjcmV0LWtleQ==" {
		t.Errorf("encryption_key must be threaded from the key arg")
	}
	if in.TokenExpiry == nil || *in.TokenExpiry != 30 {
		t.Errorf("token_expiry not mapped")
	}
	// enabled=true inverts to disabled=false.
	if in.Disabled == nil || *in.Disabled {
		t.Errorf("enabled=true must map to disabled=false, got %v", in.Disabled)
	}
}

// TestBuildJSONWebTokenConfigurationInput_KeyOmitted confirms a nil key arg
// omits the <encryption_key> element (Classic merge retains the stored key).
func TestBuildJSONWebTokenConfigurationInput_KeyOmitted(t *testing.T) {
	plan := JSONWebTokenConfigurationResourceModel{
		Name: types.StringValue("token config"),
	}
	in := buildJSONWebTokenConfigurationInput(plan, nil)
	if in.EncryptionKey != nil {
		t.Errorf("encryption_key must be nil when arg is nil, got %v", *in.EncryptionKey)
	}
}

// TestBuildJSONWebTokenConfigurationInput_EnabledInversion covers both
// polarities plus the null/unknown omission.
func TestBuildJSONWebTokenConfigurationInput_EnabledInversion(t *testing.T) {
	base := JSONWebTokenConfigurationResourceModel{Name: types.StringValue("x")}

	base.Enabled = types.BoolValue(false)
	if in := buildJSONWebTokenConfigurationInput(base, nil); in.Disabled == nil || !*in.Disabled {
		t.Errorf("enabled=false must map to disabled=true, got %v", in.Disabled)
	}

	base.Enabled = types.BoolValue(true)
	if in := buildJSONWebTokenConfigurationInput(base, nil); in.Disabled == nil || *in.Disabled {
		t.Errorf("enabled=true must map to disabled=false, got %v", in.Disabled)
	}

	base.Enabled = types.BoolNull()
	if in := buildJSONWebTokenConfigurationInput(base, nil); in.Disabled != nil {
		t.Errorf("null enabled must omit disabled, got %v", *in.Disabled)
	}

	base.Enabled = types.BoolUnknown()
	if in := buildJSONWebTokenConfigurationInput(base, nil); in.Disabled != nil {
		t.Errorf("unknown enabled must omit disabled, got %v", *in.Disabled)
	}
}

// TestBuildJSONWebTokenConfigurationInput_TokenExpiryOmitted confirms a
// null/unknown token_expiry collapses to nil so the server value is retained.
func TestBuildJSONWebTokenConfigurationInput_TokenExpiryOmitted(t *testing.T) {
	plan := JSONWebTokenConfigurationResourceModel{
		Name:        types.StringValue("x"),
		TokenExpiry: types.Int64Null(),
	}
	if in := buildJSONWebTokenConfigurationInput(plan, nil); in.TokenExpiry != nil {
		t.Errorf("null token_expiry must be omitted, got %v", *in.TokenExpiry)
	}

	plan.TokenExpiry = types.Int64Unknown()
	if in := buildJSONWebTokenConfigurationInput(plan, nil); in.TokenExpiry != nil {
		t.Errorf("unknown token_expiry must be omitted, got %v", *in.TokenExpiry)
	}
}

// TestEncryptionKeyForUpdate pins the rotation gate: the key is sent only when
// encryption_key_wo_version changed versus prior state.
func TestEncryptionKeyForUpdate(t *testing.T) {
	cfgKey := types.StringValue("bmV3LWtleQ==")

	// Version unchanged → nil (omit; server retains the stored key).
	if got := encryptionKeyForUpdate(types.Int64Value(1), types.Int64Value(1), cfgKey); got != nil {
		t.Errorf("unchanged version must omit the key, got %v", *got)
	}

	// Both null (never set) → unchanged → nil.
	if got := encryptionKeyForUpdate(types.Int64Null(), types.Int64Null(), cfgKey); got != nil {
		t.Errorf("null-to-null version must omit the key, got %v", *got)
	}

	// Version bumped → key included.
	if got := encryptionKeyForUpdate(types.Int64Value(2), types.Int64Value(1), cfgKey); got == nil || *got != "bmV3LWtleQ==" {
		t.Errorf("bumped version must include the key, got %v", got)
	}

	// Version newly set (null → 1) → key included.
	if got := encryptionKeyForUpdate(types.Int64Value(1), types.Int64Null(), cfgKey); got == nil || *got != "bmV3LWtleQ==" {
		t.Errorf("newly set version must include the key, got %v", got)
	}
}
