// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package smtp_server

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SmtpServerResourceModel is the Terraform resource model for the Jamf Pro SMTP
// Server settings singleton. The three credential blocks are typed-pointer
// SingleNestedAttributes (Optional-only, per STYLE_GUIDE §SingleNestedAttribute
// typed-pointer); exactly one is valid for the active authentication_type, and a
// plan-time ConfigValidator enforces the discriminator contract.
type SmtpServerResourceModel struct {
	ID                    types.String                    `tfsdk:"id"`
	Enabled               types.Bool                      `tfsdk:"enabled"`
	AuthenticationType    types.String                    `tfsdk:"authentication_type"`
	SenderSettings        *smtpSenderSettingsModel        `tfsdk:"sender_settings"`
	ConnectionSettings    *smtpConnectionSettingsModel    `tfsdk:"connection_settings"`
	BasicAuthCredentials  *smtpBasicCredentialsModel      `tfsdk:"basic_auth_credentials"`
	GraphAPICredentials   *smtpGraphAPICredentialsModel   `tfsdk:"graph_api_credentials"`
	GoogleMailCredentials *smtpGoogleMailCredentialsModel `tfsdk:"google_mail_credentials"`
	Timeouts              resourceTimeouts.Value          `tfsdk:"timeouts"`
}

// smtpSenderSettingsModel maps the required senderSettings block.
type smtpSenderSettingsModel struct {
	EmailAddress types.String `tfsdk:"email_address"`
	DisplayName  types.String `tfsdk:"display_name"`
}

// smtpConnectionSettingsModel maps the connectionSettings block (required for
// authentication_type NONE / BASIC; absent for GRAPH_API / GOOGLE_MAIL).
type smtpConnectionSettingsModel struct {
	Host              types.String `tfsdk:"host"`
	Port              types.Int64  `tfsdk:"port"`
	EncryptionType    types.String `tfsdk:"encryption_type"`
	ConnectionTimeout types.Int64  `tfsdk:"connection_timeout"`
}

// smtpBasicCredentialsModel maps basicAuthCredentials (authentication_type =
// BASIC). password is WriteOnly (never persisted in state, never returned by the
// server); password_wo_version is the rotation trigger.
type smtpBasicCredentialsModel struct {
	Username          types.String `tfsdk:"username"`
	Password          types.String `tfsdk:"password"`
	PasswordWoVersion types.Int64  `tfsdk:"password_wo_version"`
}

// smtpGraphAPICredentialsModel maps graphApiCredentials (authentication_type =
// GRAPH_API). client_secret is WriteOnly.
type smtpGraphAPICredentialsModel struct {
	ClientID              types.String `tfsdk:"client_id"`
	TenantID              types.String `tfsdk:"tenant_id"`
	ClientSecret          types.String `tfsdk:"client_secret"`
	ClientSecretWoVersion types.Int64  `tfsdk:"client_secret_wo_version"`
}

// smtpGoogleMailCredentialsModel maps googleMailCredentials (authentication_type
// = GOOGLE_MAIL). client_secret is WriteOnly. authentications is a read-only
// (Computed) list of server-managed OAuth grants — linking a Google sender
// account happens out of band in the Jamf Pro admin UI, not through Terraform.
type smtpGoogleMailCredentialsModel struct {
	ClientID              types.String `tfsdk:"client_id"`
	ClientSecret          types.String `tfsdk:"client_secret"`
	ClientSecretWoVersion types.Int64  `tfsdk:"client_secret_wo_version"`
	Authentications       types.List   `tfsdk:"authentications"`
}

// SmtpServerDataSourceModel is the Terraform data source model. Secrets and
// rotation triggers are absent (write-only, never readable); the data source
// surfaces only server-readable fields.
type SmtpServerDataSourceModel struct {
	ID                    types.String                      `tfsdk:"id"`
	Enabled               types.Bool                        `tfsdk:"enabled"`
	AuthenticationType    types.String                      `tfsdk:"authentication_type"`
	SenderSettings        *smtpSenderSettingsModel          `tfsdk:"sender_settings"`
	ConnectionSettings    *smtpConnectionSettingsModel      `tfsdk:"connection_settings"`
	BasicAuthCredentials  *smtpBasicCredentialsDSModel      `tfsdk:"basic_auth_credentials"`
	GraphAPICredentials   *smtpGraphAPICredentialsDSModel   `tfsdk:"graph_api_credentials"`
	GoogleMailCredentials *smtpGoogleMailCredentialsDSModel `tfsdk:"google_mail_credentials"`
	Timeouts              datasourceTimeouts.Value          `tfsdk:"timeouts"`
}

// smtpBasicCredentialsDSModel is the data-source view: only the non-secret
// username round-trips.
type smtpBasicCredentialsDSModel struct {
	Username types.String `tfsdk:"username"`
}

// smtpGraphAPICredentialsDSModel is the data-source view of graphApiCredentials.
type smtpGraphAPICredentialsDSModel struct {
	ClientID types.String `tfsdk:"client_id"`
	TenantID types.String `tfsdk:"tenant_id"`
}

// smtpGoogleMailCredentialsDSModel is the data-source view of
// googleMailCredentials.
type smtpGoogleMailCredentialsDSModel struct {
	ClientID        types.String `tfsdk:"client_id"`
	Authentications types.List   `tfsdk:"authentications"`
}

// smtpServerIdentityModel represents the identity object used on import.
type smtpServerIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
