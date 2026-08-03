// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package maintenanceactions

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
	assertAttrsPresent(t, schema, []string{"policy_id", "quantity", "period"})
	assertRequired(t, schema, []string{"policy_id", "quantity", "period"})
}

// TestFlushPolicyLogsInterval covers the join into the single path token Jamf Pro
// expects. The "+" is the endpoint's own encoding of the space in "Six Months".
func TestFlushPolicyLogsInterval(t *testing.T) {
	for _, tc := range []struct {
		quantity, period, want string
	}{
		{"Six", "Months", "Six+Months"},
		{"Zero", "Days", "Zero+Days"},
		{"Three", "Years", "Three+Years"},
	} {
		if got := logFlushInterval(tc.quantity, tc.period); got != tc.want {
			t.Errorf("logFlushInterval(%q, %q) = %q, want %q", tc.quantity, tc.period, got, tc.want)
		}
	}
}

// TestFlushPolicyLogsVocabulary pins the quantity set. It is deliberately
// non-contiguous — Jamf Pro has no Four or Five, and an out-of-set quantity is a
// server 500 rather than a validation error, so the OneOf is load-bearing.
func TestFlushPolicyLogsVocabulary(t *testing.T) {
	wantQuantities := []string{"Zero", "One", "Two", "Three", "Six"}
	if len(logFlushQuantities) != len(wantQuantities) {
		t.Fatalf("logFlushQuantities = %v, want %v", logFlushQuantities, wantQuantities)
	}
	for i, q := range wantQuantities {
		if logFlushQuantities[i] != q {
			t.Errorf("logFlushQuantities[%d] = %q, want %q", i, logFlushQuantities[i], q)
		}
	}

	wantPeriods := []string{"Days", "Weeks", "Months", "Years"}
	if len(logFlushPeriods) != len(wantPeriods) {
		t.Fatalf("logFlushPeriods = %v, want %v", logFlushPeriods, wantPeriods)
	}
	for i, p := range wantPeriods {
		if logFlushPeriods[i] != p {
			t.Errorf("logFlushPeriods[%d] = %q, want %q", i, logFlushPeriods[i], p)
		}
	}
}

// TestFlushPolicyLogsDocumentsItsValues guards the docs half of the contract:
// tfplugindocs does not render validators, so the accepted values are invisible
// to users unless the description enumerates them. Both lists are derived from
// the same slices the OneOf validators use, so this catches a slice edited
// without a matching description.
func TestFlushPolicyLogsDocumentsItsValues(t *testing.T) {
	var resp action.SchemaResponse
	NewFlushPolicyLogsAction().(*FlushPolicyLogsAction).Schema(context.Background(), action.SchemaRequest{}, &resp)

	for attr, values := range map[string][]string{
		"quantity": logFlushQuantities,
		"period":   logFlushPeriods,
	} {
		desc := resp.Schema.Attributes[attr].GetMarkdownDescription()
		for _, v := range values {
			if !strings.Contains(desc, "`"+v+"`") {
				t.Errorf("%s description does not document accepted value %q:\n%s", attr, v, desc)
			}
		}
	}
}

// TestRedeployComputerTargetValidatorsWired guards the exactly-one-of
// ConfigValidator for the three-way management_id / serial_number / udid
// selector. Without it, a config naming no computer passes plan.
func TestRedeployComputerTargetValidatorsWired(t *testing.T) {
	a, ok := NewRedeployManagementFrameworkAction().(action.ActionWithConfigValidators)
	if !ok {
		t.Fatal("redeploy_management_framework declares no ConfigValidators")
	}
	if len(a.ConfigValidators(context.Background())) == 0 {
		t.Fatal("redeploy_management_framework declares an empty ConfigValidators slice")
	}
}
