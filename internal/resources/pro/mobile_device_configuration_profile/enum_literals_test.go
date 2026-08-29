// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

import (
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/enumguard"
)

// TestEnumLiteralsComeFromTheSDK pins STYLE_GUIDE.md §"Enum values and error
// codes come from the SDK, not from literals" for this package.
//
// The distribution method and self_service.notification_location literals are
// deliberately not checked against any vocabulary. Both collide with a
// same-spelled set belonging to a different construct — the macOS profile's
// distribution method, and the patch policy's notification location — and the
// mobile-device profile spec generates neither of its own. Naming a foreign
// vocabulary in Covered just to exempt the collision would make the guard report
// a promotion the day that unrelated spec changed.
func TestEnumLiteralsComeFromTheSDK(t *testing.T) {
	got, err := enumguard.Check(enumguard.Params{
		Covered: enumguard.Union(
			proclassic.MobileDeviceConfigurationProfileGeneralLevelValues(),
		),
		Absent: map[string]string{
			"Device": "the write-side spelling for Device Level. proclassic.MobileDeviceConfigurationProfileGeneralLevel generates only the read-side pair (System / User), which the read constants alias \u2014 as does the User Level write constant, that spelling being symmetric across write and read",
		},
	})
	if err != nil {
		t.Fatalf("enumguard.Check: %v", err)
	}
	for _, problem := range got.Problems() {
		t.Error(problem)
	}
	if got.Examined == 0 {
		t.Fatal("no string literals parsed — the guard found nothing to check")
	}
}
