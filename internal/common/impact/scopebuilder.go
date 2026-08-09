// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Reasons an input cannot be evaluated during a plan. They are user-facing text
// and are kept together so the same input always reads the same way, whichever
// resource reports it.
const (
	// ReasonNetworkSegment covers network segment limitations and exclusions.
	// Jamf Pro matches these against where a device is when it checks in, so
	// membership does not exist ahead of time.
	ReasonNetworkSegment = "network segments are matched against a device's network location when it checks in"
	// ReasonIbeacon covers iBeacon limitations and exclusions, matched as a
	// device enters or leaves range.
	ReasonIbeacon = "iBeacon regions are matched as a device enters or leaves range"
	// ReasonDirectoryServiceGroup covers directory service user group scope,
	// resolved against the directory when a user logs in.
	ReasonDirectoryServiceGroup = "directory service group membership is resolved when a user logs in"
	// ReasonUserName covers directory service or local user name scope.
	ReasonUserName = "user names are matched when a user logs in"
	// ReasonUserTarget covers Jamf Pro user and user group targets, which reach
	// devices through user assignment rather than directly.
	ReasonUserTarget = "user-based targets reach devices through user assignment, which Jamf Pro resolves"
	// ReasonNotCounted covers inputs the provider does not yet count. Stated as
	// a limitation of the calculation rather than of Jamf Pro, because it is one.
	ReasonNotCounted = "not included in this calculation"
)

// ScopeBuilder assembles a Scope from Terraform collection values, recording
// unresolvable inputs and pending references as it goes.
//
// It exists so each resource family's adapter reads as a short declaration of
// which attribute means what, rather than repeating null/unknown handling. The
// builder is deliberately not aware of any resource's model type: the shared
// scope helper has several model variants, and blueprints and compliance
// benchmarks do not use it at all.
type ScopeBuilder struct {
	ctx   context.Context
	scope Scope
}

// NewScopeBuilder starts a builder for a device type.
func NewScopeBuilder(ctx context.Context, dt DeviceType) *ScopeBuilder {
	return &ScopeBuilder{ctx: ctx, scope: Scope{DeviceType: dt}}
}

// Scope returns the assembled scope.
func (b *ScopeBuilder) Scope() Scope { return b.scope }

// All records the tenant-wide target flag.
func (b *ScopeBuilder) All(v types.Bool) *ScopeBuilder {
	if !v.IsNull() && !v.IsUnknown() && v.ValueBool() {
		b.scope.All = true
	}
	return b
}

// Devices records individually scoped devices.
func (b *ScopeBuilder) Devices(attrPath string, set types.Set) *ScopeBuilder {
	ids, pending := setStrings(b.ctx, set)
	if pending {
		b.scope.PendingPaths = append(b.scope.PendingPaths, attrPath)
	}
	b.scope.DeviceIDs = append(b.scope.DeviceIDs, ids...)
	return b
}

// JamfProGroups records groups referenced by their numeric Jamf Pro id.
func (b *ScopeBuilder) JamfProGroups(attrPath string, set types.Set) *ScopeBuilder {
	ids, pending := setStrings(b.ctx, set)
	if pending {
		b.scope.PendingPaths = append(b.scope.PendingPaths, attrPath)
	}
	b.scope.JamfProGroupIDs = append(b.scope.JamfProGroupIDs, ids...)
	return b
}

// PlatformGroups records groups referenced by their Platform UUID.
func (b *ScopeBuilder) PlatformGroups(attrPath string, set types.Set) *ScopeBuilder {
	ids, pending := setStrings(b.ctx, set)
	if pending {
		b.scope.PendingPaths = append(b.scope.PendingPaths, attrPath)
	}
	b.scope.PlatformGroupIDs = append(b.scope.PlatformGroupIDs, ids...)
	return b
}

// PlatformGroupIDs records already-extracted Platform UUIDs, for callers whose
// references are not held in a Set (a blueprint's activation conditions embed
// them in an expression).
func (b *ScopeBuilder) PlatformGroupIDs(ids ...string) *ScopeBuilder {
	b.scope.PlatformGroupIDs = append(b.scope.PlatformGroupIDs, ids...)
	return b
}

