// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"sort"
	"strings"
)

// Effect is the direction an unresolvable scope input moves the true device
// count in, relative to what the provider can count.
//
// The distinction matters because Jamf Pro's scope model is not additive
// throughout: targets build the audience, limitations narrow it, and exclusions
// remove from it. An input the provider cannot evaluate therefore does not
// always mean "the real number is higher" — for a limitation it means the real
// number is lower.
type Effect int

const (
	// Narrows means the true count is at most what was counted. Limitations and
	// exclusions narrow.
	Narrows Effect = iota
	// Broadens means the true count is at least what was counted. Targets the
	// provider cannot enumerate broaden.
	Broadens
	// Ambiguous means the input can move the count in either direction, so the
	// figure bounds it from neither side. A blueprint's activation conditions are
	// the motivating case: the same expression syntax can require a group or
	// exclude one, so which way it moves the audience is not evident from the
	// group references alone.
	Ambiguous
)

// Unresolvable records one scope input that cannot be evaluated during a plan,
// with the reason stated in terms an administrator can act on.
type Unresolvable struct {
	// Path is the attribute path as written in configuration, e.g.
	// "limitations.network_segment_ids".
	Path string
	// Reason explains why the input cannot be evaluated ahead of time. It is
	// user-facing text and must not reference API plumbing.
	Reason string
	// Effect is the direction this input moves the true count in.
	Effect Effect
	// Values is how many entries the attribute holds.
	Values int
}

// Scope is a device-type-neutral description of one resource's scope, populated
// by a per-resource adapter. It deliberately does not mirror any single
// resource's schema: the Jamf Pro scope block, a blueprint's device group set
// and a compliance benchmark's targets all reduce to this.
type Scope struct {
	DeviceType DeviceType

	// All is the tenant-wide flag (all_computers / all_mobile_devices).
	All bool
	// DeviceIDs are individually scoped devices (computer_ids / mobile_device_ids).
	DeviceIDs []string
	// JamfProGroupIDs are groups referenced by numeric Jamf Pro id.
	JamfProGroupIDs []string
	// PlatformGroupIDs are groups referenced by Platform UUID.
	PlatformGroupIDs []string
	// MentionedPlatformIDs are groups the configuration refers to but which are
	// deliberately not counted, because whether they widen or narrow the audience
	// is not determinable. Their names are surfaced so a reviewer can see which
	// groups are involved without being given a total that implies more certainty
	// than there is.
	MentionedPlatformIDs []string

	// ExcludedJamfProGroupIDs and ExcludedPlatformGroupIDs are groups removed from
	// the audience. They are carried as data rather than as narrowing caveats so
	// their membership can be subtracted exactly; when membership cannot be read
	// they fall back to being reported as narrowing.
	ExcludedJamfProGroupIDs  []string
	ExcludedPlatformGroupIDs []string
	// ExcludedDeviceIDs are individually excluded devices. These are Jamf Pro
	// numeric identifiers, whereas group membership is expressed in device
	// management identifiers, so they cannot be subtracted from a membership set
	// and remain a narrowing caveat.
	ExcludedDeviceIDs []string

	// Unresolvable holds inputs that cannot be evaluated during a plan.
	Unresolvable []Unresolvable
	// PendingPaths holds attribute paths whose values are not yet known because
	// the object they reference is created by this same plan. Any entry here
	// makes the scope undeterminable — a group created by this plan has no
	// membership until after it is applied, and for a smart group that
	// membership is decided by Jamf Pro, not by the configuration.
	PendingPaths []string
}

// Empty reports whether the scope names nothing at all.
func (s Scope) Empty() bool {
	return !s.All &&
		len(s.DeviceIDs) == 0 &&
		len(s.JamfProGroupIDs) == 0 &&
		len(s.PlatformGroupIDs) == 0 &&
		len(s.MentionedPlatformIDs) == 0 &&
		len(s.ExcludedJamfProGroupIDs) == 0 &&
		len(s.ExcludedPlatformGroupIDs) == 0 &&
		len(s.ExcludedDeviceIDs) == 0 &&
		len(s.Unresolvable) == 0 &&
		len(s.PendingPaths) == 0
}

