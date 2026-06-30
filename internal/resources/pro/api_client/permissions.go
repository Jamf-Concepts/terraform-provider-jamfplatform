// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// resourceSDKMethods lists the SDK methods the API client resource's CRUD path
// calls. It mirrors the "SDK endpoints used" block in crud.go and drives the
// "Required Jamf privileges" table appended to the resource MarkdownDescription.
// permissions_test.go asserts this list stays in sync with the actual
// client.<Method> calls in crud.go and with the SDK privilege registry.
var resourceSDKMethods = []string{
	"CreateApiIntegrationV1", //gitleaks:allow — SDK method name, not a secret (generic-api-key false positive)
	"GetApiIntegrationV1",
	"UpdateApiIntegrationV1",
	"DeleteApiIntegrationV1",
	"RotateApiIntegrationClientCredentialsV1",
}

// resourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the API client resource, appended to its MarkdownDescription.
var resourcePrivileges = permissions.Section(pro.Privileges, resourceSDKMethods...)

// dataSourceSDKMethods lists the SDK methods the singular API client data
// source calls (data_source.go). A data source documents only the privileges
// it itself needs — a read, not the resource's full CRUD set.
var dataSourceSDKMethods = []string{
	"GetApiIntegrationV1",
}

// dataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the singular API client data source.
var dataSourcePrivileges = permissions.Section(pro.Privileges, dataSourceSDKMethods...)

// pluralDataSourceSDKMethods lists the SDK methods the plural API clients data
// source calls (datasource_plural.go).
var pluralDataSourceSDKMethods = []string{
	"ListApiIntegrationsV1",
}

// pluralDataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the plural API clients data source.
var pluralDataSourcePrivileges = permissions.Section(pro.Privileges, pluralDataSourceSDKMethods...)

// listResourceSDKMethods lists the SDK methods the API client list resource
// calls (list_resource.go).
var listResourceSDKMethods = []string{
	"ListApiIntegrationsV1",
}

// listResourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the API client list resource.
var listResourcePrivileges = permissions.Section(pro.Privileges, listResourceSDKMethods...)
