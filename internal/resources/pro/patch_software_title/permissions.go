// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// resourceSDKMethods lists the SDK methods the patch software title resource's
// CRUD path calls. The classic /patchsoftwaretitles CRUD runs through the
// ProClassic client (crud.go); extension-attribute read + accept run through the
// Pro v2 client (extension_attributes.go, invoked from Create/Read/Update). It
// drives the "Required Jamf privileges" table appended to the resource
// MarkdownDescription. permissions_test.go asserts this list stays in sync with
// the actual <client>.<Method> calls in crud.go + extension_attributes.go and
// with the SDK privilege registries.
var resourceSDKMethods = []string{
	"CreatePatchSoftwareTitleByID",
	"GetPatchSoftwareTitleByID",
	"UpdatePatchSoftwareTitleByID",
	"DeletePatchSoftwareTitleByID",
	"ListPatchSoftwareTitleExtensionAttributesV2",
	"UpdatePatchSoftwareTitleConfigurationV2",
}

// resourcePrivileges is the rendered "Required Jamf privileges" Markdown section
// for the patch software title resource, appended to its MarkdownDescription.
// The resource spans two SDK families (ProClassic CRUD + Pro v2 extension
// attributes), so the registries are merged.
var resourcePrivileges = permissions.Section(
	permissions.Merge(proclassic.Privileges, pro.Privileges),
	resourceSDKMethods...,
)

// dataSourceSDKMethods lists the SDK methods the patch software title data
// source calls (data_source.go). Lookup is read-only by ID.
var dataSourceSDKMethods = []string{
	"GetPatchSoftwareTitleByID",
}

// dataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the patch software title data source.
var dataSourcePrivileges = permissions.Section(proclassic.Privileges, dataSourceSDKMethods...)

// listResourceSDKMethods lists the SDK methods the patch software title list
// resource calls (list_resource.go).
var listResourceSDKMethods = []string{
	"ListPatchSoftwareTitles",
}

// listResourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the patch software title list resource.
var listResourcePrivileges = permissions.Section(proclassic.Privileges, listResourceSDKMethods...)
