// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package gsx_connection

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GsxConnectionSettingsResourceModel represents the Terraform resource model for the Jamf
// Pro GSX Connection settings singleton.
//
// The three secrets (TokenWo, KeystoreBytesWo, KeystorePasswordWo) are WriteOnly: the
// framework strips them from plan and state, so they are read from req.Config in the CRUD
// handlers and never assigned back into the model. They have no rotation companion
// (_wo_version): the GSX PUT mandates token + keystore on every write (wire-probed
// 2026-06-09 — FIELD_REQUIRED), so the "omit when unchanged" path can never fire. They are
// Required + WriteOnly; "rotation" is simply changing the config value.
type GsxConnectionSettingsResourceModel struct {
	ID                      types.String           `tfsdk:"id"`
	Enabled                 types.Bool             `tfsdk:"enabled"`
	Username                types.String           `tfsdk:"username"`
	ServiceAccountNumber    types.String           `tfsdk:"service_account_number"`
	ShipToNumber            types.String           `tfsdk:"ship_to_number"`
	TokenWo                 types.String           `tfsdk:"token_wo"`
	KeystoreBytesWo         types.String           `tfsdk:"keystore_bytes_wo"`
	KeystorePasswordWo      types.String           `tfsdk:"keystore_password_wo"`
	KeystoreName            types.String           `tfsdk:"keystore_name"`
	KeystoreErrorMessage    types.String           `tfsdk:"keystore_error_message"`
	KeystoreExpirationEpoch types.Int64            `tfsdk:"keystore_expiration_epoch"`
	Timeouts                resourceTimeouts.Value `tfsdk:"timeouts"`
}

// GsxConnectionSettingsDataSourceModel represents the Terraform data source model. It
// exposes only the non-secret fields — the GSX API never returns the token or keystore
// bytes/password, so a data source cannot surface them.
type GsxConnectionSettingsDataSourceModel struct {
	ID                      types.String             `tfsdk:"id"`
	Enabled                 types.Bool               `tfsdk:"enabled"`
	Username                types.String             `tfsdk:"username"`
	ServiceAccountNumber    types.String             `tfsdk:"service_account_number"`
	ShipToNumber            types.String             `tfsdk:"ship_to_number"`
	KeystoreName            types.String             `tfsdk:"keystore_name"`
	KeystoreErrorMessage    types.String             `tfsdk:"keystore_error_message"`
	KeystoreExpirationEpoch types.Int64              `tfsdk:"keystore_expiration_epoch"`
	Timeouts                datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// gsxConnectionSettingsIdentityModel represents the identity object used on import.
type gsxConnectionSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
