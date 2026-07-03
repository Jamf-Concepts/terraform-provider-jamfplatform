// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamfprotectactions

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// syncPlansSDKMethods lists the SDK methods the
// jamfplatform_pro_jamf_protect_plans_sync action's Invoke path calls. It
// mirrors the "SDK endpoints used" block in sync_plans.go and drives the
// "Required Jamf privileges" table appended to the action MarkdownDescription.
// permissions_test.go asserts this list stays in sync with the actual
// client.<Method> calls in sync_plans.go and with the SDK privilege registry.
var syncPlansSDKMethods = []string{
	"SyncJamfProtectPlansV1",
}

// syncPlansPrivileges is the rendered "Required Jamf privileges" Markdown
// section for the Jamf Protect plans sync action, appended to its
// MarkdownDescription.
var syncPlansPrivileges = permissions.Section(pro.Privileges, syncPlansSDKMethods...)

// retryDeploymentSDKMethods lists the SDK methods the
// jamfplatform_pro_jamf_protect_deployment_retry action's Invoke path calls
// directly. It mirrors the "SDK endpoints used" block in retry_deployment.go
// and drives the "Required Jamf privileges" table. The computer resolver
// (computertarget.ResolveComputerID) additionally needs Read Computers when a
// serial_number / management_id is supplied; that is noted in the action
// description rather than declared here, matching the redeploy action.
var retryDeploymentSDKMethods = []string{
	"ListJamfProtectDeploymentTasksV1",
	"RetryJamfProtectDeploymentTasksV1",
}

// retryDeploymentPrivileges is the rendered "Required Jamf privileges" Markdown
// section for the Jamf Protect deployment retry action, appended to its
// MarkdownDescription.
var retryDeploymentPrivileges = permissions.Section(pro.Privileges, retryDeploymentSDKMethods...)
