// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package activation_profile

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// capabilityConfig builds a configuration carrying just the two capability
// booleans, which is all atLeastOneCapabilityValidator reads.
func capabilityConfig(t *testing.T, contentControls, networkSecurity tftypes.Value) tfsdk.Config {
	t.Helper()
	s := resourceSchema(t)
	objectType := s.Type().TerraformType(context.Background()).(tftypes.Object)

	values := map[string]tftypes.Value{}
	for name, attrType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	capabilityType := objectType.AttributeTypes["capabilities"].(tftypes.Object)
	capabilityValues := map[string]tftypes.Value{}
	for name, attrType := range capabilityType.AttributeTypes {
		capabilityValues[name] = tftypes.NewValue(attrType, nil)
	}
	capabilityValues["content_controls"] = contentControls
	capabilityValues["network_security"] = networkSecurity
	values["capabilities"] = tftypes.NewValue(capabilityType, capabilityValues)

	return tfsdk.Config{
		Schema: s,
		Raw:    tftypes.NewValue(objectType, values),
	}
}

// validateCapabilities runs the validator and returns its diagnostics.
func validateCapabilities(t *testing.T, contentControls, networkSecurity tftypes.Value) string {
	t.Helper()
	req := resource.ValidateConfigRequest{Config: capabilityConfig(t, contentControls, networkSecurity)}
	resp := &resource.ValidateConfigResponse{}
	atLeastOneCapabilityValidator{}.ValidateResource(context.Background(), req, resp)
	var sb strings.Builder
	for _, d := range resp.Diagnostics {
		sb.WriteString(d.Summary())
		sb.WriteString("|")
		sb.WriteString(d.Detail())
	}
	return sb.String()
}

// TestAtLeastOneCapability_BothDisabledIsAnError covers the rule the server
// enforces as a business rule, in an error envelope nothing can parse.
func TestAtLeastOneCapability_BothDisabledIsAnError(t *testing.T) {
	got := validateCapabilities(t,
		tftypes.NewValue(tftypes.Bool, false),
		tftypes.NewValue(tftypes.Bool, false),
	)
	if !strings.Contains(got, "No service capability enabled") {
		t.Errorf("expected the no-capability error, got %q", got)
	}
}

// TestAtLeastOneCapability_EitherEnabledPasses covers both single-capability
// shapes and the both-enabled one.
func TestAtLeastOneCapability_EitherEnabledPasses(t *testing.T) {
	cases := []struct {
		name                             string
		contentControls, networkSecurity bool
	}{
		{"content controls only", true, false},
		{"network security only", false, true},
		{"both", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validateCapabilities(t,
				tftypes.NewValue(tftypes.Bool, tc.contentControls),
				tftypes.NewValue(tftypes.Bool, tc.networkSecurity),
			)
			if got != "" {
				t.Errorf("expected no diagnostics, got %q", got)
			}
		})
	}
}

// TestAtLeastOneCapability_NullCountsAsDisabled matters because both attributes
// are optional: omitting both must fail the same way as setting both false.
func TestAtLeastOneCapability_NullCountsAsDisabled(t *testing.T) {
	got := validateCapabilities(t,
		tftypes.NewValue(tftypes.Bool, nil),
		tftypes.NewValue(tftypes.Bool, nil),
	)
	if !strings.Contains(got, "No service capability enabled") {
		t.Errorf("expected the no-capability error for two omitted attributes, got %q", got)
	}
}

// TestAtLeastOneCapability_UnknownDefersRatherThanFailing keeps the validator
// quiet when a capability is derived from another resource and is not yet known,
// so a legitimate plan is not rejected before the value exists.
func TestAtLeastOneCapability_UnknownDefersRatherThanFailing(t *testing.T) {
	got := validateCapabilities(t,
		tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		tftypes.NewValue(tftypes.Bool, false),
	)
	if got != "" {
		t.Errorf("expected the validator to defer on an unknown value, got %q", got)
	}
}
