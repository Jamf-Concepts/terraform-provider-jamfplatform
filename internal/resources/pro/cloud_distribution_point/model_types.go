// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_distribution_point

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CloudDistributionPointResourceModel is the Terraform model for the Jamf Pro
// cloud distribution point singleton.
//
// `password` and `private_key` are `WriteOnly`: their plaintext is read from
// req.Config on writes and never persisted in Terraform state. `password` is
// documented write-only in the SDK (the server never returns it). `private_key`
// is the AWS CloudFront signing key (`.pem`/`.der`), required only for
// AMAZON_S3 + signed URLs; it is kept WriteOnly because its read-back shape is
// untestable here (no AMAZON_S3 credentials) and a private key does not belong
// in plaintext state — if future AWS testing shows a clean round-trip and drift
// detection is wanted, it can move to Optional+Computed (provider is unshipped). They have no `_wo_version` rotation companion: the
// API marks `password` (and `username`) as REQUIRED on every write body
// (OpenAPI `CloudDistributionPoint.required = [cdnType, password, username]`),
// so the endpoint has no partial-update / field-omission semantics for the
// secret — the SDK correctly models `Password` as a plain always-sent `string`.
// Both secrets are simply re-sent from Config on every write (idempotent); for
// JAMF_CLOUD they are empty strings, which the server accepts.
type CloudDistributionPointResourceModel struct {
	ID types.String `tfsdk:"id"`

	// CdnType is the discriminator. RequiresReplace — switching type tears down
	// (DELETE → NONE) and recreates (POST). One of JAMF_CLOUD, AMAZON_S3,
	// AKAMAI, RACKSPACE_CLOUD_FILES.
	CdnType types.String `tfsdk:"cdn_type"`

	Master types.Bool `tfsdk:"master"`

	// Non-JCDS credential / endpoint fields. Untested (no AMAZON_S3 / AKAMAI /
	// RACKSPACE credentials on the maintainer tenant) — best-guess from the SDK
	// and jamf-cli scaffold.
	Username                types.String `tfsdk:"username"`
	Password                types.String `tfsdk:"password"`
	Directory               types.String `tfsdk:"directory"`
	UploadURL               types.String `tfsdk:"upload_url"`
	DownloadURL             types.String `tfsdk:"download_url"`
	CdnURL                  types.String `tfsdk:"cdn_url"`
	RequireSignedURLs       types.Bool   `tfsdk:"require_signed_urls"`
	KeyPairID               types.String `tfsdk:"key_pair_id"`
	PrivateKey              types.String `tfsdk:"private_key"`
	ExpirationSeconds       types.Int64  `tfsdk:"expiration_seconds"`
	SecondaryAuthRequired   types.Bool   `tfsdk:"secondary_auth_required"`
	SecondaryAuthTimeToLive types.Int64  `tfsdk:"secondary_auth_time_to_live"`

	// Server-derived status echoes — Computed only, never user-set.
	SecondaryAuthStatusCode types.Int64  `tfsdk:"secondary_auth_status_code"`
	HasConnectionSucceeded  types.Bool   `tfsdk:"has_connection_succeeded"`
	Message                 types.String `tfsdk:"message"`
	InventoryID             types.String `tfsdk:"inventory_id"`

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// CloudDistributionPointDataSourceModel mirrors the resource read surface.
// WriteOnly secrets are omitted — Jamf Pro never returns them.
type CloudDistributionPointDataSourceModel struct {
	ID                      types.String             `tfsdk:"id"`
	CdnType                 types.String             `tfsdk:"cdn_type"`
	Master                  types.Bool               `tfsdk:"master"`
	Username                types.String             `tfsdk:"username"`
	Directory               types.String             `tfsdk:"directory"`
	UploadURL               types.String             `tfsdk:"upload_url"`
	DownloadURL             types.String             `tfsdk:"download_url"`
	CdnURL                  types.String             `tfsdk:"cdn_url"`
	RequireSignedURLs       types.Bool               `tfsdk:"require_signed_urls"`
	KeyPairID               types.String             `tfsdk:"key_pair_id"`
	ExpirationSeconds       types.Int64              `tfsdk:"expiration_seconds"`
	SecondaryAuthRequired   types.Bool               `tfsdk:"secondary_auth_required"`
	SecondaryAuthTimeToLive types.Int64              `tfsdk:"secondary_auth_time_to_live"`
	SecondaryAuthStatusCode types.Int64              `tfsdk:"secondary_auth_status_code"`
	HasConnectionSucceeded  types.Bool               `tfsdk:"has_connection_succeeded"`
	Message                 types.String             `tfsdk:"message"`
	InventoryID             types.String             `tfsdk:"inventory_id"`
	Timeouts                datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// cloudDistributionPointIdentityModel is the import identity for the singleton.
type cloudDistributionPointIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
