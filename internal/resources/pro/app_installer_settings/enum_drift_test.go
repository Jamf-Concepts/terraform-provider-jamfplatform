// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer_settings

import (
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

// days_of_week validates against explicit SDK constants rather than
// AppInstallerDeploymentProcessControlsDaysOfWeekValues(), so an SDK bump cannot
// silently widen what the attribute accepts. This is the tripwire for the
// converse: a value Jamf adds passing unnoticed. Seven days is a closed set in
// practice, which is exactly why a change here would be worth looking at.
func TestDaysOfWeekEnum_HasNotGrown(t *testing.T) {
	want := map[string]bool{
		pro.AppInstallerDeploymentProcessControlsDaysOfWeekMonday:    true,
		pro.AppInstallerDeploymentProcessControlsDaysOfWeekTuesday:   true,
		pro.AppInstallerDeploymentProcessControlsDaysOfWeekWednesday: true,
		pro.AppInstallerDeploymentProcessControlsDaysOfWeekThursday:  true,
		pro.AppInstallerDeploymentProcessControlsDaysOfWeekFriday:    true,
		pro.AppInstallerDeploymentProcessControlsDaysOfWeekSaturday:  true,
		pro.AppInstallerDeploymentProcessControlsDaysOfWeekSunday:    true,
	}
	got := pro.AppInstallerDeploymentProcessControlsDaysOfWeekValues()
	for _, v := range got {
		if !want[v] {
			t.Errorf("AppInstallerDeploymentProcessControlsDaysOfWeek gained value %q: update the days_of_week validator", v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("AppInstallerDeploymentProcessControlsDaysOfWeek has %d values, schema validates %d", len(got), len(want))
	}
}