// ExcludedJamfProGroups records excluded groups referenced by numeric Jamf Pro
// id, as data so their membership can be subtracted.
func (b *ScopeBuilder) ExcludedJamfProGroups(attrPath string, set types.Set) *ScopeBuilder {
	ids, pending := setStrings(b.ctx, set)
	if pending {
		b.scope.PendingPaths = append(b.scope.PendingPaths, attrPath)
	}
	b.scope.ExcludedJamfProGroupIDs = append(b.scope.ExcludedJamfProGroupIDs, ids...)
	return b
}

// ExcludedPlatformGroups records excluded groups referenced by Platform UUID.
func (b *ScopeBuilder) ExcludedPlatformGroups(attrPath string, set types.Set) *ScopeBuilder {
	ids, pending := setStrings(b.ctx, set)
	if pending {
		b.scope.PendingPaths = append(b.scope.PendingPaths, attrPath)
	}
	b.scope.ExcludedPlatformGroupIDs = append(b.scope.ExcludedPlatformGroupIDs, ids...)
	return b
}

// ExcludedDevices records individually excluded devices.
func (b *ScopeBuilder) ExcludedDevices(attrPath string, set types.Set) *ScopeBuilder {
	ids, pending := setStrings(b.ctx, set)
	if pending {
		b.scope.PendingPaths = append(b.scope.PendingPaths, attrPath)
	}
	b.scope.ExcludedDeviceIDs = append(b.scope.ExcludedDeviceIDs, ids...)
	return b
}

// Pending records an attribute path whose value this plan creates.
func (b *ScopeBuilder) Pending(attrPath string) *ScopeBuilder {
	b.scope.PendingPaths = append(b.scope.PendingPaths, attrPath)
	return b
}

// Narrows records a set-valued input that reduces the audience by an amount the
// provider cannot compute.
func (b *ScopeBuilder) Narrows(attrPath string, set types.Set, reason string) *ScopeBuilder {
	return b.unresolvable(attrPath, set, reason, Narrows)
}

// Broadens records a set-valued input that increases the audience by an amount
// the provider cannot compute.
func (b *ScopeBuilder) Broadens(attrPath string, set types.Set, reason string) *ScopeBuilder {
	return b.unresolvable(attrPath, set, reason, Broadens)
}

// BroadensIf records a tenant-wide user target flag as a broadening input.
func (b *ScopeBuilder) BroadensIf(attrPath string, v types.Bool, reason string) *ScopeBuilder {
	if !v.IsNull() && !v.IsUnknown() && v.ValueBool() {
		b.scope.Unresolvable = append(b.scope.Unresolvable, Unresolvable{
			Path: attrPath, Reason: reason, Effect: Broadens, Values: 1,
		})
	}
	return b
}

func (b *ScopeBuilder) unresolvable(attrPath string, set types.Set, reason string, e Effect) *ScopeBuilder {
	ids, pending := setStrings(b.ctx, set)
	if len(ids) == 0 && !pending {
		return b
	}
	n := len(ids)
	if pending && n == 0 {
		n = 1
	}
	b.scope.Unresolvable = append(b.scope.Unresolvable, Unresolvable{
		Path: attrPath, Reason: reason, Effect: e, Values: n,
	})
	return b
}

// setStrings extracts the known string elements of a set. It reports pending
// when the set itself, or any element, is not yet known — which is how Terraform
// represents a reference to something created by the same plan.
func setStrings(ctx context.Context, set types.Set) (ids []string, pending bool) {
	if set.IsNull() {
		return nil, false
	}
	if set.IsUnknown() {
		return nil, true
	}
	for _, el := range set.Elements() {
		if el.IsUnknown() {
			pending = true
			continue
		}
		if el.IsNull() {
			continue
		}
		sv, ok := el.(types.String)
		if !ok {
			continue
		}
		if v := sv.ValueString(); v != "" {
			ids = append(ids, v)
		}
	}
	return ids, pending
}
