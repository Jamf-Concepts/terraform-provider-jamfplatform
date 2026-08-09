// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
)

// A device group is a scopeable object in Jamf Pro's terms: scope is based on it,
// so changing who is in it changes what every policy, profile and app scoped to it
// applies to. That knock-on effect is the reason this alert exists separately from
// the one on the deployed objects themselves.
//
// No reads are needed. Current membership is already in state, and a static
// group's intended membership is in the configuration.

const (
	groupTypeSmart        = "smart"
	groupTypeStatic       = "static"
	deviceTypeMobileValue = "mobile"
)

// reportMembershipImpact emits the plan-time impact alert for a change to this
// group's membership.
//
// Runs for creates and destroys as well as updates: a new group starts out with
// members that other objects may immediately act on, and deleting a group stops
// everything scoped to it applying to those members.
func (r *DeviceGroupResource) reportMembershipImpact(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	cache := r.pd.ImpactCache()
	if !cache.Enabled() {
		return
	}
	creating := req.State.Raw.IsNull()
	destroying := req.Plan.Raw.IsNull()
	if creating && destroying {
		return
	}

	var plan, state DeviceGroupResourceModel
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

	m := impact.Membership{Noun: membershipNoun(subject.DeviceType.ValueString())}

	// Current membership comes from state. member_count is Computed without
	// UseStateForUnknown, so the planned value is unknown on every update — state
	// is the only place a usable figure exists.
	if !creating && !state.MemberCount.IsNull() && !state.MemberCount.IsUnknown() {
		m.Current = state.MemberCount.ValueInt64()
		m.CurrentKnown = true
	}

	smart := subject.GroupType.ValueString() == groupTypeSmart
	if smart {
		m.Undetermined = impact.CriteriaUndetermined
		m.Changed = criteriaDiffer(plan.Criteria, state.Criteria)
	} else {
		// A null members set means "leave membership as it is", not "no members" —
		// so it must not be read as a planned membership of zero.
		if !destroying && !plan.Members.IsNull() && !plan.Members.IsUnknown() {
			m.Next = int64(len(plan.Members.Elements()))
			m.NextKnown = true
		}
		m.Changed = !plan.Members.Equal(state.Members)
	}

	anchor := path.Root("members")
	if smart {
		anchor = path.Root("criteria")
	}

	resp.Diagnostics.Append(impact.ReportMembership(ctx, impact.MembershipRequest{
		Cache:      cache,
		Path:       anchor,
		Label:      groupLabel(subject),
		Action:     action,
		Membership: m,
	})...)
}

// membershipNoun returns what a group of this device type contains, using the
// admin UI's term.
func membershipNoun(deviceType string) string {
	if deviceType == deviceTypeMobileValue {
		return "mobile devices"
	}
	return "computers"
}

// groupLabel names the group the way the admin UI does, e.g. "smart computer
// group", so the alert reads the same as Jamf Pro's own.
func groupLabel(m DeviceGroupResourceModel) string {
	kind := groupTypeStatic
	if m.GroupType.ValueString() == groupTypeSmart {
		kind = groupTypeSmart
	}
	scope := "computer"
	if m.DeviceType.ValueString() == deviceTypeMobileValue {
		scope = "mobile device"
	}
	return kind + " " + scope + " group"
}

// criteriaDiffer reports whether two criteria lists describe different
// membership. Compared field by field rather than by length, because an edited
// operator or value changes membership without changing how many criteria there
// are.
func criteriaDiffer(a, b []DeviceGroupCriteriaModel) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if !a[i].AttributeName.Equal(b[i].AttributeName) ||
			!a[i].Operator.Equal(b[i].Operator) ||
			!a[i].AttributeValue.Equal(b[i].AttributeValue) ||
			!a[i].JoinType.Equal(b[i].JoinType) ||
			!a[i].HasOpeningParenthesis.Equal(b[i].HasOpeningParenthesis) ||
			!a[i].HasClosingParenthesis.Equal(b[i].HasClosingParenthesis) {
			return true
		}
	}
	return false
}
