// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package adcs

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AdcsResourceModel is the Terraform resource model for the Jamf Pro AD CS
// (Active Directory Certificate Services) PKI integration.
//
// The two certificate input blocks (ServerCertificate, ClientCertificate) are
// typed-pointer SingleNestedAttributes (Optional-only). Their scalar children
// data_wo / password_wo are WriteOnly (the framework strips them from plan and
// state and the server never returns them); wo_version is the per-cert rotation
// trigger. They are read from req.Config in the CRUD handlers and never assigned
// back into the model.
//
// The two *_details blocks are Computed-only and modelled as types.Object (NOT a
// typed-pointer struct): a Computed nested object has an Unknown value at plan
// time on Create, and decoding an Unknown object into a *struct errors at apply
// (see [[feedback_computed_nested_collection_typeslist]]). They are populated
// from the GET-after-write via types.ObjectValueFrom and set to ObjectNull when
// the server returns no certificate (OUTBOUND mode, or an absent block).
type AdcsResourceModel struct {
	ID                       types.String           `tfsdk:"id"`
	ConnectorMode            types.String           `tfsdk:"connector_mode"`
	DisplayName              types.String           `tfsdk:"display_name"`
	CaName                   types.String           `tfsdk:"ca_name"`
	Fqdn                     types.String           `tfsdk:"fqdn"`
	RevocationEnabled        types.Bool             `tfsdk:"revocation_enabled"`
	AdcsURL                  types.String           `tfsdk:"adcs_url"`
	APIClientID              types.String           `tfsdk:"api_client_id"`
	ServerCertificate        *adcsCertInputModel    `tfsdk:"server_certificate"`
	ClientCertificate        *adcsClientCertInput   `tfsdk:"client_certificate"`
	ServerCertificateDetails types.Object           `tfsdk:"server_certificate_details"`
	ClientCertificateDetails types.Object           `tfsdk:"client_certificate_details"`
	ConnectorLastCheckIn     types.String           `tfsdk:"connector_last_check_in"`
	Timeouts                 resourceTimeouts.Value `tfsdk:"timeouts"`
}

// adcsCertInputModel maps the server_certificate input block (.pem/.cer — public,
// password-less). data_wo is the base64-encoded certificate bytes (WriteOnly);
// filename is Optional+Computed (server echoes it on GET); wo_version is the
// rotation trigger (bump to re-send the certificate on the next apply).
type adcsCertInputModel struct {
	DataWo    types.String `tfsdk:"data_wo"`
	Filename  types.String `tfsdk:"filename"`
	WoVersion types.Int64  `tfsdk:"wo_version"`
}

// adcsClientCertInput maps the client_certificate input block (.pfx/.p12 —
// confidential, password-protected). data_wo + password_wo are WriteOnly;
// filename is Optional+Computed; wo_version is the rotation trigger.
type adcsClientCertInput struct {
	DataWo     types.String `tfsdk:"data_wo"`
	PasswordWo types.String `tfsdk:"password_wo"`
	Filename   types.String `tfsdk:"filename"`
	WoVersion  types.Int64  `tfsdk:"wo_version"`
}

// AdcsDataSourceModel is the Terraform data source model. The WriteOnly input
// blocks are absent (never readable); only server-readable scalars, the two
// *_details blocks, the connector mode, and the connector last-check-in are
// surfaced. The *_details blocks are typed-pointer here — a data source has no
// plan/apply Unknown cycle, so the Computed-object trap does not apply.
type AdcsDataSourceModel struct {
	ID                       types.String             `tfsdk:"id"`
	ConnectorMode            types.String             `tfsdk:"connector_mode"`
	DisplayName              types.String             `tfsdk:"display_name"`
	CaName                   types.String             `tfsdk:"ca_name"`
	Fqdn                     types.String             `tfsdk:"fqdn"`
	RevocationEnabled        types.Bool               `tfsdk:"revocation_enabled"`
	AdcsURL                  types.String             `tfsdk:"adcs_url"`
	APIClientID              types.String             `tfsdk:"api_client_id"`
	ServerCertificateDetails *adcsCertDetailsModel    `tfsdk:"server_certificate_details"`
	ClientCertificateDetails *adcsCertDetailsModel    `tfsdk:"client_certificate_details"`
	ConnectorLastCheckIn     types.String             `tfsdk:"connector_last_check_in"`
	Timeouts                 datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// adcsCertDetailsModel is the data-source view of a certificate metadata block.
type adcsCertDetailsModel struct {
	Filename       types.String `tfsdk:"filename"`
	SerialNumber   types.String `tfsdk:"serial_number"`
	Subject        types.String `tfsdk:"subject"`
	Issuer         types.String `tfsdk:"issuer"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
}

// adcsIdentityModel represents the identity object used on import.
type adcsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
