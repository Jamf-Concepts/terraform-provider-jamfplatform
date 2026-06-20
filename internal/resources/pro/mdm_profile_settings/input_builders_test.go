// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdm_profile_settings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildMDMProfileSettingsInput_FieldMapping pins each Terraform plan
// attribute to its exact SDK pointer. The bool labels are easy to transpose — notably
// `..._before_expiry` maps to the `...WhenDeviceIdentityCertExpiring` wire field — so
// every pairing is asserted individually rather than as a loose round-trip.
func TestBuildMDMProfileSettingsInput_FieldMapping(t *testing.T) {
	plan := MDMProfileSettingsResourceModel{
		AutoRenewComputerProfileWhenCaRenewed:     types.BoolValue(true),
		AutoRenewComputerProfileBeforeExpiry:      types.BoolValue(false),
		ComputerProfileExpirationLimitDays:        types.Int64Value(180),
		AutoRenewMobileDeviceProfileWhenCaRenewed: types.BoolValue(false),
		AutoRenewMobileDeviceProfileBeforeExpiry:  types.BoolValue(true),
		MobileDeviceProfileExpirationLimitDays:    types.Int64Value(90),
	}

	got := buildMDMProfileSettingsInput(plan)

	assertBool(t, "AutoRenewComputerMDMProfileWhenCaRenewed", got.AutoRenewComputerMDMProfileWhenCaRenewed, true)
	assertBool(t, "AutoRenewComputerMDMProfileWhenDeviceIdentityCertExpiring", got.AutoRenewComputerMDMProfileWhenDeviceIdentityCertExpiring, false)
	assertBool(t, "AutoRenewMobileDeviceMDMProfileWhenCaRenewed", got.AutoRenewMobileDeviceMDMProfileWhenCaRenewed, false)
	assertBool(t, "AutoRenewMobileDeviceMDMProfileWhenDeviceIdentityCertExpiring", got.AutoRenewMobileDeviceMDMProfileWhenDeviceIdentityCertExpiring, true)
	assertInt(t, "MDMProfileComputerExpirationLimitInDays", got.MDMProfileComputerExpirationLimitInDays, 180)
	assertInt(t, "MDMProfileMobileDeviceExpirationLimitInDays", got.MDMProfileMobileDeviceExpirationLimitInDays, 90)
}

// TestBuildMDMProfileSettingsInput_InvertedValues guards against a copy-paste
// swap between the computer and mobile-device fields by flipping every value relative
// to the case above.
func TestBuildMDMProfileSettingsInput_InvertedValues(t *testing.T) {
	plan := MDMProfileSettingsResourceModel{
		AutoRenewComputerProfileWhenCaRenewed:     types.BoolValue(false),
		AutoRenewComputerProfileBeforeExpiry:      types.BoolValue(true),
		ComputerProfileExpirationLimitDays:        types.Int64Value(7),
		AutoRenewMobileDeviceProfileWhenCaRenewed: types.BoolValue(true),
		AutoRenewMobileDeviceProfileBeforeExpiry:  types.BoolValue(false),
		MobileDeviceProfileExpirationLimitDays:    types.Int64Value(365),
	}

	got := buildMDMProfileSettingsInput(plan)

	assertBool(t, "AutoRenewComputerMDMProfileWhenCaRenewed", got.AutoRenewComputerMDMProfileWhenCaRenewed, false)
	assertBool(t, "AutoRenewComputerMDMProfileWhenDeviceIdentityCertExpiring", got.AutoRenewComputerMDMProfileWhenDeviceIdentityCertExpiring, true)
	assertBool(t, "AutoRenewMobileDeviceMDMProfileWhenCaRenewed", got.AutoRenewMobileDeviceMDMProfileWhenCaRenewed, true)
	assertBool(t, "AutoRenewMobileDeviceMDMProfileWhenDeviceIdentityCertExpiring", got.AutoRenewMobileDeviceMDMProfileWhenDeviceIdentityCertExpiring, false)
	assertInt(t, "MDMProfileComputerExpirationLimitInDays", got.MDMProfileComputerExpirationLimitInDays, 7)
	assertInt(t, "MDMProfileMobileDeviceExpirationLimitInDays", got.MDMProfileMobileDeviceExpirationLimitInDays, 365)
}

func assertBool(t *testing.T, field string, p *bool, want bool) {
	t.Helper()
	if p == nil {
		t.Fatalf("%s: expected non-nil pointer", field)
	}
	if *p != want {
		t.Errorf("%s = %v, want %v", field, *p, want)
	}
}

func assertInt(t *testing.T, field string, p *int, want int) {
	t.Helper()
	if p == nil {
		t.Fatalf("%s: expected non-nil pointer", field)
	}
	if *p != want {
		t.Errorf("%s = %d, want %d", field, *p, want)
	}
}
