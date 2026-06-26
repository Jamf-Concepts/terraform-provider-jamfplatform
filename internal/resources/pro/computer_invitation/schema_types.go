// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_invitation

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// computerInvitationTimeoutAttributeTypes defines the timeout attribute types
// for the computer invitation resource operations.
var computerInvitationTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// validInvitationTypes is the set of invitation_type values Jamf Pro recognises,
// wire-probed against a live tenant. USER_INITIATED_URL and USER_INITIATED_EMAIL
// are created through this resource. DEP_CUSTOM_ENROLL is minted by Automated
// Device Enrollment and is accepted here so a faithfully-imported ADE invitation
// validates; a from-scratch create with it is rejected by the server.
var validInvitationTypes = []string{
	"USER_INITIATED_URL",
	"USER_INITIATED_EMAIL",
	"DEP_CUSTOM_ENROLL",
}

// markdownValueList renders a slice of enum values as a backticked,
// comma-separated list for MarkdownDescription strings. Deriving the documented
// values from the same slice the OneOf validator uses keeps the docs and the
// validator from drifting apart.
func markdownValueList(vals []string) string {
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = "`" + v + "`"
	}
	return strings.Join(quoted, ", ")
}
