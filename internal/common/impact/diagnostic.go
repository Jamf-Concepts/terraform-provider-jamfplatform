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

// figure renders a count with the qualifier its bound earns.
//
// The qualifier is not a hedge for its own sake: Jamf Pro's scope model narrows
// as well as broadens, so the direction of the uncertainty is known even when
// the exact number is not.
func figure(n int64, b Bound, noun string) string {
	switch b {
	case BoundAtMost:
		return fmt.Sprintf("up to %d %s", n, noun)
	case BoundAtLeast:
		return fmt.Sprintf("%d or more %s", n, noun)
	case BoundUnknown:
		return fmt.Sprintf("an estimated %d %s", n, noun)
	default:
		return fmt.Sprintf("%d %s", n, noun)
	}
}

// summarise renders the headline count, with the tenant proportion when known.
func summarise(r Resolution) string {
	noun := r.DeviceType.Noun()
	head := figure(r.Count, r.Bound, noun)
	if r.Total > 0 {
		head = strings.Replace(head, " "+noun, fmt.Sprintf(" of %d %s", r.Total, noun), 1)
		if pct, ok := r.Percent(); ok {
			head += fmt.Sprintf(" (%d%%)", pct)
		}
	}
	return head
}

// breakdown lists what was counted, so a reviewer can see where the figure came
// from rather than having to trust it.
func breakdown(r Resolution) []string {
	var lines []string
	if len(r.Groups) > 0 {
		var named []string
		for i, g := range r.Groups {
			if i == maxListedGroups {
				named = append(named, fmt.Sprintf("and %d more", len(r.Groups)-maxListedGroups))
				break
			}
			named = append(named, fmt.Sprintf("%s (%d)", g.Name, g.MembershipCount))
		}
		lines = append(lines, fmt.Sprintf("%s: %s", plural(len(r.Groups), "group", "groups"), strings.Join(named, ", ")))
	}
	if r.DirectDevices > 0 {
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
	if len(r.Unresolvable) == 0 && len(r.MissingGroupIDs) == 0 && len(r.Mentioned) == 0 {
		return nil
	}
	var lines []string
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
		var named []string
		for i, g := range r.Mentioned {
			if i == maxListedGroups {
				named = append(named, fmt.Sprintf("and %d more", len(r.Mentioned)-maxListedGroups))
				break
			}
			named = append(named, fmt.Sprintf("%s (%d)", g.Name, g.MembershipCount))
		}
		lines = append(lines, fmt.Sprintf("Groups referred to but not counted: %s.", strings.Join(named, ", ")))
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
