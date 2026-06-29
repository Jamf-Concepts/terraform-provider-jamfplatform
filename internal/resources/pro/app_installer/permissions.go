// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// resourceSDKMethods lists the SDK methods the App Installer resource's CRUD
// path calls. It mirrors the "SDK endpoints used" block in crud.go and drives
// the "Required Jamf privileges" table appended to the resource
// MarkdownDescription. permissions_test.go asserts this list stays in sync with
// the actual client.<Method> calls in crud.go and with the SDK privilege
// registry.
var resourceSDKMethods = []string{
	"CreateAppInstallerDeploymentV1",
	"GetAppInstallerDeploymentV1",
	"UpdateAppInstallerDeploymentV1",
	"DeleteAppInstallerDeploymentV1",
}

// resourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the App Installer resource, appended to its MarkdownDescription.
var resourcePrivileges = permissions.Section(pro.Privileges, resourceSDKMethods...)

// dataSourceSDKMethods lists the registry-tracked SDK methods the singular App
// Installer data source calls. data_source.go also calls
// ResolveAppInstallerDeploymentV1IDByName, but that resolver is a list-backed
// convenience wrapper with no own privilege registry entry; it issues the same
// read:pro:mac-applications request as the deployment GET, so the GET covers the
// data source's privilege requirement. permissions_test.go filters the
// discovered calls to registry-known methods before comparing.
var dataSourceSDKMethods = []string{
	"GetAppInstallerDeploymentV1",
}

// dataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the singular App Installer data source.
var dataSourcePrivileges = permissions.Section(pro.Privileges, dataSourceSDKMethods...)

// pluralDataSourceSDKMethods lists the SDK methods the plural App Installers
// data source calls.
var pluralDataSourceSDKMethods = []string{
	"ListAppInstallerDeploymentsV1",
}

// pluralDataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the plural App Installers data source.
var pluralDataSourcePrivileges = permissions.Section(pro.Privileges, pluralDataSourceSDKMethods...)

// listResourceSDKMethods lists the SDK methods the App Installer list resource
// calls (List for the query, Get for per-item hydration when IncludeResource is
// requested).
var listResourceSDKMethods = []string{
	"ListAppInstallerDeploymentsV1",
	"GetAppInstallerDeploymentV1",
}

// listResourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the App Installer list resource.
var listResourcePrivileges = permissions.Section(pro.Privileges, listResourceSDKMethods...)
