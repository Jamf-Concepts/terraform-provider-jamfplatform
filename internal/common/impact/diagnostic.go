// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"fmt"
	"strings"
)

// snapshotNote closes every impact alert. Group membership is re-evaluated by
// Jamf Pro continuously, and a plan that also edits a group cannot see that
// edit from the object being scoped, so no figure here is a promise about what
// apply will do.
const snapshotNote = "Group membership is a snapshot taken during this plan and can change before or during apply."

// maxListedGroups caps how many group names appear in the breakdown before the
// remainder is summarised, so a scope naming dozens of groups stays readable.
const maxListedGroups = 5

// figure renders a count with the qualifier its bound earns, choosing the
// singular noun at exactly one so a delta of a single device never reads
// "1 computers".
//
// The qualifier is not a hedge for its own sake: Jamf Pro's scope model narrows
// as well as broadens, so the direction of the uncertainty is known even when
// the exact number is not.
func figure(n int64, b Bound, one, many string) string {
	noun := many
	if n == 1 {
		noun = one
	}
	switch b {
	case BoundAtMost:
		return fmt.Sprintf("up to %d %s", n, noun)
	case BoundAtLeast:
		// "or more" implies plurality whatever the count, so the plural stays.
		return fmt.Sprintf("%d or more %s", n, many)
	case BoundUnknown:
		return fmt.Sprintf("an estimated %d %s", n, noun)
	default:
		return fmt.Sprintf("%d %s", n, noun)
	}
}

// qualifier renders a bound as a leading phrase, for figures where a trailing
// "or more" cannot attach cleanly — a split across two estates, for instance.
func qualifier(b Bound) string {
	switch b {
	case BoundAtMost:
		return "up to "
	case BoundAtLeast:
		return "at least "
	case BoundUnknown:
		return "an estimated "
	default:
		return ""
	}
}

// splitEstates renders a per-estate breakdown, each side with its own
// denominator: "3 of 4 computers and 0 of 1 mobile devices".
//
// A merged figure would hide the distinction an administrator cares about most —
// three Macs and three iPads are not the same change — so a scope spanning both
// estates never reports one combined number.
func splitEstates(r Resolution) string {
	parts := make([]string, 0, 2)
	// Computers first, then mobile devices, matching the admin UI's ordering.
	for _, dt := range []DeviceType{DeviceTypeComputer, DeviceTypeMobile} {
		n, ok := r.PerEstate[dt]
		if !ok {
			continue
		}
		if total := r.totals.For(dt); total > 0 {
			parts = append(parts, fmt.Sprintf("%d of %d %s", n, total, dt.Noun()))
			continue
		}
		parts = append(parts, fmt.Sprintf("%d %s", n, dt.Noun()))
	}
	return strings.Join(parts, " and ")
}

// unmanagedNote explains a figure larger than the tenant's managed device count.
const unmanagedNote = "The scope names devices that are not managed. Only managed devices receive anything, so the figure is larger than the number that will act on this."

// summarise renders the headline count, with the tenant proportion when known.
func summarise(r Resolution) string {
	// One entry is still worth rendering this way: a resource that can span both
	// estates but names only computers should read "3 of 4 computers", not
	// "3 of 5 devices" against a denominator including mobile devices it never
	// touches.
	if len(r.PerEstate) > 0 {
		return qualifier(r.Bound) + splitEstates(r)
	}
	many := r.DeviceType.Noun()
	noun := many
	if r.Count == 1 {
		noun = singularNoun(r.DeviceType)
	}
	head := figure(r.Count, r.Bound, singularNoun(r.DeviceType), many)
	// A proportion of the managed estate makes no sense once the figure exceeds it:
	// "2 of 1 mobile devices (200%)" reads as a defect rather than as the fact that
	// some named devices are unmanaged.
	if r.Total > 0 && !r.exceedsManaged {
		head = strings.Replace(head, " "+noun, fmt.Sprintf(" of %d %s", r.Total, many), 1)
		if pct, ok := r.Percent(); ok {
			head += fmt.Sprintf(" (%d%%)", pct)
		}
	}
	return head
}

// namedGroups renders up to maxListedGroups group names with their sizes.
func namedGroups(gs []Group) string {
	var named []string
	for i, g := range gs {
		if i == maxListedGroups {
			named = append(named, fmt.Sprintf("and %d more", len(gs)-maxListedGroups))
			break
		}
		named = append(named, fmt.Sprintf("%s (%d)", g.Name, g.MembershipCount))
	}
	return strings.Join(named, ", ")
}

