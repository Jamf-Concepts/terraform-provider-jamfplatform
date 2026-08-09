// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// This file resolves the scope categories that name devices rather than groups —
// individually scoped devices, and the buildings and departments a device belongs
// to — into the same device management identifiers group membership uses.
//
// Doing so is what lets them join the exact set arithmetic instead of being added
// on as separate terms. Both problems turn out to be the same one: a Jamf Pro
// numeric identifier has to become a management identifier before it can be
// compared with a group's membership. One filtered inventory read answers both.
//
// The filter fields differ per estate, which is the only reason this needs a
// per-estate switch:
//
//   - Computers filter on userAndLocation.buildingId / departmentId and id, all
//     numeric, matching what a scope block carries.
//   - Mobile devices are not resolved here. Their inventory is a discriminated
//     union across four operating systems, and their buildings and departments
//     filter by *name* while a scope block carries ids — so the mobile estate
//     keeps the unresolved treatment until both are handled.
//
// Filters are validated server-side — an unknown field is rejected with a 400
// rather than silently matching nothing — so a filter that returns no devices
// genuinely means no devices match, which is what makes an empty result safe to
// treat as authoritative.

// deviceFilterKind names the scope category a lookup serves, so results can be
// cached per category as well as per identifier.
type deviceFilterKind string

const (
	filterKindDevice     deviceFilterKind = "device"
	filterKindBuilding   deviceFilterKind = "building"
	filterKindDepartment deviceFilterKind = "department"
)

// deviceFilter builds the inventory filter for one scope category, and reports
// whether this estate supports it at all.
func deviceFilter(dt DeviceType, kind deviceFilterKind, ids []string) (string, bool) {
	if len(ids) == 0 {
		return "", false
	}
	var field string
	switch {
	case dt == DeviceTypeComputer && kind == filterKindDevice:
		field = "id"
	case dt == DeviceTypeComputer && kind == filterKindBuilding:
		field = "userAndLocation.buildingId"
	case dt == DeviceTypeComputer && kind == filterKindDepartment:
		field = "userAndLocation.departmentId"
	default:
		return "", false
	}

	terms := make([]string, 0, len(ids))
	for _, id := range ids {
		terms = append(terms, fmt.Sprintf("%s==%s", field, id))
	}
	sort.Strings(terms)
	return strings.Join(terms, " or "), true
}

// deviceIDsFor resolves one scope category into device management identifiers,
// reporting ok=false when the category cannot be resolved for this estate — in
// which case the caller keeps its existing unresolved treatment.
func (c *Cache) deviceIDsFor(ctx context.Context, dt DeviceType, kind deviceFilterKind, ids []string) ([]string, bool, error) {
	filter, supported := deviceFilter(dt, kind, ids)
	if !supported {
		return nil, false, nil
	}
	out, err := c.devicesByFilter(ctx, filter)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

// devicesByFilter reads the management identifiers matching an inventory filter,
// once per distinct filter per plan.
func (c *Cache) devicesByFilter(ctx context.Context, filter string) ([]string, error) {
	if !c.Enabled() {
		return nil, nil
	}
	c.deviceMu.Lock()
	if c.devices == nil {
		c.devices = make(map[string]*memberSet)
	}
	entry, ok := c.devices[filter]
	if !ok {
		entry = &memberSet{}
		c.devices[filter] = entry
	}
	c.deviceMu.Unlock()

	entry.once.Do(func() {
		entry.ids, entry.err = c.src.ComputerManagementIDs(ctx, filter)
	})
	return entry.ids, entry.err
}
