// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package digicert

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// assignDigicertServerFields populates the server-derived fields of a resource
// model from a DigiCertSettingResponse. It never touches the client_certificate
// INPUT block — those carry WriteOnly bytes/password and a config-only wo_version
// the CRUD caller owns. The Computed client_certificate_details object is built
// from the response's certificate metadata.
func assignDigicertServerFields(state *DigicertResourceModel, resp *pro.DigiCertSettingResponse) diag.Diagnostics {
	state.ID = types.StringValue(resp.ID)
	state.DisplayName = types.StringValue(resp.CaName)
	state.HostName = types.StringValue(resp.Fqdn)
	state.RevocationEnabled = types.BoolValue(resp.RevocationEnabled)

	details, diags := clientCertificateDetailsObject(resp.ClientCert)
	if diags.HasError() {
		return diags
	}
	state.ClientCertificateDetails = details
	return diags
}

// assignDigicertDataSourceModel populates a data source model from a
// DigiCertSettingResponse.
func assignDigicertDataSourceModel(state *DigicertDataSourceModel, resp *pro.DigiCertSettingResponse) diag.Diagnostics {
	state.ID = types.StringValue(resp.ID)
	state.DisplayName = types.StringValue(resp.CaName)
	state.HostName = types.StringValue(resp.Fqdn)
	state.RevocationEnabled = types.BoolValue(resp.RevocationEnabled)

	details, diags := clientCertificateDetailsObject(resp.ClientCert)
	if diags.HasError() {
		return diags
	}
	state.ClientCertificateDetails = details
	return diags
}

// clientCertificateDetailsObject builds the Computed client_certificate_details
// types.Object from the response certificate metadata. Returns a null object when
// no certificate is stored.
func clientCertificateDetailsObject(cert *pro.CertificateResponse) (types.Object, diag.Diagnostics) {
	if cert == nil {
		return types.ObjectNull(clientCertificateDetailsAttrTypes), nil
	}
	attrs := map[string]attr.Value{
		"filename":        types.StringValue(cert.Filename),
		"serial_number":   types.StringValue(cert.SerialNumber),
		"subject":         types.StringValue(cert.Subject),
		"issuer":          types.StringValue(cert.Issuer),
		"expiration_date": expirationDate(cert.ExpirationDate),
	}
	return types.ObjectValue(clientCertificateDetailsAttrTypes, attrs)
}

// expirationDate maps the response *string (offset-less wire datetime, e.g.
// "2036-06-06T17:42:41") to a Computed string, null when absent. Round-tripped
// verbatim — Jamf Pro returns no timezone, so no parsing is attempted.
func expirationDate(t *string) types.String {
	if t == nil {
		return types.StringNull()
	}
	return types.StringValue(*t)
}
