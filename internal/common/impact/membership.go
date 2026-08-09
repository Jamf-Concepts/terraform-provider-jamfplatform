// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// This file implements the scopeable-object side of impact alerts: the alert on a
// group or class itself, rather than on something scoped to it.
//
// Jamf Pro keeps these two alerts separate — "Display impact alert on Save for
// deployable objects" and "Display criteria impact alert on Save for scopeable
// object edits" — because editing a group and editing something scoped to that
// group are different events. The provider follows the same split, and for a
// reason specific to Terraform: a resource's plan modifier can only see its own
// prior state and planned configuration, never a sibling's. So a plan that edits
// a smart group's criteria and a policy scoped to that group cannot produce one
// combined figure — the policy has no way to know the group is changing. Two
// alerts, one from each side, is the only honest coverage.
//
// Unlike the deployable side, this needs no reads at all: current membership is
// already in state, and a static group's intended membership is in the
// configuration.

// Membership describes how a scopeable object's membership is changing.
type Membership struct {
	// Noun is what is being counted, in the admin UI's terms — "computers",
	// "mobile devices", "users".
	Noun string

	// Current is the membership before this change. Known for anything that
	// already exists.
	Current      int64
	CurrentKnown bool

	// Next is the membership after this change, for objects whose membership the
	// configuration decides — a static group's member list.
	Next      int64
	NextKnown bool

	// Undetermined explains why the membership after this change is not knowable.
	// Set for a smart group, whose membership Jamf Pro derives from criteria.
	Undetermined string

	// Changed reports whether membership is actually being altered by this plan.
	// A rename or a description edit changes the object without changing who is in
	// it, and must not raise an alert.
	Changed bool
}

// MembershipRequest describes one scopeable object's planned change.
type MembershipRequest struct {
	// Cache gates reporting. A nil cache disables it. No reads are performed.
	Cache *Cache
	// Path anchors the diagnostic to the attribute that drives membership, so the
	// warning appears beside the criteria or member list.
	Path path.Path
	// Label names the object using the admin UI's term — "smart computer group",
	// "static mobile device group", "class".
	Label string
	// Action is the planned lifecycle operation.
	Action Action
	// Membership is how membership is changing.
	Membership Membership
	// Note is an optional extra sentence about how this object's members reach
	// devices. A user group's members are users, so what they receive depends on
	// which devices those users are assigned to.
	Note string
}

// ReportMembership produces the impact alert for a change to a scopeable object.
//
// The knock-on effect is the point of this alert. A group's membership is not
// interesting in itself; it is interesting because everything scoped to that
// group starts or stops applying to whatever joins or leaves.
func ReportMembership(_ context.Context, req MembershipRequest) diag.Diagnostics {
	var diags diag.Diagnostics
	if !req.Cache.Enabled() {
		return diags
	}
	m := req.Membership
	if req.Action == ActionUpdate && !m.Changed {
		// Jamf Pro alerts on a membership or criteria edit, not on a rename.
		return diags
	}
	if !m.CurrentKnown && !m.NextKnown && m.Undetermined == "" {
		// Nothing to say.
		return diags
	}

	diags.AddAttributeWarning(req.Path, membershipHeadline(req), membershipDetail(req))
	return diags
}

// membershipHeadline renders the one-line summary.
func membershipHeadline(req MembershipRequest) string {
	m := req.Membership
	switch req.Action {
	case ActionCreate:
		if m.NextKnown {
			return fmt.Sprintf("Impact alert — this %s will contain %s", req.Label, countOf(m.Next, m.Noun))
		}
		return fmt.Sprintf("Impact alert — this new %s's membership is decided after apply", req.Label)
	case ActionDelete:
		if m.CurrentKnown {
			return fmt.Sprintf("Impact alert — removing this %s affects %s", req.Label, countOf(m.Current, m.Noun))
		}
		return fmt.Sprintf("Impact alert — removing this %s changes what is scoped to it", req.Label)
	default:
		switch {
		case m.CurrentKnown && m.NextKnown && m.Next != m.Current:
			return fmt.Sprintf("Impact alert — this %s changes from %s to %s",
				req.Label, countOf(m.Current, m.Noun), countOf(m.Next, m.Noun))
		case m.CurrentKnown:
			return fmt.Sprintf("Impact alert — this %s currently contains %s and its membership is changing",
				req.Label, countOf(m.Current, m.Noun))
		default:
			return fmt.Sprintf("Impact alert — this %s's membership is changing", req.Label)
		}
	}
}

// membershipDetail renders the body: the knock-on effect, the arithmetic where it
// is available, and why it is not where it is not.
func membershipDetail(req MembershipRequest) string {
	m := req.Membership
	var b strings.Builder

	switch req.Action {
	case ActionDelete:
		fmt.Fprintf(&b, "Every object scoped to this %s stops applying to its members.\n", req.Label)
	default:
		fmt.Fprintf(&b, "Every object scoped to this %s applies to whatever joins it, and stops applying to whatever leaves.\n", req.Label)
	}

	if m.CurrentKnown && m.NextKnown {
		switch delta := m.Next - m.Current; {
		case delta > 0:
			fmt.Fprintf(&b, "This change adds %s.\n", countOf(delta, m.Noun))
		case delta < 0:
			fmt.Fprintf(&b, "This change removes %s.\n", countOf(-delta, m.Noun))
		default:
			fmt.Fprintf(&b, "The number of %s does not change, but which ones are members does.\n", m.Noun)
		}
	}

	if req.Note != "" {
		b.WriteString(req.Note + "\n")
	}

	// Not on a delete: there is no membership after the object is gone, so
	// explaining that Jamf Pro will re-evaluate criteria would be nonsense.
	if m.Undetermined != "" && req.Action != ActionDelete {
		fmt.Fprintf(&b, "\nMembership after this change is not known during plan: %s\n", m.Undetermined)
	}

	b.WriteString("\n" + snapshotNote)
	return b.String()
}

// countOf renders a count with a singular or plural noun.
func countOf(n int64, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", strings.TrimSuffix(noun, "s"))
	}
	return fmt.Sprintf("%d %s", n, noun)
}

// CriteriaUndetermined is the reason a smart group's post-change membership
// cannot be reported. Jamf Pro evaluates criteria against inventory on its own
// schedule, so the configuration does not determine the result.
const CriteriaUndetermined = "Jamf Pro re-evaluates the criteria against device inventory after the change is applied, so the resulting membership is not decided by this configuration."
