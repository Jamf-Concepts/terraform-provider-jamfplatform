// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_adcs

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// assignAdcsResourceModel populates a resource model from an SDK GET response.
//
// The WriteOnly certificate bytes/password are never touched (the framework
// excludes WriteOnly values from state and the server never returns them). The
// input certificate blocks' `filename` is refreshed in place from the server
// echo when the block is present, leaving `data_wo` / `wo_version` as the plan
// values so the rotation gate keeps working. The two *_details blocks are
// rebuilt as types.Object from the server metadata; connector_mode is derived
// from the wire `outbound` flag.
func assignAdcsResourceModel(ctx context.Context, state *AdcsResourceModel, s *pro.AdcsSettingsResponse) diag.Diagnostics {
	var diags diag.Diagnostics

	state.ConnectorMode = types.StringValue(outboundToConnectorMode(s.Outbound))
	state.DisplayName = types.StringValue(s.DisplayName)
	state.CaName = types.StringValue(s.CaName)
	state.Fqdn = types.StringValue(s.Fqdn)
	state.RevocationEnabled = types.BoolValue(s.RevocationEnabled)
	state.AdcsURL = types.StringValue(s.AdcsURL)
	state.APIClientID = optionalString(s.ApiClientID)
	state.ConnectorLastCheckIn = adcsTimestamp(s.ConnectorLastCheckInTimestamp)

	// Refresh the input blocks' server-echoed filename in place (never rebuild —
	// that would drop data_wo / wo_version and break the rotation gate).
	if state.ServerCertificate != nil && s.ServerCert != nil {
		state.ServerCertificate.Filename = types.StringValue(s.ServerCert.Filename)
	}
	if state.ClientCertificate != nil && s.ClientCert != nil {
		state.ClientCertificate.Filename = types.StringValue(s.ClientCert.Filename)
	}

	serverDetails, d := certDetailsObject(ctx, s.ServerCert)
	diags.Append(d...)
	state.ServerCertificateDetails = serverDetails

	clientDetails, d := certDetailsObject(ctx, s.ClientCert)
	diags.Append(d...)
	state.ClientCertificateDetails = clientDetails

	return diags
}

// assignAdcsDataSourceModel populates a data source model from an SDK GET
// response. The *_details blocks are typed-pointer (no plan/apply Unknown cycle
// in a data source).
func assignAdcsDataSourceModel(state *AdcsDataSourceModel, s *pro.AdcsSettingsResponse) {
	state.ConnectorMode = types.StringValue(outboundToConnectorMode(s.Outbound))
	state.DisplayName = types.StringValue(s.DisplayName)
	state.CaName = types.StringValue(s.CaName)
	state.Fqdn = types.StringValue(s.Fqdn)
	state.RevocationEnabled = types.BoolValue(s.RevocationEnabled)
	state.AdcsURL = types.StringValue(s.AdcsURL)
	state.APIClientID = optionalString(s.ApiClientID)
	state.ConnectorLastCheckIn = adcsTimestamp(s.ConnectorLastCheckInTimestamp)
	state.ServerCertificateDetails = certDetailsModel(s.ServerCert)
	state.ClientCertificateDetails = certDetailsModel(s.ClientCert)
}

// certDetailsObject builds a Computed *_details types.Object from a certificate
// metadata response, or ObjectNull when the server returned no certificate.
func certDetailsObject(ctx context.Context, c *pro.AdcsCertificateResponse) (types.Object, diag.Diagnostics) {
	if c == nil {
		return types.ObjectNull(adcsCertDetailsAttributeTypes), nil
	}
	return types.ObjectValueFrom(ctx, adcsCertDetailsAttributeTypes, certDetailsModel(c))
}

// certDetailsModel maps a certificate metadata response into the details model.
func certDetailsModel(c *pro.AdcsCertificateResponse) *adcsCertDetailsModel {
	if c == nil {
		return nil
	}
	return &adcsCertDetailsModel{
		Filename:       types.StringValue(c.Filename),
		SerialNumber:   types.StringValue(c.SerialNumber),
		Subject:        types.StringValue(c.Subject),
		Issuer:         types.StringValue(c.Issuer),
		ExpirationDate: adcsCertExpiration(c),
	}
}

// adcsCertExpiration maps a certificate's expiry into a Computed types.String.
// AdcsCertificateResponse.ExpirationDate is a *string holding Jamf Pro's
// offset-less wire datetime (e.g. "2036-06-06T17:42:41") — round-tripped
// verbatim, no parsing (the SDK fix changed it from *time.Time, which failed
// RFC3339 parse on the zone-less value). The connector check-in timestamp
// (adcsTimestamp) remains a genuine offset-bearing *time.Time.
func adcsCertExpiration(c *pro.AdcsCertificateResponse) types.String {
	if c == nil || c.ExpirationDate == nil {
		return types.StringNull()
	}
	return types.StringValue(*c.ExpirationDate)
}

// adcsTimestamp maps a *time.Time into a Computed RFC3339 types.String, Null when
// the server omitted it (e.g. the connector has never checked in).
func adcsTimestamp(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}
	return types.StringValue(t.Format(time.RFC3339))
}

// optionalString maps a possibly-empty server string into a Computed types.String
// — Null when empty (e.g. apiClientId is "" on an INBOUND record), else the value.
func optionalString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
