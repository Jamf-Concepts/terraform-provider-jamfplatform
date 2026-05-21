// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// proGroupsForbiddenWarningKey is the FiredOnce key for the missing-privilege
// advisory surfaced when Pro /v2/groups returns 403 during jamf_pro_id resolution.
// Stable string so all device_group constructs (resource, data source, list resource)
// share a single emission per provider invocation.
const proGroupsForbiddenWarningKey = "device_group.jamf_pro_id.forbidden"

// resolveJamfProID best-effort populates the jamf_pro_id attribute for a Platform
// device group by calling the Pro /v2/groups/{id} endpoint. It deliberately does
// not call providerdata.ConfigurePro — device_group is a Platform Services
// resource, so the Jamf Pro version gate and floor warning would be misleading
// here, and they would force a synchronous version fetch on every Platform-only
// run. Instead the resolver tolerates Pro being unavailable: a 403 surfaces a
// single warning per provider invocation (latched via providerdata.Data.FiredOnce)
// and yields a null attribute; a 404 nulls the attribute silently (covers two
// distinct races — the group being deleted between the Platform read and this
// call, and the Pro endpoint being absent on tenants without Jamf Pro). All
// other errors are propagated so genuine transport problems do not get
// swallowed.
func resolveJamfProID(ctx context.Context, proClient *pro.Client, pd *providerdata.Data, platformID string) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	if proClient == nil || pd == nil || platformID == "" {
		return types.StringNull(), diags
	}
	grp, err := proClient.GetGroupV2(ctx, platformID)
	if err != nil {
		switch {
		case helpers.IsForbiddenError(err):
			if pd.FiredOnce(proGroupsForbiddenWarningKey) {
				diags.AddWarning(
					"Platform API client lacks 'Read Groups' privilege; jamf_pro_id will be null.",
					"The provider tried to resolve the numeric Jamf Pro classic ID for one or more device groups via the Pro `/v2/groups` endpoint but received a 403 Forbidden response. Grant the API client the `Read Groups` privilege to populate `jamf_pro_id` on subsequent applies; otherwise classic-API scope references will be unable to use these groups.",
				)
			}
			tflog.Debug(ctx, "Pro groups endpoint returned 403; nulling jamf_pro_id", map[string]any{
				"platform_id": platformID,
			})
			return types.StringNull(), diags
		case helpers.IsNotFoundError(err):
			tflog.Debug(ctx, "Pro groups endpoint returned 404; nulling jamf_pro_id", map[string]any{
				"platform_id": platformID,
			})
			return types.StringNull(), diags
		default:
			diags.AddError(
				"Error resolving Jamf Pro classic ID for device group",
				err.Error(),
			)
			return types.StringNull(), diags
		}
	}
	if grp == nil || grp.GroupJamfProID == "" {
		return types.StringNull(), diags
	}
	return types.StringValue(grp.GroupJamfProID), diags
}
