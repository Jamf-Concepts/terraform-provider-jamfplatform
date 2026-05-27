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
//     internal/common/scope helper (see STYLE_GUIDE.md §Scope helper).
//   - self_service.display_notifications / notification_location are two TF
//     attributes that round-trip through a single proclassic.NotificationValue
//     (the wire emits two <notification> elements per policy).
//   - Plaintext secrets (accounts[].password, management_account.managed_password,
//     open_firmware_efi_password.of_password) are `WriteOnly` attributes: sent
//     on writes from req.Config, never persisted in state. Each carries a
//     `*_wo_version` Int64 Optional companion as rotation trigger. The
//     SHA-256 response twins (password_sha256, of_password_sha256) are no
//     longer surfaced — the classic API returns a literal 20-asterisk
//     redaction string that carries no drift-detection signal.
//   - scope.limit_to_users is intentionally NOT modelled. Probe 2026-05-24
//     confirmed the Jamf Pro server denormalises
//     `<limitations><user_groups>` into `<limit_to_users><user_groups>` on
//     every write (and vice versa on every read), so the two wire paths
//     carry identical values. deploymenttheory/terraform-provider-jamfpro
//     omits it for the same reason. SDK commit 748aa25 still fixed the
//     underlying wrapper-shape defect on the SDK type for any future
//     non-policy consumer.
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
	TriggerNetworkStateChanged types.Bool                             `tfsdk:"trigger_network_state_changed"`
	TriggerStartup             types.Bool                             `tfsdk:"trigger_startup"`
	TriggerOther               types.String                           `tfsdk:"trigger_other"`
	Frequency                  types.String                           `tfsdk:"frequency"`
	RetryEvent                 types.String                           `tfsdk:"retry_event"`
	RetryAttempts              types.Int64                            `tfsdk:"retry_attempts"`
	NotifyOnEachFailedRetry    types.Bool                             `tfsdk:"notify_on_each_failed_retry"`
	LimitToJamfProAssignedUser types.Bool                             `tfsdk:"limit_to_jamf_pro_assigned_user"`
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
// internal/common/scope helper. See STYLE_GUIDE.md §Scope helper for the canonical rules.
// User-side IDs use the UI-canonical names `user_ids` / `user_group_ids`; the
// wire elements are `<jss_users>` and `<jss_user_groups>`.
type PolicyScopeModel struct {
	AllComputers     types.Bool                   `tfsdk:"all_computers"`
	AllJssUsers      types.Bool                   `tfsdk:"all_jss_users"`
	ComputerIDs      types.Set                    `tfsdk:"computer_ids"`
	ComputerGroupIDs types.Set                    `tfsdk:"computer_group_ids"`
	BuildingIDs      types.Set                    `tfsdk:"building_ids"`
	DepartmentIDs    types.Set                    `tfsdk:"department_ids"`
	UserIDs          types.Set                    `tfsdk:"user_ids"`
	UserGroupIDs     types.Set                    `tfsdk:"user_group_ids"`
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
	UserIDs                          types.Set `tfsdk:"user_ids"`
	UserGroupIDs                     types.Set `tfsdk:"user_group_ids"`
	NetworkSegmentIDs                types.Set `tfsdk:"network_segment_ids"`
	IbeaconIDs                       types.Set `tfsdk:"ibeacon_ids"`
	DirectoryServiceOrLocalUserNames types.Set `tfsdk:"directory_service_or_local_user_names"`
	DirectoryServiceUserGroupNames   types.Set `tfsdk:"directory_service_user_group_names"`
}

