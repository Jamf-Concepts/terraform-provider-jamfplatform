// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_prestage_enrollment

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIsKnownTrue(t *testing.T) {
	cases := map[string]struct {
		in   types.Bool
		want bool
	}{
		"known true":  {types.BoolValue(true), true},
		"known false": {types.BoolValue(false), false},
		"null":        {types.BoolNull(), false},
		"unknown":     {types.BoolUnknown(), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := isKnownTrue(tc.in); got != tc.want {
				t.Errorf("isKnownTrue(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolveIntentBool(t *testing.T) {
	cases := map[string]struct {
		plan, cfg, state types.Bool
		want             types.Bool
	}{
		"known plan wins":                       {types.BoolValue(true), types.BoolValue(false), types.BoolValue(false), types.BoolValue(true)},
		"unknown plan -> config":                {types.BoolUnknown(), types.BoolValue(true), types.BoolValue(false), types.BoolValue(true)},
		"unknown plan, null config -> state":    {types.BoolUnknown(), types.BoolNull(), types.BoolValue(true), types.BoolValue(true)},
		"unknown plan, all null -> false":       {types.BoolUnknown(), types.BoolNull(), types.BoolNull(), types.BoolValue(false)},
		"unknown plan, unknown config -> state": {types.BoolUnknown(), types.BoolUnknown(), types.BoolValue(true), types.BoolValue(true)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := resolveIntentBool(tc.plan, tc.cfg, tc.state)
			if !got.Equal(tc.want) {
				t.Errorf("resolveIntentBool(%v,%v,%v) = %v, want %v", tc.plan, tc.cfg, tc.state, got, tc.want)
			}
		})
	}
}

// TestRestoreServerArbitrated_StateCarryPath models the §F9 mutual-exclusion
// state-carry case: ModifyPlan rendered both shared-iPad mode flags Unknown;
// the user's config enabled temporary_session_only and omitted
// use_storage_quota_size (carried true from prior state). Restore must send
// BOTH true on the wire (the server then forces one false) — a bare Unknown
// would serialise as false and silently disable the user's intent.
func TestRestoreServerArbitrated_StateCarryPath(t *testing.T) {
	plan := MobileDevicePrestageEnrollmentResourceModel{
		DefaultPrestage:      types.BoolUnknown(),
		UseStorageQuotaSize:  types.BoolUnknown(),
		TemporarySessionOnly: types.BoolUnknown(),
	}
	cfg := MobileDevicePrestageEnrollmentResourceModel{
		DefaultPrestage:      types.BoolValue(true),
		UseStorageQuotaSize:  types.BoolNull(), // user omitted
		TemporarySessionOnly: types.BoolValue(true),
	}
	state := MobileDevicePrestageEnrollmentResourceModel{
		UseStorageQuotaSize: types.BoolValue(true), // carried from prior apply
	}

	restoreServerArbitrated(&plan, cfg, state)

	if !plan.DefaultPrestage.Equal(types.BoolValue(true)) {
		t.Errorf("default_prestage intent lost: %v", plan.DefaultPrestage)
	}
	if !plan.UseStorageQuotaSize.Equal(types.BoolValue(true)) {
		t.Errorf("use_storage_quota_size intent lost (state-carry): %v", plan.UseStorageQuotaSize)
	}
	if !plan.TemporarySessionOnly.Equal(types.BoolValue(true)) {
		t.Errorf("temporary_session_only intent lost: %v", plan.TemporarySessionOnly)
	}
}

// TestRestoreServerArbitrated_CreateNoState confirms create (zero-value state)
// sources intent from config and defaults a wholly-absent field to false
// rather than leaking Unknown into the POST body.
func TestRestoreServerArbitrated_CreateNoState(t *testing.T) {
	plan := MobileDevicePrestageEnrollmentResourceModel{
		DefaultPrestage:      types.BoolUnknown(),
		UseStorageQuotaSize:  types.BoolValue(false),
		TemporarySessionOnly: types.BoolUnknown(),
	}
	cfg := MobileDevicePrestageEnrollmentResourceModel{
		DefaultPrestage:      types.BoolValue(true),
		UseStorageQuotaSize:  types.BoolValue(false),
		TemporarySessionOnly: types.BoolNull(),
	}

	restoreServerArbitrated(&plan, cfg, MobileDevicePrestageEnrollmentResourceModel{})

	if !plan.DefaultPrestage.Equal(types.BoolValue(true)) {
		t.Errorf("default_prestage = %v, want true (from config)", plan.DefaultPrestage)
	}
	if !plan.TemporarySessionOnly.Equal(types.BoolValue(false)) {
		t.Errorf("temporary_session_only = %v, want false (no config/state)", plan.TemporarySessionOnly)
	}
}
