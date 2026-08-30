// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
)

func assertAttrsPresent(t *testing.T, schema func(context.Context, action.SchemaRequest, *action.SchemaResponse), want []string) {
	t.Helper()
	var resp action.SchemaResponse
	schema(context.Background(), action.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range want {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func assertRequired(t *testing.T, schema func(context.Context, action.SchemaRequest, *action.SchemaResponse), required []string) {
	t.Helper()
	var resp action.SchemaResponse
	schema(context.Background(), action.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range required {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}
}

func assertTypeName(t *testing.T, meta func(context.Context, action.MetadataRequest, *action.MetadataResponse), want string) {
	t.Helper()
	req := action.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp action.MetadataResponse
	meta(context.Background(), req, &resp)
	if resp.TypeName != want {
		t.Errorf("expected type name %q, got %q", want, resp.TypeName)
	}
}

// --- send_blank_push ---

func TestSendBlankPushAction_Metadata(t *testing.T) {
	assertTypeName(t, NewSendBlankPushAction().(*SendBlankPushAction).Metadata, "jamfplatform_pro_send_blank_push")
}

func TestSendBlankPushAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewSendBlankPushAction().(*SendBlankPushAction).Schema,
		[]string{"management_ids", "serial_numbers"})
}

// --- renew_mdm_profile ---

func TestRenewMdmProfileAction_Metadata(t *testing.T) {
	assertTypeName(t, NewRenewMdmProfileAction().(*RenewMdmProfileAction).Metadata, "jamfplatform_pro_renew_mdm_profile")
}

func TestRenewMdmProfileAction_Schema(t *testing.T) {
	schema := NewRenewMdmProfileAction().(*RenewMdmProfileAction).Schema
	assertAttrsPresent(t, schema, []string{"udids"})
	assertRequired(t, schema, []string{"udids"})
}

// --- flush_mdm_commands ---

func TestFlushMdmCommandsAction_Metadata(t *testing.T) {
	assertTypeName(t, NewFlushMdmCommandsAction().(*FlushMdmCommandsAction).Metadata, "jamfplatform_pro_flush_mdm_commands")
}

func TestFlushMdmCommandsAction_Schema(t *testing.T) {
	schema := NewFlushMdmCommandsAction().(*FlushMdmCommandsAction).Schema
	assertAttrsPresent(t, schema, []string{"id_type", "id", "status"})
	assertRequired(t, schema, []string{"id_type", "id", "status"})
}

// --- plan-time device-target validation ---

// TestSendBlankPushValidatorsWired covers the at-least-one-of rule: the action
// accepts management_ids and/or serial_numbers, and previously only rejected
// "neither" once the apply was already running.
//
// It asserts which selectors the validator names, not merely that some validator
// is present. The behavioural tests for this rule are tenant-gated acceptance
// tests, so a non-empty check is the whole guard in CI — and it would pass just
// as happily if the re-pointing onto send_blank_push had picked up an unrelated
// validator when the twelve batch actions it used to ride were removed.
func TestSendBlankPushValidatorsWired(t *testing.T) {
	a, ok := NewSendBlankPushAction().(action.ActionWithConfigValidators)
	if !ok {
		t.Fatal("send_blank_push declares no ConfigValidators")
	}
	validators := a.ConfigValidators(context.Background())
	if len(validators) == 0 {
		t.Fatal("send_blank_push declares an empty ConfigValidators slice")
	}
	var described strings.Builder
	for _, v := range validators {
		described.WriteString(v.Description(context.Background()))
		described.WriteString("\n")
	}
	for _, selector := range []string{"management_ids", "serial_numbers"} {
		if !strings.Contains(described.String(), selector) {
			t.Errorf("no ConfigValidator names %q; the at-least-one-of rule is not wired to both selectors:\n%s", selector, described.String())
		}
	}
}
