// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_invitation

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mobileDeviceInvitationTimeoutAttributeTypes defines the timeout attribute
// types for the mobile device invitation resource operations.
var mobileDeviceInvitationTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// validInvitationTypes is the set of user-creatable invitation_type values for
// mobile device invitations.
var validInvitationTypes = []string{
	"USER_INITIATED_URL",
	"USER_INITIATED_EMAIL",
}
