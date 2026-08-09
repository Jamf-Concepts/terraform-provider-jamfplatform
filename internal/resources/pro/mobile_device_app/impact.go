// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// reportScopeImpact emits the plan-time impact alert for a scope change. A mobile device app
// is a deployable object in Jamf Pro's terms, so the alert reports how many
// devices the change reaches.
func (r *MobileAppResource) reportScopeImpact(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	impact.ReportPlan(ctx, req, resp, impact.PlanReport{
		Cache: r.impact,
		Path:  path.Root("scope"),
		Label: "mobile device app",
	}, func(ctx context.Context, m *MobileAppResourceModel) impact.Scope {
		return scope.MobileImpactScopeNoIbeacons(ctx, m.Scope)
	})
}
