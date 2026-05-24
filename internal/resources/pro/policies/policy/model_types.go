// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PolicyResourceModel is the Terraform resource model for a Jamf Pro classic
// policy. Mirrors proclassic.Policy field-for-field with the following
// adjustments:
//
//   - Scope target sub-blocks are flattened Set<String> of numeric IDs via the
//     internal/common/scope helper (see SCOPE_SPIKE.md §5).
//   - self_service.notification_enabled / notification_type are two TF
//     attributes that round-trip through a single proclassic.NotificationValue
//     (the wire emits two <notification> elements per policy).
//   - WriteOnly secrets are exposed as plain Optional+Sensitive attributes for
//     v1 — the wire still accepts them. The SHA-256 response twins
//     (password_sha256, managed_password_sha256, of_password_sha256) surface
//     as Computed-only strings. WriteOnly adoption is tracked as a follow-up.
//   - limit_to_users is intentionally omitted — SDK shape would emit a
//     repeating <user_groups> wrapper instead of a single wrapper around
//     repeated <user_group> children. Surfaced to maintainer for SDK fix.
type PolicyResourceModel struct {
	ID                   types.String                     `tfsdk:"id"`
	General              *PolicyGeneralModel              `tfsdk:"general"`
	Scope                *PolicyScopeModel                `tfsdk:"scope"`
	SelfService          *PolicySelfServiceModel          `tfsdk:"self_service"`
	PackageConfiguration *PolicyPackageConfigurationModel `tfsdk:"package_configuration"`
	Scripts              *PolicyScriptsModel              `tfsdk:"scripts"`
	Printers             *PolicyPrintersModel             `tfsdk:"printers"`
	DockItems            *PolicyDockItemsModel            `tfsdk:"dock_items"`
	AccountMaintenance   *PolicyAccountMaintenanceModel   `tfsdk:"account_maintenance"`
	Reboot               *PolicyRebootModel               `tfsdk:"reboot"`
	Maintenance          *PolicyMaintenanceModel          `tfsdk:"maintenance"`
	FilesProcesses       *PolicyFilesProcessesModel       `tfsdk:"files_processes"`
	UserInteraction      *PolicyUserInteractionModel      `tfsdk:"user_interaction"`
	DiskEncryption       *PolicyDiskEncryptionModel       `tfsdk:"disk_encryption"`
	Timeouts             resourceTimeouts.Value           `tfsdk:"timeouts"`
}

// PolicyGeneralModel models <policy><general>.
type PolicyGeneralModel struct {
	ID                         types.String                           `tfsdk:"id"`
	Name                       types.String                           `tfsdk:"name"`
	Enabled                    types.Bool                             `tfsdk:"enabled"`
	Trigger                    types.String                           `tfsdk:"trigger"`
	TriggerCheckin             types.Bool                             `tfsdk:"trigger_checkin"`
	TriggerEnrollmentComplete  types.Bool                             `tfsdk:"trigger_enrollment_complete"`
	TriggerLogin               types.Bool                             `tfsdk:"trigger_login"`
	TriggerLogout              types.Bool                             `tfsdk:"trigger_logout"`
	TriggerNetworkStateChanged types.Bool                             `tfsdk:"trigger_network_state_changed"`
	TriggerStartup             types.Bool                             `tfsdk:"trigger_startup"`
	TriggerOther               types.String                           `tfsdk:"trigger_other"`
	Frequency                  types.String                           `tfsdk:"frequency"`
	RetryEvent                 types.String                           `tfsdk:"retry_event"`
	RetryAttempts              types.Int64                            `tfsdk:"retry_attempts"`
	NotifyOnEachFailedRetry    types.Bool                             `tfsdk:"notify_on_each_failed_retry"`
	LocationUserOnly           types.Bool                             `tfsdk:"location_user_only"`
	TargetDrive                types.String                           `tfsdk:"target_drive"`
	Offline                    types.Bool                             `tfsdk:"offline"`
	NetworkRequirements        types.String                           `tfsdk:"network_requirements"`
	CategoryID                 types.String                           `tfsdk:"category_id"`
	CategoryName               types.String                           `tfsdk:"category_name"`
	SiteID                     types.String                           `tfsdk:"site_id"`
	SiteName                   types.String                           `tfsdk:"site_name"`
	DateTimeLimitations        *PolicyGeneralDateTimeLimitationsModel `tfsdk:"date_time_limitations"`
	NetworkLimitations         *PolicyGeneralNetworkLimitationsModel  `tfsdk:"network_limitations"`
	OverrideDefaultSettings    *PolicyGeneralOverrideDefaultsModel    `tfsdk:"override_default_settings"`
}

