// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package restricted_software

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// buildRestrictedSoftwareInput projects a plan model into an SDK
// *proclassic.RestrictedSoftware suitable for Create / Update. Scope follows
// the omission rules in STYLE_GUIDE.md §Scope helper: nil-pointer sub-blocks
// suppress wire emission; empty child collections collapse up to a nil parent.
func buildRestrictedSoftwareInput(ctx context.Context, plan RestrictedSoftwareResourceModel) (*proclassic.RestrictedSoftware, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &proclassic.RestrictedSoftware{}

	if plan.General != nil {
		out.General = buildGeneral(plan.General)
	}

	if plan.Scope != nil {
		s, d := buildScope(ctx, plan.Scope)
		diags.Append(d...)
		out.Scope = s
	}

	return out, diags
}

// buildGeneral maps the UI-aligned general model into the wire struct. The
// renamed attributes map to their classic wire names here:
// restrict_exact_process_name→MatchExactProcessName,
// send_email_notification_on_violation→SendNotification,
// delete_application→DeleteExecutable.
func buildGeneral(m *RestrictedSoftwareGeneralModel) *proclassic.RestrictedSoftwareGeneral {
	g := &proclassic.RestrictedSoftwareGeneral{
		Name:                  helpers.OptionalStringPointer(m.Name),
		ProcessName:           helpers.OptionalStringPointer(m.ProcessName),
		MatchExactProcessName: helpers.OptionalBoolPointer(m.RestrictExactProcessName),
		SendNotification:      helpers.OptionalBoolPointer(m.SendEmailNotificationOnViolation),
		KillProcess:           helpers.OptionalBoolPointer(m.KillProcess),
		DeleteExecutable:      helpers.OptionalBoolPointer(m.DeleteApplication),
		DisplayMessage:        helpers.OptionalStringPointer(m.DisplayMessage),
	}
	if siteID := helpers.StringIDPtr(m.SiteID); siteID != nil {
		g.Site = &proclassic.SiteObject{ID: siteID}
	}
	return g
}

func buildScope(ctx context.Context, m *RestrictedSoftwareScopeModel) (*proclassic.RestrictedSoftwareScope, diag.Diagnostics) {
	var diags diag.Diagnostics
	t := m.TargetsOrZero()
	s := &proclassic.RestrictedSoftwareScope{
		AllComputers: helpers.OptionalBoolPointer(t.AllComputers),
	}

	computers, d := scope.BuildIDSlice(ctx, t.ComputerIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	if computers != nil {
		s.Computers = &proclassic.RestrictedSoftwareScopeComputers{Computer: computers}
	}

	computerGroups, d := scope.BuildIDSlice(ctx, t.ComputerGroupIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	if computerGroups != nil {
		s.ComputerGroups = &proclassic.RestrictedSoftwareScopeComputerGroups{ComputerGroup: computerGroups}
	}

	buildings, d := scope.BuildIDSlice(ctx, t.BuildingIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	if buildings != nil {
		s.Buildings = &proclassic.RestrictedSoftwareScopeBuildings{Building: buildings}
	}

	departments, d := scope.BuildIDSlice(ctx, t.DepartmentIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	if departments != nil {
		s.Departments = &proclassic.RestrictedSoftwareScopeDepartments{Department: departments}
	}

	if m.Exclusions != nil {
		e, ed := buildScopeExclusions(ctx, m.Exclusions)
		diags.Append(ed...)
		s.Exclusions = e
	}

	// Omission semantics (STYLE_GUIDE.md §Scope helper): collapse to nil when
	// every child pointer is nil so the payload omits <scope> entirely.
	if s.AllComputers == nil && s.Computers == nil && s.ComputerGroups == nil &&
		s.Buildings == nil && s.Departments == nil && s.Exclusions == nil {
		return nil, diags
	}
	return s, diags
}

func buildScopeExclusions(ctx context.Context, m *RestrictedSoftwareScopeExclusionsModel) (*proclassic.RestrictedSoftwareScopeExclusions, diag.Diagnostics) {
	var diags diag.Diagnostics
	e := &proclassic.RestrictedSoftwareScopeExclusions{}

	computers, d := scope.BuildIDSlice(ctx, m.ComputerIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	if computers != nil {
		e.Computers = &proclassic.RestrictedSoftwareScopeExclusionsComputers{Computer: computers}
	}

	computerGroups, d := scope.BuildIDSlice(ctx, m.ComputerGroupIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	if computerGroups != nil {
		e.ComputerGroups = &proclassic.RestrictedSoftwareScopeExclusionsComputerGroups{ComputerGroup: computerGroups}
	}

	buildings, d := scope.BuildIDSlice(ctx, m.BuildingIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	if buildings != nil {
		e.Buildings = &proclassic.RestrictedSoftwareScopeExclusionsBuildings{Building: buildings}
	}

	departments, d := scope.BuildIDSlice(ctx, m.DepartmentIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	if departments != nil {
		e.Departments = &proclassic.RestrictedSoftwareScopeExclusionsDepartments{Department: departments}
	}

	// <users> exclusions are NAME-keyed free-text local usernames.
	users, d := scope.BuildNameSlice(ctx, m.DirectoryServiceOrLocalUserNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if users != nil {
		e.Users = &proclassic.RestrictedSoftwareScopeExclusionsUsers{User: users}
	}

	// Always emit the block when the user declared `exclusions` (the caller's
	// gate). The classic /restrictedsoftware endpoint MERGES an omitted
	// <exclusions> sub-block (wire-probed), so collapsing an all-empty block to
	// nil would retain the server's existing members. An empty
	// <exclusions></exclusions> clears every category, which is what `[]` /
	// omission means. (Target categories are direct <scope> children and replace
	// on omit — no such handling needed.)
	return e, diags
}
