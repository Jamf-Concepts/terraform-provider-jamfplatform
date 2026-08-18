// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// DependencyRequest describes one planned change to a policy dependency.
type DependencyRequest struct {
	// Cache is the shared tenant cache. A nil cache, or one without a policy
	// source, disables reporting.
	Cache *Cache
	// Path optionally anchors the diagnostic to an attribute. A dependency alert has
	// no single attribute it derives from — any change reaches every policy using the
	// object — so this is normally empty, reporting at resource level rather than
	// blaming a field that may not be the one that changed.
	Path path.Path
	// Kind is the dependency category, used to look up the reverse index.
	Kind DependencyKind
	// ID is the dependency's numeric Jamf Pro id. Empty during a create, since
	// the object does not exist yet and nothing can reference it.
	ID string
	// Name names the object in prose. Falls back to the kind when empty.
	Name string
	// Action is the planned lifecycle operation.
	Action Action
	// Changed reports whether this plan changes the object at all.
	Changed bool
}

// ReportDependency produces the impact alert for a change to a policy dependency:
// how many computers it reaches, via every policy using it.
//
// This is the deployable channel's blind spot. A policy's own alert covers a change
// to that policy and a group's covers that group, but a script edited alone produces
// neither — it has no scope, and the policies running it are not in the plan. Jamf
// Pro's save-time alert has the same gap.
//
// Advisory: an unsweepable tenant yields one notice per plan and never fails it.
//
// The wording is load-bearing. Alerts are consumed from `terraform plan -json` by
// regex, so the headline keeps the shared "affects <qualifier><n> of <m>" shape and
// the detail keeps its counts in fixed phrases. See TestDependencyAlert_IsMachineReadable
// and the CI section of docs/guides/impact-alerts.md before rewording any of it.
func ReportDependency(ctx context.Context, req DependencyRequest) diag.Diagnostics {
	var diags diag.Diagnostics
	if req.Cache == nil || req.Cache.policySrc == nil {
		return diags
	}
	// Nothing can reference an id the tenant has not issued, so a create must not
	// sweep. On a plan creating a script and a policy together the policy's own
	// alert already covers it.
	if req.Action == ActionCreate || req.ID == "" {
		return diags
	}
	if req.Action == ActionUpdate && !req.Changed {
		return diags
	}

	uses, stats, err := req.Cache.PolicyUses(ctx, req.Kind, req.ID)
	if err != nil {
		if req.Cache.noticeOnce() {
			diags.AddWarning(
				"Impact alert unavailable",
				fmt.Sprintf("Policy dependency impact could not be calculated for this plan: %s\n\n"+
					"Impact alerts are advisory, so the plan is unaffected. No further notices will be shown for this plan.", err),
			)
		}
		return diags
	}

	label := req.Name
	if label == "" {
		label = string(req.Kind)
	}

	if len(uses) == 0 {
		addDependencyWarning(&diags, req.Path,
			dependencyUnusedHeadline(req.Kind, stats),
			dependencyUnusedDetail(req.Kind, label, stats))
		return diags
	}

	res, err := resolveDependency(ctx, req.Cache, uses, stats)
	if err != nil {
		if req.Cache.noticeOnce() {
			diags.AddWarning(
				"Impact alert unavailable",
				fmt.Sprintf("Policy dependency impact could not be calculated for this plan: %s\n\n"+
					"Impact alerts are advisory, so the plan is unaffected. No further notices will be shown for this plan.", err),
			)
		}
		return diags
	}

	addDependencyWarning(&diags, req.Path,
		dependencyHeadline(req, res),
		dependencyDetail(req, res, label, stats))
	return diags
}

// dependencyUnusedHeadline renders the summary for a dependency nothing references.
//
// Only a complete sweep may state it flatly. An unread policy is precisely the thing
// that could contradict "no policy uses this", so a shortfall has to reach the
// headline rather than being buried in the body.
func dependencyUnusedHeadline(kind DependencyKind, stats SweepStats) string {
	if !stats.Complete() {
		return fmt.Sprintf("Impact alert — no policy found using this %s, but the search was incomplete", kind)
	}
	return fmt.Sprintf("Impact alert — no policy uses this %s", kind)
}

