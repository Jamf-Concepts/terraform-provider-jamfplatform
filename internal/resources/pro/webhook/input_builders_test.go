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
// defaults apply). With authentication_type unknown, username and header are
// also omitted: the server resolves the type and clears inactive fields
// itself, and an unknown type gives the builder nothing to scope them by.
func TestBuildWebhookInput_OmitsUnsetOptionals(t *testing.T) {
	plan := WebhookResourceModel{
		Name:               types.StringValue("hook"),
		URL:                types.StringValue("https://e.com/x"),
		Event:              types.StringValue("ComputerAdded"),
		Enabled:            types.BoolNull(),
		ContentType:        types.StringNull(),
		AuthenticationType: types.StringUnknown(),
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

// TestAuthScopedField pins the per-type encoding of username / header: emitted
// empty (clears) under every type that does not require the field, sent only
// as configured under the type that does (the server refuses an empty element
// there), and sent only as configured when the type is unknown.
func TestAuthScopedField(t *testing.T) {
	tests := []struct {
		name      string
		field     types.String
		auth      types.String
		requiring string
		wantNil   bool
		want      string
	}{
		{"null username under NONE emits empty", types.StringNull(), types.StringValue(authTypeNone), authTypeBasic, false, ""},
		{"null username under HEADER emits empty", types.StringNull(), types.StringValue(authTypeHeader), authTypeBasic, false, ""},
		{"null username under HASH_SIGNATURE emits empty", types.StringNull(), types.StringValue(authTypeHashSignature), authTypeBasic, false, ""},
		{"null username under BASIC omitted", types.StringNull(), types.StringValue(authTypeBasic), authTypeBasic, true, ""},
		{"set username under BASIC sent", types.StringValue("bob"), types.StringValue(authTypeBasic), authTypeBasic, false, "bob"},
		{"null header under HEADER omitted", types.StringNull(), types.StringValue(authTypeHeader), authTypeHeader, true, ""},
		{"null header under BASIC emits empty", types.StringNull(), types.StringValue(authTypeBasic), authTypeHeader, false, ""},
		{"set header under HEADER sent", types.StringValue("{}"), types.StringValue(authTypeHeader), authTypeHeader, false, "{}"},
		{"null field with unknown type omitted", types.StringNull(), types.StringUnknown(), authTypeBasic, true, ""},
		{"set field with unknown type sent", types.StringValue("bob"), types.StringUnknown(), authTypeBasic, false, "bob"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authScopedField(tt.field, tt.auth, tt.requiring)
			switch {
			case tt.wantNil && got != nil:
				t.Errorf("want nil, got %q", *got)
			case !tt.wantNil && got == nil:
				t.Errorf("want %q, got nil", tt.want)
			case !tt.wantNil && *got != tt.want:
				t.Errorf("want %q, got %q", tt.want, *got)
			}
		})
	}
}
