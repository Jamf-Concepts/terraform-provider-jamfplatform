// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_initiated_enrollment_settings

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// userInitiatedEnrollmentSettingsTimeoutAttributeTypes defines the timeout
// attribute types.
var userInitiatedEnrollmentSettingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// accessGroupAttrTypes describes the object element type of the access_group
// nested set.
var accessGroupAttrTypes = map[string]attr.Type{
	"id":                                     types.StringType,
	"directory_service_group_id":             types.StringType,
	"ldap_server_id":                         types.StringType,
	"name":                                   types.StringType,
	"site_id":                                types.StringType,
	"enterprise_enrollment_enabled":          types.BoolType,
	"personal_enrollment_enabled":            types.BoolType,
	"account_driven_user_enrollment_enabled": types.BoolType,
	"require_eula":                           types.BoolType,
}

// allDirectoryServiceUsersID is the server id of the built-in "All Directory
// Service Users" access group. It always exists and cannot be created or
// deleted; the reconcile only ever updates it.
const allDirectoryServiceUsersID = "1"
