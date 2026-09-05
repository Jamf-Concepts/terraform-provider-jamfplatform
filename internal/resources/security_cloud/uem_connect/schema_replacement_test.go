// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package uem_connect

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// uem_server_url must force replacement when the practitioner changes an address
// they wrote, and never when they wrote none.
//
// The distinction matters because the attribute is Optional+Computed and the
// platform_tenant form never configures it: Jamf Security Cloud resolves the
// address from the tenant, and the resource refuses the pair. The framework marks
// an unconfigured Computed attribute UNKNOWN on every plan where anything else
// changes, and plain RequiresReplace tests only whether the planned value equals
// the prior one — an unknown never does, whatever the prior value is. So a plain
// RequiresReplace here fired on every in-place edit of a platform_tenant
// connector, planning a destroy and recreate for a toggled `enabled` or an edited
// field mapping, which strands the JSC Connector API integration on the Jamf Pro
// tenant. RequiresReplaceIfConfigured asks the intended question instead.
//
// The unknown planned value in the unconfigured cases below is not an invention:
// it is what fwserver.MarkComputedNilsAsUnknown produces for a Computed attribute
// whose configuration value is null, on any plan whose proposed new state differs
// from prior state. Verified end to end by driving
// providerserver.PlanResourceChange over this attribute's modifier chain — a
// no-op plan is clean either way, an `enabled`-only change planned a replacement
// before this test's expectations and none after, and a changed configured
// address still replaces.
//
// This tests the chain's behaviour rather than the identity of the modifier, so
// it fails on a revert to RequiresReplace and stays true under any equivalent
// spelling.
func TestUEMServerURL_ReplacementFollowsWhatThePractitionerConfigured(t *testing.T) {
	const url, otherURL = "https://a.jamfcloud.com", "https://b.jamfcloud.com"

	tests := []struct {
		name        string
		config      types.String
		state       types.String
		plan        types.String
		wantReplace bool
	}{
		{
			name:   "platform_tenant form, an unrelated field changed",
			config: types.StringNull(),
			state:  types.StringNull(),
			plan:   types.StringUnknown(),
		},
		{
			name:   "platform_tenant form adopted before the address was dropped from state",
			config: types.StringNull(),
			state:  types.StringValue(url),
			plan:   types.StringUnknown(),
		},
		{
			name:   "oauth form, address untouched",
			config: types.StringValue(url),
			state:  types.StringValue(url),
			plan:   types.StringValue(url),
		},
		{
			name:        "oauth form, address changed",
			config:      types.StringValue(otherURL),
			state:       types.StringValue(url),
			plan:        types.StringValue(otherURL),
			wantReplace: true,
		},
	}

	attr, ok := resourceSchema(t).Attributes["uem_server_url"].(schema.StringAttribute)
	if !ok {
		t.Fatal("uem_server_url must be a StringAttribute")
	}
	if len(attr.PlanModifiers) == 0 {
		t.Fatal("uem_server_url declares no plan modifiers, so this guard would pass vacuously")
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				Path:           path.Root("uem_server_url"),
				PathExpression: path.MatchRoot("uem_server_url"),
				Config:         tfsdk.Config{Schema: resourceSchema(t), Raw: nonNullRaw()},
				State:          tfsdk.State{Schema: resourceSchema(t), Raw: nonNullRaw()},
				Plan:           tfsdk.Plan{Schema: resourceSchema(t), Raw: nonNullRaw()},
				ConfigValue:    tc.config,
				StateValue:     tc.state,
				PlanValue:      tc.plan,
			}

			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			for _, m := range attr.PlanModifiers {
				m.PlanModifyString(context.Background(), req, resp)
				req.PlanValue = resp.PlanValue
				if resp.Diagnostics.HasError() {
					t.Fatalf("plan modifier diagnostics: %v", resp.Diagnostics.Errors())
				}
			}

			if resp.RequiresReplace != tc.wantReplace {
				t.Errorf("RequiresReplace = %v, want %v — an unconfigured address must never force a replacement, "+
					"and a changed configured one always must", resp.RequiresReplace, tc.wantReplace)
			}
		})
	}
}

// nonNullRaw is a non-null object of the resource's own type, which is all the
// modifiers under test read the Config/State/Plan wrappers for: a null State.Raw
// means "being created" and a null Plan.Raw means "being destroyed", and both
// short-circuit the modifiers before they decide anything.
func nonNullRaw() tftypes.Value {
	return tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})
}
