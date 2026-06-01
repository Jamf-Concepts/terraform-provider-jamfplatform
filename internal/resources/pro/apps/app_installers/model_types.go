// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installers

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AppInstallersDataSourceModel is the plural data source model. `deployments`
// carries the expanded list shape; `name_substring` narrows it client-side (the
// deployments endpoint has no server-side filter).
type AppInstallersDataSourceModel struct {
	ID            types.String      `tfsdk:"id"`
	NameSubstring types.String      `tfsdk:"name_substring"`
	Deployments   []DeploymentModel `tfsdk:"deployments"`
	Timeouts      timeouts.Value    `tfsdk:"timeouts"`
}

// DeploymentModel is one expanded deployment list entry.
type DeploymentModel struct {
	ID               types.String           `tfsdk:"id"`
	Name             types.String           `tfsdk:"name"`
	Enabled          types.Bool             `tfsdk:"enabled"`
	DeploymentType   types.String           `tfsdk:"deployment_type"`
	UpdateBehavior   types.String           `tfsdk:"update_behavior"`
	App              *AppModel              `tfsdk:"app"`
	Site             *NamedRefModel         `tfsdk:"site"`
	Category         *NamedRefModel         `tfsdk:"category"`
	SmartGroup       *NamedRefModel         `tfsdk:"smart_group"`
	ComputerStatuses *ComputerStatusesModel `tfsdk:"computer_statuses"`
}

// AppModel is the resolved catalog app reference on a list entry.
type AppModel struct {
	ID                  types.String `tfsdk:"id"`
	BundleID            types.String `tfsdk:"bundle_id"`
	IconURL             types.String `tfsdk:"icon_url"`
	LatestVersion       types.String `tfsdk:"latest_version"`
	SelectedVersion     types.String `tfsdk:"selected_version"`
	DeployedVersion     types.String `tfsdk:"deployed_version"`
	MediaSourceType     types.String `tfsdk:"media_source_type"`
	TitleAvailableInAis types.Bool   `tfsdk:"title_available_in_ais"`
	VersionRemoved      types.Bool   `tfsdk:"version_removed"`
}

// NamedRefModel is a resolved {id, name} reference (site / category / smart group).
type NamedRefModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// ComputerStatusesModel carries the per-deployment computer status counts.
type ComputerStatusesModel struct {
	Available   types.Int64 `tfsdk:"available"`
	Failed      types.Int64 `tfsdk:"failed"`
	InProgress  types.Int64 `tfsdk:"in_progress"`
	Installed   types.Int64 `tfsdk:"installed"`
	Unqualified types.Int64 `tfsdk:"unqualified"`
}
