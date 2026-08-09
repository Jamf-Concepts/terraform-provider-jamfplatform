// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// Action is the lifecycle operation planned for the object being reported on.
type Action int

const (
	// ActionCreate is a new object entering management.
	ActionCreate Action = iota
	// ActionUpdate is a change to an existing object.
	ActionUpdate
	// ActionDelete is removal of an existing object.
	ActionDelete
)

// Kind mirrors the two channels Jamf Pro's impact alert notifications
// distinguish. Jamf Pro keeps them separate because editing a group and editing
// something scoped to that group are different events with different audiences,
// and the provider follows suit: a plan that does both produces one alert from
// each side rather than one combined figure.
type Kind int

const (
	// Deployable is an object deployed to devices — a policy, a configuration
	// profile, an app, a blueprint, a compliance benchmark.
	Deployable Kind = iota
	// Scopeable is an object that scope can be based on — a smart or static
	// group, or a class.
	Scopeable
)

// Request describes one object's planned change for impact reporting.
type Request struct {
	// Cache is the shared tenant cache. A nil cache disables reporting.
	Cache *Cache
	// Path anchors the diagnostic to the attribute the figure derives from, so
	// the warning appears next to the scope block rather than the whole resource.
	Path path.Path
	// Kind selects the deployable or scopeable channel.
	Kind Kind
	// Label names the object in prose, using the admin UI's term for it
	// (e.g. "policy", "configuration profile", "smart computer group").
	Label string
	// Action is the planned lifecycle operation.
	Action Action
	// Prior is the scope currently in state. Zero for a create.
	Prior Scope
	// Planned is the scope this plan intends. Zero for a delete.
	Planned Scope
}

// Report produces the impact alert for one planned change, as advisory warning
// diagnostics.
//
// It returns no diagnostics when reporting is disabled, when the scope is
// unchanged, or when the tenant could not be read — impact reporting is
// advisory and must never be the reason a plan fails. A tenant that cannot be
// read produces a single notice for the whole plan.
func Report(ctx context.Context, req Request) diag.Diagnostics {
	var diags diag.Diagnostics
	if !req.Cache.Enabled() {
		return diags
	}

	subject := req.Planned
	if req.Action == ActionDelete {
		subject = req.Prior
	}
	if subject.Empty() {
		return diags
	}
	if req.Action == ActionUpdate && req.Prior.equal(req.Planned) {
		// Jamf Pro alerts on save, not on view. A plan that leaves scope alone
		// gets no alert, otherwise every plan would carry one warning per
		// scoped object.
		return diags
	}

	res, err := Resolve(ctx, req.Cache, subject)
	if err != nil {
		if req.Cache.noticeOnce() {
			diags.AddWarning(
				"Impact alert unavailable",
				fmt.Sprintf("Scope impact could not be calculated for this plan: %s\n\n"+
					"Impact alerts are advisory, so the plan is unaffected. No further notices will be shown for this plan.", err),
			)
		}
		return diags
	}

	if !res.Determinable {
		diags.AddAttributeWarning(req.Path, headline(req, res), pendingDetail(res.PendingPaths))
		return diags
	}

	diags.AddAttributeWarning(req.Path, headline(req, res), detail(ctx, req, res))
	return diags
}

// headline renders the one-line summary Terraform shows above the detail.
func headline(req Request, res Resolution) string {
	if !res.Determinable {
		return fmt.Sprintf("Impact alert — affected %s cannot be determined during plan", res.DeviceType.Noun())
	}
	f := summarise(res)
	switch req.Action {
	case ActionCreate:
		return fmt.Sprintf("Impact alert — this %s will be scoped to %s", req.Label, f)
	case ActionDelete:
		return fmt.Sprintf("Impact alert — removing this %s affects %s", req.Label, f)
	default:
		return fmt.Sprintf("Impact alert — this %s affects %s", req.Label, f)
	}
}

