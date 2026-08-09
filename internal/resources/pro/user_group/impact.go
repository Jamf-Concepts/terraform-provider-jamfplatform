// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
)

// A user group is a scopeable object: policies, profiles and apps can be scoped to
// it, so changing who is in it changes what those objects apply to.
//
// Its members are users rather than devices, so the alert counts users and says
// plainly that the devices reached depend on user assignment. Reporting a device
// figure here would require resolving every member's assigned devices, and would
// still be a guess about which of them a user is signed in to.

const (
	groupTypeSmartValue = "smart"
	// userGroupNote explains the indirection, since a user count is not a device
	// count and should not be read as one.
	userGroupNote = "Members are users; which devices they affect depends on the devices those users are assigned to."
)

// reportMembershipImpact emits the plan-time impact alert for a change to this
// group's membership. Runs for creates and destroys as well as updates.
func (r *UserGroupResource) reportMembershipImpact(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.pd == nil {
		return
	}
	cache := r.pd.ImpactCache()
	if !cache.Enabled() {
		return
	}
	creating := req.State.Raw.IsNull()
	destroying := req.Plan.Raw.IsNull()
	if creating && destroying {
		return
	}

	var plan, state UserGroupResourceModel
	if !creating {
		if diags := req.State.Get(ctx, &state); diags.HasError() {
			return
		}
	}
	if !destroying {
		if diags := req.Plan.Get(ctx, &plan); diags.HasError() {
			return
		}
	}

	subject := plan
	action := impact.ActionUpdate
	switch {
	case creating:
		action = impact.ActionCreate
	case destroying:
		subject = state
		action = impact.ActionDelete
	}

	m := impact.Membership{Noun: "users"}
	if !creating && !state.MemberCount.IsNull() && !state.MemberCount.IsUnknown() {
		m.Current = state.MemberCount.ValueInt64()
		m.CurrentKnown = true
	}

	smart := subject.GroupType.ValueString() == groupTypeSmartValue
	if smart {
		m.Undetermined = impact.CriteriaUndetermined
		m.Changed = criterionListsDiffer(plan.Criteria, state.Criteria)
	} else {
		if !destroying && !plan.Members.IsNull() && !plan.Members.IsUnknown() {
			m.Next = int64(len(plan.Members.Elements()))
			m.NextKnown = true
		}
		m.Changed = !plan.Members.Equal(state.Members)
	}

	anchor := path.Root("members")
	label := "static user group"
	if smart {
		anchor = path.Root("criteria")
		label = "smart user group"
	}

	resp.Diagnostics.Append(impact.ReportMembership(ctx, impact.MembershipRequest{
		Cache:      cache,
		Path:       anchor,
		Label:      label,
		Action:     action,
		Membership: m,
		Note:       userGroupNote,
	})...)
}

// criterionListsDiffer reports whether two criteria lists describe different
// membership. Compared field by field, because an edited operator or value
// changes membership without changing how many criteria there are.
func criterionListsDiffer(a, b []UserGroupCriterionModel) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if !a[i].Name.Equal(b[i].Name) ||
			!a[i].SearchType.Equal(b[i].SearchType) ||
			!a[i].Value.Equal(b[i].Value) ||
			!a[i].AndOr.Equal(b[i].AndOr) ||
			!a[i].HasOpeningParenthesis.Equal(b[i].HasOpeningParenthesis) ||
			!a[i].HasClosingParenthesis.Equal(b[i].HasClosingParenthesis) {
			return true
		}
	}
	return false
}