// patchManagementBoundary names the delivery channel the policy sweep does not read.
//
// Packages alone need it: `jamfplatform_pro_patch_software_title` assigns a package
// per software version and `jamfplatform_pro_patch_policy` carries its own scope, so
// a package can be delivered with no ordinary policy referencing it — and a denial
// phrased "reaches no computers through a policy" would then read as an all-clear.
// The other five kinds have no second consumer, so their wording must not take this
// on.
const patchManagementBoundary = "Patch Management version-to-package assignments are not searched, " +
	"so a package delivered only by a patch policy is not counted here."

// dependencyUnusedDetail renders the body for a dependency nothing references.
func dependencyUnusedDetail(kind DependencyKind, label string, stats SweepStats) string {
	var b strings.Builder
	switch {
	case !stats.Complete():
		fmt.Fprintf(&b, "No policy that could be read references %s. The sweep did not cover the whole "+
			"tenant, so a policy that went unread may use it.\n\n", label)
		if kind == DependencyPackage {
			fmt.Fprintf(&b, "%s\n\n", patchManagementBoundary)
		}
	case kind == DependencyPackage:
		fmt.Fprintf(&b, "No policy in this tenant references %s. %s\n\n", label, patchManagementBoundary)
	default:
		fmt.Fprintf(&b, "No policy in this tenant references %s, so this change reaches no computers "+
			"through a policy.\n\n", label)
	}
	fmt.Fprintf(&b, "%s\n\n%s", sweepSentence(stats), dependencyNote)
	return b.String()
}

// sweepSentence states what the sweep covered, disclosing any shortfall.
//
// "Searched <n> polic(y|ies)" stays first and intact in both branches: CI pipelines
// anchor a regex on that phrase, so a partial sweep is disclosed after it rather
// than rewritten into it. Reporting the successful count as what was searched — and
// the listed total beside it — is what stops an alert claiming to have read policies
// it could not.
func sweepSentence(stats SweepStats) string {
	if stats.Complete() {
		return fmt.Sprintf("Searched %s.", plural(stats.Searched, "policy", "policies"))
	}
	return fmt.Sprintf("Searched %s of %d (%d could not be read, so this answer may be incomplete).",
		plural(stats.Searched, "policy", "policies"), stats.Listed(), stats.Unreadable)
}

// addDependencyWarning attaches the alert to an attribute when one was named, and
// to the resource otherwise.
func addDependencyWarning(diags *diag.Diagnostics, p path.Path, summary, detail string) {
	if p.Equal(path.Empty()) {
		diags.AddWarning(summary, detail)
		return
	}
	diags.AddAttributeWarning(p, summary, detail)
}

// dependencyNote closes every dependency alert. Distinct from snapshotNote because
// the sweep adds its own staleness: which policies use the object is also a
// point-in-time reading.
const dependencyNote = "Policy usage and group membership are a plan-time snapshot and can change before apply."

// DependencyResolution is the combined reach of the policies using one
// dependency.
type DependencyResolution struct {
	// Resolution is the union of the using policies' scopes.
	Resolution
	// Enabled and Disabled split the using policies. A disabled policy references the
	// object but delivers nothing, so it is disclosed separately, never counted.
	Enabled  []PolicyUse
	Disabled []PolicyUse
}

