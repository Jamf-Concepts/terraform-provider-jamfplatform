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

// proGroupsTransientWarningKey is the FiredOnce key for the transient-failure
// advisory surfaced when Pro /v2/groups returns an unexpected status (anything
// other than 200, 403, or 404) during jamf_pro_id resolution. Separate from the
// forbidden key so the operator can distinguish "tenant lacks privilege"
// (actionable) from "Pro endpoint had a hiccup" (likely transient).
const proGroupsTransientWarningKey = "device_group.jamf_pro_id.transient"

// resolveJamfProID best-effort populates the jamf_pro_id attribute for a Platform
// device group by calling the Pro /v2/groups/{id} endpoint. It deliberately does
// not call providerdata.ConfigurePro — device_group is a Platform Services
// resource, so the Jamf Pro version gate and floor warning would be misleading
// here, and they would force a synchronous version fetch on every Platform-only
// run.
//
// All failure modes degrade to (null, warning) rather than (zero, error) so the
// Platform Create/Update/Read result that called us is never discarded:
//
//   - 403 Forbidden — Platform API client lacks the `Read Groups` privilege.
//     Surfaces a single actionable warning per provider invocation (latched via
//     providerdata.Data.FiredOnce with proGroupsForbiddenWarningKey).
//   - 404 Not Found — group deleted between the Platform read and this call, or
//     the Pro endpoint is absent on a Platform-only tenant. Silently null; no
//     warning, since either case is benign for the device_group resource itself.
//   - Anything else (transport error, 5xx, unexpected status) — null with a
//     single transient-failure warning per provider invocation, latched via
//     proGroupsTransientWarningKey. Returning an error here would cause Create
//     to skip resp.State.Set and orphan a successfully-created group on the
//     Platform side.
//
// Callers must therefore tolerate a null result without treating it as a hard
// failure.
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
			if pd.FiredOnce(proGroupsTransientWarningKey) {
				diags.AddWarning(
					"Failed to resolve Jamf Pro classic ID; jamf_pro_id will be null.",
					"The provider tried to resolve the numeric Jamf Pro classic ID for one or more device groups via the Pro `/v2/groups` endpoint but the call failed: "+err.Error()+". This is treated as a transient bridging failure so the Platform device group operation is not rolled back; subsequent applies will retry.",
				)
			}
			tflog.Debug(ctx, "Pro groups endpoint returned unexpected error; nulling jamf_pro_id", map[string]any{
				"platform_id": platformID,
				"error":       err.Error(),
			})
			return types.StringNull(), diags
		}
	}
	if grp == nil || grp.GroupJamfProID == "" {
		return types.StringNull(), diags
	}
	return types.StringValue(grp.GroupJamfProID), diags
}
