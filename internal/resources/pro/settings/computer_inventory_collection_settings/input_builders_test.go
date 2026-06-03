// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_inventory_collection_settings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOptBool(t *testing.T) {
	if optBool(types.BoolNull()) != nil {
		t.Error("null bool must map to nil (merge-patch omit)")
	}
	if optBool(types.BoolUnknown()) != nil {
		t.Error("unknown bool must map to nil (merge-patch omit)")
	}
	tr := optBool(types.BoolValue(true))
	if tr == nil || *tr != true {
		t.Errorf("true bool must map to &true, got %v", tr)
	}
	fa := optBool(types.BoolValue(false))
	if fa == nil || *fa != false {
		t.Errorf("false bool must map to &false (explicit off is sent), got %v", fa)
	}
}

// TestBuildInput_OmitsUnsetSendsSet verifies the merge-patch contract: attributes the
// user did not set (null) are omitted from the payload, while set attributes round-trip
// as concrete pointers. applicationPaths must never be carried in the settings payload.
func TestBuildInput_OmitsUnsetSendsSet(t *testing.T) {
	plan := ComputerInventoryCollectionSettingsResourceModel{
		CollectLocalUserAccounts: types.BoolValue(true),  // set → sent
		CollectPrinters:          types.BoolValue(false), // set → sent (explicit off)
		IncludeHiddenAccounts:    types.BoolNull(),       // unset → omitted
	}
	in := buildComputerInventoryCollectionSettingsInput(plan)
	if in.ApplicationPaths != nil {
		t.Error("settings payload must never carry applicationPaths")
	}
	p := in.ComputerInventoryCollectionPreferences
	if p.IncludeAccounts == nil || *p.IncludeAccounts != true {
		t.Errorf("IncludeAccounts: want &true, got %v", p.IncludeAccounts)
	}
	if p.IncludePrinters == nil || *p.IncludePrinters != false {
		t.Errorf("IncludePrinters: want &false, got %v", p.IncludePrinters)
	}
	if p.IncludeHiddenAccounts != nil {
		t.Errorf("IncludeHiddenAccounts: unset attr must be omitted (nil), got %v", *p.IncludeHiddenAccounts)
	}
}
