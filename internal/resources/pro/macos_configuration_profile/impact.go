// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// reportScopeImpact emits the plan-time impact alert for a scope change. A configuration profile
// is a deployable object in Jamf Pro's terms, so the alert reports how many
// devices the change reaches.
func (r *Resource) reportScopeImpact(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	impact.ReportPlan(ctx, req, resp, impact.PlanReport{
		Cache: r.impact,
		Path:  path.Root("scope"),
		Label: "configuration profile",
	}, func(ctx context.Context, m *ResourceModel) impact.Scope {
		return scope.ComputerImpactScope(ctx, m.Scope)
	})
}
