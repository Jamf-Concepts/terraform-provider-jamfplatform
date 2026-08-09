// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
)

func membershipReq(action Action, m Membership) MembershipRequest {
	return MembershipRequest{
		Cache:      NewCache(testSource()),
		Path:       path.Root("members"),
		Label:      "static computer group",
		Action:     action,
		Membership: m,
	}
}

func TestReportMembershipDisabledCacheIsSilent(t *testing.T) {
	req := membershipReq(ActionUpdate, Membership{Noun: "computers", Current: 4, CurrentKnown: true, Changed: true})
	req.Cache = nil
	if diags := ReportMembership(context.Background(), req); len(diags) != 0 {
		t.Fatalf("a disabled cache must produce nothing, got %d", len(diags))
	}
}

func TestReportMembershipSkipsUnchangedMembership(t *testing.T) {
	// A rename changes the group without changing who is in it, and must not alert.
	diags := ReportMembership(context.Background(), membershipReq(ActionUpdate, Membership{
		Noun: "computers", Current: 4, CurrentKnown: true, Changed: false,
	}))
	if len(diags) != 0 {
		t.Fatalf("an unchanged membership must not alert, got %q", diags[0].Summary())
	}
}

func TestReportMembershipStaticGrowthStatesTheDelta(t *testing.T) {
	diags := ReportMembership(context.Background(), membershipReq(ActionUpdate, Membership{
		Noun: "computers", Current: 4, CurrentKnown: true, Next: 7, NextKnown: true, Changed: true,
	}))
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if s := diags[0].Summary(); !strings.Contains(s, "changes from 4 computers to 7 computers") {
		t.Fatalf("summary must state both figures: %q", s)
	}
	d := diags[0].Detail()
	if !strings.Contains(d, "adds 3 computers") {
		t.Fatalf("detail must state the delta: %q", d)
	}
	if !strings.Contains(d, "applies to whatever joins it") {
		t.Fatalf("detail must name the knock-on effect: %q", d)
	}
	if !strings.Contains(d, snapshotNote) {
		t.Fatalf("every alert carries the snapshot caveat: %q", d)
	}
}

func TestReportMembershipStaticShrinkageStatesTheDelta(t *testing.T) {
	diags := ReportMembership(context.Background(), membershipReq(ActionUpdate, Membership{
		Noun: "computers", Current: 7, CurrentKnown: true, Next: 4, NextKnown: true, Changed: true,
	}))
	if d := diags[0].Detail(); !strings.Contains(d, "removes 3 computers") {
		t.Fatalf("a shrinking group must say what it removes: %q", d)
	}
}

func TestReportMembershipSameSizeDifferentMembers(t *testing.T) {
	// Swapping one member for another leaves the count alone but still changes what
	// every scoped object applies to, so it must not read as "no change".
	diags := ReportMembership(context.Background(), membershipReq(ActionUpdate, Membership{
		Noun: "computers", Current: 4, CurrentKnown: true, Next: 4, NextKnown: true, Changed: true,
	}))
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if d := diags[0].Detail(); !strings.Contains(d, "which ones are members does") {
		t.Fatalf("detail must explain that membership changed without the count changing: %q", d)
	}
}

func TestReportMembershipSmartGroupCannotStateTheOutcome(t *testing.T) {
	req := membershipReq(ActionUpdate, Membership{
		Noun: "computers", Current: 4, CurrentKnown: true, Changed: true,
		Undetermined: CriteriaUndetermined,
	})
	req.Label = "smart computer group"
	diags := ReportMembership(context.Background(), req)
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if s := diags[0].Summary(); !strings.Contains(s, "currently contains 4 computers") {
		t.Fatalf("summary must give the present figure: %q", s)
	}
	d := diags[0].Detail()
	if !strings.Contains(d, "not known during plan") {
		t.Fatalf("detail must say the outcome is unknown: %q", d)
	}
	if strings.Contains(d, "adds") || strings.Contains(d, "removes") {
		t.Fatalf("a smart group must not claim a delta it cannot compute: %q", d)
	}
}

