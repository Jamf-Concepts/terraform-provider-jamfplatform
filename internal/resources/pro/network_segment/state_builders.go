// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package network_segment

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignNetworkSegmentResourceModel populates a resource model from a NetworkSegment
// response. state.ID is only overwritten when the API ID is non-nil so a transient
// GET that drops the ID does not clobber the value persisted from Create. Optional
// writable string fields (building, department) and the override bools are reconciled
// through helpers.ReconcileOptional*Pointer so the explicit-null vs API-empty
// distinction the user set in config is preserved across refreshes. Computed-only
// server-derived fields (distribution_point, distribution_server, swu_server, url)
// are populated unconditionally with the API value (or null when absent).
func assignNetworkSegmentResourceModel(state *NetworkSegmentResourceModel, s *proclassic.NetworkSegment) {
	if s == nil {
		return
	}
	if s.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(s.ID)
	}
	if s.Name != nil {
		state.Name = helpers.StringPointerValueOrNull(s.Name)
	}
	if s.StartingAddress != nil {
		state.StartingAddress = helpers.StringPointerValueOrNull(s.StartingAddress)
	}
	if s.EndingAddress != nil {
		state.EndingAddress = helpers.StringPointerValueOrNull(s.EndingAddress)
	}
	state.Building = helpers.ReconcileOptionalStringPointer(s.Building, state.Building)
	state.Department = helpers.ReconcileOptionalStringPointer(s.Department, state.Department)
	state.OverrideBuildings = helpers.ReconcileOptionalBoolPointer(s.OverrideBuildings, state.OverrideBuildings)
	state.OverrideDepartments = helpers.ReconcileOptionalBoolPointer(s.OverrideDepartments, state.OverrideDepartments)
	state.DistributionPoint = helpers.StringPointerValueOrNull(s.DistributionPoint)
	state.DistributionServer = helpers.StringPointerValueOrNull(s.DistributionServer)
	state.SwuServer = helpers.StringPointerValueOrNull(s.SwuServer)
	state.URL = helpers.StringPointerValueOrNull(s.URL)
}

// assignNetworkSegmentDataSourceModel populates a data source model from a NetworkSegment
// response. Symmetric with assignNetworkSegmentResourceModel: nil API fields preserve the
// caller-supplied selector (id or name) — DS accepts either as input so silently nulling
// the non-supplied one if the SDK omits it would be hostile.
func assignNetworkSegmentDataSourceModel(state *NetworkSegmentDataSourceModel, s *proclassic.NetworkSegment) {
	if s == nil {
		return
	}
	if s.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(s.ID)
	}
	if s.Name != nil {
		state.Name = helpers.StringPointerValueOrNull(s.Name)
	}
	state.StartingAddress = helpers.StringPointerValueOrNull(s.StartingAddress)
	state.EndingAddress = helpers.StringPointerValueOrNull(s.EndingAddress)
	state.Building = helpers.StringPointerValueOrNull(s.Building)
	state.Department = helpers.StringPointerValueOrNull(s.Department)
	state.OverrideBuildings = helpers.BoolPointerValueOrNull(s.OverrideBuildings)
	state.OverrideDepartments = helpers.BoolPointerValueOrNull(s.OverrideDepartments)
	state.DistributionPoint = helpers.StringPointerValueOrNull(s.DistributionPoint)
	state.DistributionServer = helpers.StringPointerValueOrNull(s.DistributionServer)
	state.SwuServer = helpers.StringPointerValueOrNull(s.SwuServer)
	state.URL = helpers.StringPointerValueOrNull(s.URL)
}
