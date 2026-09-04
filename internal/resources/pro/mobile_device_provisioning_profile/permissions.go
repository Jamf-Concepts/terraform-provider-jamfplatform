// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_provisioning_profile

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// resourceSDKMethods lists the SDK methods the provisioning profile resource's
// CRUD path (crud.go) calls. It drives the "Required Jamf permissions" table
// appended to the resource MarkdownDescription. permissions_test.go asserts this
// list stays in sync with the actual client.<Method> calls in crud.go and with
// the SDK privilege registry.
var resourceSDKMethods = []string{
	"CreateMobileDeviceProvisioningProfileByID",
	"GetMobileDeviceProvisioningProfileByID",
	"DeleteMobileDeviceProvisioningProfileByID",
}

// resourcePrivileges is the rendered "Required Jamf permissions" Markdown
// section for the provisioning profile resource, appended to its MarkdownDescription.
var resourcePrivileges = permissions.Section(proclassic.Privileges, resourceSDKMethods...)

// dataSourceSDKMethods lists the SDK methods the provisioning profile data
// source (data_source.go) calls for its id / name / uuid lookups.
var dataSourceSDKMethods = []string{
	"GetMobileDeviceProvisioningProfileByID",
	"GetMobileDeviceProvisioningProfileByName",
	"GetMobileDeviceProvisioningProfileByUUID",
}

// dataSourcePrivileges is the rendered "Required Jamf permissions" Markdown
// section for the provisioning profile data source, appended to its MarkdownDescription.
var dataSourcePrivileges = permissions.Section(proclassic.Privileges, dataSourceSDKMethods...)

// listResourceSDKMethods lists the SDK methods the provisioning profile list
// resource (list_resource.go) calls.
var listResourceSDKMethods = []string{
	"ListMobileDeviceProvisioningProfiles",
}

// listResourcePrivileges is the rendered "Required Jamf permissions" Markdown
// section for the provisioning profile list resource, appended to its description.
var listResourcePrivileges = permissions.Section(proclassic.Privileges, listResourceSDKMethods...)
