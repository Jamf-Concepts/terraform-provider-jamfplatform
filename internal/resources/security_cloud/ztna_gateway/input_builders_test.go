// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const probeTenant = "928260f5-01f3-4881-bd2e-f28faa0dbab2"

// TestBuildGatewayCreateInput_InternetGatewayDerivesDedicatedFlag pins the derived
// discriminator on the way out. The API requires exactly one of the
// dedicated-egress flag or `ipsec`; with no `ipsec` block, the flag has to be sent
// as true or the create is rejected as configured as neither.
func TestBuildGatewayCreateInput_InternetGatewayDerivesDedicatedFlag(t *testing.T) {
	plan := internetGatewayPlan(t)

	got, diags := buildGatewayCreateInput(context.Background(), plan, plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	if got.Ipsec != nil {
		t.Error("an internet gateway must send no ipsec block")
	}
	if got.DedicatedIps == nil || !got.DedicatedIps.Enabled {
		t.Fatalf("an internet gateway must send dedicatedIps.enabled = true, got %+v", got.DedicatedIps)
	}
	if got.AvailabilityZones != nil {
		t.Error("an internet gateway must send no availability zones — the server refuses the combination")
	}
}

// TestBuildGatewayCreateInput_IPSecGatewayOmitsDedicatedFlag is the other half of
// the same rule: with an `ipsec` block, the flag must not be set, because the
// server refuses both together.
func TestBuildGatewayCreateInput_IPSecGatewayOmitsDedicatedFlag(t *testing.T) {
	plan := ipsecGatewayPlan(t, 1)

	got, diags := buildGatewayCreateInput(context.Background(), plan, plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	if got.DedicatedIps != nil {
		t.Errorf("an IPsec gateway must not send dedicatedIps, got %+v", got.DedicatedIps)
	}
	if got.Ipsec == nil {
		t.Fatal("an IPsec gateway must send the ipsec block")
	}
	if got.Ipsec.Left.Secret == nil || *got.Ipsec.Left.Secret != "probe-secret" {
		t.Errorf("create must carry the pre-shared key, got %v", got.Ipsec.Left.Secret)
	}
}

// TestBuildGatewayCreateInput_CollapsesCipherValuesToSingleElementArrays pins the
// boundary translation: the schema takes one algorithm per field, the wire takes
// an array, and the server rejects an array of any size but one.
func TestBuildGatewayCreateInput_CollapsesCipherValuesToSingleElementArrays(t *testing.T) {
	plan := ipsecGatewayPlan(t, 1)

	got, diags := buildGatewayCreateInput(context.Background(), plan, plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	for phase, suite := range map[string]struct {
		encryption []string
		integrity  []string
		dhGroups   []string
	}{
		"ike": {got.Ipsec.Ike.Encryption, got.Ipsec.Ike.Integrity, got.Ipsec.Ike.DhGroups},
		"esp": {got.Ipsec.Esp.Encryption, got.Ipsec.Esp.Integrity, got.Ipsec.Esp.DhGroups},
	} {
		for name, values := range map[string][]string{
			"encryption": suite.encryption,
			"integrity":  suite.integrity,
			"dhGroups":   suite.dhGroups,
		} {
			if len(values) != 1 {
				t.Errorf("%s.%s = %v, want exactly one element", phase, name, values)
			}
		}
	}

	if len(got.Ipsec.Left.Subnets) != 1 {
		t.Errorf("the Jamf-side subnet must go out as exactly one element, got %v", got.Ipsec.Left.Subnets)
	}
}

// TestBuildGatewayPatchInput_WithholdsSecretUntilRotationTriggerMoves is the
// important one for the credential path. The key cannot be read back, so
// re-sending it on every update would silently rotate whatever the config happens
// to hold; withholding it means Jamf keeps the stored key until the operator asks
// for a change.
func TestBuildGatewayPatchInput_WithholdsSecretUntilRotationTriggerMoves(t *testing.T) {
	state := ipsecGatewayPlan(t, 1)
	unchanged := ipsecGatewayPlan(t, 1)
	rotated := ipsecGatewayPlan(t, 2)

	got, diags := buildGatewayPatchInput(context.Background(), unchanged, state, unchanged)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if got.Ipsec.Left.Secret != nil {
		t.Error("an unchanged rotation trigger must withhold the pre-shared key")
	}

	got, diags = buildGatewayPatchInput(context.Background(), rotated, state, rotated)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if got.Ipsec.Left.Secret == nil {
		t.Fatal("a bumped rotation trigger must carry the pre-shared key")
	}
	if *got.Ipsec.Left.Secret != "probe-secret" {
		t.Errorf("secret = %q, want the value from config", *got.Ipsec.Left.Secret)
	}
}

// TestSharedSecretRotated_NoPriorBlockCountsAsRotation covers the import path: a
// gateway imported into state has no prior IPsec block, and the first update after
// that is the only chance to supply the key the wire never returned.
func TestSharedSecretRotated_NoPriorBlockCountsAsRotation(t *testing.T) {
	plan := ipsecGatewayPlan(t, 1)
	if !sharedSecretRotated(plan.IPSec, nil) {
		t.Error("with no prior IPsec block the secret must be sent, so an imported gateway can be given its key")
	}
	if sharedSecretRotated(nil, plan.IPSec) {
		t.Error("with no planned IPsec block there is no secret to send")
	}
}

// TestBuildGatewayPatchInput_ClearsAvailabilityZonesWhenUnset pins the one place a
// merge-patch omission would be wrong: removing the zones from configuration has
// to send an empty array, because omitting the field preserves what is stored.
func TestBuildGatewayPatchInput_ClearsAvailabilityZonesWhenUnset(t *testing.T) {
	state := ipsecGatewayPlan(t, 1)
	plan := ipsecGatewayPlan(t, 1)
	plan.AvailabilityZones = types.SetNull(types.StringType)

	got, diags := buildGatewayPatchInput(context.Background(), plan, state, plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if got.AvailabilityZones == nil {
		t.Fatal("removing availability zones must send an empty array, not omit the field")
	}
	if len(*got.AvailabilityZones) != 0 {
		t.Errorf("availabilityZones = %v, want empty", *got.AvailabilityZones)
	}
}

// internetGatewayPlan builds a dedicated internet gateway plan.
func internetGatewayPlan(t *testing.T) GatewayResourceModel {
	t.Helper()
	return GatewayResourceModel{
		ID:         types.StringValue("e045"),
		Name:       types.StringValue("tf-internet"),
		Datacenter: types.StringValue("eu-west-2"),
		Contact:    &ContactModel{Name: types.StringValue("TF"), Email: types.StringValue("tf@example.com")},
		Enabled:    types.BoolValue(true),
		TenantIDs:  stringSet(t, probeTenant),
	}
}

// ipsecGatewayPlan builds a dedicated IPsec gateway plan with the given rotation
// trigger.
func ipsecGatewayPlan(t *testing.T, woVersion int64) GatewayResourceModel {
	t.Helper()
	return GatewayResourceModel{
		ID:                types.StringValue("c08e"),
		Name:              types.StringValue("tf-ipsec"),
		Datacenter:        types.StringValue("eu-west-2"),
		Contact:           &ContactModel{Name: types.StringValue("TF"), Email: types.StringValue("tf@example.com")},
		Enabled:           types.BoolValue(true),
		TenantIDs:         stringSet(t, probeTenant),
		AvailabilityZones: stringSet(t, "3.9.67.90"),
		IPSec: &IPSecModel{
			KeyExchange: types.StringValue("ikev2"),
			IKE:         cipherSuitePlan(),
			ESP:         cipherSuitePlan(),
			JamfSide: &JamfSideModel{
				Host:                  types.StringValue("%any"),
				IKEID:                 types.StringValue("wpa.wandera.com"),
				Subnet:                types.StringValue("172.16.0.0/12"),
				SharedSecret:          types.StringValue("probe-secret"),
				SharedSecretWoVersion: types.Int64Value(woVersion),
			},
			CustomerSide: &CustomerSideModel{
				Host:    types.StringValue("198.51.100.7"),
				IKEID:   types.StringValue("peer.example.com"),
				Subnets: stringSet(t, "0.0.0.0/0"),
				Vendor:  types.StringValue("strongSwan"),
			},
		},
	}
}

// cipherSuitePlan builds a cipher-suite phase.
func cipherSuitePlan() *CipherSuiteModel {
	return &CipherSuiteModel{
		Encryption:      types.StringValue("aes256"),
		Integrity:       types.StringValue("sha512"),
		DHGroup:         types.StringValue("modp2048"),
		LifetimeSeconds: types.Int64Value(28800),
	}
}

// stringSet assembles a set of strings the way the framework would.
func stringSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	set, diags := types.SetValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}
	return set
}
