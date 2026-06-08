// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_assignment

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// intStringOrNull maps a nil/negative *int to a null TF String, else its decimal
// form. The server returns vpp_admin_account_id = -1 for an assignment created
// without a valid account (wire-probed); treat that as absent.
func intStringOrNull(p *int) types.String {
	if p == nil || *p < 0 {
		return types.StringNull()
	}
	return types.StringValue(strconv.Itoa(*p))
}

// derefString returns "" for a nil *string.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
