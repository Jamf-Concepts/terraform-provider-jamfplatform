// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_invitation

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// buildConfig renders a VPPInvitationResourceModel into a tfsdk.Config the
// ConfigValidator can read. Only the fields the validator inspects are set;
// everything else is null.
func buildConfig(t *testing.T, m VPPInvitationResourceModel) tfsdk.Config {
	t.Helper()
	r := NewVPPInvitationResource()
	var sr resource.SchemaResponse
	r.(*VPPInvitationResource).Schema(context.Background(), resource.SchemaRequest{}, &sr)
	if sr.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", sr.Diagnostics)
	}

	objType := sr.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for name, ty := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(ty, nil)
	}
	setStr := func(name string, s types.String) {
		if s.IsNull() {
			return
		}
		vals[name] = tftypes.NewValue(tftypes.String, s.ValueString())
	}
	setStr("distribution_method", m.DistributionMethod)
	setStr("sender_name", m.SenderName)
	setStr("sender_email_address", m.SenderEmailAddress)
	setStr("subject", m.Subject)
	setStr("message", m.Message)
	return tfsdk.Config{Raw: tftypes.NewValue(objType, vals), Schema: sr.Schema}
}

func runEmailValidator(t *testing.T, m VPPInvitationResourceModel) int {
	t.Helper()
	cfg := buildConfig(t, m)
	resp := &resource.ValidateConfigResponse{}
	emailModeRequiresFieldsValidator{}.ValidateResource(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	return len(resp.Diagnostics.Errors())
}

func TestEmailValidator_SendEmailsMissingFields(t *testing.T) {
	errs := runEmailValidator(t, VPPInvitationResourceModel{
		DistributionMethod: types.StringValue(distributionMethodSendEmails),
	})
	if errs != 4 {
		t.Errorf("expected 4 errors (all email fields missing), got %d", errs)
	}
}

func TestEmailValidator_SendEmailsAllPresent(t *testing.T) {
	errs := runEmailValidator(t, VPPInvitationResourceModel{
		DistributionMethod: types.StringValue(distributionMethodSendEmails),
		SenderName:         types.StringValue("IT"),
		SenderEmailAddress: types.StringValue("it@example.com"),
		Subject:            types.StringValue("Register"),
		Message:            types.StringValue("link %@"),
	})
	if errs != 0 {
		t.Errorf("expected 0 errors, got %d", errs)
	}
}

func TestEmailValidator_SelfServiceNoRequirement(t *testing.T) {
	errs := runEmailValidator(t, VPPInvitationResourceModel{
		DistributionMethod: types.StringValue("Make available in Self Service only"),
	})
	if errs != 0 {
		t.Errorf("non-email mode must not require email fields, got %d errors", errs)
	}
}

// TestEmailValidator_DefersOnUnknownFields is the §436 regression guard: with
// distribution_method = "Send emails" (known) but the required sender / subject
// / message fields UNKNOWN (variable/for_each/resource-driven), the validator
// MUST defer, not treat unknown as missing and error. (buildConfig's setStr
// cannot represent unknown, so this builds the config directly.) See
// STYLE_GUIDE "Config-time validators MUST defer on unknown values".
func TestEmailValidator_DefersOnUnknownFields(t *testing.T) {
	r := NewVPPInvitationResource()
	var sr resource.SchemaResponse
	r.(*VPPInvitationResource).Schema(context.Background(), resource.SchemaRequest{}, &sr)
	if sr.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", sr.Diagnostics)
	}
	objType := sr.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for name, ty := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(ty, nil)
	}
	vals["distribution_method"] = tftypes.NewValue(tftypes.String, distributionMethodSendEmails)
	for _, f := range []string{"sender_name", "sender_email_address", "subject", "message"} {
		vals[f] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	}
	cfg := tfsdk.Config{Raw: tftypes.NewValue(objType, vals), Schema: sr.Schema}

	resp := &resource.ValidateConfigResponse{}
	emailModeRequiresFieldsValidator{}.ValidateResource(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if errs := len(resp.Diagnostics.Errors()); errs != 0 {
		t.Errorf("validator must defer when required email fields are unknown, got %d errors: %v", errs, resp.Diagnostics)
	}
}
