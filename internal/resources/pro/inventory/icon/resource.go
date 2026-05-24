// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package icon implements the jamfplatform_pro_icon resource backed by the
// Jamf Pro icon upload API.
package icon

import (
	"context"
	"io"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/files"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this resource.
// Empty string skips the version check — the icon endpoint has been stable since
// well before the provider's overall floor (11.0.0), so no per-resource gate is needed.
const minJamfProVersion = ""

// IconResource implements the Terraform resource for Jamf Pro icons.
type IconResource struct {
	client *pro.Client
}

var (
	_ resource.Resource                = &IconResource{}
	_ resource.ResourceWithImportState = &IconResource{}
	_ resource.ResourceWithIdentity    = &IconResource{}
	_ resource.ResourceWithModifyPlan  = &IconResource{}
)

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
)

// iconIdentityModel is the identity struct for Jamf Pro icon resources.
type iconIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// NewIconResource returns a new instance of IconResource.
func NewIconResource() resource.Resource {
	return &IconResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *IconResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_icon"
}

// IdentitySchema defines the identifier used for import.
func (r *IconResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro icon ID used to uniquely reference the icon.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the icon resource.
func (r *IconResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Jamf Pro icon. Icons are uploaded via the ` + "`/v1/icon`" + ` endpoint and referenced by ID from Self Service branding configurations.

**Source-driven change detection**: the provider opens ` + "`icon_file_source`" + ` during every plan, computes a SHA-256 of the bytes, and stores it as ` + "`source_hash`" + `. When the hash changes, Terraform replaces the resource (no in-place update — Jamf Pro has no icon update endpoint). When the hash is unchanged the resource is stable.

**Source types**:
- **Local file**: ` + "`icon_file_source = \"./icon.png\"`" + `. Provider reads bytes on every plan. Stable across plans unless file content changes.
- **URL**: ` + "`icon_file_source = \"https://cdn.example.com/icon.png\"`" + `. Provider downloads on every plan (~tens of KB). Triggers replacement when the remote content changes — useful for tracking upstream icons (e.g. App Store CDN).
- **Frozen URL behaviour**: if you need a URL-sourced icon to NOT track upstream changes, download the icon locally and switch ` + "`icon_file_source`" + ` to the local path.

**No DELETE endpoint**: Jamf Pro does not expose a delete API for icons. ` + "`terraform destroy`" + ` and replacements both remove the resource from Terraform state only; the icon record persists on the tenant.

**Import workflow** (no spurious replacement):
1. ` + "`terraform import jamfplatform_pro_icon.example 42`" + `. Provider downloads the icon bytes via the CDN URL and stores the SHA-256 in state.
2. Download the icon locally from the URL stored in state (e.g. ` + "`curl -o ./icon.png \"$(terraform state show jamfplatform_pro_icon.example | awk '/^[[:space:]]*url/{print $3}' | tr -d '\\\"')\"`" + `). **You must download from the URL — do NOT reuse the file you originally uploaded.** Jamf Pro re-encodes uploaded PNGs server-side (different zlib compression and/or metadata), so the bytes served back from the CDN are not byte-identical to what you uploaded.
3. Add ` + "`icon_file_source = \"./icon.png\"`" + ` to your config.
4. ` + "`terraform plan`" + ` shows an in-place update on ` + "`icon_file_source`" + ` (null → path). No replacement because the local bytes now match what was stored on import.

If you skip step 2 and point ` + "`icon_file_source`" + ` at your original upload file instead of the CDN-downloaded copy, the first plan after import will show a **replacement** — the local bytes' hash will not match the import-stored hash (Jamf transformed them).`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Jamf Pro icon ID assigned on upload. Changes when the resource is replaced.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"icon_file_source": schema.StringAttribute{
				MarkdownDescription: "Local filesystem path or `http(s)://` URL to the icon image. Read by the provider during every plan to compute `source_hash`. Required.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"source_hash": schema.StringAttribute{
				MarkdownDescription: "Provider-computed SHA-256 of the icon bytes, prefixed `sha256:`. Replacement is triggered when this value changes.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "CDN URL returned by Jamf Pro after upload.",
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
func (r *IconResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_icon")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ModifyPlan implements provider-driven change detection for the icon
// resource. On every plan (when icon_file_source is known) the provider
// opens the source, reads the bytes, computes a SHA-256, and either:
//
//   - sets the computed source_hash on Create plans;
//   - leaves the plan unchanged when the hash matches state (no-op plan);
//   - triggers a RequiresReplace on source_hash when the hash differs.
//
// The destroy path (req.Plan.Raw.IsNull()) returns early before touching
// the source — destroys never need byte access.
func (r *IconResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy path — nothing to do.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan IconResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Skip if icon_file_source is not yet known. The Required validator
	// catches the null/empty case at later phases; an Unknown value here
	// means a deferred reference (e.g. from another resource) that we
	// cannot resolve at plan time.
	if plan.IconFileSource.IsNull() || plan.IconFileSource.IsUnknown() || plan.IconFileSource.ValueString() == "" {
		return
	}

	file, _, cleanup, openErr := files.OpenUploadSource(ctx, plan.IconFileSource.ValueString(), files.DefaultMaxBytes)
	if openErr != nil {
		resp.Diagnostics.AddError("Error opening icon source during plan", openErr.Error())
		return
	}
	defer cleanup()

	data, readErr := io.ReadAll(file)
	if readErr != nil {
		resp.Diagnostics.AddError("Error reading icon source during plan", readErr.Error())
		return
	}
	newHash := computeSourceHashString(data)

	// Create case — set computed source_hash on the plan.
	if req.State.Raw.IsNull() {
		plan.SourceHash = types.StringValue(newHash)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
		return
	}

	// Update/replace case — compare against stored hash.
	var state IconResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.SourceHash.ValueString() == newHash {
		// Hashes match — UseStateForUnknown has already carried state
		// forward into the plan. Do nothing; do NOT call resp.Plan.Set
		// because re-writing the plan can introduce spurious diffs.
		return
	}

	// Hashes differ — request replacement. Mark computed attributes
	// Unknown so the framework re-derives them on apply.
	plan.SourceHash = types.StringValue(newHash)
	plan.ID = types.StringUnknown()
	plan.URL = types.StringUnknown()
	resp.RequiresReplace = append(resp.RequiresReplace, path.Root("source_hash"))
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// ImportState handles import by the Jamf Pro icon ID.
func (r *IconResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