// resolveDependency unions the scopes of every enabled policy using a dependency.
//
// Over member sets, not by summing per-policy counts: scopes overlap heavily, so a
// computer in three affected policies must count once. Summing would inflate the
// figure without bound, the more so the better-organised the tenant.
//
// A policy whose scope will not resolve exactly degrades the bound rather than the
// figure, so one network-segment limitation does not discard the rest.
//
// An incomplete sweep degrades the bound the other way, upwards: see below.
func resolveDependency(ctx context.Context, c *Cache, uses []PolicyUse, stats SweepStats) (DependencyResolution, error) {
	out := DependencyResolution{}
	for _, u := range uses {
		if u.Enabled {
			out.Enabled = append(out.Enabled, u)
			continue
		}
		out.Disabled = append(out.Disabled, u)
	}

	// With nothing enabled there is no figure to render: the headline says the object is
	// used only by disabled policies and the detail says it reaches nothing yet, and
	// neither shows a count. Resolve would buy nothing, and a tenant read that failed
	// here could only turn a complete answer into an "unavailable" notice.
	//
	// Note how narrow this is. An *enabled* policy with an empty scope does render a
	// figure, and has to go through Resolve to get its denominator — the earlier version
	// of this early return keyed on the combined scope being empty, which caught both
	// cases and left that alert as the only dependency headline with no "of N" clause.
	if len(out.Enabled) == 0 {
		out.Resolution = Resolution{DeviceType: DeviceTypeComputer, Determinable: true}
		return out, nil
	}

	// Only enabled policies deliver anything, so only they contribute devices.
	combined := combineScopes(scopesOf(out.Enabled))

	if !stats.Complete() {
		// An unread policy can only add audience: it cannot take a device away from a
		// policy that was read, and its own exclusions would not combine anyway. So an
		// incomplete sweep makes the figure a lower bound — a known direction, which is
		// exactly what Bound exists to express — and it goes in through the same channel
		// as every other unquantifiable input rather than living only in a prose
		// sentence. Resolve folds it into the bound for free and caveats() lists it
		// beside the rest.
		//
		// Copied rather than appended in place: with one contributor combineScopes
		// returns that policy's own Scope verbatim, and its Unresolvable slice may have
		// spare capacity, so appending would rewrite what the index holds for that
		// policy for the rest of the plan.
		combined.Unresolvable = append(append([]Unresolvable(nil), combined.Unresolvable...),
			Unresolvable{
				Path:   "unread policies",
				Reason: reasonUnreadPolicies,
				Effect: Broadens,
				Values: stats.Unreadable,
			})
	}

	res, err := Resolve(ctx, c, combined)
	if err != nil {
		return out, err
	}
	out.Resolution = res
	return out, nil
}

// scopesOf extracts the scopes of a set of uses.
func scopesOf(uses []PolicyUse) []Scope {
	out := make([]Scope, 0, len(uses))
	for _, u := range uses {
		out = append(out, u.Scope)
	}
	return out
}

// reasonUnreadPolicies is the caveat text for policies the sweep could not read. Their
// references are missing from the index, so any audience they add is missing from the
// figure — which is why the figure becomes a lower bound rather than merely a hedged
// one.
const reasonUnreadPolicies = "could not be read during this plan, so any dependency they use is missing from this figure"

// exclusionsAreNotCombinable is the caveat text for exclusions dropped because only
// some of the contributing policies carry them.
const exclusionsAreNotCombinable = "exclusions apply per policy, so an exclusion only some of them carry cannot be subtracted from a combined audience"

// combineScopes merges every contributing policy's scope into one whose audience is
// their union.
//
// Targets union straightforwardly; exclusions do not. An exclusion belongs to the
// policy declaring it, so a computer excluded from A but targeted by B still receives
// the dependency through B, and subtracting that exclusion from the combined target
// set undercounts. An exclusion every contributor carries is the exception, and is
// kept — see combineExclusions. The rest become a single aggregated narrowing caveat:
// stating the direction of the error is honest, applying it to the wrong audience is
// not.
//
// One pass, not a pairwise fold. Folding re-examined the accumulator each step and
// re-emitted its caveats per remaining policy — 56 copies of one line on a real tenant.
//
// Only the fields policyWireScope can populate are carried across. AllEstates,
// MentionedPlatformIDs and PendingPaths are dropped silently, which is correct only
// because a policy scope never sets them: policies are computers-only, and arrive
// fully resolved from the wire so nothing can pend. Teach policyWireScope to set any
// of those and this loop has to learn it too, or the union will quietly under-report.
func combineScopes(scopes []Scope) Scope {
	switch len(scopes) {
	case 0:
		return Scope{DeviceType: DeviceTypeComputer}
	case 1:
		return scopes[0]
	}

	out := Scope{DeviceType: DeviceTypeComputer}
	for _, s := range scopes {
		out.All = out.All || s.All
		out.DeviceIDs = unionStrings(out.DeviceIDs, s.DeviceIDs)
		out.BuildingIDs = unionStrings(out.BuildingIDs, s.BuildingIDs)
		out.DepartmentIDs = unionStrings(out.DepartmentIDs, s.DepartmentIDs)
		out.PlatformGroupIDs = unionStrings(out.PlatformGroupIDs, s.PlatformGroupIDs)
		out.ProGroups = unionRefs(out.ProGroups, s.ProGroups)
		out.Unresolvable = append(out.Unresolvable, s.Unresolvable...)
	}

	shared, droppedExclusions := combineExclusions(scopes)
	out.ExcludedProGroups = shared.ExcludedProGroups
	out.ExcludedPlatformGroupIDs = shared.ExcludedPlatformGroupIDs
	out.ExcludedDeviceIDs = shared.ExcludedDeviceIDs
	out.ExcludedBuildingIDs = shared.ExcludedBuildingIDs
	out.ExcludedDepartmentIDs = shared.ExcludedDepartmentIDs
	if droppedExclusions > 0 {
		out.Unresolvable = append(out.Unresolvable, Unresolvable{
			Path:   "policy exclusions",
			Reason: exclusionsAreNotCombinable,
			Effect: Narrows,
			Values: droppedExclusions,
		})
	}
	// Identical inputs collapse into one entry carrying the total, so 56 policies
	// sharing a network-segment limitation yield one line, not 56.
	out.Unresolvable = aggregateUnresolvable(out.Unresolvable)
	return out
}

