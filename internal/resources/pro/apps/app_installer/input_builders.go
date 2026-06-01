// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildAppInstallerInput projects a plan model into an SDK
// *pro.AppInstallerDeploymentCreate suitable for both create and update. The two
// nested blocks are full-replace on the wire: a managed block emits every field;
// an omitted (nil) block is left nil so the payload drops it. update_behavior is
// server-required, so it is always set from the (known) plan value rather than
// routed through the Optional helper.
//
// appTitleID is the catalog title ID resolved from app_title_name by the caller
// (Create/Update) before the build; the plan's app_title_id is Computed and not
// populated until after the follow-up GET. selected_version is not emitted: the
// server controls the version (it coerces any submitted value to the latest
// available), so the field is Computed-only.
func buildAppInstallerInput(plan AppInstallerResourceModel, appTitleID string) *pro.AppInstallerDeploymentCreate {
	out := &pro.AppInstallerDeploymentCreate{
		Name:                            plan.Name.ValueString(),
		AppTitleID:                      appTitleID,
		DeploymentType:                  plan.DeploymentType.ValueString(),
		Enabled:                         optionalBoolPointer(plan.Enabled),
		CategoryID:                      helpers.OptionalStringPointer(plan.CategoryID),
		SiteID:                          helpers.OptionalStringPointer(plan.SiteID),
		SmartGroupID:                    helpers.OptionalStringPointer(plan.SmartGroupID),
		InstallPredefinedConfigProfiles: optionalBoolPointer(plan.InstallPredefinedConfigProfiles),
		TriggerAdminNotifications:       optionalBoolPointer(plan.TriggerAdminNotifications),
	}

	updateBehavior := plan.UpdateBehavior.ValueString()
	out.UpdateBehavior = &updateBehavior

	if plan.NotificationSettings != nil {
		out.NotificationSettings = buildNotificationSettings(plan.NotificationSettings)
	}

	if plan.SelfServiceSettings != nil {
		out.SelfServiceSettings = buildSelfServiceSettings(plan.SelfServiceSettings)
	}

	return out
}

// buildNotificationSettings maps the notification block. Each field is
// independently optional on the wire: an unset (null) field is OMITTED (nil),
// never sent as a zero value — the server rejects a blank message string or a
// non-positive interval/delay whenever the key is present, so zero-filling would
// 400. Fields the user did set are sent and replace the server value.
func buildNotificationSettings(m *NotificationSettingsModel) *pro.AppInstallerNotificationSettings {
	return &pro.AppInstallerNotificationSettings{
		NotificationMessage:  helpers.OptionalStringPointer(m.NotificationMessage),
		NotificationInterval: optionalIntPointer(m.NotificationInterval),
		DeadlineMessage:      helpers.OptionalStringPointer(m.DeadlineMessage),
		Deadline:             optionalIntPointer(m.Deadline),
		QuitDelay:            optionalIntPointer(m.QuitDelay),
		CompleteMessage:      helpers.OptionalStringPointer(m.CompleteMessage),
		Relaunch:             optionalBoolPointer(m.Relaunch),
		Suppress:             optionalBoolPointer(m.Suppress),
	}
}

// buildSelfServiceSettings maps the Self Service block. description is optional
// (omitted when unset, like the notification strings). The booleans are
// server-defaulted to false, so they are always emitted. categories is always
// emitted as a slice — empty (not nil) when there are none — so clearing
// categories is deterministic (the block is full-replace).
func buildSelfServiceSettings(m *SelfServiceSettingsModel) *pro.AppInstallerSelfServiceSettings {
	cats := make([]pro.SelfServiceCategory, 0, len(m.Categories))
	for _, c := range m.Categories {
		id := c.CategoryID.ValueString()
		cats = append(cats, pro.SelfServiceCategory{
			ID:       &id,
			Featured: boolPointerOrFalse(c.Featured),
		})
	}
	return &pro.AppInstallerSelfServiceSettings{
		Description:                 helpers.OptionalStringPointer(m.Description),
		ForceViewDescription:        boolPointerOrFalse(m.ForceViewDescription),
		IncludeInComplianceCategory: boolPointerOrFalse(m.IncludeInComplianceCategory),
		IncludeInFeaturedCategory:   boolPointerOrFalse(m.IncludeInFeaturedCategory),
		Categories:                  &cats,
	}
}
