// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Wire enum values for the top-level `<key_type>` element. Audit reference:
// local-testing/diskencryption/AUDIT_FINDINGS.md §1.
//
// The server returns `Individual and Institutional` (lowercase `and`)
// regardless of input case. The provider's plan modifier
// (keyTypePlanModifier) canonicalises any case variant the user supplies
// to the wire form on input so TF state stays stable.
const (
	keyTypeIndividual              = "Individual"
	keyTypeInstitutional           = "Institutional"
	keyTypeIndividualInstitutional = "Individual and Institutional"
)

// allKeyTypeWireValues lists the canonical wire-form `key_type` values.
// Used by stringvalidator.OneOfCaseInsensitive in the resource schema so
// user input is accepted in either case (e.g. `Individual And
// Institutional` vs `Individual and Institutional`).
var allKeyTypeWireValues = []string{
	keyTypeIndividual,
	keyTypeInstitutional,
	keyTypeIndividualInstitutional,
}

// Wire enum values for the top-level `<file_vault_enabled_users>` element.
const (
	fileVaultEnabledUsersCurrentOrNext   = "Current or Next User"
	fileVaultEnabledUsersManagementAccnt = "Management Account"
)

// allFileVaultEnabledUsersValues lists the accepted file_vault_enabled_users
// wire enum values.
var allFileVaultEnabledUsersValues = []string{
	fileVaultEnabledUsersCurrentOrNext,
	fileVaultEnabledUsersManagementAccnt,
}

// canonicalKeyType maps any case variant of a recognised key_type to the
// canonical wire form returned by the server. Used by both the input
// builder (so the wire payload uses the canonical spelling) and the
// keyTypePlanModifier (so plan-time state matches the post-apply read).
// Unrecognised values pass through unchanged — stringvalidator.OneOfCaseInsensitive
// surfaces them as a schema error at plan time.
func canonicalKeyType(v string) string {
	switch strings.ToLower(v) {
	case strings.ToLower(keyTypeIndividual):
		return keyTypeIndividual
	case strings.ToLower(keyTypeInstitutional):
		return keyTypeInstitutional
	case strings.ToLower(keyTypeIndividualInstitutional):
		return keyTypeIndividualInstitutional
	}
	return v
}

// keyTypePlanModifier rewrites the planned `key_type` value to the
// canonical wire form so users may write `Individual And Institutional`
// (Title Case) in their config while TF state stays on the server's
// canonical `Individual and Institutional` (lowercase `and`). Per
// STYLE_GUIDE §Asymmetric server normalisation on `type`-style
// discriminator fields.
type keyTypePlanModifier struct{}

func (keyTypePlanModifier) Description(context.Context) string {
	return "normalise key_type to the wire-canonical spelling (Individual and Institutional with lowercase `and`) so TF state matches what the server returns on read"
}

func (m keyTypePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (keyTypePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	resp.PlanValue = types.StringValue(canonicalKeyType(req.ConfigValue.ValueString()))
}

// Compile-time interface assertion.
var _ planmodifier.String = keyTypePlanModifier{}