func TestReportMembershipCreateAndDelete(t *testing.T) {
	create := ReportMembership(context.Background(), membershipReq(ActionCreate, Membership{
		Noun: "computers", Next: 3, NextKnown: true,
	}))
	if len(create) != 1 || !strings.Contains(create[0].Summary(), "will contain 3 computers") {
		t.Fatalf("create wording wrong: %+v", create)
	}

	del := ReportMembership(context.Background(), membershipReq(ActionDelete, Membership{
		Noun: "computers", Current: 4, CurrentKnown: true,
	}))
	if len(del) != 1 || !strings.Contains(del[0].Summary(), "removing this static computer group affects 4 computers") {
		t.Fatalf("delete wording wrong: %+v", del)
	}
	if d := del[0].Detail(); !strings.Contains(d, "stops applying to its members") {
		t.Fatalf("delete detail must state what stops: %q", d)
	}
}

func TestReportMembershipNewSmartGroupHasNoFigure(t *testing.T) {
	req := membershipReq(ActionCreate, Membership{Noun: "computers", Undetermined: CriteriaUndetermined})
	req.Label = "smart computer group"
	diags := ReportMembership(context.Background(), req)
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if s := diags[0].Summary(); !strings.Contains(s, "membership is decided after apply") {
		t.Fatalf("a new smart group has no membership to report: %q", s)
	}
}

func TestReportMembershipSingularNoun(t *testing.T) {
	diags := ReportMembership(context.Background(), membershipReq(ActionDelete, Membership{
		Noun: "computers", Current: 1, CurrentKnown: true,
	}))
	if s := diags[0].Summary(); !strings.Contains(s, "1 computer") || strings.Contains(s, "1 computers") {
		t.Fatalf("a single member must read in the singular: %q", s)
	}
}

func TestReportMembershipNoteIsIncluded(t *testing.T) {
	req := membershipReq(ActionUpdate, Membership{
		Noun: "users", Current: 2, CurrentKnown: true, Next: 5, NextKnown: true, Changed: true,
	})
	req.Label = "static user group"
	req.Note = "Members are users; which devices they affect depends on the devices those users are assigned to."
	diags := ReportMembership(context.Background(), req)
	if d := diags[0].Detail(); !strings.Contains(d, "depends on the devices those users are assigned to") {
		t.Fatalf("the note must be surfaced: %q", d)
	}
}

func TestReportMembershipNothingKnownIsSilent(t *testing.T) {
	diags := ReportMembership(context.Background(), membershipReq(ActionUpdate, Membership{
		Noun: "computers", Changed: true,
	}))
	if len(diags) != 0 {
		t.Fatalf("with no figure and no reason there is nothing to say, got %q", diags[0].Summary())
	}
}

func TestReportMembershipDeleteDoesNotExplainFutureMembership(t *testing.T) {
	// A deleted group has no membership afterwards, so explaining that Jamf Pro
	// will re-evaluate its criteria reads as nonsense. Caught on a live destroy plan.
	req := membershipReq(ActionDelete, Membership{
		Noun: "computers", Current: 2, CurrentKnown: true,
		Undetermined: CriteriaUndetermined,
	})
	req.Label = "smart computer group"
	diags := ReportMembership(context.Background(), req)
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	d := diags[0].Detail()
	if strings.Contains(d, "Membership after this change") {
		t.Fatalf("a delete must not describe membership after the change: %q", d)
	}
	if !strings.Contains(d, "stops applying to its members") {
		t.Fatalf("a delete must still state what stops: %q", d)
	}
}

func TestCountOfUsesTheCallerSingular(t *testing.T) {
	// "classes" trimmed of its "s" is "classe", so an irregular plural must come
	// with its singular form from the caller.
	req := membershipReq(ActionDelete, Membership{
		Noun: "classes", NounSingular: "class", Current: 1, CurrentKnown: true,
	})
	req.Label = "class"
	diags := ReportMembership(context.Background(), req)
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if s := diags[0].Summary(); !strings.Contains(s, "affects 1 class") || strings.Contains(s, "classe") {
		t.Fatalf("the singular must come from the caller, not from trimming: %q", s)
	}
}
