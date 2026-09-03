// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestConnectionComparisonsCoverEveryConfigurableAttribute is the guard that keeps
// replacement-on-change honest.
//
// An attribute an operator can set but connectionComparisons does not list plans
// as an in-place update, and Jamf's update endpoint refuses every one — so the
// omission surfaces as an apply failure on that single attribute, which no other
// test would catch. Adding an attribute without listing it fails here instead.
//
// Two names are exempt, for opposite reasons. `client_secret` is WriteOnly, so it
// is never in state and comparing it would report a change on every plan;
// rotation runs through `client_secret_wo_version`, which is compared. `timeouts`
// is provider-side configuration Jamf never sees, so changing it must not touch
// the connection.
func TestConnectionComparisonsCoverEveryConfigurableAttribute(t *testing.T) {
	exempt := map[string]string{
		"client_secret": "WriteOnly, never in state; rotation goes through client_secret_wo_version",
		"timeouts":      "provider-side configuration with no counterpart in Jamf Account",
		// These two carry stringplanmodifier.RequiresReplace() on the schema
		// already, because Jamf refuses to move a connection to another provider
		// family or region even when the endpoint works. They replace for a
		// reason that outlives the broken endpoint, so they stay in the schema
		// rather than moving into the temporary comparison list — and comparing
		// them in both places would be the one duplication this guard exists to
		// prevent.
		"connection_type": "already RequiresReplace in the schema, independently of the update endpoint",
		"hosting_region":  "already RequiresReplace in the schema, independently of the update endpoint",
	}

	compared := map[string]bool{}
	for _, comparison := range connectionComparisons(ConnectionResourceModel{}, ConnectionResourceModel{}) {
		compared[comparison.name] = true
	}

	var schemaResp resource.SchemaResponse
	NewConnectionResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	settable := map[string]bool{}
	for name, attribute := range schemaResp.Schema.Attributes {
		if attribute.IsRequired() || attribute.IsOptional() {
			settable[name] = true
		}
	}
	for name := range schemaResp.Schema.Blocks {
		settable[name] = true
	}
	if len(settable) == 0 {
		t.Fatal("the schema reported no settable attributes, so this guard would pass vacuously")
	}

	for name := range settable {
		if reason, ok := exempt[name]; ok {
			if compared[name] {
				t.Errorf("%s is compared but documented as exempt (%s)", name, reason)
			}
			continue
		}
		if !compared[name] {
			t.Errorf("%s can be set but is not compared, so changing it would plan an in-place update "+
				"that Jamf's update endpoint refuses; add it to connectionComparisons", name)
		}
	}

	for name := range compared {
		if !settable[name] {
			t.Errorf("%s is compared but is not a settable attribute; the list has drifted from the schema", name)
		}
	}
}