// combineExclusions splits the contributors' exclusions into the ones every single
// contributor carries — which can be subtracted from the union — and a count of the
// rest, which cannot.
//
// The intersection is sound where the individual exclusions are not: a device excluded
// by every contributing policy receives the object from none of them, so removing it
// from the union removes something that was never in it. Dropping those too made the
// bound jump discontinuously at the 1→2 boundary — two policies both targeting every
// computer and both excluding one large kiosk group went from an exact "400 of 1000"
// to "up to 1000 of 1000" purely because a second policy joined, which is the figure
// an administrator is least able to act on.
//
// Matched by reference, not by resolved membership. Policy A excluding a group and
// policy B excluding a device inside that group is not treated as agreement, so such
// a pair still falls into the caveat. That under-subtracts, which errs towards the
// larger figure the caveat is already warning about — the safe direction.
//
// References are deduplicated within each policy before they are counted, so the same
// group named twice in one policy's exclusions cannot masquerade as two policies
// agreeing.
func combineExclusions(scopes []Scope) (shared Scope, dropped int) {
	n := len(scopes)
	groups := make(map[string]ProGroupRef)
	groupCounts := make(map[string]int)
	deviceCounts := make(map[string]int)
	buildingCounts := make(map[string]int)
	departmentCounts := make(map[string]int)
	platformGroupCounts := make(map[string]int)

	countUnique := func(counts map[string]int, ids []string) {
		local := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			if _, dup := local[id]; dup {
				continue
			}
			local[id] = struct{}{}
			counts[id]++
		}
	}
	for _, s := range scopes {
		local := make(map[string]struct{}, len(s.ExcludedProGroups))
		for _, g := range s.ExcludedProGroups {
			key := g.key()
			if _, dup := local[key]; dup {
				continue
			}
			local[key] = struct{}{}
			groups[key] = g
			groupCounts[key]++
		}
		countUnique(deviceCounts, s.ExcludedDeviceIDs)
		countUnique(buildingCounts, s.ExcludedBuildingIDs)
		countUnique(departmentCounts, s.ExcludedDepartmentIDs)
		countUnique(platformGroupCounts, s.ExcludedPlatformGroupIDs)
	}

	keys := make([]string, 0, len(groupCounts))
	for key, c := range groupCounts {
		if c == n {
			keys = append(keys, key)
			continue
		}
		dropped += c
	}
	// Sorted, because map iteration order would otherwise reshuffle the excluded-group
	// list between plans and make the same change read differently each time.
	sort.Strings(keys)
	for _, key := range keys {
		shared.ExcludedProGroups = append(shared.ExcludedProGroups, groups[key])
	}
	keep := func(counts map[string]int, into *[]string) {
		for id, c := range counts {
			if c == n {
				*into = append(*into, id)
				continue
			}
			dropped += c
		}
		sort.Strings(*into)
	}
	keep(deviceCounts, &shared.ExcludedDeviceIDs)
	keep(buildingCounts, &shared.ExcludedBuildingIDs)
	keep(departmentCounts, &shared.ExcludedDepartmentIDs)
	keep(platformGroupCounts, &shared.ExcludedPlatformGroupIDs)
	return shared, dropped
}

