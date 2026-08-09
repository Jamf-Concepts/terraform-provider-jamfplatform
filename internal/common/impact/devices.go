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
//   - Mobile devices filter on mobileDeviceId, also numeric, but on building and
//     department by *name*. A scope block carries ids, so those two are translated
//     through the tenant's building and department lists first — one cached read
//     each, and only when a mobile scope actually names one.
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
//
// values are already in whatever form the field expects — numeric ids for the
// computer fields and mobileDeviceId, names for the mobile building and
// department fields. Names are quoted; numeric ids need no quoting, and the API
// accepts either.
func deviceFilter(dt DeviceType, kind deviceFilterKind, values []string) (string, bool) {
	if len(values) == 0 {
		return "", false
	}
	var (
		field  string
		quoted bool
	)
	switch {
	case dt == DeviceTypeComputer && kind == filterKindDevice:
		field = "id"
	case dt == DeviceTypeComputer && kind == filterKindBuilding:
		field = "userAndLocation.buildingId"
	case dt == DeviceTypeComputer && kind == filterKindDepartment:
		field = "userAndLocation.departmentId"
	case dt == DeviceTypeMobile && kind == filterKindDevice:
		field = "mobileDeviceId"
	case dt == DeviceTypeMobile && kind == filterKindBuilding:
		field, quoted = "building", true
	case dt == DeviceTypeMobile && kind == filterKindDepartment:
		field, quoted = "department", true
	default:
		return "", false
	}

	terms := make([]string, 0, len(values))
	for _, v := range values {
		if quoted {
			terms = append(terms, fmt.Sprintf("%s==%q", field, v))
			continue
		}
		terms = append(terms, fmt.Sprintf("%s==%s", field, v))
	}
	sort.Strings(terms)
	return strings.Join(terms, " or "), true
}

// deviceIDsFor resolves one scope category into device management identifiers,
// reporting ok=false when the category cannot be resolved for this estate — in
// which case the caller keeps its existing unresolved treatment.
func (c *Cache) deviceIDsFor(ctx context.Context, dt DeviceType, kind deviceFilterKind, ids []string) ([]string, bool, error) {
	values, resolvable, err := c.filterValues(ctx, dt, kind, ids)
	if err != nil || !resolvable {
		return nil, false, err
	}
	filter, supported := deviceFilter(dt, kind, values)
	if !supported {
		return nil, false, nil
	}
	out, err := c.devicesByFilter(ctx, dt, filter)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

// filterValues converts a scope category's identifiers into the values its filter
// field expects. Only the mobile building and department fields need translating,
// because they match on name.
//
// An identifier with no matching name is dropped rather than guessed at, and if
// that leaves nothing the category reports as unresolvable — better an honest
// caveat than a filter that silently matches everything or nothing.
func (c *Cache) filterValues(ctx context.Context, dt DeviceType, kind deviceFilterKind, ids []string) ([]string, bool, error) {
	if dt != DeviceTypeMobile || (kind != filterKindBuilding && kind != filterKindDepartment) {
		return ids, true, nil
	}
	places, err := c.placeNames(ctx)
	if err != nil {
		// Names unavailable, so the category keeps its unresolved treatment.
		return nil, false, nil
	}
	lookup := places.Buildings
	if kind == filterKindDepartment {
		lookup = places.Departments
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := lookup[id]; ok && name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, false, nil
	}
	return names, true, nil
}

// placeNames reads the building and department names once per plan.
func (c *Cache) placeNames(ctx context.Context) (Places, error) {
	c.placeOnce.Do(func() {
		c.places, c.placeErr = c.src.PlaceNames(ctx)
	})
	return c.places, c.placeErr
}

// devicesByFilter reads the management identifiers matching an inventory filter,
// once per distinct filter per plan. Keyed by estate as well as filter, since the
// two estates have separate inventories.
func (c *Cache) devicesByFilter(ctx context.Context, dt DeviceType, filter string) ([]string, error) {
	if !c.Enabled() {
		return nil, nil
	}
	key := string(dt) + ":" + filter
	c.deviceMu.Lock()
	if c.devices == nil {
		c.devices = make(map[string]*memberSet)
	}
	entry, ok := c.devices[key]
	if !ok {
		entry = &memberSet{}
		c.devices[key] = entry
	}
	c.deviceMu.Unlock()

	entry.once.Do(func() {
		if dt == DeviceTypeMobile {
			entry.ids, entry.err = c.src.MobileManagementIDs(ctx, filter)
			return
		}
		entry.ids, entry.err = c.src.ComputerManagementIDs(ctx, filter)
	})
	return entry.ids, entry.err
}
