// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildWebhookInput_FieldMapping verifies the flat attributes project into
// the correct wire fields.
func TestBuildWebhookInput_FieldMapping(t *testing.T) {
	plan := WebhookResourceModel{
		Name:               types.StringValue("hook"),
		Enabled:            types.BoolValue(true),
		URL:                types.StringValue("https://e.com/x"),
		Event:              types.StringValue("ComputerAdded"),
		ContentType:        types.StringValue("application/json"),
		AuthenticationType: types.StringValue(authTypeBasic),
		ConnectionTimeout:  types.Int64Value(7),
		ReadTimeout:        types.Int64Value(9),
		Username:           types.StringValue("bob"),
		Header:             types.StringNull(),
		HashAlgorithm:      types.StringValue("SHA256"),
		SmartGroupID:       types.Int64Null(),
	}
	in := buildWebhookInput(plan, new("secret"))

	if in.Name == nil || *in.Name != "hook" {
		t.Errorf("name not mapped")
	}
	if in.Enabled == nil || !*in.Enabled {
		t.Errorf("enabled not mapped")
	}
	if in.URL == nil || *in.URL != "https://e.com/x" {
		t.Errorf("url not mapped")
	}
	if in.Event == nil || *in.Event != "ComputerAdded" {
		t.Errorf("event not mapped")
	}
	if in.ContentType == nil || *in.ContentType != "application/json" {
		t.Errorf("content_type not mapped")
	}
	if in.AuthenticationType == nil || *in.AuthenticationType != authTypeBasic {
		t.Errorf("authentication_type not mapped")
	}
	if in.ConnectionTimeout == nil || *in.ConnectionTimeout != 7 {
		t.Errorf("connection_timeout not mapped")
	}
	if in.ReadTimeout == nil || *in.ReadTimeout != 9 {
		t.Errorf("read_timeout not mapped")
	}
	if in.Username == nil || *in.Username != "bob" {
		t.Errorf("username not mapped")
	}
	if in.Password == nil || *in.Password != "secret" {
		t.Errorf("password must be threaded from the password arg")
	}
	if in.HashAlgorithm == nil || *in.HashAlgorithm != "SHA256" {
		t.Errorf("hash_algorithm not mapped")
	}
	// Unset smart_group_id is always emitted as the -1 sentinel ("none") so a
	// smart→non-smart transition clears any retained group under Classic merge.
	if in.SmartGroupID == nil || *in.SmartGroupID != -1 {
		t.Errorf("unset smart_group_id must emit -1, got %v", in.SmartGroupID)
	}
}

// TestBuildWebhookInput_PasswordOmitted confirms a nil password arg omits the
// <password> element (Classic merge retains the stored secret).
func TestBuildWebhookInput_PasswordOmitted(t *testing.T) {
	plan := WebhookResourceModel{
		Name:  types.StringValue("hook"),
		URL:   types.StringValue("https://e.com/x"),
		Event: types.StringValue("ComputerAdded"),
	}
	in := buildWebhookInput(plan, nil)
	if in.Password != nil {
		t.Errorf("password must be nil when arg is nil, got %v", *in.Password)
	}
}

// TestBuildWebhookInput_SmartGroupSentinel verifies a smart event with no
// configured group emits the -1 sentinel so Update can clear a prior group,
// while a configured group is passed through.
func TestBuildWebhookInput_SmartGroupSentinel(t *testing.T) {
	base := WebhookResourceModel{
		Name:  types.StringValue("hook"),
		URL:   types.StringValue("https://e.com/x"),
		Event: types.StringValue("SmartGroupComputerMembershipChange"),
	}

	// No configured group → -1.
	base.SmartGroupID = types.Int64Null()
	in := buildWebhookInput(base, nil)
	if in.SmartGroupID == nil || *in.SmartGroupID != -1 {
		t.Errorf("smart event with no group must emit -1, got %v", in.SmartGroupID)
	}

	// Configured group → passed through.
	base.SmartGroupID = types.Int64Value(29)
	in = buildWebhookInput(base, nil)
	if in.SmartGroupID == nil || *in.SmartGroupID != 29 {
		t.Errorf("configured smart_group_id must pass through, got %v", in.SmartGroupID)
	}
}

// TestBuildWebhookInput_OmitsUnsetOptionals confirms null Optional+Computed
// fields collapse to nil so the classic omitempty tags drop them (server
// defaults apply).
func TestBuildWebhookInput_OmitsUnsetOptionals(t *testing.T) {
	plan := WebhookResourceModel{
		Name:               types.StringValue("hook"),
		URL:                types.StringValue("https://e.com/x"),
		Event:              types.StringValue("ComputerAdded"),
		Enabled:            types.BoolNull(),
		ContentType:        types.StringNull(),
		AuthenticationType: types.StringNull(),
		ConnectionTimeout:  types.Int64Null(),
		ReadTimeout:        types.Int64Null(),
		HashAlgorithm:      types.StringNull(),
		Username:           types.StringNull(),
		Header:             types.StringNull(),
	}
	in := buildWebhookInput(plan, nil)
	if in.Enabled != nil || in.ContentType != nil || in.AuthenticationType != nil ||
		in.ConnectionTimeout != nil || in.ReadTimeout != nil || in.HashAlgorithm != nil ||
		in.Username != nil || in.Header != nil {
		t.Errorf("unset optionals must be nil, got %+v", in)
	}
}
