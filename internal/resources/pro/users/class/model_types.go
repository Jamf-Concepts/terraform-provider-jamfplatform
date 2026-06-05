// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package class

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ClassResourceModel is the Terraform resource model for a Jamf Pro class (the
// "Classes" item under the Computers sidebar — Apple Classroom). Membership is
// modelled directly (classes do not use the shared scope helper):
//
//   - students / teachers are authored by username; Jamf Pro resolves the
//     matching user records and echoes the resolved IDs in student_ids /
//     teacher_ids (Computed). Unknown usernames are auto-created by the server.
//   - the *_group_ids sets reference existing Jamf Pro groups by ID.
//
// The device list and primary mobile_device_group shown in the admin UI are
// server-derived from the assigned mobile device groups' membership, so they are
// not modelled here. meeting_times / apple_tvs (roster-sync only) and the
// Restrictions / Home Screen tabs (separate endpoints) are out of scope.
type ClassResourceModel struct {
	ID                   types.String           `tfsdk:"id"`
	Name                 types.String           `tfsdk:"name"`
	Description          types.String           `tfsdk:"description"`
	SiteID               types.String           `tfsdk:"site_id"`
	SiteName             types.String           `tfsdk:"site_name"`
	Source               types.String           `tfsdk:"source"`
	Students             types.Set              `tfsdk:"students"`
	Teachers             types.Set              `tfsdk:"teachers"`
	StudentGroupIDs      types.Set              `tfsdk:"student_group_ids"`
	TeacherGroupIDs      types.Set              `tfsdk:"teacher_group_ids"`
	MobileDeviceGroupIDs types.Set              `tfsdk:"mobile_device_group_ids"`
	StudentIDs           types.Set              `tfsdk:"student_ids"`
	TeacherIDs           types.Set              `tfsdk:"teacher_ids"`
	Timeouts             resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ClassDataSourceModel is the Terraform data source model for a Jamf Pro class.
// Either id or name must be supplied (enforced by ExactlyOneOf at config
// validation). Every attribute is read-only.
type ClassDataSourceModel struct {
	ID                   types.String             `tfsdk:"id"`
	Name                 types.String             `tfsdk:"name"`
	Description          types.String             `tfsdk:"description"`
	SiteID               types.String             `tfsdk:"site_id"`
	SiteName             types.String             `tfsdk:"site_name"`
	Source               types.String             `tfsdk:"source"`
	Students             types.Set                `tfsdk:"students"`
	Teachers             types.Set                `tfsdk:"teachers"`
	StudentGroupIDs      types.Set                `tfsdk:"student_group_ids"`
	TeacherGroupIDs      types.Set                `tfsdk:"teacher_group_ids"`
	MobileDeviceGroupIDs types.Set                `tfsdk:"mobile_device_group_ids"`
	StudentIDs           types.Set                `tfsdk:"student_ids"`
	TeacherIDs           types.Set                `tfsdk:"teacher_ids"`
	Timeouts             datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// classIdentityModel is the identity object for the resource and list results.
type classIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
