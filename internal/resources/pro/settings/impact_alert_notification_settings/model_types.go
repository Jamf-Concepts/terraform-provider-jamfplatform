// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact_alert_notification_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ImpactAlertNotificationSettingsResourceModel represents the Terraform resource model
// for Jamf Pro Impact Alert Notification settings.
type ImpactAlertNotificationSettingsResourceModel struct {
	ID                                       types.String           `tfsdk:"id"`
	DeployableObjectsAlertEnabled            types.Bool             `tfsdk:"deployable_objects_alert_enabled"`
	DeployableObjectsConfirmationCodeEnabled types.Bool             `tfsdk:"deployable_objects_confirmation_code_enabled"`
	ScopeableObjectsAlertEnabled             types.Bool             `tfsdk:"scopeable_objects_alert_enabled"`
	ScopeableObjectsConfirmationCodeEnabled  types.Bool             `tfsdk:"scopeable_objects_confirmation_code_enabled"`
	Timeouts                                 resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ImpactAlertNotificationSettingsDataSourceModel represents the Terraform data source model.
type ImpactAlertNotificationSettingsDataSourceModel struct {
	ID                                       types.String             `tfsdk:"id"`
	DeployableObjectsAlertEnabled            types.Bool               `tfsdk:"deployable_objects_alert_enabled"`
	DeployableObjectsConfirmationCodeEnabled types.Bool               `tfsdk:"deployable_objects_confirmation_code_enabled"`
	ScopeableObjectsAlertEnabled             types.Bool               `tfsdk:"scopeable_objects_alert_enabled"`
	ScopeableObjectsConfirmationCodeEnabled  types.Bool               `tfsdk:"scopeable_objects_confirmation_code_enabled"`
	Timeouts                                 datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// impactAlertNotificationSettingsIdentityModel represents the identity object used on import.
type impactAlertNotificationSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
