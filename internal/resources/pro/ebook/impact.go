// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ebook

import (
	"context"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// An ebook is the one scope-bearing resource that targets both estates in a
// single block: computers and mobile devices side by side, plus classes. The
// audience is their union, so the figure is measured against the whole managed
// estate rather than one half of it, and each group reference carries the estate
// it belongs to — numeric group ids repeat across the two.
func ebookImpactScope(ctx context.Context, m *EbookResourceModel) impact.Scope {
	if m == nil || m.Scope == nil {
		return impact.Scope{DeviceType: impact.DeviceTypeAny}
	}
	s := m.Scope
	in := scope.ImpactInputs{
		DeviceType:  impact.DeviceTypeAny,
		DeviceAttr:  "computer_ids",
		GroupAttr:   "computer_group_ids",
		GroupEstate: impact.DeviceTypeComputer,
	}
	if t := s.Targets; t != nil {
		in.All = t.AllComputers
		in.DeviceIDs = t.ComputerIDs
		in.GroupIDs = t.ComputerGroupIDs
		in.AllJssUsers = t.AllJssUsers
		in.BuildingIDs = t.BuildingIDs
		in.DepartmentIDs = t.DepartmentIDs
		in.UserIDs = t.UserIDs
		in.UserGroupIDs = t.UserGroupIDs
		in.ClassIDs = t.ClassIDs
		in.SecondaryDevices = &scope.SecondaryEstate{
			DeviceType: impact.DeviceTypeMobile,
			DeviceAttr: "mobile_device_ids",
			GroupAttr:  "mobile_device_group_ids",
			All:        t.AllMobileDevices,
			DeviceIDs:  t.MobileDeviceIDs,
			GroupIDs:   t.MobileDeviceGroupIDs,
		}
	}
	if l := s.Limitations; l != nil {
		in.LimitNetworkSegmentIDs = l.NetworkSegmentIDs
		in.LimitUserNames = l.DirectoryServiceOrLocalUserNames
		in.LimitDSGroupNames = l.DirectoryServiceUserGroupNames
	}
	if e := s.Exclusions; e != nil {
		in.ExcludeDeviceIDs = e.ComputerIDs
		in.ExcludeGroupIDs = e.ComputerGroupIDs
		in.ExcludeBuildingIDs = e.BuildingIDs
		in.ExcludeDepartmentIDs = e.DepartmentIDs
		in.ExcludeUserIDs = e.UserIDs
		in.ExcludeUserGroupIDs = e.UserGroupIDs
		in.ExcludeNetworkSegmentIDs = e.NetworkSegmentIDs
		in.ExcludeUserNames = e.DirectoryServiceOrLocalUserNames
		in.ExcludeDSGroupNames = e.DirectoryServiceUserGroupNames
		in.ExcludeSecondaryDevices = &scope.SecondaryEstate{
			DeviceType: impact.DeviceTypeMobile,
			DeviceAttr: "mobile_device_ids",
			GroupAttr:  "mobile_device_group_ids",
			DeviceIDs:  e.MobileDeviceIDs,
			GroupIDs:   e.MobileDeviceGroupIDs,
		}
	}
	return scope.BuildImpactScope(ctx, in)
}

// reportScopeImpact emits the plan-time impact alert for a scope change. A ebook
// is a deployable object in Jamf Pro's terms, so the alert reports how many
// devices the change reaches.
func (r *EbookResource) reportScopeImpact(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	impact.ReportPlan(ctx, req, resp, impact.PlanReport{
		Cache: r.impact,
		Path:  path.Root("scope"),
		Label: "ebook",
	}, ebookImpactScope)
}
