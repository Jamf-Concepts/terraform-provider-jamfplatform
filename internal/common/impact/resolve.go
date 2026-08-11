// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"fmt"
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

// ProGroupRef identifies a Jamf Pro group by estate and numeric id — the only
// combination that is unique, since the two estates are numbered independently.
type ProGroupRef struct {
	DeviceType DeviceType
	ID         string
}

// key renders a reference for set comparison.
func (r ProGroupRef) key() string { return string(r.DeviceType) + ":" + r.ID }

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

	// All is the tenant-wide flag (all_computers / all_mobile_devices) for a
	// scope fixed to one estate.
	All bool
	// AllEstates lists the estates targeted tenant-wide, for a scope that can
	// span both estates at once. An ebook can set all_computers while scoping
	// mobile devices to one group, and folding both flags into All would claim
	// the whole combined estate; carrying the estate keeps the two sides apart.
	AllEstates []DeviceType
	// DeviceIDs are individually scoped devices (computer_ids / mobile_device_ids).
	DeviceIDs []string
	// BuildingIDs and DepartmentIDs are the buildings and departments the scope
	// targets. Carried as data so the devices assigned to them can be resolved and
	// unioned exactly; when an estate cannot resolve them they fall back to being
	// reported as broadening.
	BuildingIDs   []string
	DepartmentIDs []string
	// ProGroups are groups referenced by numeric Jamf Pro id. The estate is part of
	// the reference because numeric group ids repeat across the computer and mobile
	// device estates, so an id alone does not identify a group. Carrying it per
	// reference also lets a resource target both estates at once, which an ebook
	// does.
	ProGroups []ProGroupRef
	// PlatformGroupIDs are groups referenced by Platform UUID.
	PlatformGroupIDs []string
	// MentionedPlatformIDs are groups the configuration refers to but which are
	// deliberately not counted, because whether they widen or narrow the audience
	// is not determinable. Their names are surfaced so a reviewer can see which
	// groups are involved without being given a total that implies more certainty
	// than there is.
	MentionedPlatformIDs []string

	// ExcludedProGroups and ExcludedPlatformGroupIDs are groups removed from the
	// audience. They are carried as data rather than as narrowing caveats so their
	// membership can be subtracted exactly; when membership cannot be read they
	// fall back to being reported as narrowing.
	ExcludedProGroups        []ProGroupRef
	ExcludedPlatformGroupIDs []string
	// ExcludedDeviceIDs, ExcludedBuildingIDs and ExcludedDepartmentIDs are the
	// device-naming exclusion categories, carried as data for the same reason as
	// their target counterparts.
	ExcludedDeviceIDs     []string
	ExcludedBuildingIDs   []string
	ExcludedDepartmentIDs []string

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
		len(s.AllEstates) == 0 &&
		len(s.DeviceIDs) == 0 &&
		len(s.ProGroups) == 0 &&
		len(s.PlatformGroupIDs) == 0 &&
		len(s.MentionedPlatformIDs) == 0 &&
		len(s.ExcludedProGroups) == 0 &&
		len(s.ExcludedPlatformGroupIDs) == 0 &&
		len(s.BuildingIDs) == 0 &&
		len(s.DepartmentIDs) == 0 &&
		len(s.ExcludedDeviceIDs) == 0 &&
		len(s.ExcludedBuildingIDs) == 0 &&
		len(s.ExcludedDepartmentIDs) == 0 &&
		len(s.Unresolvable) == 0 &&
		len(s.PendingPaths) == 0
}

// namesDevicesDirectly reports whether the scope names devices other than through
// group membership — individually, or by the building or department they are in.
func (s Scope) namesDevicesDirectly() bool {
	return len(s.DeviceIDs) > 0 || len(s.BuildingIDs) > 0 || len(s.DepartmentIDs) > 0
}

