// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// resourceSDKMethods lists the SDK methods the API role resource's CRUD path
// calls. It mirrors the "SDK endpoints used" block in crud.go and drives the
// "Required Jamf privileges" table appended to the resource MarkdownDescription.
// permissions_test.go asserts this list stays in sync with the actual
// client.<Method> calls in crud.go and with the SDK privilege registry.
var resourceSDKMethods = []string{
	"CreateApiRoleV1",
	"GetApiRoleV1",
	"UpdateApiRoleV1",
	"DeleteApiRoleV1",
}

// resourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the API role resource, appended to its MarkdownDescription.
var resourcePrivileges = permissions.Section(pro.Privileges, resourceSDKMethods...)

// dataSourceSDKMethods lists the SDK methods the singular API role data source
// calls (data_source.go). It documents only the privileges that construct needs.
var dataSourceSDKMethods = []string{
	"GetApiRoleV1",
}

// dataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the singular API role data source.
var dataSourcePrivileges = permissions.Section(pro.Privileges, dataSourceSDKMethods...)

// pluralDataSourceSDKMethods lists the SDK methods the plural API roles data
// source calls (datasource_plural.go).
var pluralDataSourceSDKMethods = []string{
	"ListApiRolesV1",
}

// pluralDataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the plural API roles data source.
var pluralDataSourcePrivileges = permissions.Section(pro.Privileges, pluralDataSourceSDKMethods...)

// listResourceSDKMethods lists the SDK methods the API role list resource calls
// (list_resource.go).
var listResourceSDKMethods = []string{
	"ListApiRolesV1",
}

// listResourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the API role list resource.
var listResourcePrivileges = permissions.Section(pro.Privileges, listResourceSDKMethods...)