// aggregateUnresolvable collapses entries sharing a path, reason and direction into
// one carrying the summed count, keeping first-seen order so wording stays stable.
func aggregateUnresolvable(in []Unresolvable) []Unresolvable {
	if len(in) < 2 {
		return in
	}
	type key struct {
		path, reason string
		effect       Effect
	}
	seen := make(map[key]int, len(in))
	out := make([]Unresolvable, 0, len(in))
	for _, u := range in {
		k := key{u.Path, u.Reason, u.Effect}
		if i, ok := seen[k]; ok {
			out[i].Values += u.Values
			continue
		}
		seen[k] = len(out)
		out = append(out, u)
	}
	return out
}

// unionStrings merges two id slices as a set, preserving first-seen order so a
// diagnostic reads the same across plans.
//
// The empty-side fast paths return the other input by reference, so the result can
// alias a PolicyUse's own backing array. Callers must treat it as read-only:
// appending to it in place would rewrite the scope the index holds for that policy,
// and the index outlives this call for the whole plan.
func unionStrings(a, b []string) []string {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, v := range append(append([]string(nil), a...), b...) {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// unionRefs merges two group-reference slices as a set. Aliases an input on the
// empty-side fast paths, exactly as unionStrings does, and with the same read-only
// obligation on the caller.
func unionRefs(a, b []ProGroupRef) []ProGroupRef {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]ProGroupRef, 0, len(a)+len(b))
	for _, v := range append(append([]ProGroupRef(nil), a...), b...) {
		if _, ok := seen[v.key()]; ok {
			continue
		}
		seen[v.key()] = struct{}{}
		out = append(out, v)
	}
	return out
}

// dependencyHeadline renders the one-line summary.
func dependencyHeadline(req DependencyRequest, res DependencyResolution) string {
	if len(res.Enabled) == 0 {
		return fmt.Sprintf("Impact alert — this %s is used only by disabled policies", req.Kind)
	}
	if !res.Determinable {
		return "Impact alert — affected computers cannot be determined during plan"
	}
	via := plural(len(res.Enabled), "policy", "policies")
	if req.Action == ActionDelete {
		return fmt.Sprintf("Impact alert — removing this %s affects %s, via %s",
			req.Kind, summarise(res.Resolution), via)
	}
	return fmt.Sprintf("Impact alert — this %s affects %s, via %s",
		req.Kind, summarise(res.Resolution), via)
}

// dependencyDetail renders the body: which policies carry the change, what the figure
// was counted from, and what could not be evaluated.
//
// No lead sentence explaining that a dependency change reaches every policy using it —
// the headline already says "affects N computers, via M policies", and repeating it
// pushed the policy list below the fold.
func dependencyDetail(req DependencyRequest, res DependencyResolution, label string, stats SweepStats) string {
	var b strings.Builder

	// "Used by N enabled policies" and "Searched N policies" are fixed phrases that
	// CI regexes anchor on; keep the counts where they are.
	if len(res.Enabled) > 0 {
		fmt.Fprintf(&b, "Used by %s: %s.\n",
			plural(len(res.Enabled), "enabled policy", "enabled policies"),
			policyNames(res.Enabled, maxListedPolicies))
	}
	if len(res.Disabled) > 0 {
		// Disclosed rather than hidden: staging a change on a disabled policy is
		// common, and its audience is what the figure becomes once enabled.
		fmt.Fprintf(&b, "Also %s, delivering nothing until enabled: %s.\n",
			plural(len(res.Disabled), "disabled policy", "disabled policies"),
			policyNames(res.Disabled, maxListedDisabledPolicies))
	}

	if len(res.Enabled) == 0 {
		fmt.Fprintf(&b, "\nNo enabled policy uses %s, so this reaches no computers yet.\n", label)
		fmt.Fprintf(&b, "\n%s %s", sweepSentence(stats), dependencyNote)
		return b.String()
	}

	if !res.Determinable {
		fmt.Fprintf(&b, "\n%s", pendingDetail(res.PendingPaths))
		return b.String()
	}

	lines := breakdown(res.Resolution)
	if len(lines) > 0 {
		fmt.Fprintf(&b, "Counted from their combined scope: %s.\n", strings.Join(lines, "; "))
	}
	if l := excludedLine(res.Resolution); l != "" {
		fmt.Fprintf(&b, "%s\n", l)
	}
	if len(res.Enabled) > 1 && len(lines) > 0 {
		// Says why the total is not the sum of the parts — the first thing a reader
		// checks when they know each policy's own audience. Withheld when the breakdown
		// produced nothing, since there is then no union for it to be talking about:
		// two unscoped policies reach nobody, and claiming an overlap over an empty
		// audience invites the reader to look for a figure that is not there.
		b.WriteString("Computers in more than one are counted once.\n")
	}

	if lines := caveats(res.Resolution); len(lines) > 0 {
		fmt.Fprintf(&b, "\n%s\n", strings.Join(lines, "\n"))
	}

	fmt.Fprintf(&b, "\n%s %s", sweepSentence(stats), dependencyNote)
	return b.String()
}

