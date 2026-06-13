// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patchactions

import (
	"context"
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

// --- retry_patch_policy_logs ---

func TestRetryPatchPolicyLogsAction_Metadata(t *testing.T) {
	assertTypeName(t, NewRetryPatchPolicyLogsAction().(*RetryPatchPolicyLogsAction).Metadata, "jamfplatform_pro_retry_patch_policy_logs")
}

func TestRetryPatchPolicyLogsAction_Schema(t *testing.T) {
	schema := NewRetryPatchPolicyLogsAction().(*RetryPatchPolicyLogsAction).Schema
	assertAttrsPresent(t, schema, []string{"patch_policy_id", "device_ids"})
	assertRequired(t, schema, []string{"patch_policy_id"})
}
