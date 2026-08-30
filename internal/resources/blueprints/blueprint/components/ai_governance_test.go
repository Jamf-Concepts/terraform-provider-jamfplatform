// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestAIGovernanceToRawConfiguration pins the wire shape established by probing: the wire names are
// policyId and versionNumber, not the Terraform attribute names, and versionNumber is a number.
func TestAIGovernanceToRawConfiguration(t *testing.T) {
	component := &AIGovernanceComponent{
		Policies: []AIGovernancePolicyReference{
			{PolicyID: types.StringValue("6ac0fe71-3f82-4aed-aed2-d98d440fffa1"), Version: types.Int64Value(1)},
			{PolicyID: types.StringValue("44c2efe6-e59b-41d0-94fb-89b17f3c8999"), Version: types.Int64Value(3)},
		},
	}

	raw, err := component.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration: %v", err)
	}
	const want = `{"policies":[{"policyId":"6ac0fe71-3f82-4aed-aed2-d98d440fffa1","versionNumber":1},{"policyId":"44c2efe6-e59b-41d0-94fb-89b17f3c8999","versionNumber":3}]}`
	if string(raw) != want {
		t.Errorf("configuration =\n  %s\nwant\n  %s", raw, want)
	}
}

// TestAIGovernanceEmptyPoliciesStillSendsAnArray pins that an empty list marshals as [] rather than
// null. The platform refuses both, but null is refused as a missing list — a worse diagnostic than
// the one an empty array earns.
func TestAIGovernanceEmptyPoliciesStillSendsAnArray(t *testing.T) {
	raw, err := (&AIGovernanceComponent{}).ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration: %v", err)
	}
	if string(raw) != `{"policies":[]}` {
		t.Errorf("configuration = %s, want {\"policies\":[]}", raw)
	}
}

func TestAIGovernanceFromRawConfiguration(t *testing.T) {
	var component AIGovernanceComponent
	raw := json.RawMessage(`{"policies":[{"policyId":"6ac0fe71-3f82-4aed-aed2-d98d440fffa1","versionNumber":2}]}`)
	if err := component.FromRawConfiguration(raw); err != nil {
		t.Fatalf("FromRawConfiguration: %v", err)
	}
	if len(component.Policies) != 1 {
		t.Fatalf("got %d policies, want 1", len(component.Policies))
	}
	if got := component.Policies[0].PolicyID.ValueString(); got != "6ac0fe71-3f82-4aed-aed2-d98d440fffa1" {
		t.Errorf("policy_id = %q", got)
	}
	if got := component.Policies[0].Version.ValueInt64(); got != 2 {
		t.Errorf("version = %d, want 2", got)
	}
}

// TestAIGovernanceRoundTrip pins that a configuration read back from the platform reproduces the
// same component, so a blueprint holding this component does not drift on refresh.
func TestAIGovernanceRoundTrip(t *testing.T) {
	original := &AIGovernanceComponent{
		Policies: []AIGovernancePolicyReference{
			{PolicyID: types.StringValue("a"), Version: types.Int64Value(1)},
			{PolicyID: types.StringValue("b"), Version: types.Int64Value(2)},
		},
	}
	raw, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration: %v", err)
	}

	var round AIGovernanceComponent
	if err := round.FromRawConfiguration(raw); err != nil {
		t.Fatalf("FromRawConfiguration: %v", err)
	}
	if len(round.Policies) != len(original.Policies) {
		t.Fatalf("round trip changed the policy count: %d then %d", len(original.Policies), len(round.Policies))
	}
	for i := range original.Policies {
		if round.Policies[i] != original.Policies[i] {
			t.Errorf("policy %d round-tripped as %+v, want %+v", i, round.Policies[i], original.Policies[i])
		}
	}
}

func TestAIGovernanceIdentifierAndClientComponent(t *testing.T) {
	component := &AIGovernanceComponent{
		Policies: []AIGovernancePolicyReference{{PolicyID: types.StringValue("x"), Version: types.Int64Value(1)}},
	}
	if got := component.GetIdentifier(); got != "com.jamf.ai-governance" {
		t.Errorf("identifier = %q", got)
	}
	client, err := component.ToClientComponent()
	if err != nil {
		t.Fatalf("ToClientComponent: %v", err)
	}
	if client.Identifier != "com.jamf.ai-governance" {
		t.Errorf("client identifier = %q", client.Identifier)
	}
	if len(client.Configuration) == 0 {
		t.Error("client configuration is empty")
	}
}

func TestAIGovernanceSchemaShape(t *testing.T) {
	attributes := AIGovernanceComponentSchema()
	policies, ok := attributes["policies"]
	if !ok {
		t.Fatal("missing the policies attribute")
	}
	if !policies.IsRequired() {
		t.Error("policies must be required — the platform refuses a component without them")
	}
}
