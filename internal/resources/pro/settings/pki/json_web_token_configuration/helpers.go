// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package json_web_token_configuration

import (
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// extractJSONWebTokenConfigurationID returns the assigned ID as a string from a
// Create/GET response. The classic endpoint echoes the integer ID at the top
// level (<json_web_token_configuration><id>).
func extractJSONWebTokenConfigurationID(c *proclassic.JsonWebTokenConfiguration) string {
	if c == nil || c.ID == nil {
		return ""
	}
	return strconv.Itoa(*c.ID)
}

// int64FromIntPtr renders an *int as a types.Int64, null for nil.
func int64FromIntPtr(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}

// derefString returns the underlying string for a non-nil *string, or "".
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
