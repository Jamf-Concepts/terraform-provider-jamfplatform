// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package disk_encryption_configuration implements the
// jamfplatform_pro_disk_encryption_configuration resource, data source,
// and list resource backed by the Jamf ProClassic
// /diskencryptionconfigurations API.
package disk_encryption_configuration

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty: classic /diskencryptionconfigurations predates the
// provider's overall floor. The provider-level advisory still fires
// through providerdata.ConfigureProClassic when the tenant is below
// ProviderMinJamfProVersion.
const minJamfProVersion = ""

// DiskEncryptionConfigurationResource implements the Terraform resource for
// Jamf Pro disk encryption configurations.
type DiskEncryptionConfigurationResource struct {
	client *proclassic.Client
}

var (
	_ resource.Resource                     = &DiskEncryptionConfigurationResource{}
	_ resource.ResourceWithImportState      = &DiskEncryptionConfigurationResource{}
	_ resource.ResourceWithIdentity         = &DiskEncryptionConfigurationResource{}
	_ resource.ResourceWithConfigValidators = &DiskEncryptionConfigurationResource{}
)

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewDiskEncryptionConfigurationResource returns a new instance of the
// resource.
func NewDiskEncryptionConfigurationResource() resource.Resource {
	return &DiskEncryptionConfigurationResource{}
}

// Metadata sets the resource type name.
func (r *DiskEncryptionConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_disk_encryption_configuration"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *DiskEncryptionConfigurationResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro disk encryption configuration ID used to uniquely reference the configuration.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the resource.
func (r *DiskEncryptionConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro disk encryption configuration. Disk encryption configurations describe how Jamf-managed Macs derive a FileVault recovery key. The flat envelope (`name`, `key_type`, `file_vault_enabled_users`) is paired with an optional `institutional_recovery_key` block carrying the recovery certificate when `key_type` selects `Institutional` or `Individual and Institutional`.\n\n**Wire quirks worth knowing:**\n\n- `key_type` uses the **read-canonical** spellings (lowercase `and` in `Individual and Institutional`) in Terraform state. The classic write endpoint asymmetrically demands Title-Case `Individual And Institutional` and rejects the lowercase form with HTTP 409; the provider translates one-way at the input boundary so users only ever see the lowercase wire form.\n- The classic POST/PUT endpoints reject an `institutional_recovery_key` block without `certificate_type` (`Certificate type is required if a recovery key is specified`). The schema marks `certificate_type` Required inside the IRK block so any plan that supplies the block also supplies the format declarator.\n- `institutional_recovery_key.password_sha256` is **not** a real SHA-256 hash — the server returns the literal `********************` (20 asterisks) whenever a password is set, empty otherwise. Useful only as an \"is-set\" hint; drift detection against the user-supplied plaintext is impossible.\n- `institutional_recovery_key.password` is **write-only**. The Jamf Pro server never echoes the plaintext on reads, so the provider deliberately does not overwrite this attribute from API responses.\n- The Classic `/diskencryptionconfigurations` PUT endpoint **cannot clear** the `institutional_recovery_key` block once set — an empty `<institutional_recovery_key/>` on PUT is treated as a no-op. Transitioning `key_type` from `Institutional`/`Individual and Institutional` back to `Individual` does not remove the stored cert on the server. This is a known server limitation; destroy and recreate the resource to fully clear the IRK material.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Disk encryption configuration ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "**\"Display Name\"** in the Jamf Pro admin UI. Disk encryption configuration name. Must not be empty.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"key_type": schema.StringAttribute{
				MarkdownDescription: "**\"Recovery Key Type\"** in the Jamf Pro admin UI. Selects which recovery key Jamf provisions when the Mac enables FileVault. Wire-canonical values (must be supplied verbatim — the server normalises any case variant on read so writing the wire form keeps state stable): `\"Individual\"`, `\"Institutional\"`, `\"Individual and Institutional\"` (note the lowercase `and`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(allKeyTypeWireValues...),
				},
			},
			"file_vault_enabled_users": schema.StringAttribute{
				MarkdownDescription: "**\"Enabled FileVault 2 User\"** in the Jamf Pro admin UI. Account allowed to unlock FileVault. Accepted values: `\"Current or Next User\"`, `\"Management Account\"`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(allFileVaultEnabledUsersValues...),
				},
			},
			"institutional_recovery_key": schema.SingleNestedAttribute{
				MarkdownDescription: "**\"Institutional Recovery Key\"** in the Jamf Pro admin UI. Optional block carrying the recovery certificate that issues FileVault keys when `key_type` is `Institutional` or `Individual and Institutional`. The block is **required** in those modes; supplying it for `Individual` is allowed but meaningless (the server stores the cert but never issues it).\n\nThis block stays Optional-only (not Computed) because the framework cannot fit an Unknown value into a typed pointer model.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"key": schema.StringAttribute{
						MarkdownDescription: "Server-derived Subject DN extracted from the uploaded certificate (`data`). Read-only — the user's input is ignored and overwritten by the server on every read. Sample: `C=US, O=jamf-tf-provider, CN=tf-audit-probe`.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"certificate_type": schema.StringAttribute{
						MarkdownDescription: "Certificate format declarator. Required by the Jamf Pro classic POST endpoint whenever a recovery key is supplied (server: `Certificate type is required if a recovery key is specified`). Accepted values: `\"PKCS12\"` (private-key-containing .p12 upload), `\"DER\"` (public-cert binary), `\"PEM\"` (public-cert text). Use `\"PKCS12\"` (with `password` set) when `key_type` is `Institutional` or `Individual and Institutional` — only PKCS12 carries the private key Jamf needs to derive per-Mac recovery keys.",
						Required:            true,
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "PKCS12 import password. **Write-only** — the Jamf Pro server never echoes the plaintext on reads; only the masked `password_sha256` sentinel is returned (see below). The provider deliberately does not overwrite this attribute from API responses, so the user-supplied value remains in state until the user changes it. Wrap in `sensitive(...)` to keep it out of Terraform output.\n\nRequired when uploading a `.p12` certificate (the `data` payload contains the private key wrapped by this password). Omit for `.cer` / `.pem` uploads.",
						Optional:            true,
						Sensitive:           true,
					},
					"password_sha256": schema.StringAttribute{
						MarkdownDescription: "Server-side **redaction sentinel** — **NOT a real hash**. The Jamf Pro server returns the literal `********************` (20 asterisks) whenever a password is set on the stored IRK, and an empty string otherwise. This value is useful only as an \"is-set\" hint; it cannot be compared against any hash of the user-supplied `password`, so drift detection against rotated passwords is impossible. The provider surfaces it verbatim so out-of-band password clearing (sentinel transitions to empty) is observable.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"data": schema.StringAttribute{
						MarkdownDescription: "Base64-encoded recovery certificate payload. Required whenever the IRK block is supplied. Jamf Pro accepts `.p12` (PKCS12 with private key — required for the IRK to issue keys), `.cer` (DER binary), and `.pem` (PEM text). For PKCS12 uploads, also set `password`. Round-trips exactly on read. **Sensitive** because PKCS12 payloads contain the wrapped private key.",
						Required:            true,
						Sensitive:           true,
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// ConfigValidators returns the cross-field validators evaluated against the
// user's config at plan time.
func (r *DiskEncryptionConfigurationResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		institutionalKeyTypeRequiresIRKConfigValidator{},
	}
}

// Configure wires the Jamf ProClassic client into the resource.
func (r *DiskEncryptionConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_disk_encryption_configuration")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import by the Jamf Pro disk encryption configuration ID.
func (r *DiskEncryptionConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
