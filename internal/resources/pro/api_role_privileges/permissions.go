// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role_privileges

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// dataSourceSDKMethods lists the SDK methods the api_role_privileges data
// source's Read path calls. It mirrors the client.<Method> calls in
// data_source.go and drives the "Required Jamf privileges" table appended to
// the data source MarkdownDescription. permissions_test.go asserts this list
// stays in sync with those calls and with the SDK privilege registry.
var dataSourceSDKMethods = []string{
	"ListApiRolePrivilegesV1",
	"SearchApiRolePrivilegesV1",
}

// dataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the api_role_privileges data source, appended to its
// MarkdownDescription.
var dataSourcePrivileges = permissions.Section(pro.Privileges, dataSourceSDKMethods...)
