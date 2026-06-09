// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package venafi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildVenafiInput_KnownValues(t *testing.T) {
	plan := PkiVenafiResourceModel{
		Name:              types.StringValue("Venafi CA"),
		ProxyAddress:      types.StringValue("proxy.example.com:8443"),
		ClientID:          types.StringValue("client-abc"),
		RevocationEnabled: types.BoolValue(true),
	}
	token := "secret-token"
	got := buildVenafiInput(plan, &token)

	if got.Name != "Venafi CA" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.ProxyAddress == nil || *got.ProxyAddress != "proxy.example.com:8443" {
		t.Errorf("ProxyAddress = %v", got.ProxyAddress)
	}
	if got.ClientID == nil || *got.ClientID != "client-abc" {
		t.Errorf("ClientID = %v", got.ClientID)
	}
	if got.RevocationEnabled == nil || !*got.RevocationEnabled {
		t.Errorf("RevocationEnabled should be &true")
	}
	if got.RefreshToken == nil || *got.RefreshToken != "secret-token" {
		t.Errorf("RefreshToken = %v", got.RefreshToken)
	}
}

func TestBuildVenafiInput_OmitsNullUnknownForMergePreserve(t *testing.T) {
	plan := PkiVenafiResourceModel{
		Name:              types.StringValue("Venafi CA"),
		ProxyAddress:      types.StringNull(),
		ClientID:          types.StringUnknown(),
		RevocationEnabled: types.BoolNull(),
	}
	got := buildVenafiInput(plan, nil)

	if got.ProxyAddress != nil {
		t.Errorf("null ProxyAddress must be omitted (merge=preserve), got %v", *got.ProxyAddress)
	}
	if got.ClientID != nil {
		t.Errorf("unknown ClientID must be omitted, got %v", *got.ClientID)
	}
	if got.RevocationEnabled != nil {
		t.Errorf("null RevocationEnabled must be omitted, got %v", *got.RevocationEnabled)
	}
	if got.RefreshToken != nil {
		t.Errorf("nil token arg must omit RefreshToken, got %v", *got.RefreshToken)
	}
}

func TestBuildVenafiInput_EmptyStringClears(t *testing.T) {
	plan := PkiVenafiResourceModel{
		Name:         types.StringValue("Venafi CA"),
		ProxyAddress: types.StringValue(""),
	}
	got := buildVenafiInput(plan, nil)

	if got.ProxyAddress == nil {
		t.Fatalf("empty-string ProxyAddress must be emitted (\"\" clears), got nil")
	}
	if *got.ProxyAddress != "" {
		t.Errorf("ProxyAddress = %q, want empty string", *got.ProxyAddress)
	}
}

func TestShouldRotate(t *testing.T) {
	cases := []struct {
		name        string
		plan, state types.Int64
		want        bool
	}{
		{"null plan", types.Int64Null(), types.Int64Value(1), false},
		{"unknown plan", types.Int64Unknown(), types.Int64Value(1), false},
		{"new trigger from null state", types.Int64Value(1), types.Int64Null(), true},
		{"changed value", types.Int64Value(2), types.Int64Value(1), true},
		{"unchanged value", types.Int64Value(1), types.Int64Value(1), false},
	}
	for _, c := range cases {
		if got := shouldRotate(c.plan, c.state); got != c.want {
			t.Errorf("%s: shouldRotate = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestHasProxyTrustStore(t *testing.T) {
	if hasProxyTrustStore(types.StringNull()) {
		t.Errorf("null must be false")
	}
	if hasProxyTrustStore(types.StringUnknown()) {
		t.Errorf("unknown must be false")
	}
	if hasProxyTrustStore(types.StringValue("")) {
		t.Errorf("empty string must be false")
	}
	if !hasProxyTrustStore(types.StringValue("-----BEGIN-----")) {
		t.Errorf("non-empty must be true")
	}
}
