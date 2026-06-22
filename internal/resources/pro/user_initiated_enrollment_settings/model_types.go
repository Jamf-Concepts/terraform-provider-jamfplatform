// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_initiated_enrollment_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UserInitiatedEnrollmentSettingsResourceModel is the Terraform model for
// jamfplatform_pro_user_initiated_enrollment_settings.
//
// The resource is a singleton mapping the Jamf Pro User-Initiated Enrollment
// settings page (General, Computers and Devices tabs) plus the nested
// collection of directory-service Access Groups. It also owns two embedded
// signing certificate sub-blocks managed inline with the settings write.
//
// The resource owns the General/Computers/Devices fields. The Re-enrollment
// page's "clear …" toggles share the same backing record but are managed by
// the jamfplatform_pro_re_enrollment_settings resource; this resource
// round-trips those values unchanged and never models them.
type UserInitiatedEnrollmentSettingsResourceModel struct {
	ID types.String `tfsdk:"id"`

	// General tab.
	SkipCertificateInstallation types.Bool `tfsdk:"skip_certificate_installation"`
	RestrictReenrollment        types.Bool `tfsdk:"restrict_reenrollment"`
	SigningMdmProfileEnabled    types.Bool `tfsdk:"signing_mdm_profile_enabled"`

	// Computers tab.
	EnableComputerEnrollment         types.Bool   `tfsdk:"enable_computer_enrollment"`
	CreateManagementAccount          types.Bool   `tfsdk:"create_management_account"`
	ManagementUsername               types.String `tfsdk:"management_username"`
	HideManagementAccount            types.Bool   `tfsdk:"hide_management_account"`
	AllowSshOnlyManagementAccount    types.Bool   `tfsdk:"allow_ssh_only_management_account"`
	EnsureSshRunning                 types.Bool   `tfsdk:"ensure_ssh_running"`
	LaunchSelfService                types.Bool   `tfsdk:"launch_self_service"`
	SignQuickaddPackage              types.Bool   `tfsdk:"sign_quickadd_package"`
	AccountDrivenDeviceEnrollmentMac types.Bool   `tfsdk:"account_driven_device_enrollment_macos"`

	// Devices tab.
	ProfileDrivenEnrollmentViaURLInstitutional types.Bool `tfsdk:"profile_driven_enrollment_via_url_institutional"`
	ProfileDrivenEnrollmentViaURLPersonal      types.Bool `tfsdk:"profile_driven_enrollment_via_url_personal"`
	AccountDrivenUserEnrollment                types.Bool `tfsdk:"account_driven_user_enrollment"`
	AccountDrivenUserEnrollmentVisionos        types.Bool `tfsdk:"account_driven_user_enrollment_visionos"`
	MergeManagedAppleAccountUsernames          types.Bool `tfsdk:"merge_managed_apple_account_usernames"`
	AccountDrivenDeviceEnrollmentIos           types.Bool `tfsdk:"account_driven_device_enrollment_ios"`
	AccountDrivenDeviceEnrollmentVisionos      types.Bool `tfsdk:"account_driven_device_enrollment_visionos"`

	// Deprecated — server ignores input and always returns USERENROLLMENT.
	PersonalDeviceEnrollmentType types.String `tfsdk:"personal_device_enrollment_type"`

	// Embedded cert sub-blocks.
	MdmSigningCertificate *certificateModel `tfsdk:"mdm_signing_certificate"`
	DeveloperCertificate  *certificateModel `tfsdk:"developer_certificate"`

	// Nested Access-Group collection. Optional+Computed: null when the user did
	// not author the collection (round-tripped, not managed); a known set when
	// the user manages the groups.
	AccessGroups types.Set `tfsdk:"access_group"`

	// Nested per-language enrollment-messaging collection (UI: Messaging tab),
	// keyed by ISO 639-1 language code. A map (not a set) so Terraform correlates
	// elements by the always-known language-code key rather than by value — sets
	// and lists mis-correlate when a collection of objects with unknown Computed
	// leaves changes membership. Optional+Computed with the same
	// managed-vs-unmanaged semantics as AccessGroups: null when the user does not
	// author the collection.
	MessagingLanguages types.Map `tfsdk:"messaging_languages"`

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// UserInitiatedEnrollmentSettingsDataSourceModel mirrors the resource model
// with every attribute Computed and the WriteOnly cert inputs dropped.
type UserInitiatedEnrollmentSettingsDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	SkipCertificateInstallation types.Bool `tfsdk:"skip_certificate_installation"`
	RestrictReenrollment        types.Bool `tfsdk:"restrict_reenrollment"`
	SigningMdmProfileEnabled    types.Bool `tfsdk:"signing_mdm_profile_enabled"`

	EnableComputerEnrollment         types.Bool   `tfsdk:"enable_computer_enrollment"`
	CreateManagementAccount          types.Bool   `tfsdk:"create_management_account"`
	ManagementUsername               types.String `tfsdk:"management_username"`
	HideManagementAccount            types.Bool   `tfsdk:"hide_management_account"`
	AllowSshOnlyManagementAccount    types.Bool   `tfsdk:"allow_ssh_only_management_account"`
	EnsureSshRunning                 types.Bool   `tfsdk:"ensure_ssh_running"`
	LaunchSelfService                types.Bool   `tfsdk:"launch_self_service"`
	SignQuickaddPackage              types.Bool   `tfsdk:"sign_quickadd_package"`
	AccountDrivenDeviceEnrollmentMac types.Bool   `tfsdk:"account_driven_device_enrollment_macos"`

	ProfileDrivenEnrollmentViaURLInstitutional types.Bool `tfsdk:"profile_driven_enrollment_via_url_institutional"`
	ProfileDrivenEnrollmentViaURLPersonal      types.Bool `tfsdk:"profile_driven_enrollment_via_url_personal"`
	AccountDrivenUserEnrollment                types.Bool `tfsdk:"account_driven_user_enrollment"`
	AccountDrivenUserEnrollmentVisionos        types.Bool `tfsdk:"account_driven_user_enrollment_visionos"`
	MergeManagedAppleAccountUsernames          types.Bool `tfsdk:"merge_managed_apple_account_usernames"`
	AccountDrivenDeviceEnrollmentIos           types.Bool `tfsdk:"account_driven_device_enrollment_ios"`
	AccountDrivenDeviceEnrollmentVisionos      types.Bool `tfsdk:"account_driven_device_enrollment_visionos"`

	PersonalDeviceEnrollmentType types.String `tfsdk:"personal_device_enrollment_type"`

	MdmSigningCertificate *certificateReadOnlyModel `tfsdk:"mdm_signing_certificate"`
	DeveloperCertificate  *certificateReadOnlyModel `tfsdk:"developer_certificate"`

	AccessGroups       types.Set `tfsdk:"access_group"`
	MessagingLanguages types.Map `tfsdk:"messaging_languages"`

	Timeouts datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// certificateModel maps an embedded *CertificateIdentityV2 sub-block.
//
// keystore_file and keystore_password are WriteOnly — sent to Jamf Pro on
// writes but never persisted in Terraform state. The server returns the cert
// identity object as null on GET, so keystore_file_name is preserved from
// prior state on readback rather than echoed. subject and serial_number come
// from the matching *Details object and only populate while the toggle is
// enabled.
type certificateModel struct {
	KeystoreFile     types.String `tfsdk:"keystore_file"`
	KeystoreFileName types.String `tfsdk:"keystore_file_name"`

	KeystorePassword          types.String `tfsdk:"keystore_password"`
	KeystorePasswordWoVersion types.Int64  `tfsdk:"keystore_password_wo_version"`

	// Computed details.
	Subject      types.String `tfsdk:"subject"`
	SerialNumber types.String `tfsdk:"serial_number"`
}

// certificateReadOnlyModel is the data-source projection: the cert sub-block
// minus the WriteOnly inputs and rotation companion.
type certificateReadOnlyModel struct {
	KeystoreFileName types.String `tfsdk:"keystore_file_name"`
	Subject          types.String `tfsdk:"subject"`
	SerialNumber     types.String `tfsdk:"serial_number"`
}

// accessGroupModel maps a pro.EnrollmentAccessGroupPreview item.
type accessGroupModel struct {
	ID                                 types.String `tfsdk:"id"`
	DirectoryServiceGroupID            types.String `tfsdk:"directory_service_group_id"`
	LdapServerID                       types.String `tfsdk:"ldap_server_id"`
	Name                               types.String `tfsdk:"name"`
	SiteID                             types.String `tfsdk:"site_id"`
	EnterpriseEnrollmentEnabled        types.Bool   `tfsdk:"enterprise_enrollment_enabled"`
	PersonalEnrollmentEnabled          types.Bool   `tfsdk:"personal_enrollment_enabled"`
	AccountDrivenUserEnrollmentEnabled types.Bool   `tfsdk:"account_driven_user_enrollment_enabled"`
	RequireEula                        types.Bool   `tfsdk:"require_eula"`
}

// messagingLanguageModel maps one per-language enrollment-messaging item
// (pro.EnrollmentProcessTextObject) from the Messaging tab. Attribute names
// mirror the Jamf Pro "Edit language" modal; the cryptic wire field each maps
// to is noted in a trailing comment. The 4 personal* wire fields
// (personalText/personalButton/personalProfileName/personalProfileDescription)
// have no control in the current UI and are intentionally not modelled — the
// read-merge write path preserves them on the wire rather than clearing them.
type messagingLanguageModel struct {
	// Identity. The language code is the MAP KEY, not a field here.
	Name      types.String `tfsdk:"name"`       // name (Computed, derived from code)
	PageTitle types.String `tfsdk:"page_title"` // title

	// Login.
	LoginPageText   types.String `tfsdk:"login_page_text"`   // loginDescription
	UsernameText    types.String `tfsdk:"username_text"`     // username
	PasswordText    types.String `tfsdk:"password_text"`     // password
	LoginButtonText types.String `tfsdk:"login_button_text"` // loginButton

	// Device ownership.
	DeviceOwnershipPageText                  types.String `tfsdk:"device_ownership_page_text"`                  // deviceClassDescription
	PersonalDeviceButtonName                 types.String `tfsdk:"personal_device_button_name"`                 // deviceClassPersonal
	InstitutionalOwnershipButtonName         types.String `tfsdk:"institutional_ownership_button_name"`         // deviceClassEnterprise
	PersonalDeviceManagementDescription      types.String `tfsdk:"personal_device_management_description"`      // deviceClassPersonalDescription
	InstitutionalDeviceManagementDescription types.String `tfsdk:"institutional_device_management_description"` // deviceClassEnterpriseDescription
	EnrollDeviceButtonName                   types.String `tfsdk:"enroll_device_button_name"`                   // deviceClassButton

	// End User License Agreement.
	PersonalEula         types.String `tfsdk:"personal_eula"`           // personalEula
	InstitutionalEula    types.String `tfsdk:"institutional_eula"`      // enterpriseEula
	EulaAcceptButtonText types.String `tfsdk:"eula_accept_button_text"` // eulaButton

	// Sites.
	SiteSelectionText types.String `tfsdk:"site_selection_text"` // siteDescription

	// Certificate.
	CaCertificateInstallationText  types.String `tfsdk:"ca_certificate_installation_text"`   // certificateText
	CaCertificateName              types.String `tfsdk:"ca_certificate_name"`                // certificateProfileName
	CaCertificateDescription       types.String `tfsdk:"ca_certificate_description"`         // certificateProfileDescription
	CaCertificateInstallButtonName types.String `tfsdk:"ca_certificate_install_button_name"` // certificateButton

	// Institutional MDM.
	InstitutionalMdmInstallationText   types.String `tfsdk:"institutional_mdm_installation_text"`   // enterpriseText
	InstitutionalMdmProfileName        types.String `tfsdk:"institutional_mdm_profile_name"`        // enterpriseProfileName
	InstitutionalMdmProfileDescription types.String `tfsdk:"institutional_mdm_profile_description"` // enterpriseProfileDescription
	InstitutionalMdmPendingText        types.String `tfsdk:"institutional_mdm_pending_text"`        // enterprisePending
	InstitutionalMdmInstallButtonName  types.String `tfsdk:"institutional_mdm_install_button_name"` // enterpriseButton

	// User Enrollment MDM.
	UserEnrollmentMdmInstallationText   types.String `tfsdk:"user_enrollment_mdm_installation_text"`   // userEnrollmentText
	UserEnrollmentMdmProfileName        types.String `tfsdk:"user_enrollment_mdm_profile_name"`        // userEnrollmentProfileName
	UserEnrollmentMdmProfileDescription types.String `tfsdk:"user_enrollment_mdm_profile_description"` // userEnrollmentProfileDescription
	UserEnrollmentMdmInstallButtonName  types.String `tfsdk:"user_enrollment_mdm_install_button_name"` // userEnrollmentButton

	// QuickAdd.
	QuickaddInstallationText  types.String `tfsdk:"quickadd_installation_text"`   // quickAddText
	QuickaddName              types.String `tfsdk:"quickadd_name"`                // quickAddName
	QuickaddProgressText      types.String `tfsdk:"quickadd_progress_text"`       // quickAddPending
	QuickaddInstallButtonName types.String `tfsdk:"quickadd_install_button_name"` // quickAddButton

	// Complete.
	EnrollmentCompleteText         types.String `tfsdk:"enrollment_complete_text"`           // completeMessage
	EnrollmentFailedText           types.String `tfsdk:"enrollment_failed_text"`             // failedMessage
	TryAgainButtonName             types.String `tfsdk:"try_again_button_name"`              // tryAgainButton
	ViewEnrollmentStatusButtonName types.String `tfsdk:"view_enrollment_status_button_name"` // checkNowButton
	ViewEnrollmentStatusText       types.String `tfsdk:"view_enrollment_status_text"`        // checkEnrollmentMessage
	LogOutButtonName               types.String `tfsdk:"log_out_button_name"`                // logoutButton
}

// userInitiatedEnrollmentSettingsIdentityModel is the identity object used on
// import.
type userInitiatedEnrollmentSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
