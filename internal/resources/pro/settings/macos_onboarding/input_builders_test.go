// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_onboarding

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildOnboardingInput_StampsPriorityFromIndex verifies priority is derived from
// the list index (1-based) regardless of input order, and that only the three writable
// fields are emitted (id and the echoes are left unset).
func TestBuildOnboardingInput_StampsPriorityFromIndex(t *testing.T) {
	items := []onboardingItemModel{
		{EntityID: types.StringValue("4"), SelfServiceEntityType: types.StringValue(entityTypePolicy), Priority: types.Int64Value(99), EntityName: types.StringValue("ignored")},
		{EntityID: types.StringValue("11"), SelfServiceEntityType: types.StringValue(entityTypeConfigProfile)},
		{EntityID: types.StringValue("87"), SelfServiceEntityType: types.StringValue(entityTypeMacApp)},
	}

	got := buildOnboardingInput(true, items)

	if !got.Enabled {
		t.Errorf("enabled = false, want true")
	}
	if len(got.OnboardingItems) != 3 {
		t.Fatalf("got %d items, want 3", len(got.OnboardingItems))
	}
	for i, it := range got.OnboardingItems {
		wantPriority := i + 1
		if it.Priority != wantPriority {
			t.Errorf("item %d priority = %d, want %d (index-derived, ignoring any planned priority)", i, it.Priority, wantPriority)
		}
		if it.ID != nil || it.EntityName != nil || it.ScopeDescription != nil || it.SiteDescription != nil {
			t.Errorf("item %d: server-derived fields must be unset on write, got id=%v name=%v", i, it.ID, it.EntityName)
		}
	}
	if got.OnboardingItems[0].EntityID != "4" || got.OnboardingItems[0].SelfServiceEntityType != entityTypePolicy {
		t.Errorf("item 0 = %+v, want entityId 4 / %s", got.OnboardingItems[0], entityTypePolicy)
	}
}

// TestBuildOnboardingInput_EmptyItems verifies an empty list yields a non-nil empty
// array (the full-replace "clear all" body) with the enabled flag preserved.
func TestBuildOnboardingInput_EmptyItems(t *testing.T) {
	got := buildOnboardingInput(false, []onboardingItemModel{})
	if got.Enabled {
		t.Errorf("enabled = true, want false")
	}
	if got.OnboardingItems == nil {
		t.Errorf("OnboardingItems is nil, want non-nil empty slice (clears all items on full-replace)")
	}
	if len(got.OnboardingItems) != 0 {
		t.Errorf("got %d items, want 0", len(got.OnboardingItems))
	}
}