// allEstateSet returns the estates the scope targets tenant-wide. The
// single-estate All flag maps to the scope's own estate; All on an
// estate-spanning scope, and DeviceTypeAny entries, mean both estates.
func (s Scope) allEstateSet() map[DeviceType]struct{} {
	out := make(map[DeviceType]struct{}, 2)
	add := func(dt DeviceType) {
		if dt == DeviceTypeAny {
			out[DeviceTypeComputer] = struct{}{}
			out[DeviceTypeMobile] = struct{}{}
			return
		}
		out[dt] = struct{}{}
	}
	if s.All {
		add(s.DeviceType)
	}
	for _, dt := range s.AllEstates {
		add(dt)
	}
	return out
}

// tenantWideOrder renders a tenant-wide estate set in the admin UI's order,
// computers first.
func tenantWideOrder(set map[DeviceType]struct{}) []DeviceType {
	out := make([]DeviceType, 0, len(set))
	for _, dt := range []DeviceType{DeviceTypeComputer, DeviceTypeMobile} {
		if _, ok := set[dt]; ok {
			out = append(out, dt)
		}
	}
	return out
}

// withoutTenantWideEstates drops the groups whose estate is targeted
// tenant-wide, since their membership is subsumed by that estate's total.
func withoutTenantWideEstates(groups []Group, all map[DeviceType]struct{}) []Group {
	if len(all) == 0 {
		return groups
	}
	out := make([]Group, 0, len(groups))
	for _, g := range groups {
		if _, ok := all[g.DeviceType]; ok {
			continue
		}
		out = append(out, g)
	}
	return out
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
	// PerEstate splits Count by estate, and is populated only when the scope spans
	// both. A merged "3 of 5 devices" hides whether three Macs or three iPads are
	// affected, which is the first thing an administrator wants to know.
	//
	// The split is sound rather than approximate: a device belongs to exactly one
	// estate, so computer-group members and mobile-device-group members are
	// disjoint by construction and the union partitions cleanly.
	PerEstate map[DeviceType]int64

	// tenantWide lists the estates the scope targeted in full, so the breakdown
	// can say so rather than listing nothing.
	tenantWide []DeviceType
	// members is the resolved member set — the union of the targets less the
	// exclusions — kept only when it is complete, so an update's delta can be
	// taken from real membership rather than from which references changed. A
	// tenant-wide estate has no enumerated membership, so it leaves this nil.
	members map[string]DeviceType
	// totals carries both estates' sizes, so a split figure can give each side its
	// own denominator.
	totals Totals
	// resolvedNamed counts the device-naming categories that resolved exactly, so
	// the breakdown can mention them and the estate split can include their estate.
	resolvedNamed int
	// namedDescribed renders those categories for the breakdown.
	namedDescribed []string
	// exceedsManaged records that the exact figure is larger than the tenant's
	// managed device count, which means the scope names devices that are not
	// managed. Unlike the approximate path — where an excess means overlapping
	// counts were summed and the estate is a valid ceiling — this excess is real
	// information and must not be clamped away.
	exceedsManaged bool
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
	res.totals = totals

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
	targets, err := res.resolveGroups(ctx, c, s, s.ProGroups, s.PlatformGroupIDs, false)
	if err != nil {
		return Resolution{}, err
	}
	excluded, err := res.resolveGroups(ctx, c, s, s.ExcludedProGroups, s.ExcludedPlatformGroupIDs, true)
	if err != nil {
		return Resolution{}, err
	}
	res.Groups = targets
	res.ExcludedGroups = excluded
	res.tenantWide = tenantWideOrder(s.allEstateSet())

	sortGroups(res.Groups)
	sortGroups(res.ExcludedGroups)
	sort.Strings(res.MissingGroupIDs)

	if exact, ok, err := res.countExactly(ctx, c, s, targets, excluded); err != nil {
		return Resolution{}, err
	} else if ok {
		// Everything countable is already in the union, so nothing is added on top.
		res.Count = exact
		res.Exact = true
		res.exceedsManaged = res.Total > 0 && res.Count > res.Total
		return res, nil
	}

	res.countApproximately(s, targets, excluded)
	res.namedDeviceCaveats(s)
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
func (r *Resolution) resolveGroups(ctx context.Context, c *Cache, s Scope, proRefs []ProGroupRef, platformIDs []string, isExclusion bool) ([]Group, error) {
	out := make([]Group, 0, len(proRefs)+len(platformIDs))
	seen := make(map[string]struct{}, len(proRefs)+len(platformIDs))
	add := func(dt DeviceType, id string, byPlatform bool) error {
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
			g, found, err = c.GroupByJamfProID(ctx, dt, id)
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
	for _, ref := range proRefs {
		if err := add(ref.DeviceType, ref.ID, false); err != nil {
			return nil, err
		}
	}
	for _, id := range platformIDs {
		if err := add(DeviceTypeAny, id, true); err != nil {
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
// across both estates, so the set arithmetic itself needs no per-estate handling.
// The estate each member came from is still tracked, so a scope spanning both can
// be reported as a split rather than one merged figure.
//
// A tenant-wide estate contributes its total, less its excluded members, rather
// than enumerated membership — and only that estate. A dual-estate scope with
// all_computers set still counts its mobile side from the mobile groups it
// names, so one estate's flag never claims the combined estate.
func (r *Resolution) countExactly(ctx context.Context, c *Cache, s Scope, targets, excluded []Group) (int64, bool, error) {
	allEstates := s.allEstateSet()
	if len(allEstates) == 0 && len(targets) == 0 && !s.namesDevicesDirectly() {
		// Nothing to union at all. Fall back so a caveat-only scope keeps its
		// existing treatment.
		return 0, false, nil
	}

	// gather unions the membership of a set of groups, remembering which estate
	// each member arrived from.
	gather := func(groups []Group) (map[string]DeviceType, bool, error) {
		out := make(map[string]DeviceType)
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
				out[id] = g.DeviceType
			}
		}
		return out, true, nil
	}

	excludedMembers, ok, err := gather(excluded)
	if err != nil || !ok {
		return 0, false, err
	}
	// Device-naming exclusions join the same set, so an excluded computer or an
	// excluded building genuinely reduces the figure rather than being reported as
	// an unquantified narrowing.
	exclNamed := addNamedDevices(ctx, c, s.DeviceType, excludedMembers, Narrows,
		"exclusions device ids", s.ExcludedDeviceIDs, s.ExcludedBuildingIDs, s.ExcludedDepartmentIDs)

	if s.DeviceType != DeviceTypeAny && len(allEstates) > 0 {
		// Tenant-wide target on a single-estate scope: everything in the estate,
		// less the exclusions.
		count := max(r.Total-int64(len(excludedMembers)), 0)
		r.commitNamed(exclNamed)
		return count, true, nil
	}

	// Groups in a tenant-wide estate are subsumed by that estate's total, so
	// their membership is not read.
	remaining := withoutTenantWideEstates(targets, allEstates)
	targetMembers, ok, err := gather(remaining)
	if err != nil || !ok {
		return 0, false, err
	}
	targetNamed := addNamedDevices(ctx, c, s.DeviceType, targetMembers, Broadens,
		"targets device ids", s.DeviceIDs, s.BuildingIDs, s.DepartmentIDs)
	if len(allEstates) == 0 && len(remaining) == 0 && targetNamed.resolved == 0 {
		// Nothing was resolvable, so there is no exact figure to report — fall back
		// rather than presenting a confident zero.
		return 0, false, nil
	}

	var count int64
	perEstate := estateKeys(remaining)
	if targetNamed.resolved > 0 {
		// The estate is named by the device categories too, so it belongs in the
		// split even when no group named it.
		perEstate[s.DeviceType] += 0
	}
	members := make(map[string]DeviceType, len(targetMembers))
	for id, dt := range targetMembers {
		if _, isExcluded := excludedMembers[id]; isExcluded {
			continue
		}
		members[id] = dt
		count++
		perEstate[dt]++
	}
	// A tenant-wide estate contributes its whole total, less the excluded
	// members that sit in it.
	for _, dt := range tenantWideOrder(allEstates) {
		var excludedHere int64
		for _, mdt := range excludedMembers {
			if mdt == dt {
				excludedHere++
			}
		}
		n := max(r.totals.For(dt)-excludedHere, 0)
		count += n
		perEstate[dt] += n
	}
	if len(allEstates) == 0 {
		// Member identities are complete only when no estate was counted by its
		// total, so only then can a delta be taken from them.
		r.members = members
	}
	r.setPerEstate(s, perEstate)
	r.commitNamed(targetNamed)
	r.commitNamed(exclNamed)
	r.resolvedNamed = targetNamed.resolved
	return count, true, nil
}

// commitNamed records an outcome's caveats onto the resolution, once the exact
// strategy has committed to succeeding.
func (r *Resolution) commitNamed(o namedOutcome) {
	for _, u := range o.unresolved {
		r.Unresolvable = append(r.Unresolvable, u)
		r.Bound = r.Bound.with(u.Effect)
	}
	r.namedDescribed = append(r.namedDescribed, o.described...)
}

// reasonPlaceNotFound covers building or department ids with no match in the
// tenant's building and department lists. Unexported because only the resolver
// emits it; the Reason* constants the scope adapters share live in scopebuilder.go.
const reasonPlaceNotFound = "not found among this tenant's buildings and departments, so their devices are not counted here"

// namedCategory is one device-naming scope category awaiting resolution.
type namedCategory struct {
	kind deviceFilterKind
	ids  []string
	path string
}

// describe renders this category for the breakdown: how many were named, and how
// many devices they turned out to hold.
func (c namedCategory) describe(dt DeviceType, named, devices int) string {
	switch c.kind {
	case filterKindBuilding:
		return fmt.Sprintf("%s (%d)", plural(named, "building", "buildings"), devices)
	case filterKindDepartment:
		return fmt.Sprintf("%s (%d)", plural(named, "department", "departments"), devices)
	default:
		return fmt.Sprintf("%s named individually", plural(named, singularNoun(dt), dt.Noun()))
	}
}

// namedOutcome is what resolving the device-naming categories produced. It is
// returned rather than written straight onto the resolution, because the exact
// strategy may still abandon the attempt — and a caveat recorded by an abandoned
// attempt would then be reported alongside the approximate figure that replaced it.
type namedOutcome struct {
	// unresolved holds the categories this estate cannot resolve.
	unresolved []Unresolvable
	// resolved counts the categories that did resolve.
	resolved int
	// described names what resolved, for the breakdown — otherwise a figure counting
	// named computers or a building would have nothing explaining where it came from.
	described []string
}

// addNamedDevices folds the device-naming scope categories — individual devices,
// buildings, departments — into a membership set, having resolved them to the same
// device management identifiers group membership uses.
//
// A failed inventory read degrades that one category to a caveat, the same
// advisory contract as a failed membership read. Propagating it would abort the
// whole resolution and consume the plan-wide "impact unavailable" notice,
// silencing every other resource's alert over one unreadable category.
func addNamedDevices(ctx context.Context, c *Cache, dt DeviceType, into map[string]DeviceType, side Effect, deviceAttr string, deviceIDs, buildingIDs, departmentIDs []string) namedOutcome {
	var out namedOutcome
	categories := []namedCategory{
		{filterKindDevice, deviceIDs, deviceAttr},
		{filterKindBuilding, buildingIDs, "building_ids"},
		{filterKindDepartment, departmentIDs, "department_ids"},
	}
	for _, cat := range categories {
		if len(cat.ids) == 0 {
			continue
		}
		ids, dropped, supported, err := c.deviceIDsFor(ctx, dt, cat.kind, cat.ids)
		if err != nil || !supported {
			out.unresolved = append(out.unresolved, Unresolvable{
				Path: cat.path, Reason: ReasonNotCounted, Effect: side, Values: len(cat.ids),
			})
			continue
		}
		for _, id := range ids {
			into[id] = dt
		}
		// A partially translated category is described by what actually resolved,
		// and the dropped remainder is caveated in the direction the category
		// already pushes — its devices are missing from the figure either way.
		resolved := len(cat.ids) - dropped
		out.resolved += resolved
		out.described = append(out.described, cat.describe(dt, resolved, len(ids)))
		if dropped > 0 {
			out.unresolved = append(out.unresolved, Unresolvable{
				Path: cat.path, Reason: reasonPlaceNotFound, Effect: side, Values: dropped,
			})
		}
	}
	return out
}

// estateKeys seeds a per-estate tally with every estate the scope names, so an
// estate that contributes no devices still reports as zero.
//
// An empty group is exactly the case worth showing: naming a mobile device group
// that currently holds nothing is a real fact about the change, and reporting
// only the computer side would silently hide that the mobile side was targeted at
// all.
func estateKeys(groups []Group) map[DeviceType]int64 {
	out := make(map[DeviceType]int64, 2)
	for _, g := range groups {
		out[g.DeviceType] += 0
	}
	return out
}

// setPerEstate records the per-estate tally for a scope that can span both
// estates. A scope fixed to one estate already says which in its noun, so it is
// left merged.
func (r *Resolution) setPerEstate(s Scope, perEstate map[DeviceType]int64) {
	if s.DeviceType != DeviceTypeAny || len(perEstate) == 0 {
		return
	}
	r.PerEstate = perEstate
}

// groupsShareAnEstate reports whether any two of the groups sit in the same
// estate, which is the only way their membership counts can overlap.
func groupsShareAnEstate(groups []Group) bool {
	seen := make(map[DeviceType]struct{}, 2)
	for _, g := range groups {
		if _, dup := seen[g.DeviceType]; dup {
			return true
		}
		seen[g.DeviceType] = struct{}{}
	}
	return false
}

// countApproximately sums membership counts, clamps to the estate, and reports
// exclusions as narrowing. Used when membership could not be read.
func (r *Resolution) countApproximately(s Scope, targets, excluded []Group) {
	allEstates := s.allEstateSet()
	if s.DeviceType != DeviceTypeAny && len(allEstates) > 0 {
		r.Count = r.Total
	} else {
		// Groups in a tenant-wide estate are subsumed by that estate's total, and
		// each tenant-wide estate contributes its own total — never the combined
		// estate on the strength of one flag.
		remaining := withoutTenantWideEstates(targets, allEstates)
		perEstate := estateKeys(remaining)
		for _, dt := range tenantWideOrder(allEstates) {
			n := r.totals.For(dt)
			r.Count += n
			perEstate[dt] += n
		}
		for _, g := range remaining {
			r.Count += g.MembershipCount
			perEstate[g.DeviceType] += g.MembershipCount
		}
		r.setPerEstate(s, perEstate)
		// Summed counts can double-count only where two groups could share a member,
		// which means two groups in the same estate: a device belongs to exactly one
		// estate, so a computer group and a mobile device group never overlap.
		if groupsShareAnEstate(remaining) {
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

// namedDeviceCaveats reports the device-naming categories as unresolved, used on
// the approximate path where they could not be turned into membership.
func (r *Resolution) namedDeviceCaveats(s Scope) {
	report := func(path string, n int, e Effect, reason string) {
		if n == 0 {
			return
		}
		r.Unresolvable = append(r.Unresolvable, Unresolvable{
			Path: path, Reason: reason, Effect: e, Values: n,
		})
		r.Bound = r.Bound.with(e)
	}
	report("targets.building_ids", len(s.BuildingIDs), Broadens, ReasonNotCounted)
	report("targets.department_ids", len(s.DepartmentIDs), Broadens, ReasonNotCounted)
	report("exclusions.building_ids", len(s.ExcludedBuildingIDs), Narrows, ReasonNotCounted)
	report("exclusions.department_ids", len(s.ExcludedDepartmentIDs), Narrows, ReasonNotCounted)
}

// finishDirectDevices folds individually named devices into the figure on the
// approximate path. They are added rather than unioned there, because without a
// membership set a device that is also a group member is counted twice — hence
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
	added.ProGroups = refsMissingFrom(planned.ProGroups, prior.ProGroups)
	added.PlatformGroupIDs = missingFrom(planned.PlatformGroupIDs, prior.PlatformGroupIDs)
	added.BuildingIDs = missingFrom(planned.BuildingIDs, prior.BuildingIDs)
	added.DepartmentIDs = missingFrom(planned.DepartmentIDs, prior.DepartmentIDs)
	added.PendingPaths = planned.PendingPaths
	added.Unresolvable = unresolvableDiff(planned.Unresolvable, prior.Unresolvable)
	added.All = planned.All && !prior.All
	added.AllEstates = estatesMissingFrom(planned.AllEstates, prior.AllEstates)
	// An exclusion that is being LIFTED adds devices, so it belongs on this side —
	// as a target, since those devices are entering scope.
	added.ProGroups = append(added.ProGroups,
		refsMissingFrom(prior.ExcludedProGroups, planned.ExcludedProGroups)...)
	added.PlatformGroupIDs = append(added.PlatformGroupIDs,
		missingFrom(prior.ExcludedPlatformGroupIDs, planned.ExcludedPlatformGroupIDs)...)
	added.BuildingIDs = append(added.BuildingIDs,
		missingFrom(prior.ExcludedBuildingIDs, planned.ExcludedBuildingIDs)...)
	added.DepartmentIDs = append(added.DepartmentIDs,
		missingFrom(prior.ExcludedDepartmentIDs, planned.ExcludedDepartmentIDs)...)

	removed.DeviceIDs = missingFrom(prior.DeviceIDs, planned.DeviceIDs)
	removed.ProGroups = refsMissingFrom(prior.ProGroups, planned.ProGroups)
	removed.PlatformGroupIDs = missingFrom(prior.PlatformGroupIDs, planned.PlatformGroupIDs)
	removed.BuildingIDs = missingFrom(prior.BuildingIDs, planned.BuildingIDs)
	removed.DepartmentIDs = missingFrom(prior.DepartmentIDs, planned.DepartmentIDs)
	removed.Unresolvable = unresolvableDiff(prior.Unresolvable, planned.Unresolvable)
	removed.All = prior.All && !planned.All
	removed.AllEstates = estatesMissingFrom(prior.AllEstates, planned.AllEstates)
	// A newly ADDED exclusion removes devices, so it belongs on the removal side.
	removed.ProGroups = append(removed.ProGroups,
		refsMissingFrom(planned.ExcludedProGroups, prior.ExcludedProGroups)...)
	removed.PlatformGroupIDs = append(removed.PlatformGroupIDs,
		missingFrom(planned.ExcludedPlatformGroupIDs, prior.ExcludedPlatformGroupIDs)...)
	removed.BuildingIDs = append(removed.BuildingIDs,
		missingFrom(planned.ExcludedBuildingIDs, prior.ExcludedBuildingIDs)...)
	removed.DepartmentIDs = append(removed.DepartmentIDs,
		missingFrom(planned.ExcludedDepartmentIDs, prior.ExcludedDepartmentIDs)...)

	return added, removed
}

// estatesMissingFrom returns the estates in a that do not appear in b.
func estatesMissingFrom(a, b []DeviceType) []DeviceType {
	if len(a) == 0 {
		return nil
	}
	in := make(map[DeviceType]struct{}, len(b))
	for _, v := range b {
		in[v] = struct{}{}
	}
	var out []DeviceType
	for _, v := range a {
		if _, ok := in[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}

// refsMissingFrom returns the references in a that do not appear in b, sorted.
func refsMissingFrom(a, b []ProGroupRef) []ProGroupRef {
	if len(a) == 0 {
		return nil
	}
	in := make(map[string]struct{}, len(b))
	for _, v := range b {
		in[v.key()] = struct{}{}
	}
	var out []ProGroupRef
	for _, v := range a {
		if _, ok := in[v.key()]; !ok {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key() < out[j].key() })
	return out
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
