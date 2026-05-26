// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_failover_url

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ssoFailoverURLTimeoutAttributeTypes defines the timeout attribute types.
var ssoFailoverURLTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}