// excludedLine renders the exclusion side of the breakdown.
//
// Without it a figure of 1 sitting under "counted from 1 group: All Managed
// Clients (4)" reads as a contradiction rather than as a subtraction.
func excludedLine(r Resolution) string {
	if len(r.ExcludedGroups) == 0 {
		return ""
	}
	return fmt.Sprintf("Less %s excluded: %s.",
		plural(len(r.ExcludedGroups), "group", "groups"), namedGroups(r.ExcludedGroups))
}

// breakdown lists what was counted, so a reviewer can see where the figure came
// from rather than having to trust it.
func breakdown(r Resolution) []string {
	var lines []string
	for _, dt := range r.tenantWide {
		lines = append(lines, fmt.Sprintf("every managed %s (%d)", singularNoun(dt), r.totals.For(dt)))
	}
	if len(r.Groups) > 0 {
		lines = append(lines, fmt.Sprintf("%s: %s", plural(len(r.Groups), "group", "groups"), namedGroups(r.Groups)))
	}
	lines = append(lines, r.namedDescribed...)
	if r.DirectDevices > 0 {
		// The approximate path adds named devices on rather than resolving them, so it
		// reports them separately from the resolved categories above.
		lines = append(lines, fmt.Sprintf("%s scoped individually",
			plural(r.DirectDevices, singularNoun(r.DeviceType), r.DeviceType.Noun())))
	}
	return lines
}

// singularNoun returns the singular admin-UI noun for a device type.
func singularNoun(d DeviceType) string {
	switch d {
	case DeviceTypeMobile:
		return "mobile device"
	case DeviceTypeComputer:
		return "computer"
	default:
		return "device"
	}
}

// plural renders a count with the right noun form.
func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// caveats renders the inputs that could not be evaluated, grouped by the
// direction they move the figure in so the reader learns which way it is wrong.
func caveats(r Resolution) []string {
	if len(r.Unresolvable) == 0 && len(r.MissingGroupIDs) == 0 && len(r.Mentioned) == 0 && !r.exceedsManaged {
		return nil
	}
	var lines []string
	if r.exceedsManaged {
		lines = append(lines, fmt.Sprintf("%s The tenant has %s under management.",
			unmanagedNote, plural(int(r.Total), singularNoun(r.DeviceType), r.DeviceType.Noun())))
	}
	var narrows, broadens, ambiguous []Unresolvable
	for _, u := range r.Unresolvable {
		switch u.Effect {
		case Narrows:
			narrows = append(narrows, u)
		case Broadens:
			broadens = append(broadens, u)
		default:
			ambiguous = append(ambiguous, u)
		}
	}
	emit := func(us []Unresolvable, heading string) {
		if len(us) == 0 {
			return
		}
		lines = append(lines, heading)
		for _, u := range us {
			lines = append(lines, fmt.Sprintf("  · %s (%d) — %s", u.Path, u.Values, u.Reason))
		}
	}
	emit(narrows, "Not resolvable during plan; the true figure may be lower:")
	emit(broadens, "Not resolvable during plan; the true figure may be higher:")
	emit(ambiguous, "Not resolvable during plan; may move the figure either way:")
	if len(r.Mentioned) > 0 {
		lines = append(lines, fmt.Sprintf("Groups referred to but not counted: %s.", namedGroups(r.Mentioned)))
	}
	if len(r.MissingGroupIDs) > 0 {
		lines = append(lines, fmt.Sprintf(
			"Referenced but not found in this tenant, so not counted: %s.",
			strings.Join(r.MissingGroupIDs, ", ")))
	}
	return lines
}

// pendingDetail renders the case where impact cannot be counted at all because
// the scope names something this plan creates.
func pendingDetail(paths []string) string {
	return fmt.Sprintf(
		"Impact cannot be determined during this plan: %s %s a group this plan creates. "+
			"A new group has no membership until it has been applied, and for a smart group that membership is decided by Jamf Pro from its criteria rather than by configuration.\n\n%s",
		joinPaths(paths),
		verbFor(len(paths)),
		snapshotNote,
	)
}

func verbFor(n int) string {
	if n == 1 {
		return "references"
	}
	return "reference"
}

func joinPaths(paths []string) string {
	switch len(paths) {
	case 0:
		return "the scope"
	case 1:
		return paths[0]
	default:
		return strings.Join(paths[:len(paths)-1], ", ") + " and " + paths[len(paths)-1]
	}
}
