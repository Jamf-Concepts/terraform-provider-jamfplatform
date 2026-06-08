// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package volume_purchasing_notification

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mustStringSet(t *testing.T, vals ...string) types.Set {
	t.Helper()
	if vals == nil {
		vals = []string{}
	}
	set, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("string set build diags: %v", diags)
	}
	return set
}

func mustExternalSet(t *testing.T, recipients ...ExternalRecipientModel) types.Set {
	t.Helper()
	if recipients == nil {
		recipients = []ExternalRecipientModel{}
	}
	set, diags := types.SetValueFrom(context.Background(), externalRecipientObjectType, recipients)
	if diags.HasError() {
		t.Fatalf("external set build diags: %v", diags)
	}
	return set
}

// TestBuildInput_FullReplaceEmitsEmptyCollections verifies that null collections
// emit empty (non-nil) slices so the full-replace write clears the field, and that
// site_id defaults to the no-site sentinel.
func TestBuildInput_FullReplaceEmitsEmptyCollections(t *testing.T) {
	plan := VolumePurchasingNotificationResourceModel{
		Name:               types.StringValue("Test Notification"),
		Enabled:            types.BoolNull(),
		Triggers:           types.SetNull(types.StringType),
		LocationIDs:        types.SetNull(types.StringType),
		InternalRecipients: types.SetNull(types.StringType),
		ExternalRecipients: types.SetNull(externalRecipientObjectType),
		SiteID:             types.StringNull(),
	}
	in, diags := buildVolumePurchasingNotificationInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if in.Name != "Test Notification" {
		t.Errorf("name not carried: %q", in.Name)
	}
	// enabled null → omitted (nil) so the server applies its default.
	if in.Enabled != nil {
		t.Errorf("enabled must be nil when null, got %v", *in.Enabled)
	}
	for name, got := range map[string]*[]string{"triggers": in.Triggers, "locationIds": in.LocationIds} {
		if got == nil {
			t.Fatalf("%s pointer must be non-nil (always-emit)", name)
		}
		if len(*got) != 0 {
			t.Errorf("%s must be empty, got %v", name, *got)
		}
	}
	if in.InternalRecipients == nil || len(*in.InternalRecipients) != 0 {
		t.Errorf("internalRecipients must be empty non-nil, got %v", in.InternalRecipients)
	}
	if in.ExternalRecipients == nil || len(*in.ExternalRecipients) != 0 {
		t.Errorf("externalRecipients must be empty non-nil, got %v", in.ExternalRecipients)
	}
	if in.SiteID == nil || *in.SiteID != siteNone {
		t.Errorf("site_id must default to %q, got %v", siteNone, in.SiteID)
	}
}

// TestBuildInput_PopulatedAndDailyFrequency verifies scalar carry-through and that
// every internal recipient is sent with frequency DAILY.
func TestBuildInput_PopulatedAndDailyFrequency(t *testing.T) {
	plan := VolumePurchasingNotificationResourceModel{
		Name:               types.StringValue("Populated"),
		Enabled:            types.BoolValue(false),
		Triggers:           mustStringSet(t, triggerNoMoreLicenses),
		LocationIDs:        mustStringSet(t, "3", "5"),
		InternalRecipients: mustStringSet(t, "66", "67"),
		ExternalRecipients: mustExternalSet(t, ExternalRecipientModel{Email: types.StringValue("x@example.com"), Name: types.StringValue("X Person")}),
		SiteID:             types.StringValue("12"),
	}
	in, diags := buildVolumePurchasingNotificationInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if in.Enabled == nil || *in.Enabled != false {
		t.Errorf("enabled must be sent as false, got %v", in.Enabled)
	}
	if in.Triggers == nil || len(*in.Triggers) != 1 || (*in.Triggers)[0] != triggerNoMoreLicenses {
		t.Errorf("triggers not carried: %v", in.Triggers)
	}
	if in.LocationIds == nil || len(*in.LocationIds) != 2 {
		t.Errorf("location_ids not carried: %v", in.LocationIds)
	}
	if in.InternalRecipients == nil || len(*in.InternalRecipients) != 2 {
		t.Fatalf("internal recipients not carried: %v", in.InternalRecipients)
	}
	for _, ir := range *in.InternalRecipients {
		if ir.Frequency == nil || *ir.Frequency != frequencyDaily {
			t.Errorf("recipient %q frequency must be %q, got %v", ir.AccountID, frequencyDaily, ir.Frequency)
		}
	}
	if in.ExternalRecipients == nil || len(*in.ExternalRecipients) != 1 {
		t.Fatalf("external recipients not carried: %v", in.ExternalRecipients)
	}
	if got := (*in.ExternalRecipients)[0]; got.Email != "x@example.com" || got.Name != "X Person" {
		t.Errorf("external recipient mismatch: %+v", got)
	}
	if in.SiteID == nil || *in.SiteID != "12" {
		t.Errorf("site_id not carried: %v", in.SiteID)
	}
}
