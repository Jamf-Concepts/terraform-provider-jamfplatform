// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty: /v1/packages predates the provider's overall floor; the
// shared providerdata.ConfigurePro helper still surfaces the global floor
// advisory when the tenant is older.
const minJamfProVersion = ""

// PackageResource implements the Terraform resource for Jamf Pro packages.
type PackageResource struct {
	client *pro.Client
}

var (
	_ resource.Resource                = &PackageResource{}
	_ resource.ResourceWithImportState = &PackageResource{}
	_ resource.ResourceWithIdentity    = &PackageResource{}
)

const (
	// Upload + verification can take minutes on multi-GB binaries — 30m
	// matches the v1 spike default. Read/Delete remain on the standard
	// 60s budget; no upload path runs.
	defaultCreateTimeout = 30 * time.Minute
	defaultUpdateTimeout = 30 * time.Minute
	defaultReadTimeout   = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewPackageResource returns a new instance of PackageResource.
func NewPackageResource() resource.Resource {
	return &PackageResource{}
}

// Metadata sets the resource type name.
func (r *PackageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_package"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *PackageResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro package ID used to uniquely reference the package record.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the package resource.
func (r *PackageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro package. A package record carries the metadata (name, category, restart requirement, OS requirement, info/notes, optional manifest, optional hashes) that Jamf Pro joins with a binary on a distribution point.\n\n**Three operating modes are inferred from the configuration**:\n\n- **JCDS upload** — set `package_file_source` (local path or `https?://` URL). The provider creates the metadata record, streams the binary to the Jamf Cloud Distribution Point, then polls until JCDS finishes computing every server-side hash and `cloud_transfer_status` becomes `READY`. In this mode the hash attributes (`sha3_512`, `sha256`, `md5`, `size`, `hash_type`, `hash_value`) are server-populated — supplying any of them in config errors at plan time (`ConflictsWith`).\n- **File-share DP with user-supplied hashes (FSDP-with-hashes)** — omit `package_file_source` and supply the hash attributes directly. The provider PUTs them verbatim; the server stores whatever the user supplies without validation. Use this when the binary lives on a customer-managed share and hashes are computed off-cluster.\n- **Pure metadata-only (FSDP)** — omit `package_file_source` and all hash attributes. The provider manages only the JSS metadata record; no upload, no hash compute, no verification poll.\n\n**Hash behaviour notes:**\n\n- `package_file_source_checksum` (optional) is a user-supplied SHA-3-512 hint validated locally before any bytes leave the workstation. A mismatch errors out without uploading — useful for guarding against on-disk corruption.\n- `size` is **Computed-only**. The server rejects user-supplied size PUTs even on FSDP records (audit §13.8 A.7).\n- Re-uploads are hash-aware: changing only metadata (`info`, `notes`, `priority`, ...) issues one `PUT /packages/{id}`; the binary is not re-uploaded.\n\n**Manifest sub-resource**: `manifest_file_source` (Optional) uploads a `.plist` to `POST /packages/{id}/manifest`. Setting the source after a state without one uploads; clearing the source deletes server-side. Re-upload only fires when the freshly-loaded source content differs from the stored `manifest` body.\n\n**URL sources**: both `package_file_source` and `manifest_file_source` accept `http(s)://` URLs. The provider streams the URL into a sanitised tempfile under the OS tempdir, enforces an 8 GiB download cap, and follows up to 10 redirects.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Package ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "**\"Display Name\"** in the Jamf Pro admin UI. Wire field `packageName`. Required.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"file_name": schema.StringAttribute{
				MarkdownDescription: "**\"Filename\"** in the Jamf Pro admin UI. The on-disk filename Jamf Pro associates with the binary on the distribution point. Required.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"category_id": schema.StringAttribute{
				MarkdownDescription: "**\"Category\"** in the Jamf Pro admin UI. Server default `\"-1\"` (None).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"info": schema.StringAttribute{
				MarkdownDescription: "**\"Info\"** in the Jamf Pro admin UI. Free-form metadata field. Server returns `\"\"` when null — provider reconciles to keep state stable.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"notes": schema.StringAttribute{
				MarkdownDescription: "**\"Notes\"** in the Jamf Pro admin UI. Free-form notes field. Server returns `\"\"` when null — provider reconciles to keep state stable.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "**\"Priority\"** in the Jamf Pro admin UI. Server default `10`. Lower values install first when multiple packages are queued.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 20),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"fill_user_template": schema.BoolAttribute{
				MarkdownDescription: "**\"Fill user templates (FUT)\"** in the Jamf Pro admin UI. Wire field `fillUserTemplate`. Server default `false`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"fill_existing_users": schema.BoolAttribute{
				MarkdownDescription: "**\"Fill existing user home directories (FEU)\"** in the Jamf Pro admin UI. Wire field `fillExistingUsers`. Server default `false`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"reboot_required": schema.BoolAttribute{
				MarkdownDescription: "**\"Requires restart\"** in the Jamf Pro admin UI. Wire field `rebootRequired`. Server default `false`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"os_requirements": schema.StringAttribute{
				MarkdownDescription: "**\"Operating system requirement\"** in the Jamf Pro admin UI (Limitations tab). Comma-separated string, e.g. `\"13.5.2, 13.6.x, 14.3\"`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"available_in_software_update": schema.BoolAttribute{
				MarkdownDescription: "**\"Install only if available in Software Update\"** in the Jamf Pro admin UI. Wire field `swu`. Server default `false`. NOT the same as the deferred `osInstall` flag.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},

			// Upload-source inputs (no wire field — pure provider plumbing).
			"package_file_source": schema.StringAttribute{
				MarkdownDescription: "Optional local filesystem path or `http(s)://` URL pointing to the package binary. Setting this triggers a JCDS upload during Create/Update and gates the verification poll. When omitted, the resource manages only the JSS metadata record (FSDP modes). Mutually exclusive with the hash attributes — supplying both errors at plan time.",
				Optional:            true,
			},
			"package_file_source_checksum": schema.StringAttribute{
				MarkdownDescription: "Optional user-supplied SHA-3-512 hex digest validated against the locally-computed hash before upload. Mismatch errors out without uploading. Mutually exclusive with `stream_url_directly`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("stream_url_directly")),
				},
			},
			"stream_url_directly": schema.BoolAttribute{
				MarkdownDescription: "When `true` and `package_file_source` is an `http(s)://` URL, stream the body straight to JCDS instead of staging to a tempfile. Skips disk usage at the cost of: no 429 retry, no pre-upload checksum validation (`ConflictsWith` `package_file_source_checksum`), no `Content-Length` precompute (chunked transfer encoding), and no recovery from mid-stream origin failure. Use on disk-constrained runners with multi-GB binaries. Ignored for local-path sources.",
				Optional:            true,
			},
			"manifest_file_source": schema.StringAttribute{
				MarkdownDescription: "**\"Manifest file\"** in the Jamf Pro admin UI. Optional local path or `http(s)://` URL to a `.plist` manifest. The provider uploads when this is set and the stored `manifest` body differs from the freshly-loaded source content. Clearing this attribute deletes the manifest server-side.",
				Optional:            true,
			},

			// Server-echoed manifest fields (Computed).
			"manifest": schema.StringAttribute{
				MarkdownDescription: "The raw plist body Jamf Pro stores for the package manifest. Empty when no manifest is uploaded. Direct-equality compared against the freshly-loaded `manifest_file_source` to decide whether to re-upload.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					resetIfSourceChangedString(path.MatchRoot("manifest_file_source")),
				},
			},
			"manifest_file_name": schema.StringAttribute{
				MarkdownDescription: "The filename Jamf Pro echoed back for the most recent manifest upload.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					resetIfSourceChangedString(path.MatchRoot("manifest_file_source")),
				},
			},

			// Hash attrs: Optional+Computed in both JCDS and FSDP modes.
			"sha3_512": schema.StringAttribute{
				MarkdownDescription: "SHA-3-512 hex digest of the package binary. JCDS uploads populate this server-side after verification; FSDP-mode users may supply it. Mutually exclusive with `package_file_source`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("package_file_source")),
				},
				PlanModifiers: []planmodifier.String{
					resetIfSourceChangedString(path.MatchRoot("package_file_source")),
				},
			},
			"sha256": schema.StringAttribute{
				MarkdownDescription: "SHA-256 hex digest of the package binary. JCDS uploads populate this server-side; FSDP-mode users may supply it. Mutually exclusive with `package_file_source`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("package_file_source")),
				},
				PlanModifiers: []planmodifier.String{
					resetIfSourceChangedString(path.MatchRoot("package_file_source")),
				},
			},
			"md5": schema.StringAttribute{
				MarkdownDescription: "MD5 hex digest of the package binary. JCDS uploads populate this server-side; FSDP-mode users may supply it. Mutually exclusive with `package_file_source`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("package_file_source")),
				},
				PlanModifiers: []planmodifier.String{
					resetIfSourceChangedString(path.MatchRoot("package_file_source")),
				},
			},
			"hash_type": schema.StringAttribute{
				MarkdownDescription: "Hash algorithm advertised for the package. Allowed user-set values: `\"MD5\"`, `\"SHA_256\"`, `\"SHA3_512\"`. The server may also return the legacy default `\"SHA_512\"` on records that have never been uploaded — this value is accepted on reads but not on writes. Mutually exclusive with `package_file_source` (JCDS mode populates `\"SHA3_512\"` post-verification).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(AllowedHashTypeValues...),
					stringvalidator.ConflictsWith(path.MatchRoot("package_file_source")),
				},
				PlanModifiers: []planmodifier.String{
					resetIfSourceChangedString(path.MatchRoot("package_file_source")),
				},
			},
			"hash_value": schema.StringAttribute{
				MarkdownDescription: "Primary hash value used by Jamf Pro to match the package on a distribution point. JCDS mode populates this with the SHA-3-512 of the verified upload; FSDP mode users may supply a digest matching `hash_type`. Mutually exclusive with `package_file_source`. Requires `hash_type` to be set when user-supplied.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("package_file_source")),
					stringvalidator.AlsoRequires(path.MatchRoot("hash_type")),
				},
				PlanModifiers: []planmodifier.String{
					resetIfSourceChangedString(path.MatchRoot("package_file_source")),
				},
			},

			// Server-only / Computed-only fields.
			"size": schema.StringAttribute{
				MarkdownDescription: "Package binary size in bytes (server-derived; the wire type is `string`). Populated automatically by JCDS uploads. **Cannot be set by the user** — audit §13.8 A.7 confirmed the server silently drops user-supplied `size` PUTs.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					// Watch both upload sources — DeletePackageManifestV1
					// has a server-side side effect of clearing the
					// package's size field, so a manifest_file_source
					// transition set→null must also invalidate the planned
					// size value.
					resetIfSourceChangedString(
						path.MatchRoot("package_file_source"),
						path.MatchRoot("manifest_file_source"),
					),
				},
			},
			"install_language": schema.StringAttribute{
				MarkdownDescription: "Server-derived locale tag, default `\"en_US\"`. Not exposed in the Jamf Pro admin UI; surfaced Computed-only to avoid drift on refresh.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_package_id": schema.StringAttribute{
				MarkdownDescription: "Server-derived parent package ID, default `\"-1\"` (no parent). Computed-only.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"self_healing_action": schema.StringAttribute{
				MarkdownDescription: "Server-derived self-healing action, default `\"nothing\"`. Computed-only.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"self_heal_notify": schema.BoolAttribute{
				MarkdownDescription: "Server-derived self-healing notification flag. Default `false`. Computed-only.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cloud_transfer_status": schema.StringAttribute{
				MarkdownDescription: "JCDS transfer status — populated as the cloud distribution point processes an upload. The verification poll converges on `\"READY\"` for JCDS uploads; FSDP records leave this empty.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					resetIfSourceChangedString(path.MatchRoot("package_file_source")),
				},
			},
			"indexed": schema.BoolAttribute{
				MarkdownDescription: "Distribution-point indexing telemetry. Server-derived. Computed-only.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"format": schema.StringAttribute{
				MarkdownDescription: "Distribution-point format string. Server-derived. Computed-only.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

// Configure wires the Jamf Pro client into the resource via the shared
// providerdata.ConfigurePro helper.
func (r *PackageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_package")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import by the Jamf Pro package ID.
func (r *PackageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
