// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// assignAppInstallerResourceModel populates a resource model from the flat
// single-deployment GET response. The scalar fields are always refreshed; the
// nested notification_settings / self_service_settings blocks are only refreshed
// when the caller (plan or current state) already manages them. The server
// echoes both blocks populated even when the user never set them, so refreshing
// an unmanaged block would violate the framework's "produced inconsistent result
// after apply" check (plan said null, we'd return a populated object). See
// feedback_optional_computed_nested_object.
func assignAppInstallerResourceModel(state *AppInstallerResourceModel, d *pro.AppInstallerDeployment) {
	if d == nil {
		return
	}

	state.ID = types.StringValue(d.ID)
	state.Name = types.StringValue(d.Name)
	state.Enabled = types.BoolValue(d.Enabled)
	state.AppTitleID = types.StringValue(d.AppTitleID)
	state.DeploymentType = types.StringValue(d.DeploymentType)
	state.UpdateBehavior = types.StringValue(d.UpdateBehavior)
	state.SelectedVersion = types.StringValue(d.SelectedVersion)
	state.LatestAvailableVersion = types.StringValue(d.LatestAvailableVersion)
	state.TitleAvailableInAis = types.BoolValue(d.TitleAvailableInAis)
	state.VersionRemoved = types.BoolValue(d.VersionRemoved)
	state.CategoryID = types.StringValue(d.CategoryID)
	state.SiteID = types.StringValue(d.SiteID)
	state.SmartGroupID = types.StringValue(d.SmartGroupID)
	state.InstallPredefinedConfigProfiles = types.BoolValue(d.InstallPredefinedConfigProfiles)
	state.TriggerAdminNotifications = types.BoolValue(d.TriggerAdminNotifications)

	if state.NotificationSettings != nil && d.NotificationSettings != nil {
		state.NotificationSettings = flattenNotificationSettings(d.NotificationSettings)
	}
	if state.SelfServiceSettings != nil && d.SelfServiceSettings != nil {
		state.SelfServiceSettings = flattenSelfServiceSettings(d.SelfServiceSettings)
	}
}

// flattenNotificationSettings maps the notification block. Every field is
// nullable on the wire (the server echoes null for an unset field, never a zero
// value), so null is preserved as a TF null — matching the Optional-only schema
// and the omit-when-unset input builder, which keeps state consistent with config.
func flattenNotificationSettings(n *pro.AppInstallerNotificationSettings) *NotificationSettingsModel {
	return &NotificationSettingsModel{
		NotificationMessage:  stringPtrValueOrNull(n.NotificationMessage),
		NotificationInterval: int64PtrValueOrNull(n.NotificationInterval),
		DeadlineMessage:      stringPtrValueOrNull(n.DeadlineMessage),
		Deadline:             int64PtrValueOrNull(n.Deadline),
		QuitDelay:            int64PtrValueOrNull(n.QuitDelay),
		CompleteMessage:      stringPtrValueOrNull(n.CompleteMessage),
		Relaunch:             boolPtrValueOrNull(n.Relaunch),
		Suppress:             boolPtrValueOrNull(n.Suppress),
	}
}

// flattenSelfServiceSettings maps the full Self Service block. The server echoes
// every field including per-category featured, so plain value assignment is
// safe; categories is keyed by id (an unordered Set membership).
func flattenSelfServiceSettings(s *pro.AppInstallerSelfServiceSettings) *SelfServiceSettingsModel {
	out := &SelfServiceSettingsModel{
		Description:                 stringPtrValueOrNull(s.Description),
		ForceViewDescription:        boolPtrValueOrFalse(s.ForceViewDescription),
		IncludeInComplianceCategory: boolPtrValueOrFalse(s.IncludeInComplianceCategory),
		IncludeInFeaturedCategory:   boolPtrValueOrFalse(s.IncludeInFeaturedCategory),
	}

	out.Categories = []SelfServiceCategoryModel{}
	if s.Categories != nil {
		out.Categories = make([]SelfServiceCategoryModel, 0, len(*s.Categories))
		for _, c := range *s.Categories {
			id := ""
			if c.ID != nil {
				id = *c.ID
			}
			out.Categories = append(out.Categories, SelfServiceCategoryModel{
				CategoryID: types.StringValue(id),
				Featured:   boolPtrValueOrFalse(c.Featured),
			})
		}
	}
	return out
}

// assignAppInstallerDataSourceModel projects the flat GET shape into the
// singular data source model (scalar fields only).
func assignAppInstallerDataSourceModel(state *AppInstallerDataSourceModel, d *pro.AppInstallerDeployment) {
	if d == nil {
		return
	}
	state.ID = types.StringValue(d.ID)
	state.Name = types.StringValue(d.Name)
	state.Enabled = types.BoolValue(d.Enabled)
	state.AppTitleID = types.StringValue(d.AppTitleID)
	state.DeploymentType = types.StringValue(d.DeploymentType)
	state.UpdateBehavior = types.StringValue(d.UpdateBehavior)
	state.SelectedVersion = types.StringValue(d.SelectedVersion)
	state.LatestAvailableVersion = types.StringValue(d.LatestAvailableVersion)
	state.TitleAvailableInAis = types.BoolValue(d.TitleAvailableInAis)
	state.VersionRemoved = types.BoolValue(d.VersionRemoved)
	state.CategoryID = types.StringValue(d.CategoryID)
	state.SiteID = types.StringValue(d.SiteID)
	state.SmartGroupID = types.StringValue(d.SmartGroupID)
	state.InstallPredefinedConfigProfiles = types.BoolValue(d.InstallPredefinedConfigProfiles)
	state.TriggerAdminNotifications = types.BoolValue(d.TriggerAdminNotifications)
}

// stringPtrValueOrNull maps an SDK *string to a TF String, nil → null. Used for
// nullable fields the server echoes as null when unset.
func stringPtrValueOrNull(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

// int64PtrValueOrNull maps an SDK *int to a TF Int64, nil → null.
func int64PtrValueOrNull(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}

// boolPtrValueOrNull maps an SDK *bool to a TF Bool, nil → null. Used for
// nullable notification booleans the server echoes as null when unset.
func boolPtrValueOrNull(p *bool) types.Bool {
	if p == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*p)
}

// boolPtrValueOrFalse maps an SDK *bool to a TF Bool, nil → false. Used for the
// Self Service booleans, which the server defaults to false.
func boolPtrValueOrFalse(p *bool) types.Bool {
	if p == nil {
		return types.BoolValue(false)
	}
	return types.BoolValue(*p)
}
