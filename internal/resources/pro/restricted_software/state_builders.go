// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package restricted_software

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignRestrictedSoftwareResourceModel populates a resource model from the SDK
// RestrictedSoftware response. general is always refreshed (required block); the
// optional scope block is only refreshed when the caller (plan or current
// state) already manages it. The classic server echoes <scope> on GET with
// default values, so populating an unmanaged block would violate the
// framework's "produced inconsistent result after apply" check (plan said null,
// we'd return a populated object). See feedback_server_derived_echo_attrs.
//
// includeUnmanaged inverts that gate for the list resource's config-generation
// path (terraform query -generate-config-out): there is no plan to stay
// consistent with, so a wire-present scope is allocated and hydrated, yielding a
// complete exported config rather than a general-only one. CRUD callers pass
// false. The flatteners use the PreferCurrent* helpers (which adopt the wire
// value when the current state is null), so allocating an empty section is
// sufficient for it to fully hydrate.
func assignRestrictedSoftwareResourceModel(ctx context.Context, state *RestrictedSoftwareResourceModel, rs *proclassic.RestrictedSoftware, includeUnmanaged bool) diag.Diagnostics {
	var diags diag.Diagnostics
	if rs == nil {
		return diags
	}

	if id := extractRestrictedSoftwareID(rs); id != "" {
		state.ID = types.StringValue(id)
	}

	if state.General == nil {
		state.General = &RestrictedSoftwareGeneralModel{}
	}
	flattenGeneral(rs.General, state.General)

	if includeUnmanaged && state.Scope == nil && rs.Scope != nil {
		state.Scope = &RestrictedSoftwareScopeModel{}
	}
	if state.Scope != nil && rs.Scope != nil {
		flattenScope(ctx, rs.Scope, state.Scope, includeUnmanaged)
	}

	return diags
}

func flattenGeneral(g *proclassic.RestrictedSoftwareGeneral, state *RestrictedSoftwareGeneralModel) {
	if g == nil {
		return
	}
	state.ID = helpers.StringValueFromIntPtr(g.ID)
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.ProcessName = helpers.StringPointerValueOrNull(g.ProcessName)
	state.RestrictExactProcessName = helpers.PreferCurrentBoolPointer(g.MatchExactProcessName, state.RestrictExactProcessName)
	state.SendEmailNotificationOnViolation = helpers.PreferCurrentBoolPointer(g.SendNotification, state.SendEmailNotificationOnViolation)
	state.KillProcess = helpers.PreferCurrentBoolPointer(g.KillProcess, state.KillProcess)
	state.DeleteApplication = helpers.PreferCurrentBoolPointer(g.DeleteExecutable, state.DeleteApplication)
	state.DisplayMessage = helpers.PreferCurrentStringPointer(g.DisplayMessage, state.DisplayMessage)

	if g.Site != nil {
		state.SiteID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(g.Site.ID), state.SiteID)
		state.SiteName = helpers.DerivedRefName(g.Site.ID, g.Site.Name)
	} else {
		state.SiteID = helpers.PreferCurrentStringPointer(nil, state.SiteID)
		state.SiteName = types.StringNull()
	}
}

// flattenScope refreshes the scope sub-blocks the caller already manages. When
// includeUnmanaged is set (config generation) every wire-present sub-block is
// first allocated so the from-scratch read hydrates the full scope rather than
// leaving unmanaged targets/exclusions null. Restricted software has no
// limitations tab.
func flattenScope(ctx context.Context, s *proclassic.RestrictedSoftwareScope, state *RestrictedSoftwareScopeModel, includeUnmanaged bool) {
	if includeUnmanaged {
		if state.Targets == nil {
			state.Targets = &RestrictedSoftwareScopeTargetsModel{}
		}
		if state.Exclusions == nil && s.Exclusions != nil {
			state.Exclusions = &RestrictedSoftwareScopeExclusionsModel{}
		}
	}

	if state.Targets != nil {
		state.Targets.AllComputers = helpers.PreferCurrentBoolPointer(s.AllComputers, state.Targets.AllComputers)
		state.Targets.ComputerIDs = scope.FlattenIDNameSet(ctx, computerSlice(s.Computers))
		state.Targets.ComputerGroupIDs = scope.FlattenIDNameSet(ctx, computerGroupSlice(s.ComputerGroups))
		state.Targets.BuildingIDs = scope.FlattenIDNameSet(ctx, buildingSlice(s.Buildings))
		state.Targets.DepartmentIDs = scope.FlattenIDNameSet(ctx, departmentSlice(s.Departments))
	}

	if state.Exclusions != nil && s.Exclusions != nil {
		e := s.Exclusions
		state.Exclusions.ComputerIDs = scope.FlattenIDNameSet(ctx, exclComputerSlice(e.Computers))
		state.Exclusions.ComputerGroupIDs = scope.FlattenIDNameSet(ctx, exclComputerGroupSlice(e.ComputerGroups))
		state.Exclusions.BuildingIDs = scope.FlattenIDNameSet(ctx, exclBuildingSlice(e.Buildings))
		state.Exclusions.DepartmentIDs = scope.FlattenIDNameSet(ctx, exclDepartmentSlice(e.Departments))
		state.Exclusions.DirectoryServiceOrLocalUserNames = scope.FlattenNameSet(ctx, exclUsersSlice(e.Users))
	}
}

// ---- scope sub-slice accessors -------------------------------------------------

func computerSlice(c *proclassic.RestrictedSoftwareScopeComputers) *[]proclassic.IDName {
	if c == nil {
		return nil
	}
	return c.Computer
}

func computerGroupSlice(g *proclassic.RestrictedSoftwareScopeComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func buildingSlice(b *proclassic.RestrictedSoftwareScopeBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func departmentSlice(d *proclassic.RestrictedSoftwareScopeDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func exclComputerSlice(c *proclassic.RestrictedSoftwareScopeExclusionsComputers) *[]proclassic.IDName {
	if c == nil {
		return nil
	}
	return c.Computer
}

func exclComputerGroupSlice(g *proclassic.RestrictedSoftwareScopeExclusionsComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func exclBuildingSlice(b *proclassic.RestrictedSoftwareScopeExclusionsBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func exclDepartmentSlice(d *proclassic.RestrictedSoftwareScopeExclusionsDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func exclUsersSlice(u *proclassic.RestrictedSoftwareScopeExclusionsUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

// ---- set flatteners ------------------------------------------------------------
