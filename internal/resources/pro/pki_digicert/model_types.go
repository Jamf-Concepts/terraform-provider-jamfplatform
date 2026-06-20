// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_digicert

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DigicertResourceModel represents the Terraform resource model for a DigiCert
// Trust Lifecycle Manager integration.
//
// The client certificate is modelled as two separate nested attributes:
//   - ClientCertificate is the user-authored INPUT block (Optional). It carries
//     the WriteOnly cert bytes/password (never persisted), the Optional+Computed
//     filename, and the WoVersion rotation companion. Because data_wo/password_wo
//     are WriteOnly the framework strips them from plan and state; the CRUD
//     handlers read them from req.Config.
//   - ClientCertificateDetails is the Computed server-echo metadata block. It is a
//     types.Object (not a Go struct) — Computed nested objects backed by a typed
//     struct trip an Unknown-at-plan conversion error that only surfaces under
//     acceptance apply.
type DigicertResourceModel struct {
	ID                       types.String             `tfsdk:"id"`
	DisplayName              types.String             `tfsdk:"display_name"`
	HostName                 types.String             `tfsdk:"host_name"`
	RevocationEnabled        types.Bool               `tfsdk:"revocation_enabled"`
	ClientCertificate        *DigicertClientCertModel `tfsdk:"client_certificate"`
	ClientCertificateDetails types.Object             `tfsdk:"client_certificate_details"`
	Timeouts                 resourceTimeouts.Value   `tfsdk:"timeouts"`
}

// DigicertClientCertModel is the user-authored input block for the client
// certificate. DataWo / PasswordWo are WriteOnly (config-only, never in state).
// WoVersion is the rotation trigger.
type DigicertClientCertModel struct {
	DataWo     types.String `tfsdk:"data_wo"`
	PasswordWo types.String `tfsdk:"password_wo"`
	Filename   types.String `tfsdk:"filename"`
	WoVersion  types.Int64  `tfsdk:"wo_version"`
}

// DigicertDataSourceModel represents the Terraform data source model. It exposes
// only the non-secret scalars and the Computed cert-metadata block — the DigiCert
// API never returns the certificate bytes or password.
type DigicertDataSourceModel struct {
	ID                       types.String             `tfsdk:"id"`
	DisplayName              types.String             `tfsdk:"display_name"`
	HostName                 types.String             `tfsdk:"host_name"`
	RevocationEnabled        types.Bool               `tfsdk:"revocation_enabled"`
	ClientCertificateDetails types.Object             `tfsdk:"client_certificate_details"`
	Timeouts                 datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// digicertIdentityModel represents the identity object for import.
type digicertIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
