// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SsoSettingsResourceModel is the Terraform model for jamfplatform_pro_sso_settings.
//
// The resource is a singleton wrapping /v3/sso (settings) plus an embedded
// signing_certificate sub-block that drives /v2/sso/cert. configurationType
// gates which of the SAML / OIDC sub-blocks the server consults; plan-time
// validators enforce the cross-field rules described in SSO_SPIKE.md §3.
type SsoSettingsResourceModel struct {
	ID types.String `tfsdk:"id"`

	SsoEnabled                                     types.Bool   `tfsdk:"sso_enabled"`
	SsoBypassAllowed                               types.Bool   `tfsdk:"sso_bypass_allowed"`
	SsoForEnrollmentEnabled                        types.Bool   `tfsdk:"sso_for_enrollment_enabled"`
	SsoForMacOsSelfServiceEnabled                  types.Bool   `tfsdk:"sso_for_macos_self_service_enabled"`
	EnrollmentSsoForAccountDrivenEnrollmentEnabled types.Bool   `tfsdk:"enrollment_sso_for_account_driven_enrollment_enabled"`
	GroupEnrollmentAccessEnabled                   types.Bool   `tfsdk:"group_enrollment_access_enabled"`
	GroupEnrollmentAccessName                      types.String `tfsdk:"group_enrollment_access_name"`

	ConfigurationType types.String `tfsdk:"configuration_type"`

	OidcSettings        *oidcSettingsModel        `tfsdk:"oidc_settings"`
	SamlSettings        *samlSettingsModel        `tfsdk:"saml_settings"`
	EnrollmentSsoConfig *enrollmentSsoConfigModel `tfsdk:"enrollment_sso_config"`
	SigningCertificate  *signingCertificateModel  `tfsdk:"signing_certificate"`

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// SsoSettingsDataSourceModel mirrors the resource model with every attribute Computed.
type SsoSettingsDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	SsoEnabled                                     types.Bool   `tfsdk:"sso_enabled"`
	SsoBypassAllowed                               types.Bool   `tfsdk:"sso_bypass_allowed"`
	SsoForEnrollmentEnabled                        types.Bool   `tfsdk:"sso_for_enrollment_enabled"`
	SsoForMacOsSelfServiceEnabled                  types.Bool   `tfsdk:"sso_for_macos_self_service_enabled"`
	EnrollmentSsoForAccountDrivenEnrollmentEnabled types.Bool   `tfsdk:"enrollment_sso_for_account_driven_enrollment_enabled"`
	GroupEnrollmentAccessEnabled                   types.Bool   `tfsdk:"group_enrollment_access_enabled"`
	GroupEnrollmentAccessName                      types.String `tfsdk:"group_enrollment_access_name"`

	ConfigurationType types.String `tfsdk:"configuration_type"`

	OidcSettings        *oidcSettingsModel               `tfsdk:"oidc_settings"`
	SamlSettings        *samlSettingsModel               `tfsdk:"saml_settings"`
	EnrollmentSsoConfig *enrollmentSsoConfigModel        `tfsdk:"enrollment_sso_config"`
	SigningCertificate  *signingCertificateReadOnlyModel `tfsdk:"signing_certificate"`

	Timeouts datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// oidcSettingsModel maps to pro.OidcSettings.
type oidcSettingsModel struct {
	UserMapping                   types.String `tfsdk:"user_mapping"`
	JamfIDAuthenticationEnabled   types.Bool   `tfsdk:"jamf_id_authentication_enabled"`
	UsernameAttributeClaimMapping types.String `tfsdk:"username_attribute_claim_mapping"`
}

// samlSettingsModel maps to pro.SamlSettings.
//
// FederationMetadataFile carries the raw base64 string the user passes
// (typically via HCL filebase64()). Probe A21 confirmed Jamf preserves the
// bytes verbatim across PUT→GET, so no hash convergence is needed.
type samlSettingsModel struct {
	IdpProviderType         types.String `tfsdk:"idp_provider_type"`
	OtherProviderTypeName   types.String `tfsdk:"other_provider_type_name"`
	EntityID                types.String `tfsdk:"entity_id"`
	MetadataSource          types.String `tfsdk:"metadata_source"`
	IdpURL                  types.String `tfsdk:"idp_url"`
	FederationMetadataFile  types.String `tfsdk:"federation_metadata_file"`
	MetadataFileName        types.String `tfsdk:"metadata_file_name"`
	SessionTimeout          types.Int64  `tfsdk:"session_timeout"`
	TokenExpirationDisabled types.Bool   `tfsdk:"token_expiration_disabled"`
	UserMapping             types.String `tfsdk:"user_mapping"`
	UserAttributeEnabled    types.Bool   `tfsdk:"user_attribute_enabled"`
	UserAttributeName       types.String `tfsdk:"user_attribute_name"`
	GroupAttributeName      types.String `tfsdk:"group_attribute_name"`
	GroupRdnKey             types.String `tfsdk:"group_rdn_key"`
}

// enrollmentSsoConfigModel maps to pro.EnrollmentSsoConfig.
type enrollmentSsoConfigModel struct {
	Hosts          types.Set    `tfsdk:"hosts"`
	ManagementHint types.String `tfsdk:"management_hint"`
}

// signingCertificateModel drives /v2/sso/cert.
//
// keystore_password and password are both WriteOnly. The server returns
// neither on GET (probe E3 — keys absent, not redacted sentinels). Each has
// its own _wo_version rotation companion: bumping the integer re-sends the
// current value on the next Update PUT.
type signingCertificateModel struct {
	SetupType        types.String `tfsdk:"setup_type"`
	Type             types.String `tfsdk:"type"`
	Key              types.String `tfsdk:"key"`
	KeystoreFile     types.String `tfsdk:"keystore_file"`
	KeystoreFileName types.String `tfsdk:"keystore_file_name"`

	KeystorePassword          types.String `tfsdk:"keystore_password"`
	KeystorePasswordWoVersion types.Int64  `tfsdk:"keystore_password_wo_version"`
	Password                  types.String `tfsdk:"password"`
	PasswordWoVersion         types.Int64  `tfsdk:"password_wo_version"`

	// Computed echo / details.
	SerialNumber types.String `tfsdk:"serial_number"`
	Subject      types.String `tfsdk:"subject"`
	Issuer       types.String `tfsdk:"issuer"`
	Expiration   types.String `tfsdk:"expiration"`
	Keys         types.List   `tfsdk:"keys"`
}

// signingCertificateReadOnlyModel is the DS-side projection: same fields as
// the resource minus the WriteOnly inputs + rotation companions.
type signingCertificateReadOnlyModel struct {
	SetupType        types.String `tfsdk:"setup_type"`
	Type             types.String `tfsdk:"type"`
	Key              types.String `tfsdk:"key"`
	KeystoreFileName types.String `tfsdk:"keystore_file_name"`

	SerialNumber types.String `tfsdk:"serial_number"`
	Subject      types.String `tfsdk:"subject"`
	Issuer       types.String `tfsdk:"issuer"`
	Expiration   types.String `tfsdk:"expiration"`
	Keys         types.List   `tfsdk:"keys"`
}

// ssoSettingsIdentityModel is the identity object used on import.
type ssoSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