// detail renders the body: what was counted, what is changing, what could not be
// evaluated, and the snapshot caveat that closes every alert.
func detail(ctx context.Context, req Request, res Resolution) string {
	var b strings.Builder

	switch req.Kind {
	case Scopeable:
		fmt.Fprintf(&b, "Changing this %s changes what every object scoped to it applies to.\n\n", req.Label)
	default:
		if req.Action == ActionDelete {
			fmt.Fprintf(&b, "These %s will stop receiving this %s.\n\n", res.DeviceType.Noun(), req.Label)
		}
	}

	if lines := breakdown(res); len(lines) > 0 {
		b.WriteString("Counted from " + strings.Join(lines, "; ") + ".\n")
	}
	if l := excludedLine(res); l != "" {
		b.WriteString(l + "\n")
	}

	if req.Action == ActionUpdate {
		if d := deltaLine(ctx, req); d != "" {
			b.WriteString(d + "\n")
		}
	}

	if lines := caveats(res); len(lines) > 0 {
		b.WriteString("\n" + strings.Join(lines, "\n") + "\n")
	}

	b.WriteString("\n" + snapshotNote)
	return b.String()
}

// deltaLine renders the devices entering and leaving scope — the figure Jamf
// Pro's own alert leads with, and the one a plan is uniquely able to produce
// because it holds the prior and intended scope together.
//
// A failure here is silently dropped: the headline figure has already been
// resolved successfully, so a partial delta is not worth surfacing an error for.
func deltaLine(ctx context.Context, req Request) string {
	added, removed := Delta(req.Prior, req.Planned)

	var parts []string
	if !added.Empty() {
		if r, err := Resolve(ctx, req.Cache, added); err == nil && r.Determinable && r.Count > 0 {
			parts = append(parts, "adding "+figure(r.Count, r.Bound, r.DeviceType.Noun()))
		}
	}
	if !removed.Empty() {
		if r, err := Resolve(ctx, req.Cache, removed); err == nil && r.Determinable && r.Count > 0 {
			parts = append(parts, "removing "+figure(r.Count, r.Bound, r.DeviceType.Noun()))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "This change is " + strings.Join(parts, " and ") + "."
}

// equal reports whether two scopes name the same things, so an unchanged scope
// can be skipped.
func (s Scope) equal(o Scope) bool {
	if s.All != o.All || s.DeviceType != o.DeviceType {
		return false
	}
	if !sameStrings(s.DeviceIDs, o.DeviceIDs) ||
		!sameRefs(s.ProGroups, o.ProGroups) ||
		!sameStrings(s.PlatformGroupIDs, o.PlatformGroupIDs) ||
		!sameStrings(s.ExcludedDeviceIDs, o.ExcludedDeviceIDs) ||
		!sameRefs(s.ExcludedProGroups, o.ExcludedProGroups) ||
		!sameStrings(s.ExcludedPlatformGroupIDs, o.ExcludedPlatformGroupIDs) ||
		!sameStrings(s.PendingPaths, o.PendingPaths) {
		return false
	}
	if len(s.Unresolvable) != len(o.Unresolvable) {
		return false
	}
	left := unresolvableKeys(s.Unresolvable)
	right := unresolvableKeys(o.Unresolvable)
	return sameStrings(left, right)
}

func unresolvableKeys(us []Unresolvable) []string {
	out := make([]string, 0, len(us))
	for _, u := range us {
		out = append(out, fmt.Sprintf("%s=%d", u.Path, u.Values))
	}
	return out
}

// sameRefs compares two group-reference slices as sets.
func sameRefs(a, b []ProGroupRef) bool {
	if len(a) != len(b) {
		return false
	}
	x := make([]string, 0, len(a))
	y := make([]string, 0, len(b))
	for _, v := range a {
		x = append(x, v.key())
	}
	for _, v := range b {
		y = append(y, v.key())
	}
	return sameStrings(x, y)
}

// sameStrings compares two slices as sets, tolerating order differences because
// scope collections are unordered.
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]string(nil), a...)
	y := append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
