// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_store_country_codes

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// dataSourceSDKMethods lists the SDK methods the App Store country codes data
// source's Read path calls. It mirrors the client.<Method> calls in
// data_source.go and drives the "Required Jamf privileges" table appended to
// the data source MarkdownDescription. permissions_test.go asserts this list
// stays in sync with the actual client.<Method> calls and with the SDK
// privilege registry.
var dataSourceSDKMethods = []string{
	"ListAppStoreCountryCodesV1",
}

// dataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the App Store country codes data source, appended to its
// MarkdownDescription.
var dataSourcePrivileges = permissions.Section(pro.Privileges, dataSourceSDKMethods...)