// Bound describes how a counted figure relates to the true device count.
type Bound int

const (
	// BoundExact means every input was countable.
	BoundExact Bound = iota
	// BoundAtMost means the true count is the counted figure or lower.
	BoundAtMost
	// BoundAtLeast means the true count is the counted figure or higher.
	BoundAtLeast
	// BoundUnknown means inputs push in both directions, so the figure is
	// neither an upper nor a lower bound.
	BoundUnknown
)

// with folds an additional effect into a bound.
func (b Bound) with(e Effect) Bound {
	if e == Ambiguous {
		return BoundUnknown
	}
	next := BoundAtMost
	if e == Broadens {
		next = BoundAtLeast
	}
	switch b {
	case BoundExact:
		return next
	case next:
		return b
	default:
		return BoundUnknown
	}
}

// Resolution is the outcome of counting a Scope against the tenant.
type Resolution struct {
	DeviceType DeviceType
	// Count is the number of devices the countable inputs resolve to.
	Count int64
	// Total is the tenant's managed device count for this device type, or zero
	// when unknown.
	Total int64
	// Bound describes how Count relates to the true figure.
	Bound Bound
	// Determinable is false when the scope references something this plan
	// creates, in which case Count carries no meaning and must not be shown.
	Determinable bool

	// Groups are the target groups that were found and counted.
	Groups []Group
	// ExcludedGroups are the excluded groups that were found. Rendered in the
	// breakdown so a figure smaller than the targets suggest is explicable.
	ExcludedGroups []Group
	// tenantWide records that the scope targeted the whole estate, so the
	// breakdown can say so rather than listing nothing.
	tenantWide bool
	// DirectDevices is the number of individually scoped devices counted.
	DirectDevices int
	// MissingGroupIDs are referenced groups that are not present in the tenant.
	MissingGroupIDs []string
	// Mentioned are groups the configuration refers to without them being
	// counted. Shown by name so the reader can see which groups are involved.
	Mentioned []Group
	// Exact reports whether Count came from set arithmetic over real membership
	// rather than from summed counts. Purely informational — the bound already
	// carries whether the figure can be trusted as-is.
	Exact bool

	Unresolvable []Unresolvable
	PendingPaths []string
}

// Percent returns Count as a whole-number percentage of Total, and whether a
// percentage is meaningful.
func (r Resolution) Percent() (int, bool) {
	if r.Total <= 0 || !r.Determinable {
		return 0, false
	}
	return int((r.Count * 100) / r.Total), true
}

