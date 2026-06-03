// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_check_in_settings

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

// buildComputerCheckInSettingsInput converts the Terraform plan model into an SDK payload.
// The Client Check-In API is a full-replace PUT, so every field is emitted on every
// write.
func buildComputerCheckInSettingsInput(plan ComputerCheckInSettingsResourceModel) *pro.ClientCheckInV3 {
	checkInFrequency := int(plan.CheckInFrequency.ValueInt64())
	createStartupScript := plan.CreateStartupScript.ValueBool()
	startupLog := plan.StartupLog.ValueBool()
	startupPolicies := plan.StartupPolicies.ValueBool()
	startupSsh := plan.StartupSsh.ValueBool()
	createHooks := plan.CreateLoginHook.ValueBool()
	hookLog := plan.LoginHookLog.ValueBool()
	hookPolicies := plan.LoginHookPolicies.ValueBool()
	enableLocalConfigurationProfiles := plan.AllowNetworkStateChangeTriggers.ValueBool()

	return &pro.ClientCheckInV3{
		CheckInFrequency:                 &checkInFrequency,
		CreateStartupScript:              &createStartupScript,
		StartupLog:                       &startupLog,
		StartupPolicies:                  &startupPolicies,
		StartupSsh:                       &startupSsh,
		CreateHooks:                      &createHooks,
		HookLog:                          &hookLog,
		HookPolicies:                     &hookPolicies,
		EnableLocalConfigurationProfiles: &enableLocalConfigurationProfiles,
	}
}
