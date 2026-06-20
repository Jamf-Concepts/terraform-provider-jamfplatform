// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_venafi

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PkiVenafiResourceModel represents the Terraform resource model for a Jamf Pro
// Venafi certificate authority (Settings → Global → PKI certificates).
//
// `RefreshTokenWo` is `WriteOnly`: the user-supplied secret is sent on writes
// (sourced from req.Config) but never persisted in Terraform state. The
// companion `RefreshTokenWoVersion` is the rotation trigger — bump it to
// re-send the token; leave it unchanged to preserve the stored value.
// `JamfPublicKeyRotation` triggers regeneration of the Jamf-minted public key.
type PkiVenafiResourceModel struct {
	ID                     types.String           `tfsdk:"id"`
	Name                   types.String           `tfsdk:"name"`
	ProxyAddress           types.String           `tfsdk:"proxy_address"`
	ClientID               types.String           `tfsdk:"client_id"`
	RevocationEnabled      types.Bool             `tfsdk:"revocation_enabled"`
	RefreshTokenWo         types.String           `tfsdk:"refresh_token_wo"`
	RefreshTokenWoVersion  types.Int64            `tfsdk:"refresh_token_wo_version"`
	RefreshTokenConfigured types.Bool             `tfsdk:"refresh_token_configured"`
	JamfPublicKey          types.String           `tfsdk:"jamf_public_key"`
	JamfPublicKeyRotation  types.Int64            `tfsdk:"jamf_public_key_rotation"`
	ProxyTrustStore        types.String           `tfsdk:"proxy_trust_store"`
	Timeouts               resourceTimeouts.Value `tfsdk:"timeouts"`
}

// PkiVenafiDataSourceModel represents the Terraform data source model for a Jamf
// Pro Venafi certificate authority. The refresh token is never exposed — the
// API never returns it on read.
type PkiVenafiDataSourceModel struct {
	ID                     types.String             `tfsdk:"id"`
	Name                   types.String             `tfsdk:"name"`
	ProxyAddress           types.String             `tfsdk:"proxy_address"`
	ClientID               types.String             `tfsdk:"client_id"`
	RevocationEnabled      types.Bool               `tfsdk:"revocation_enabled"`
	RefreshTokenConfigured types.Bool               `tfsdk:"refresh_token_configured"`
	JamfPublicKey          types.String             `tfsdk:"jamf_public_key"`
	ProxyTrustStore        types.String             `tfsdk:"proxy_trust_store"`
	Timeouts               datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// pkiVenafiIdentityModel represents the identity object for Venafi CA resources.
type pkiVenafiIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
