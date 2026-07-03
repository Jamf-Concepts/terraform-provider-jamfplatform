// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package maintenanceactions

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

// --- redeploy_management_framework ---

func TestRedeployManagementFrameworkAction_Metadata(t *testing.T) {
	assertTypeName(t, NewRedeployManagementFrameworkAction().(*RedeployManagementFrameworkAction).Metadata, "jamfplatform_pro_redeploy_management_framework")
}

func TestRedeployManagementFrameworkAction_Schema(t *testing.T) {
	schema := NewRedeployManagementFrameworkAction().(*RedeployManagementFrameworkAction).Schema
	assertAttrsPresent(t, schema, []string{"management_id", "serial_number", "udid"})
}

// --- flush_policy_logs ---

func TestFlushPolicyLogsAction_Metadata(t *testing.T) {
	assertTypeName(t, NewFlushPolicyLogsAction().(*FlushPolicyLogsAction).Metadata, "jamfplatform_pro_flush_policy_logs")
}

func TestFlushPolicyLogsAction_Schema(t *testing.T) {
	schema := NewFlushPolicyLogsAction().(*FlushPolicyLogsAction).Schema
	assertAttrsPresent(t, schema, []string{"policy_id", "interval"})
	assertRequired(t, schema, []string{"policy_id", "interval"})
}