// PolicySelfServiceModel models <policy><self_service>. DisplayNotifications
// and NotificationLocation project into a single proclassic.NotificationValue
// — see helpers.go for the split/join logic.
type PolicySelfServiceModel struct {
	UseForSelfService          types.Bool                      `tfsdk:"use_for_self_service"`
	SelfServiceDisplayName     types.String                    `tfsdk:"self_service_display_name"`
	InstallButtonText          types.String                    `tfsdk:"install_button_text"`
	ReinstallButtonText        types.String                    `tfsdk:"reinstall_button_text"`
	SelfServiceDescription     types.String                    `tfsdk:"self_service_description"`
	EnsureUsersViewDescription types.Bool                      `tfsdk:"ensure_users_view_description"`
	IncludeInFeaturedCategory  types.Bool                      `tfsdk:"include_in_featured_category"`
	DisplayNotifications       types.Bool                      `tfsdk:"display_notifications"`
	NotificationLocation       types.String                    `tfsdk:"notification_location"`
	NotificationSubject        types.String                    `tfsdk:"notification_subject"`
	NotificationMessage        types.String                    `tfsdk:"notification_message"`
	SelfServiceIcon            *PolicySelfServiceIconModel     `tfsdk:"self_service_icon"`
	Category                   *PolicySelfServiceCategoryModel `tfsdk:"category"`
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
	LeaveExistingDefault types.Bool               `tfsdk:"leave_existing_default"`
	Printers             []PolicyPrinterItemModel `tfsdk:"printers"`
}

// PolicyPrinterItemModel models a single <printer>. The Action field carries
// the UI-canonical value (`Map` / `Unmap`); the input/output builders
// translate to the wire `install` / `uninstall` form.
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

// PolicyAccountItemModel models a single <account>. `Password` is
// `WriteOnly` (never persisted in state); `PasswordWoVersion` is the
// rotation trigger companion.
type PolicyAccountItemModel struct {
	Action                         types.String `tfsdk:"action"`
	Username                       types.String `tfsdk:"username"`
	Realname                       types.String `tfsdk:"realname"`
	Password                       types.String `tfsdk:"password"`
	PasswordWoVersion              types.Int64  `tfsdk:"password_wo_version"`
	PermanentlyDeleteHomeDirectory types.Bool   `tfsdk:"permanently_delete_home_directory"`
	ArchiveHomeDirectoryTo         types.String `tfsdk:"archive_home_directory_to"`
	Home                           types.String `tfsdk:"home"`
	Hint                           types.String `tfsdk:"hint"`
	Picture                        types.String `tfsdk:"picture"`
	Admin                          types.Bool   `tfsdk:"admin"`
	FilevaultEnabled               types.Bool   `tfsdk:"filevault_enabled"`
	SecureTokenAllowed             types.Bool   `tfsdk:"secure_token_allowed"`
}

// PolicyDirectoryBindingItemModel models a single <binding>.
type PolicyDirectoryBindingItemModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// PolicyManagementAccountModel models <management_account>. `ManagedPassword`
// is `WriteOnly`; `ManagedPasswordWoVersion` is the rotation trigger.
type PolicyManagementAccountModel struct {
	Action                   types.String `tfsdk:"action"`
	ManagedPassword          types.String `tfsdk:"managed_password"`
	ManagedPasswordWoVersion types.Int64  `tfsdk:"managed_password_wo_version"`
	ManagedPasswordLength    types.Int64  `tfsdk:"managed_password_length"`
}

// PolicyOpenFirmwareEfiPasswordModel models <open_firmware_efi_password>.
// `OfPassword` is `WriteOnly`; `OfPasswordWoVersion` is the rotation trigger.
type PolicyOpenFirmwareEfiPasswordModel struct {
	OfMode              types.String `tfsdk:"of_mode"`
	OfPassword          types.String `tfsdk:"of_password"`
	OfPasswordWoVersion types.Int64  `tfsdk:"of_password_wo_version"`
}

