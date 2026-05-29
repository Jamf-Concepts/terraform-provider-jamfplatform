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

// defaultLanguageCode is the language code of the built-in English messaging
// language. It always exists, is the default shown when no language matches a
// device's locale, and cannot be deleted; the reconcile only ever updates it
// (mirrors allDirectoryServiceUsersID for the access-group collection).
const defaultLanguageCode = "en"

// messagingLanguageAttrTypes describes the VALUE object type of the
// messaging_languages nested map. The map KEY is the ISO 639-1 language code, so
// language_code is not an attribute here. All values are strings (the wire type
// is 44 *string fields; the 4 unmodelled personal* fields ride along on the
// wire).
var messagingLanguageAttrTypes = map[string]attr.Type{
	"name":       types.StringType,
	"page_title": types.StringType,

	"login_page_text":   types.StringType,
	"username_text":     types.StringType,
	"password_text":     types.StringType,
	"login_button_text": types.StringType,

	"device_ownership_page_text":                  types.StringType,
	"personal_device_button_name":                 types.StringType,
	"institutional_ownership_button_name":         types.StringType,
	"personal_device_management_description":      types.StringType,
	"institutional_device_management_description": types.StringType,
	"enroll_device_button_name":                   types.StringType,

	"personal_eula":           types.StringType,
	"institutional_eula":      types.StringType,
	"eula_accept_button_text": types.StringType,

	"site_selection_text": types.StringType,

	"ca_certificate_installation_text":   types.StringType,
	"ca_certificate_name":                types.StringType,
	"ca_certificate_description":         types.StringType,
	"ca_certificate_install_button_name": types.StringType,

	"institutional_mdm_installation_text":   types.StringType,
	"institutional_mdm_profile_name":        types.StringType,
	"institutional_mdm_profile_description": types.StringType,
	"institutional_mdm_pending_text":        types.StringType,
	"institutional_mdm_install_button_name": types.StringType,

	"user_enrollment_mdm_installation_text":   types.StringType,
	"user_enrollment_mdm_profile_name":        types.StringType,
	"user_enrollment_mdm_profile_description": types.StringType,
	"user_enrollment_mdm_install_button_name": types.StringType,

	"quickadd_installation_text":   types.StringType,
	"quickadd_name":                types.StringType,
	"quickadd_progress_text":       types.StringType,
	"quickadd_install_button_name": types.StringType,

	"enrollment_complete_text":           types.StringType,
	"enrollment_failed_text":             types.StringType,
	"try_again_button_name":              types.StringType,
	"view_enrollment_status_button_name": types.StringType,
	"view_enrollment_status_text":        types.StringType,
	"log_out_button_name":                types.StringType,
}