// Resolve counts a Scope against the cached tenant group list and device
// totals.
//
// Two strategies, preferred in order:
//
//   - Exact. Read the membership of every group the scope names, union the
//     targets, subtract the exclusions, and count what is left. Overlapping
//     groups are handled correctly rather than double-counted, and exclusions
//     genuinely reduce the figure. Costs one read per group named by a changing
//     scope, cached for the plan.
//   - Approximate. Sum the groups' membership counts, clamp to the size of the
//     estate, and report exclusions as narrowing rather than subtracting them.
//     Used when membership cannot be read, so a privilege gap or a transient
//     failure degrades the figure instead of losing it.
//
// Direct device references stay outside the exact arithmetic in both cases:
// `computer_ids` and `mobile_device_ids` carry Jamf Pro numeric identifiers,
// while group membership is expressed in device management identifiers, so the
// two cannot be combined without a further lookup. A scope naming both groups
// and individual devices therefore keeps its upper-bound framing.
func Resolve(ctx context.Context, c *Cache, s Scope) (Resolution, error) {
	res := Resolution{
		DeviceType:   s.DeviceType,
		Determinable: len(s.PendingPaths) == 0,
		Unresolvable: s.Unresolvable,
		PendingPaths: s.PendingPaths,
	}
	if !c.Enabled() {
		return res, nil
	}

	totals, err := c.DeviceTotals(ctx)
	if err != nil {
		return Resolution{}, err
	}
	res.Total = totals.For(s.DeviceType)

	for _, u := range s.Unresolvable {
		res.Bound = res.Bound.with(u.Effect)
	}

	// Mentioned groups are resolved for their names only; they never move Count.
	for _, id := range s.MentionedPlatformIDs {
		if g, found, err := c.GroupByPlatformID(ctx, id); err != nil {
			return Resolution{}, err
		} else if found {
			res.Mentioned = append(res.Mentioned, g)
		}
	}
	sort.Slice(res.Mentioned, func(i, j int) bool { return res.Mentioned[i].Name < res.Mentioned[j].Name })

	if !res.Determinable {
		return res, nil
	}

	// Resolve the group references on both sides to rows, so names and counts are
	// available whichever strategy ends up being used.
	targets, err := res.resolveGroups(ctx, c, s, s.JamfProGroupIDs, s.PlatformGroupIDs, false)
	if err != nil {
		return Resolution{}, err
	}
	excluded, err := res.resolveGroups(ctx, c, s, s.ExcludedJamfProGroupIDs, s.ExcludedPlatformGroupIDs, true)
	if err != nil {
		return Resolution{}, err
	}
	res.Groups = targets
	res.ExcludedGroups = excluded
	res.tenantWide = s.All

	sortGroups(res.Groups)
	sortGroups(res.ExcludedGroups)
	sort.Strings(res.MissingGroupIDs)

	if exact, ok, err := res.countExactly(ctx, c, s, targets, excluded); err != nil {
		return Resolution{}, err
	} else if ok {
		res.Count = exact
		res.Exact = true
		res.finishDirectDevices(s)
		return res, nil
	}

	res.countApproximately(s, targets, excluded)
	res.finishDirectDevices(s)
	return res, nil
}

// sortGroups orders groups by size then name, so a breakdown reads consistently.
func sortGroups(gs []Group) {
	sort.Slice(gs, func(i, j int) bool {
		if gs[i].MembershipCount != gs[j].MembershipCount {
			return gs[i].MembershipCount > gs[j].MembershipCount
		}
		return gs[i].Name < gs[j].Name
	})
}

