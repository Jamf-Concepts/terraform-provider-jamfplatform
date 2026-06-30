// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package baselines

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/compliancebenchmarks"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// dataSourceSDKMethods lists the SDK methods the baselines data source's Read
// path calls. It mirrors the client.<Method> calls in data_source.go and drives
// the "Required Jamf privileges" table appended to the data source
// MarkdownDescription. permissions_test.go asserts this list stays in sync with
// the actual client.<Method> calls in data_source.go and with the SDK privilege
// registry.
var dataSourceSDKMethods = []string{
	"ListBaselines",
}

// dataSourcePrivileges is the rendered "Required Jamf privileges" Markdown
// section for the baselines data source, appended to its MarkdownDescription.
var dataSourcePrivileges = permissions.Section(compliancebenchmarks.Privileges, dataSourceSDKMethods...)
