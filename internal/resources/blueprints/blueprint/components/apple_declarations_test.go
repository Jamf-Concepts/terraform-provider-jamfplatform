// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAppleDeclarations_GetIdentifier(t *testing.T) {
	c := &AppleDeclarationsComponent{}
	if c.GetIdentifier() != "com.jamf.ddm-strict" {
		t.Errorf("expected 'com.jamf.ddm-strict', got %q", c.GetIdentifier())
	}
}

// declarationsFromRaw decodes a rendered configuration back into the wire shape the assertions
// read, failing the test rather than returning an error so each case stays a single statement.
func declarationsFromRaw(t *testing.T, raw json.RawMessage) []map[string]any {
	t.Helper()

	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}
	entries, ok := config["declarations"].([]any)
	if !ok {
		t.Fatalf("expected declarations to be a slice, got %T", config["declarations"])
	}

	declarations := make([]map[string]any, 0, len(entries))
	for idx, entry := range entries {
		declaration, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("declaration[%d]: expected a map, got %T", idx, entry)
		}
		declarations = append(declarations, declaration)
	}
	return declarations
}

func TestAppleDeclarations_ToRawConfiguration_PayloadKeyIsOneBasedPosition(t *testing.T) {
	c := &AppleDeclarationsComponent{
		Declarations: []AppleDeclarationModel{
			{ChannelType: types.StringValue("SYSTEM"), Payload: types.StringValue(`{}`), Type: types.StringValue("com.apple.configuration.a")},
			{ChannelType: types.StringValue("SYSTEM"), Payload: types.StringValue(`{}`), Type: types.StringValue("com.apple.configuration.b")},
			{ChannelType: types.StringValue("USER"), Payload: types.StringValue(`{}`), Type: types.StringValue("com.apple.configuration.c")},
		},
	}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	declarations := declarationsFromRaw(t, rawCfg)
	if len(declarations) != 3 {
		t.Fatalf("expected 3 declarations, got %d", len(declarations))
	}

	for idx, declaration := range declarations {
		if declaration["payloadKey"] != float64(idx+1) {
			t.Errorf("declaration[%d]: expected payloadKey %d, got %v", idx, idx+1, declaration["payloadKey"])
		}
		if declaration["type"] != c.Declarations[idx].Type.ValueString() {
			t.Errorf("declaration[%d]: expected type %q, got %v", idx, c.Declarations[idx].Type.ValueString(), declaration["type"])
		}
	}
}

func TestAppleDeclarations_ToRawConfiguration_KindDerivedFromType(t *testing.T) {
	cases := []struct {
		declarationType string
		wantKind        string
	}{
		{"com.apple.configuration.passcode.settings", "CONFIGURATION"},
		{"com.apple.asset.data", "ASSET"},
		{"com.apple.activation.simple", "ACTIVATION"},
		{"com.apple.management.organization-info", "MANAGEMENT"},
	}

	for _, testCase := range cases {
		t.Run(testCase.declarationType, func(t *testing.T) {
			c := &AppleDeclarationsComponent{
				Declarations: []AppleDeclarationModel{
					{
						ChannelType: types.StringValue("SYSTEM"),
						Payload:     types.StringValue(`{}`),
						Type:        types.StringValue(testCase.declarationType),
					},
				},
			}

			rawCfg, err := c.ToRawConfiguration()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			declarations := declarationsFromRaw(t, rawCfg)
			if len(declarations) != 1 {
				t.Fatalf("expected 1 declaration, got %d", len(declarations))
			}
			if declarations[0]["kind"] != testCase.wantKind {
				t.Errorf("expected kind %q, got %v", testCase.wantKind, declarations[0]["kind"])
			}
		})
	}
}

func TestAppleDeclarations_ToRawConfiguration_Empty(t *testing.T) {
	c := &AppleDeclarationsComponent{}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(rawCfg) != `{"declarations":[]}` {
		t.Errorf(`expected {"declarations":[]}, got %s`, rawCfg)
	}
}

func TestAppleDeclarations_ToRawConfiguration_InvalidPayloadJSON(t *testing.T) {
	c := &AppleDeclarationsComponent{
		Declarations: []AppleDeclarationModel{
			{
				ChannelType: types.StringValue("SYSTEM"),
				Payload:     types.StringValue("not-valid-json"),
				Type:        types.StringValue("com.apple.configuration.test"),
			},
		},
	}

	if _, err := c.ToRawConfiguration(); err == nil {
		t.Error("expected error for invalid payload JSON")
	}
}

