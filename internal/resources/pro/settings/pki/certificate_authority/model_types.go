// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package certificate_authority

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CertificateAuthorityDataSourceModel is the Terraform model for the
// jamfplatform_pro_pki_certificate_authority data source. A single-record lookup:
// omit `id` to read the active Certificate Authority, or set it to read a specific CA by
// id. Every X.509 field is Computed (read-only); the cert blob is exposed as Computed
// `pem`.
type CertificateAuthorityDataSourceModel struct {
	ID                    types.String             `tfsdk:"id"`
	SubjectX500Principal  types.String             `tfsdk:"subject_x500_principal"`
	IssuerX500Principal   types.String             `tfsdk:"issuer_x500_principal"`
	SerialNumber          types.String             `tfsdk:"serial_number"`
	Version               types.Int64              `tfsdk:"version"`
	NotAfter              types.Int64              `tfsdk:"not_after"`
	NotBefore             types.Int64              `tfsdk:"not_before"`
	KeyUsage              types.List               `tfsdk:"key_usage"`
	KeyUsageExtended      types.List               `tfsdk:"key_usage_extended"`
	Sha1Fingerprint       types.String             `tfsdk:"sha1_fingerprint"`
	Sha256Fingerprint     types.String             `tfsdk:"sha256_fingerprint"`
	SignatureAlgorithm    types.String             `tfsdk:"signature_algorithm"`
	SignatureAlgorithmOid types.String             `tfsdk:"signature_algorithm_oid"`
	SignatureValue        types.String             `tfsdk:"signature_value"`
	Pem                   types.String             `tfsdk:"pem"`
	Timeouts              datasourceTimeouts.Value `tfsdk:"timeouts"`
}