// resolveGroups turns group references into rows, recording references that do
// not resolve. Unresolvable target references make the figure short; unresolvable
// exclusions make it long, so the direction depends on which side they sit on.
func (r *Resolution) resolveGroups(ctx context.Context, c *Cache, s Scope, proIDs, platformIDs []string, isExclusion bool) ([]Group, error) {
	if len(proIDs) > 0 && s.DeviceType == DeviceTypeAny {
		// Numeric group ids repeat across the two estates, so without knowing which
		// estate is meant they cannot be resolved, in either direction.
		r.Unresolvable = append(r.Unresolvable, Unresolvable{
			Path:   "group ids",
			Reason: "numeric group ids are only unique within the computer or mobile device estate, so they cannot be counted for a scope that spans both",
			Effect: Ambiguous,
			Values: len(proIDs),
		})
		r.Bound = r.Bound.with(Ambiguous)
		proIDs = nil
	}

	out := make([]Group, 0, len(proIDs)+len(platformIDs))
	seen := make(map[string]struct{}, len(proIDs)+len(platformIDs))
	add := func(id string, byPlatform bool) error {
		if id == "" {
			return nil
		}
		var (
			g     Group
			found bool
			err   error
		)
		if byPlatform {
			g, found, err = c.GroupByPlatformID(ctx, id)
		} else {
			g, found, err = c.GroupByJamfProID(ctx, s.DeviceType, id)
		}
		if err != nil {
			return err
		}
		if !found || !s.DeviceType.accepts(g.DeviceType) {
			r.MissingGroupIDs = append(r.MissingGroupIDs, id)
			if isExclusion {
				r.Bound = r.Bound.with(Narrows)
			} else {
				r.Bound = r.Bound.with(Broadens)
			}
			return nil
		}
		if _, dup := seen[g.PlatformID]; dup {
			return nil
		}
		seen[g.PlatformID] = struct{}{}
		out = append(out, g)
		return nil
	}
	for _, id := range proIDs {
		if err := add(id, false); err != nil {
			return nil, err
		}
	}
	for _, id := range platformIDs {
		if err := add(id, true); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// countExactly unions the target groups' membership, subtracts the exclusions'
// and returns the size of what remains. Reports ok=false when any membership read
// fails, so the caller can fall back to counts.
//
// Membership is expressed in device management identifiers, which are unique
// across both estates, so the set arithmetic needs no per-estate handling.
func (r *Resolution) countExactly(ctx context.Context, c *Cache, s Scope, targets, excluded []Group) (int64, bool, error) {
	if !s.All && len(targets) == 0 {
		// Nothing to union. Fall back so a device-only or caveat-only scope keeps
		// its existing treatment.
		return 0, false, nil
	}

	gather := func(groups []Group) (map[string]struct{}, bool, error) {
		out := make(map[string]struct{})
		for _, g := range groups {
			ids, err := c.Members(ctx, g.PlatformID)
			if err != nil {
				// A membership read that fails is not a plan error — it just means the
				// approximate strategy is used instead.
				return nil, false, nil
			}
			// A membership set that disagrees with the group's own count is not
			// trustworthy enough for exact arithmetic; the two figures come from
			// different services and may be momentarily out of step.
			if int64(len(ids)) != g.MembershipCount {
				return nil, false, nil
			}
			for _, id := range ids {
				out[id] = struct{}{}
			}
		}
		return out, true, nil
	}

	excludedMembers, ok, err := gather(excluded)
	if err != nil || !ok {
		return 0, false, err
	}

	if s.All {
		// Tenant-wide target: everything in the estate, less the exclusions.
		count := max(r.Total-int64(len(excludedMembers)), 0)
		return count, true, nil
	}

	targetMembers, ok, err := gather(targets)
	if err != nil || !ok {
		return 0, false, err
	}
	var count int64
	for id := range targetMembers {
		if _, isExcluded := excludedMembers[id]; !isExcluded {
			count++
		}
	}
	return count, true, nil
}

// countApproximately sums membership counts, clamps to the estate, and reports
// exclusions as narrowing. Used when membership could not be read.
func (r *Resolution) countApproximately(s Scope, targets, excluded []Group) {
	if s.All {
		r.Count = r.Total
	} else {
		for _, g := range targets {
			r.Count += g.MembershipCount
		}
		// More than one group means members can be counted twice.
		if len(targets) > 1 {
			r.Bound = r.Bound.with(Narrows)
		}
	}

	if len(excluded) > 0 {
		names := make([]string, 0, len(excluded))
		for _, g := range excluded {
			names = append(names, g.Name)
		}
		sort.Strings(names)
		r.Unresolvable = append(r.Unresolvable, Unresolvable{
			Path:   "excluded groups",
			Reason: "membership of " + strings.Join(names, ", ") + " could not be read, so the exclusion is not subtracted here",
			Effect: Narrows,
			Values: len(excluded),
		})
		r.Bound = r.Bound.with(Narrows)
	}

	// Overlapping groups can sum past the size of the estate, which would read as
	// "up to 7 of 4 computers (175%)". The estate is a hard ceiling on the true
	// figure, so clamping to it is the more accurate statement, not a cosmetic
	// one — and the breakdown still shows the per-group counts it came from.
	if r.Total > 0 && r.Count > r.Total {
		r.Count = r.Total
		r.Bound = r.Bound.with(Narrows)
	}
}

// finishDirectDevices folds individually named devices into the figure. They are
// added rather than unioned, because they are identified differently from group
// membership, so a device that is also a group member is counted twice — hence
// the upper bound whenever both are present.
func (r *Resolution) finishDirectDevices(s Scope) {
	if n := len(s.DeviceIDs); n > 0 {
		r.DirectDevices = n
		r.Count += int64(n)
		if len(r.Groups) > 0 || s.All {
			r.Bound = r.Bound.with(Narrows)
			r.Exact = false
		}
	}
	if n := len(s.ExcludedDeviceIDs); n > 0 {
		r.Unresolvable = append(r.Unresolvable, Unresolvable{
			Path:   "excluded devices",
			Reason: "individually excluded devices are identified differently from group membership, so they are not subtracted here",
			Effect: Narrows,
			Values: n,
		})
		r.Bound = r.Bound.with(Narrows)
		r.Exact = false
	}
	if r.Total > 0 && r.Count > r.Total {
		r.Count = r.Total
		r.Bound = r.Bound.with(Narrows)
	}
}

// Delta returns the countable additions and removals between two scopes: the
// group and device references present in planned but not prior, and vice versa.
//
// This is the figure Jamf Pro's own impact alert leads with — the number of
// devices being added or removed — and it is the one number Terraform is better
// placed to produce than the admin UI, because a plan holds both the prior and
// the intended scope at once.
//
// Unresolvable and pending inputs are carried onto whichever side introduces
// them, so an addition that brings a new limitation with it is still reported as
// bounded.
//
// Exclusions invert: a newly added exclusion removes devices, and a lifted
// exclusion adds them, so each is counted on the opposite side from where it is
// written.
func Delta(prior, planned Scope) (added, removed Scope) {
	added = Scope{DeviceType: planned.DeviceType}
	removed = Scope{DeviceType: prior.DeviceType}

	added.DeviceIDs = missingFrom(planned.DeviceIDs, prior.DeviceIDs)
	added.JamfProGroupIDs = missingFrom(planned.JamfProGroupIDs, prior.JamfProGroupIDs)
	added.PlatformGroupIDs = missingFrom(planned.PlatformGroupIDs, prior.PlatformGroupIDs)
	added.PendingPaths = planned.PendingPaths
	added.Unresolvable = unresolvableDiff(planned.Unresolvable, prior.Unresolvable)
	added.All = planned.All && !prior.All
	// An exclusion that is being LIFTED adds devices, so it belongs on this side —
	// as a target, since those devices are entering scope.
	added.JamfProGroupIDs = append(added.JamfProGroupIDs,
		missingFrom(prior.ExcludedJamfProGroupIDs, planned.ExcludedJamfProGroupIDs)...)
	added.PlatformGroupIDs = append(added.PlatformGroupIDs,
		missingFrom(prior.ExcludedPlatformGroupIDs, planned.ExcludedPlatformGroupIDs)...)

	removed.DeviceIDs = missingFrom(prior.DeviceIDs, planned.DeviceIDs)
	removed.JamfProGroupIDs = missingFrom(prior.JamfProGroupIDs, planned.JamfProGroupIDs)
	removed.PlatformGroupIDs = missingFrom(prior.PlatformGroupIDs, planned.PlatformGroupIDs)
	removed.Unresolvable = unresolvableDiff(prior.Unresolvable, planned.Unresolvable)
	removed.All = prior.All && !planned.All
	// A newly ADDED exclusion removes devices, so it belongs on the removal side.
	removed.JamfProGroupIDs = append(removed.JamfProGroupIDs,
		missingFrom(planned.ExcludedJamfProGroupIDs, prior.ExcludedJamfProGroupIDs)...)
	removed.PlatformGroupIDs = append(removed.PlatformGroupIDs,
		missingFrom(planned.ExcludedPlatformGroupIDs, prior.ExcludedPlatformGroupIDs)...)

	return added, removed
}

// missingFrom returns the members of a that do not appear in b, sorted.
func missingFrom(a, b []string) []string {
	if len(a) == 0 {
		return nil
	}
	in := make(map[string]struct{}, len(b))
	for _, v := range b {
		in[v] = struct{}{}
	}
	var out []string
	for _, v := range a {
		if _, ok := in[v]; !ok {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// unresolvableDiff returns the entries of a whose path/value-count differs from
// b, i.e. the unresolvable inputs this side of the change introduces.
func unresolvableDiff(a, b []Unresolvable) []Unresolvable {
	if len(a) == 0 {
		return nil
	}
	prev := make(map[string]int, len(b))
	for _, u := range b {
		prev[u.Path] = u.Values
	}
	var out []Unresolvable
	for _, u := range a {
		if n, ok := prev[u.Path]; !ok || n != u.Values {
			out = append(out, u)
		}
	}
	return out
}