func TestAppleDeclarations_FromRawConfiguration_DropsKindAndPayloadKey(t *testing.T) {
	rawMap := map[string]any{
		"declarations": []any{
			map[string]any{
				"channelType": "USER",
				"kind":        "CONFIGURATION",
				"payload":     map[string]any{"setting": true},
				"payloadKey":  float64(7),
				"type":        "com.apple.asset.data",
			},
		},
	}
	raw, err := json.Marshal(rawMap)
	if err != nil {
		t.Fatalf("failed to marshal fixture: %v", err)
	}

	c := &AppleDeclarationsComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(c.Declarations))
	}
	if c.Declarations[0].ChannelType.ValueString() != "USER" {
		t.Errorf("expected ChannelType 'USER', got %q", c.Declarations[0].ChannelType.ValueString())
	}
	if c.Declarations[0].Type.ValueString() != "com.apple.asset.data" {
		t.Errorf("expected Type 'com.apple.asset.data', got %q", c.Declarations[0].Type.ValueString())
	}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	declarations := declarationsFromRaw(t, rawCfg)
	if declarations[0]["kind"] != "ASSET" {
		t.Errorf("expected the kind to be re-derived as 'ASSET', got %v", declarations[0]["kind"])
	}
	if declarations[0]["payloadKey"] != float64(1) {
		t.Errorf("expected the payloadKey to be re-derived as 1, got %v", declarations[0]["payloadKey"])
	}
}

// htmlEscapedPayload is what Terraform's jsonencode() actually emits for a payload containing &,
// < and >: it HTML-escapes all three, exactly as Go's encoding/json does by default. That match is
// what makes the payload attribute round-trip, since it is byte-compared with no semantic equality.
//
// Verified on the wire rather than assumed: switching FromRawConfiguration to an encoder with
// SetEscapeHTML(false) makes `terraform apply` fail its FIRST apply with "Provider produced
// inconsistent result after apply", because the config value carries \u0026 and the state value
// then carries a bare &. TestAccResource_Blueprint_AppleDeclarations_HTMLEscapedPayload is the
// acceptance guard; these two are the fast ones.
//
// Do not "fix" this to emit unescaped output. It looks like a bug and is not one.
const htmlEscapedPayload = `{"Reference":{"ContentType":"application/zip","DataURL":"https://cdn.example.com/ddm?a=1\u0026b=2"},"note":"a\u003cb and b\u003ea"}`

func TestAppleDeclarations_FromRawConfiguration_PayloadKeepsHTMLEscaping(t *testing.T) {
	original := &AppleDeclarationsComponent{
		Declarations: []AppleDeclarationModel{
			{
				ChannelType: types.StringValue("SYSTEM"),
				Payload:     types.StringValue(htmlEscapedPayload),
				Type:        types.StringValue("com.apple.asset.data"),
			},
		},
	}

	rawCfg, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	restored := &AppleDeclarationsComponent{}
	if err := restored.FromRawConfiguration(rawCfg); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}
	if len(restored.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(restored.Declarations))
	}
	if got := restored.Declarations[0].Payload.ValueString(); got != htmlEscapedPayload {
		t.Errorf("payload did not round-trip byte-identically:\n want %s\n  got %s", htmlEscapedPayload, got)
	}
}

func TestCustomDeclarations_FromRawConfiguration_PayloadKeepsHTMLEscaping(t *testing.T) {
	original := &CustomDeclarationsComponent{
		Declarations: []CustomDeclarationModel{
			{
				ChannelType: types.StringValue("SYSTEM"),
				Kind:        types.StringValue("ASSET"),
				Payload:     types.StringValue(htmlEscapedPayload),
				Type:        types.StringValue("com.apple.asset.data"),
			},
		},
	}

	rawCfg, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	restored := &CustomDeclarationsComponent{}
	if err := restored.FromRawConfiguration(rawCfg); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}
	if len(restored.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(restored.Declarations))
	}
	if got := restored.Declarations[0].Payload.ValueString(); got != htmlEscapedPayload {
		t.Errorf("payload did not round-trip byte-identically:\n want %s\n  got %s", htmlEscapedPayload, got)
	}
}

func TestAppleDeclarations_ToClientComponent(t *testing.T) {
	c := &AppleDeclarationsComponent{
		Declarations: []AppleDeclarationModel{
			{
				ChannelType: types.StringValue("SYSTEM"),
				Payload:     types.StringValue(`{"key":"val"}`),
				Type:        types.StringValue("com.apple.configuration.test"),
			},
		},
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Identifier != "com.jamf.ddm-strict" {
		t.Errorf("expected identifier 'com.jamf.ddm-strict', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
