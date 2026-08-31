// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// resourceSDKMethods lists the SDK methods the patch policy resource's CRUD path
// calls. It mirrors the "SDK endpoints used" block in crud.go and drives the
// "Required Jamf permissions" table appended to the resource MarkdownDescription.
// permissions_test.go asserts this list stays in sync with the actual
// client.<Method> calls in crud.go and with the SDK privilege registry.
//
// Note: the resource is bucketed under the pro/ tree to mirror the Jamf Pro
// admin UI, but its SDK client is *proclassic.Client, so the privileges come
// from the proclassic registry.
var resourceSDKMethods = []string{
	"CreatePatchPolicyBySoftwareTitleConfigID",
	"GetPatchPolicyByID",
	"UpdatePatchPolicyByID",
	"DeletePatchPolicyByID",
}

// resourcePrivileges is the rendered "Required Jamf permissions" Markdown
// section for the patch policy resource, appended to its MarkdownDescription.
var resourcePrivileges = permissions.Section(proclassic.Privileges, resourceSDKMethods...)

// dataSourceSDKMethods lists the SDK methods the patch policy data source calls.
// A data source documents only the privileges IT needs (a single read), not the
// resource's full CRUD set.
var dataSourceSDKMethods = []string{
	"GetPatchPolicyByID",
}

// dataSourcePrivileges is the rendered "Required Jamf permissions" Markdown
// section for the patch policy data source.
var dataSourcePrivileges = permissions.Section(proclassic.Privileges, dataSourceSDKMethods...)

// listResourceSDKMethods lists the SDK methods the patch policy list resource
// calls: the list endpoint plus the per-item GET issued during config
// generation (IncludeResource).
var listResourceSDKMethods = []string{
	"ListPatchPolicies",
	"GetPatchPolicyByID",
}

// listResourcePrivileges is the rendered "Required Jamf permissions" Markdown
// section for the patch policy list resource.
var listResourcePrivileges = permissions.Section(proclassic.Privileges, listResourceSDKMethods...)