// PolicyGeneralDateTimeLimitationsModel models <general><date_time_limitations>.
type PolicyGeneralDateTimeLimitationsModel struct {
	ActivationDate types.String `tfsdk:"activation_date"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	NoExecuteOn    types.Set    `tfsdk:"no_execute_on"`
	NoExecuteStart types.String `tfsdk:"no_execute_start"`
	NoExecuteEnd   types.String `tfsdk:"no_execute_end"`
}

// PolicyGeneralNetworkLimitationsModel models <general><network_limitations>.
type PolicyGeneralNetworkLimitationsModel struct {
	MinimumNetworkConnection types.String `tfsdk:"minimum_network_connection"`
	AnyIPAddress             types.Bool   `tfsdk:"any_ip_address"`
	NetworkSegmentIDs        types.Set    `tfsdk:"network_segment_ids"`
}

// PolicyGeneralOverrideDefaultsModel models <general><override_default_settings>.
type PolicyGeneralOverrideDefaultsModel struct {
	TargetDrive       types.String `tfsdk:"target_drive"`
	DistributionPoint types.String `tfsdk:"distribution_point"`
	ForceAfpSmb       types.Bool   `tfsdk:"force_afp_smb"`
	Sus               types.String `tfsdk:"sus"`
}

// PolicyScopeModel models <policy><scope>. Every target category is a flat
// Set<String> of numeric Jamf Pro classic IDs (or names for the
// directory-service / limit-to flavours), composed via the
// internal/common/scope helper. See SCOPE_SPIKE.md §5 for the canonical rules.
type PolicyScopeModel struct {
	AllComputers     types.Bool                   `tfsdk:"all_computers"`
	AllJssUsers      types.Bool                   `tfsdk:"all_jss_users"`
	ComputerIDs      types.Set                    `tfsdk:"computer_ids"`
	ComputerGroupIDs types.Set                    `tfsdk:"computer_group_ids"`
	BuildingIDs      types.Set                    `tfsdk:"building_ids"`
	DepartmentIDs    types.Set                    `tfsdk:"department_ids"`
	JssUserIDs       types.Set                    `tfsdk:"jss_user_ids"`
	JssUserGroupIDs  types.Set                    `tfsdk:"jss_user_group_ids"`
	Limitations      *PolicyScopeLimitationsModel `tfsdk:"limitations"`
	Exclusions       *PolicyScopeExclusionsModel  `tfsdk:"exclusions"`
}

// PolicyScopeLimitationsModel models <scope><limitations>.
type PolicyScopeLimitationsModel struct {
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	IbeaconIDs                       types.Set `tfsdk:"ibeacon_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// PolicyScopeExclusionsModel models <scope><exclusions>.
type PolicyScopeExclusionsModel struct {
	ComputerIDs                      types.Set `tfsdk:"computer_ids"`
	ComputerGroupIDs                 types.Set `tfsdk:"computer_group_ids"`
	BuildingIDs                      types.Set `tfsdk:"building_ids"`
	DepartmentIDs                    types.Set `tfsdk:"department_ids"`
	JssUserIDs                       types.Set `tfsdk:"jss_user_ids"`
	JssUserGroupIDs                  types.Set `tfsdk:"jss_user_group_ids"`
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	IbeaconIDs                       types.Set `tfsdk:"ibeacon_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// PolicySelfServiceModel models <policy><self_service>. NotificationEnabled
// and NotificationType project into a single proclassic.NotificationValue
// — see helpers.go for the split/join logic.
type PolicySelfServiceModel struct {
	UseForSelfService           types.Bool                      `tfsdk:"use_for_self_service"`
	SelfServiceDisplayName      types.String                    `tfsdk:"self_service_display_name"`
	InstallButtonText           types.String                    `tfsdk:"install_button_text"`
	ReinstallButtonText         types.String                    `tfsdk:"reinstall_button_text"`
	SelfServiceDescription      types.String                    `tfsdk:"self_service_description"`
	ForceUsersToViewDescription types.Bool                      `tfsdk:"force_users_to_view_description"`
	FeatureOnMainPage           types.Bool                      `tfsdk:"feature_on_main_page"`
	NotificationEnabled         types.Bool                      `tfsdk:"notification_enabled"`
	NotificationType            types.String                    `tfsdk:"notification_type"`
	NotificationSubject         types.String                    `tfsdk:"notification_subject"`
	NotificationMessage         types.String                    `tfsdk:"notification_message"`
	SelfServiceIcon             *PolicySelfServiceIconModel     `tfsdk:"self_service_icon"`
	Category                    *PolicySelfServiceCategoryModel `tfsdk:"category"`
}

// PolicySelfServiceIconModel models <self_service><self_service_icon>.
type PolicySelfServiceIconModel struct {
	ID       types.String `tfsdk:"id"`
	URI      types.String `tfsdk:"uri"`
	Filename types.String `tfsdk:"filename"`
}

// PolicySelfServiceCategoryModel models <self_service><self_service_categories><category>.
// SDK collapses the wrapper into a single child — schema follows.
type PolicySelfServiceCategoryModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	DisplayIn types.Bool   `tfsdk:"display_in"`
	FeatureIn types.Bool   `tfsdk:"feature_in"`
}

// PolicyPackageConfigurationModel models <policy><package_configuration>.
type PolicyPackageConfigurationModel struct {
	DistributionPoint types.String             `tfsdk:"distribution_point"`
	Packages          []PolicyPackageItemModel `tfsdk:"packages"`
}

// PolicyPackageItemModel models a single <package>.
type PolicyPackageItemModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Action        types.String `tfsdk:"action"`
	Fut           types.Bool   `tfsdk:"fut"`
	Feu           types.Bool   `tfsdk:"feu"`
	UpdateAutorun types.Bool   `tfsdk:"update_autorun"`
}

// PolicyScriptsModel models <policy><scripts>.
type PolicyScriptsModel struct {
	Scripts []PolicyScriptItemModel `tfsdk:"scripts"`
}

// PolicyScriptItemModel models a single <script>.
type PolicyScriptItemModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Priority    types.String `tfsdk:"priority"`
	Parameter4  types.String `tfsdk:"parameter4"`
	Parameter5  types.String `tfsdk:"parameter5"`
	Parameter6  types.String `tfsdk:"parameter6"`
	Parameter7  types.String `tfsdk:"parameter7"`
	Parameter8  types.String `tfsdk:"parameter8"`
	Parameter9  types.String `tfsdk:"parameter9"`
	Parameter10 types.String `tfsdk:"parameter10"`
	Parameter11 types.String `tfsdk:"parameter11"`
}

// PolicyPrintersModel models <policy><printers>.
type PolicyPrintersModel struct {
	Size                 types.Int64              `tfsdk:"size"`
	LeaveExistingDefault types.Bool               `tfsdk:"leave_existing_default"`
	Printers             []PolicyPrinterItemModel `tfsdk:"printers"`
}

// PolicyPrinterItemModel models a single <printer>.
type PolicyPrinterItemModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Action      types.String `tfsdk:"action"`
	MakeDefault types.Bool   `tfsdk:"make_default"`
}

// PolicyDockItemsModel models <policy><dock_items>.
type PolicyDockItemsModel struct {
	DockItems []PolicyDockItemModel `tfsdk:"dock_items"`
}

// PolicyDockItemModel models a single <dock_item>.
type PolicyDockItemModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Action types.String `tfsdk:"action"`
}

// PolicyAccountMaintenanceModel models <policy><account_maintenance>.
type PolicyAccountMaintenanceModel struct {
	Accounts                []PolicyAccountItemModel            `tfsdk:"accounts"`
	DirectoryBindings       []PolicyDirectoryBindingItemModel   `tfsdk:"directory_bindings"`
	ManagementAccount       *PolicyManagementAccountModel       `tfsdk:"management_account"`
	OpenFirmwareEfiPassword *PolicyOpenFirmwareEfiPasswordModel `tfsdk:"open_firmware_efi_password"`
}

// PolicyAccountItemModel models a single <account>. password is Optional+
// Sensitive (no WriteOnly in v1; tracked as follow-up); password_sha256 is
// Computed-only.
type PolicyAccountItemModel struct {
	Action                 types.String `tfsdk:"action"`
	Username               types.String `tfsdk:"username"`
	Realname               types.String `tfsdk:"realname"`
	Password               types.String `tfsdk:"password"`
	PasswordSha256         types.String `tfsdk:"password_sha256"`
	ArchiveHomeDirectory   types.Bool   `tfsdk:"archive_home_directory"`
	ArchiveHomeDirectoryTo types.String `tfsdk:"archive_home_directory_to"`
	Home                   types.String `tfsdk:"home"`
	Hint                   types.String `tfsdk:"hint"`
	Picture                types.String `tfsdk:"picture"`
	Admin                  types.Bool   `tfsdk:"admin"`
	FilevaultEnabled       types.Bool   `tfsdk:"filevault_enabled"`
	SecureTokenAllowed     types.Bool   `tfsdk:"secure_token_allowed"`
}

// PolicyDirectoryBindingItemModel models a single <binding>.
type PolicyDirectoryBindingItemModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// PolicyManagementAccountModel models <management_account>.
type PolicyManagementAccountModel struct {
	Action                types.String `tfsdk:"action"`
	ManagedPassword       types.String `tfsdk:"managed_password"`
	ManagedPasswordLength types.Int64  `tfsdk:"managed_password_length"`
}

// PolicyOpenFirmwareEfiPasswordModel models <open_firmware_efi_password>.
type PolicyOpenFirmwareEfiPasswordModel struct {
	OfMode           types.String `tfsdk:"of_mode"`
	OfPassword       types.String `tfsdk:"of_password"`
	OfPasswordSha256 types.String `tfsdk:"of_password_sha256"`
}

// PolicyRebootModel models <policy><reboot>.
type PolicyRebootModel struct {
	Message                     types.String `tfsdk:"message"`
	StartupDisk                 types.String `tfsdk:"startup_disk"`
	SpecifyStartup              types.String `tfsdk:"specify_startup"`
	NoUserLoggedIn              types.String `tfsdk:"no_user_logged_in"`
	UserLoggedIn                types.String `tfsdk:"user_logged_in"`
	MinutesUntilReboot          types.Int64  `tfsdk:"minutes_until_reboot"`
	StartRebootTimerImmediately types.Bool   `tfsdk:"start_reboot_timer_immediately"`
	FileVault2Reboot            types.Bool   `tfsdk:"file_vault_2_reboot"`
}

// PolicyMaintenanceModel models <policy><maintenance>.
type PolicyMaintenanceModel struct {
	Recon                    types.Bool `tfsdk:"recon"`
	ResetName                types.Bool `tfsdk:"reset_name"`
	InstallAllCachedPackages types.Bool `tfsdk:"install_all_cached_packages"`
	Heal                     types.Bool `tfsdk:"heal"`
	Prebindings              types.Bool `tfsdk:"prebindings"`
	Permissions              types.Bool `tfsdk:"permissions"`
	Byhost                   types.Bool `tfsdk:"byhost"`
	SystemCache              types.Bool `tfsdk:"system_cache"`
	UserCache                types.Bool `tfsdk:"user_cache"`
	Verify                   types.Bool `tfsdk:"verify"`
}

// PolicyFilesProcessesModel models <policy><files_processes>.
type PolicyFilesProcessesModel struct {
	SearchByPath         types.String `tfsdk:"search_by_path"`
	DeleteFile           types.Bool   `tfsdk:"delete_file"`
	LocateFile           types.String `tfsdk:"locate_file"`
	UpdateLocateDatabase types.Bool   `tfsdk:"update_locate_database"`
	SpotlightSearch      types.String `tfsdk:"spotlight_search"`
	SearchForProcess     types.String `tfsdk:"search_for_process"`
	KillProcess          types.Bool   `tfsdk:"kill_process"`
	RunCommand           types.String `tfsdk:"run_command"`
}

// PolicyUserInteractionModel models <policy><user_interaction>.
type PolicyUserInteractionModel struct {
	MessageStart          types.String `tfsdk:"message_start"`
	AllowUsersToDefer     types.Bool   `tfsdk:"allow_users_to_defer"`
	AllowDeferralUntilUtc types.String `tfsdk:"allow_deferral_until_utc"`
	AllowDeferralMinutes  types.Int64  `tfsdk:"allow_deferral_minutes"`
	MessageFinish         types.String `tfsdk:"message_finish"`
}

// PolicyDiskEncryptionModel models <policy><disk_encryption>.
type PolicyDiskEncryptionModel struct {
	Action                                 types.String `tfsdk:"action"`
	DiskEncryptionConfigurationID          types.Int64  `tfsdk:"disk_encryption_configuration_id"`
	AuthRestart                            types.Bool   `tfsdk:"auth_restart"`
	RemediateKeyType                       types.String `tfsdk:"remediate_key_type"`
	RemediateDiskEncryptionConfigurationID types.Int64  `tfsdk:"remediate_disk_encryption_configuration_id"`
}

// policyIdentityModel is the identity object for policy resources and list results.
type policyIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