// PolicyRebootModel models <policy><reboot>.
type PolicyRebootModel struct {
	Message                     types.String `tfsdk:"message"`
	StartupDisk                 types.String `tfsdk:"startup_disk"`
	SpecifyStartup              types.String `tfsdk:"specify_startup"`
	NoUserLoggedIn              types.String `tfsdk:"no_user_logged_in"`
	UserLoggedIn                types.String `tfsdk:"user_logged_in"`
	DelayMinutes                types.Int64  `tfsdk:"delay_minutes"`
	StartRebootTimerImmediately types.Bool   `tfsdk:"start_reboot_timer_immediately"`
	FileVault2Reboot            types.Bool   `tfsdk:"file_vault_2_reboot"`
}

// PolicyMaintenanceModel models <policy><maintenance>. Attribute names mirror
// the Jamf Pro admin UI checkbox labels; wire element names are noted below.
type PolicyMaintenanceModel struct {
	UpdateInventory       types.Bool `tfsdk:"update_inventory"`        // wire: <recon>
	ResetComputerNames    types.Bool `tfsdk:"reset_computer_names"`    // wire: <reset_name>
	InstallCachedPackages types.Bool `tfsdk:"install_cached_packages"` // wire: <install_all_cached_packages>
	FixDiskPermissions    types.Bool `tfsdk:"fix_disk_permissions"`    // wire: <permissions>
	FixByhostFiles        types.Bool `tfsdk:"fix_byhost_files"`        // wire: <byhost>
	FlushSystemCaches     types.Bool `tfsdk:"flush_system_caches"`     // wire: <system_cache>
	FlushUserCaches       types.Bool `tfsdk:"flush_user_caches"`       // wire: <user_cache>
	VerifyStartupDisk     types.Bool `tfsdk:"verify_startup_disk"`     // wire: <verify>
}

// PolicyFilesProcessesModel models <policy><files_processes>. Attribute names
// mirror the Jamf Pro admin UI labels; wire element names are noted below.
type PolicyFilesProcessesModel struct {
	SearchByPath         types.String `tfsdk:"search_by_path"`
	DeleteFileIfFound    types.Bool   `tfsdk:"delete_file_if_found"` // wire: <delete_file>
	SearchByFilename     types.String `tfsdk:"search_by_filename"`   // wire: <locate_file>
	UpdateLocateDatabase types.Bool   `tfsdk:"update_locate_database"`
	SearchBySpotlight    types.String `tfsdk:"search_by_spotlight"` // wire: <spotlight_search>
	SearchForProcess     types.String `tfsdk:"search_for_process"`
	KillProcessIfFound   types.Bool   `tfsdk:"kill_process_if_found"` // wire: <kill_process>
	ExecuteCommand       types.String `tfsdk:"execute_command"`       // wire: <run_command>
}

// PolicyUserInteractionModel models <policy><user_interaction>. Attribute names
// mirror the Jamf Pro admin UI labels on the "User Interaction" tab.
//
// The UI surfaces a single "Deferral Type" dropdown with three options
// (None / Date / Duration); the wire splits this across
// <allow_users_to_defer>, <allow_deferral_until_utc>, and
// <allow_deferral_minutes>. The provider hides that split:
//
//   - `deferral_type = "none"`     → wire: defer=false, until="", minutes=0
//   - `deferral_type = "date"`     → wire: defer=true,  until=<deferral_until_utc>, minutes=0
//   - `deferral_type = "duration"` → wire: defer=true,  until="", minutes=<deferral_days>*1440
//
// The wire `<allow_deferral_minutes>` granularity (multiples of 1440) and the
// raw `<allow_users_to_defer>` master switch are no longer exposed; the
// admin UI matches the same abstraction.
type PolicyUserInteractionModel struct {
	StartMessage     types.String `tfsdk:"start_message"`      // wire: <message_start>
	DeferralType     types.String `tfsdk:"deferral_type"`      // synthetic: none/date/duration
	DeferralUntilUtc types.String `tfsdk:"deferral_until_utc"` // wire: <allow_deferral_until_utc>
	DeferralDays     types.Int64  `tfsdk:"deferral_days"`      // wire: <allow_deferral_minutes>/1440
	CompleteMessage  types.String `tfsdk:"complete_message"`   // wire: <message_finish>
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
