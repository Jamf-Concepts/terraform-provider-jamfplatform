// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package uemconnectactions

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// synchronizeSDKMethods lists the SDK methods the
// jamfplatform_security_cloud_uem_connect_synchronize action's Invoke path calls.
// It mirrors the "SDK endpoints used" block in synchronize.go and drives the
// "Required Jamf privileges" table appended to the action MarkdownDescription.
// permissions_test.go asserts this list stays in sync with the actual
// client.<Method> calls and with the SDK privilege registry.
//
// The list read is declared because the action falls back to it to find the
// tenant's only integration when no ID is configured — so a caller granted only
// the update privilege would still fail on the common, ID-less form.
var synchronizeSDKMethods = []string{
	"TriggerUemConnectorSyncV1",
	"ListUemConnectorsV1",
}

// synchronizePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the synchronize action.
var synchronizePrivileges = permissions.Section(securitycloud.Privileges, synchronizeSDKMethods...)
