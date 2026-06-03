// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_check_in_settings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildComputerCheckInSettingsInput verifies every plan field maps to the correct SDK
// pointer. Distinct bool values per field catch a swapped mapping.
func TestBuildComputerCheckInSettingsInput(t *testing.T) {
	plan := ComputerCheckInSettingsResourceModel{
		CheckInFrequency:                types.Int64Value(60),
		CreateStartupScript:             types.BoolValue(true),
		StartupLog:                      types.BoolValue(false),
		StartupPolicies:                 types.BoolValue(true),
		StartupSsh:                      types.BoolValue(false),
		CreateLoginHook:                 types.BoolValue(true),
		LoginHookLog:                    types.BoolValue(false),
		LoginHookPolicies:               types.BoolValue(true),
		AllowNetworkStateChangeTriggers: types.BoolValue(false),
	}

	out := buildComputerCheckInSettingsInput(plan)

	if out.CheckInFrequency == nil || *out.CheckInFrequency != 60 {
		t.Errorf("CheckInFrequency = %v, want 60", out.CheckInFrequency)
	}

	checks := []struct {
		name string
		got  *bool
		want bool
	}{
		{"CreateStartupScript", out.CreateStartupScript, true},
		{"StartupLog", out.StartupLog, false},
		{"StartupPolicies", out.StartupPolicies, true},
		{"StartupSsh", out.StartupSsh, false},
		{"CreateHooks (create_login_hook)", out.CreateHooks, true},
		{"HookLog (login_hook_log)", out.HookLog, false},
		{"HookPolicies (login_hook_policies)", out.HookPolicies, true},
		{"EnableLocalConfigurationProfiles (allow_network_state_change_triggers)", out.EnableLocalConfigurationProfiles, false},
	}
	for _, c := range checks {
		if c.got == nil {
			t.Errorf("%s: nil pointer, want %v", c.name, c.want)
			continue
		}
		if *c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, *c.got, c.want)
		}
	}
}
