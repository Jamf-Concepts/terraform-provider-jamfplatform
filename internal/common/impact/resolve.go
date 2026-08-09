// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"sort"
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

	// Groups are the groups that were found and counted.
	Groups []Group
	// DirectDevices is the number of individually scoped devices counted.
	DirectDevices int
	// MissingGroupIDs are referenced groups that are not present in the tenant.
	MissingGroupIDs []string
	// Mentioned are groups the configuration refers to without them being
	// counted. Shown by name so the reader can see which groups are involved.
	Mentioned []Group

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
// Overlap is not deduplicated. Two groups can share members, so summing
// membership counts can exceed the number of distinct devices — which is why a
// scope naming more than one countable source yields BoundAtMost. Deduplicating
// would require reading the full membership of every referenced group, and the
// result would still be a snapshot, so the sum plus an honest upper-bound
// framing is preferred over the extra reads.
//
// Exclusions are not subtracted for the same reason in reverse: subtracting a
// count that partly overlaps the targets would understate the result. Adapters
// therefore report exclusions as narrowing Unresolvable entries rather than as
// negative numbers.
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

	if s.All {
		// A tenant-wide target needs no group arithmetic: the denominator is the
		// count. Narrowing inputs still apply, and are already folded in above.
		res.Count = res.Total
		return res, nil
	}

	sources := 0
	seen := make(map[string]struct{})
	lookup := func(id string, byPlatform bool) error {
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
			res.MissingGroupIDs = append(res.MissingGroupIDs, id)
			// An uncounted group can only mean the figure is short.
			res.Bound = res.Bound.with(Broadens)
			return nil
		}
		if _, dup := seen[g.PlatformID]; dup {
			return nil
		}
		seen[g.PlatformID] = struct{}{}
		res.Groups = append(res.Groups, g)
		res.Count += g.MembershipCount
		sources++
		return nil
	}

	if len(s.JamfProGroupIDs) > 0 && s.DeviceType == DeviceTypeAny {
		// Numeric group ids repeat across the two estates, so without knowing
		// which estate is meant they cannot be counted, in either direction.
		res.Unresolvable = append(res.Unresolvable, Unresolvable{
			Path:   "group ids",
			Reason: "numeric group ids are only unique within the computer or mobile device estate, so they cannot be counted for a scope that spans both",
			Effect: Ambiguous,
			Values: len(s.JamfProGroupIDs),
		})
		res.Bound = res.Bound.with(Ambiguous)
	} else {
		for _, id := range s.JamfProGroupIDs {
			if err := lookup(id, false); err != nil {
				return Resolution{}, err
			}
		}
	}
	for _, id := range s.PlatformGroupIDs {
		if err := lookup(id, true); err != nil {
			return Resolution{}, err
		}
	}

	if n := len(s.DeviceIDs); n > 0 {
		res.DirectDevices = n
		res.Count += int64(n)
		sources++
	}

	// More than one countable source means members can be counted twice.
	if sources > 1 {
		res.Bound = res.Bound.with(Narrows)
	}

	// Overlapping groups can sum past the size of the estate, which would read as
	// "up to 7 of 4 computers (175%)". The estate is a hard ceiling on the true
	// figure, so clamping to it is the more accurate statement, not a cosmetic
	// one — and the breakdown still shows the per-group counts it came from.
	if res.Total > 0 && res.Count > res.Total {
		res.Count = res.Total
		res.Bound = res.Bound.with(Narrows)
	}

	sort.Slice(res.Groups, func(i, j int) bool {
		if res.Groups[i].MembershipCount != res.Groups[j].MembershipCount {
			return res.Groups[i].MembershipCount > res.Groups[j].MembershipCount
		}
		return res.Groups[i].Name < res.Groups[j].Name
	})
	sort.Strings(res.MissingGroupIDs)

	return res, nil
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
func Delta(prior, planned Scope) (added, removed Scope) {
	added = Scope{DeviceType: planned.DeviceType}
	removed = Scope{DeviceType: prior.DeviceType}

	added.DeviceIDs = missingFrom(planned.DeviceIDs, prior.DeviceIDs)
	added.JamfProGroupIDs = missingFrom(planned.JamfProGroupIDs, prior.JamfProGroupIDs)
	added.PlatformGroupIDs = missingFrom(planned.PlatformGroupIDs, prior.PlatformGroupIDs)
	added.PendingPaths = planned.PendingPaths
	added.Unresolvable = unresolvableDiff(planned.Unresolvable, prior.Unresolvable)
	added.All = planned.All && !prior.All

	removed.DeviceIDs = missingFrom(prior.DeviceIDs, planned.DeviceIDs)
	removed.JamfProGroupIDs = missingFrom(prior.JamfProGroupIDs, planned.JamfProGroupIDs)
	removed.PlatformGroupIDs = missingFrom(prior.PlatformGroupIDs, planned.PlatformGroupIDs)
	removed.Unresolvable = unresolvableDiff(prior.Unresolvable, planned.Unresolvable)
	removed.All = prior.All && !planned.All

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
