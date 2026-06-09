// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package gsx_connection

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// gsxConnectionSettingsTimeoutAttributeTypes defines the timeout attribute types for the resource.
var gsxConnectionSettingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// maxAccountNumberLength is the OpenAPI maxLength for serviceAccountNo and shipToNo.
const maxAccountNumberLength = 10
