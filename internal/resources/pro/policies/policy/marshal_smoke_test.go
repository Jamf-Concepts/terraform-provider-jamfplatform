// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestMarshal_RepresentativePolicy round-trips a representative PolicyPost
// through encoding/xml and asserts the load-bearing wire invariants
// codified in STYLE_GUIDE.md §Scope helper and the SDK round-trip tests.
// This protects the canary against future SDK regressions during the Phase 5
// fan-out — every new scope-bearing resource that consumes internal/common/scope
// inherits these invariants and the test guards them in one place.
func TestMarshal_RepresentativePolicy(t *testing.T) {
	t.Parallel()
	plan := PolicyResourceModel{
		General: &PolicyGeneralModel{
			Name:           types.StringValue("tf-acc-smoke"),
			Enabled:        types.BoolValue(true),
			TriggerCheckin: types.BoolValue(true),
			Frequency:      types.StringValue("Once per computer"),
		},
		Scope: &PolicyScopeModel{
			ComputerGroupIDs: stringSet(t, "11", "22"),
			BuildingIDs:      stringSet(t, "7"),
		},
		SelfService: &PolicySelfServiceModel{
			UseForSelfService:    types.BoolValue(true),
			DisplayNotifications: types.BoolValue(true),
			NotificationLocation: types.StringValue("Self Service"),
		},
	}
	post, diags := buildPolicyInput(context.Background(), plan, noSecrets())
	if diags.HasError() {
		t.Fatalf("buildPolicyInput diagnostics: %v", diags)
	}

	out, err := xml.MarshalIndent(post, "", "  ")
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	wire := string(out)

	// Root element must be <policy>.
	if !strings.HasPrefix(strings.TrimSpace(wire), "<policy>") {
		t.Fatalf("expected wire to start with <policy>, got:\n%s", wire)
	}
	// Scope tree must include the two computer_group IDs and the building ID,
	// each wrapped in their canonical child elements.
	for _, want := range []string{
		"<scope>",
		"<computer_groups>",
		"<computer_group>",
		"<id>11</id>",
		"<id>22</id>",
		"<buildings>",
		"<building>",
		"<id>7</id>",
	} {
		if !strings.Contains(wire, want) {
			t.Fatalf("expected wire to contain %q, got:\n%s", want, wire)
		}
	}
	// Self Service notification: one <notification>bool</notification>
	// element plus a sibling <notification_type>method</notification_type>
	// — NOT two <notification> tags. The SDK NotificationValue.Method leg is
	// deliberately unused; method travels via PolicySelfService.NotificationType
	// (the wire-side sibling that the provider models as NotificationLocation).
	if c := strings.Count(wire, "<notification>"); c != 1 {
		t.Fatalf("expected exactly 1 <notification> element, got %d:\n%s", c, wire)
	}
	if !strings.Contains(wire, "<notification>true</notification>") {
		t.Fatalf("expected <notification>true</notification>, got:\n%s", wire)
	}
	if !strings.Contains(wire, "<notification_type>Self Service</notification_type>") {
		t.Fatalf("expected <notification_type>Self Service</notification_type>, got:\n%s", wire)
	}
}

// TestMarshal_EmptyScopeOmitsElement asserts the omission semantics from
// STYLE_GUIDE.md §Scope helper: an empty TF scope block must collapse to a
// nil pointer so the wire payload omits <scope> entirely rather than emitting
// an empty element.
func TestMarshal_EmptyScopeOmitsElement(t *testing.T) {
	t.Parallel()
	plan := PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc-empty-scope")},
		Scope:   &PolicyScopeModel{},
	}
	post, diags := buildPolicyInput(context.Background(), plan, noSecrets())
	if diags.HasError() {
		t.Fatalf("buildPolicyInput diagnostics: %v", diags)
	}
	if post.Scope != nil {
		t.Fatalf("expected Scope to collapse to nil for empty scope block, got %+v", post.Scope)
	}
	out, err := xml.Marshal(post)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	if strings.Contains(string(out), "<scope>") {
		t.Fatalf("expected no <scope> element on wire, got:\n%s", string(out))
	}
}
