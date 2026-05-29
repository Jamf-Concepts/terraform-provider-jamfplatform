// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package re_enrollment_settings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildReenrollmentInput_FullRoundTrip confirms all six fields are
// populated from a fully-authored plan, with the bools sent as concrete
// pointers and the enum passed through verbatim.
func TestBuildReenrollmentInput_FullRoundTrip(t *testing.T) {
	plan := ReEnrollmentSettingsResourceModel{
		ClearPolicyLogs:                 types.BoolValue(true),
		ClearLocationInformation:        types.BoolValue(false),
		ClearLocationInformationHistory: types.BoolValue(true),
		ClearExtensionAttributes:        types.BoolValue(false),
		ClearSoftwareUpdatePlans:        types.BoolValue(true),
		ClearManagementHistory:          types.StringValue("DELETE_EVERYTHING"),
	}

	out := buildReenrollmentInput(plan)

	if out.IsFlushPolicyHistoryEnabled == nil || *out.IsFlushPolicyHistoryEnabled != true {
		t.Errorf("IsFlushPolicyHistoryEnabled = %v, want true", out.IsFlushPolicyHistoryEnabled)
	}
	if out.IsFlushLocationInformationEnabled == nil || *out.IsFlushLocationInformationEnabled != false {
		t.Errorf("IsFlushLocationInformationEnabled = %v, want false", out.IsFlushLocationInformationEnabled)
	}
	if out.IsFlushLocationInformationHistoryEnabled == nil || *out.IsFlushLocationInformationHistoryEnabled != true {
		t.Errorf("IsFlushLocationInformationHistoryEnabled = %v, want true", out.IsFlushLocationInformationHistoryEnabled)
	}
	if out.IsFlushExtensionAttributesEnabled == nil || *out.IsFlushExtensionAttributesEnabled != false {
		t.Errorf("IsFlushExtensionAttributesEnabled = %v, want false", out.IsFlushExtensionAttributesEnabled)
	}
	if out.IsFlushSoftwareUpdatePlansEnabled == nil || *out.IsFlushSoftwareUpdatePlansEnabled != true {
		t.Errorf("IsFlushSoftwareUpdatePlansEnabled = %v, want true", out.IsFlushSoftwareUpdatePlansEnabled)
	}
	if out.FlushMDMQueue != "DELETE_EVERYTHING" {
		t.Errorf("FlushMDMQueue = %q, want DELETE_EVERYTHING", out.FlushMDMQueue)
	}
}

// TestBuildReenrollmentInput_AllBoolsPopulated confirms that even when the
// flush bools are null/unknown they are sent as concrete (false) pointers —
// the wire write is full-replace, so a nil pointer (omit) is never produced.
func TestBuildReenrollmentInput_AllBoolsPopulated(t *testing.T) {
	plan := ReEnrollmentSettingsResourceModel{
		ClearPolicyLogs:                 types.BoolNull(),
		ClearLocationInformation:        types.BoolUnknown(),
		ClearLocationInformationHistory: types.BoolNull(),
		ClearExtensionAttributes:        types.BoolNull(),
		ClearSoftwareUpdatePlans:        types.BoolNull(),
		ClearManagementHistory:          types.StringValue("DELETE_NOTHING"),
	}

	out := buildReenrollmentInput(plan)

	for name, p := range map[string]*bool{
		"IsFlushPolicyHistoryEnabled":              out.IsFlushPolicyHistoryEnabled,
		"IsFlushLocationInformationEnabled":        out.IsFlushLocationInformationEnabled,
		"IsFlushLocationInformationHistoryEnabled": out.IsFlushLocationInformationHistoryEnabled,
		"IsFlushExtensionAttributesEnabled":        out.IsFlushExtensionAttributesEnabled,
		"IsFlushSoftwareUpdatePlansEnabled":        out.IsFlushSoftwareUpdatePlansEnabled,
	} {
		if p == nil {
			t.Errorf("%s must be a non-nil pointer (full-replace write)", name)
			continue
		}
		if *p != false {
			t.Errorf("%s = %v, want false for null/unknown plan input", name, *p)
		}
	}
}

// TestBuildReenrollmentInput_EnumPassthrough confirms the Required
// clear_management_history enum is sent verbatim. It is always known (Required),
// so there is no default-substitution path.
func TestBuildReenrollmentInput_EnumPassthrough(t *testing.T) {
	for _, v := range validClearManagementHistory {
		t.Run(v, func(t *testing.T) {
			plan := ReEnrollmentSettingsResourceModel{ClearManagementHistory: types.StringValue(v)}
			out := buildReenrollmentInput(plan)
			if out.FlushMDMQueue != v {
				t.Errorf("FlushMDMQueue = %q, want %q", out.FlushMDMQueue, v)
			}
		})
	}
}
