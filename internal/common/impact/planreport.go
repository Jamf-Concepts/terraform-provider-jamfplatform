// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// PlanReport is the per-resource configuration for the shared plan hook.
type PlanReport struct {
	// Cache is the shared tenant cache. A nil cache disables reporting.
	Cache *Cache
	// Path anchors the diagnostic to the attribute the figure derives from.
	Path path.Path
	// Kind selects the deployable or scopeable channel.
	Kind Kind
	// Label names the object using the admin UI's term for it.
	Label string
}

// ReportPlan is the shared ModifyPlan hook for a scope-bearing resource.
//
// It handles the lifecycle bookkeeping every resource would otherwise repeat:
// reading prior state and planned configuration only when each exists, deciding
// whether this is a create, update or delete, and skipping when reporting is off.
// The caller supplies only how to read a scope out of its own model.
//
// Runs on creates and deletes as well as updates. An object entering management
// starts applying to its scope, and one leaving stops — both are worth seeing
// before they happen, and neither is visible from the resource diff alone.
func ReportPlan[M any](
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
	rep PlanReport,
	extract func(context.Context, *M) Scope,
) {
	if !rep.Cache.Enabled() {
		return
	}
	creating := req.State.Raw.IsNull()
	destroying := req.Plan.Raw.IsNull()
	if creating && destroying {
		return
	}

	var prior, planned Scope
	if !creating {
		var state M
		if diags := req.State.Get(ctx, &state); diags.HasError() {
			// Impact reporting is advisory: a model that will not decode is the
			// resource's own problem to report, not a reason to add noise here.
			return
		}
		prior = extract(ctx, &state)
	}
	if !destroying {
		var plan M
		if diags := req.Plan.Get(ctx, &plan); diags.HasError() {
			return
		}
		planned = extract(ctx, &plan)
	}

	action := ActionUpdate
	switch {
	case creating:
		action = ActionCreate
	case destroying:
		action = ActionDelete
	}

	resp.Diagnostics.Append(Report(ctx, Request{
		Cache:   rep.Cache,
		Path:    rep.Path,
		Kind:    rep.Kind,
		Label:   rep.Label,
		Action:  action,
		Prior:   prior,
		Planned: planned,
		// Any diff at all counts. A payload edit reaches every device in scope just
		// as a scope edit does.
		Changed: !req.Plan.Raw.Equal(req.State.Raw),
	})...)
}
