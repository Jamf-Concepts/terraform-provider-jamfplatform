// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// resourceSDKMethods lists the SDK methods the package resource's CRUD path
// calls (in crud.go). It drives the "Required Jamf permissions" table appended
// to the resource MarkdownDescription. permissions_test.go asserts this list
// stays in sync with the actual client.<Method> calls in crud.go and with the
// SDK privilege registry.
var resourceSDKMethods = []string{
	"CreatePackageV1",
	"GetPackageV1",
	"UpdatePackageV1",
	"DeletePackageV1",
	"UploadPackageV1",
	"UploadPackageManifestV1",
	"DeletePackageManifestV1",
}

// resourcePrivileges is the rendered "Required Jamf permissions" Markdown
// section for the package resource, appended to its MarkdownDescription.
var resourcePrivileges = permissions.Section(pro.Privileges, resourceSDKMethods...)

// dataSourceSDKMethods lists the SDK methods the package data source (in
// data_source.go) calls. The data source also calls ResolvePackageV1ByName,
// but that is an SDK resolver helper layered over the GET /v1/packages list
// endpoint — it requires no privilege beyond packages:read, which
// GetPackageV1 already documents — so it carries no registry entry and is not
// listed here.
var dataSourceSDKMethods = []string{
	"GetPackageV1",
}

// dataSourcePrivileges is the rendered "Required Jamf permissions" Markdown
// section for the package data source, appended to its MarkdownDescription.
var dataSourcePrivileges = permissions.Section(pro.Privileges, dataSourceSDKMethods...)

// listResourceSDKMethods lists the SDK methods the package list resource (in
// list_resource.go) calls.
var listResourceSDKMethods = []string{
	"ListPackagesV1",
}

// listResourcePrivileges is the rendered "Required Jamf permissions" Markdown
// section for the package list resource, appended to its Description.
var listResourcePrivileges = permissions.Section(pro.Privileges, listResourceSDKMethods...)