// maxListedPolicies caps listed policy names so a widely-used script stays readable.
const maxListedPolicies = 8

// maxListedDisabledPolicies caps the disabled aside more tightly than the primary
// list. It qualifies the figure rather than carrying it, so sharing the larger cap
// let nine disabled policies print eight full names — as much room as the "Used by"
// line the aside is subordinate to.
const maxListedDisabledPolicies = 3

// policyNames renders the using policies, capped at most names.
func policyNames(uses []PolicyUse, most int) string {
	names := make([]string, 0, len(uses))
	for _, u := range uses {
		names = append(names, sanitisePolicyName(u.Name))
	}
	sort.Strings(names)
	if len(names) > most {
		rest := len(names) - most
		names = append(names[:most], fmt.Sprintf("and %d more", rest))
	}
	return strings.Join(names, ", ")
}

// sanitisePolicyName neutralises control characters in an administrator-supplied
// policy name.
//
// Names are interpolated into the same string that carries the phrases CI pipelines
// match on, and "Searched N policies." renders after them. A name holding a raw
// newline could therefore forge an anchor line ahead of the real one and win a
// first-match extraction. Replacing with a space rather than stripping keeps the name
// recognisable to whoever has to go and find it.
func sanitisePolicyName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, name)
}

// DependencyPlanReport is the per-resource configuration for the shared
// dependency plan hook.
type DependencyPlanReport struct {
	// Cache is the shared tenant cache.
	Cache *Cache
	// Path anchors the diagnostic.
	Path path.Path
	// Kind is the dependency category.
	Kind DependencyKind
}

// ReportDependencyPlan is the shared ModifyPlan hook for a policy dependency
// resource. It mirrors ReportPlan's lifecycle bookkeeping; the caller supplies only
// how to read the id and name out of its own model.
func ReportDependencyPlan[M any](
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
	rep DependencyPlanReport,
	identify func(context.Context, *M) (id string, name string),
) {
	if rep.Cache == nil || rep.Cache.policySrc == nil {
		return
	}
	creating := req.State.Raw.IsNull()
	destroying := req.Plan.Raw.IsNull()
	if creating && destroying {
		return
	}
	// Nothing references an object this plan is creating, so there is no sweep to
	// justify. Bail before touching the tenant.
	if creating {
		return
	}

	action := ActionUpdate
	if destroying {
		action = ActionDelete
	}

	// The id is server-assigned and so only ever present in prior state; a
	// planned-only read would be unknown on an update that replaces the object.
	var state M
	if diags := req.State.Get(ctx, &state); diags.HasError() {
		return
	}
	id, name := identify(ctx, &state)

	resp.Diagnostics.Append(ReportDependency(ctx, DependencyRequest{
		Cache:   rep.Cache,
		Path:    rep.Path,
		Kind:    rep.Kind,
		ID:      id,
		Name:    name,
		Action:  action,
		Changed: !req.Plan.Raw.Equal(req.State.Raw),
	})...)
}
