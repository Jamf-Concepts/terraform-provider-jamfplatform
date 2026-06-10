// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package self_service_branding_image implements the
// jamfplatform_pro_self_service_branding_image resource backed by the Jamf Pro
// Self Service branding image upload API. Uploaded images are referenced by ID
// from jamfplatform_pro_self_service_branding_macos (icon_id /
// banner_image_id) and jamfplatform_pro_self_service_branding_ios (icon_id).
package self_service_branding_image

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

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty string skips the version check — the Self Service branding
// image endpoint predates the provider's overall floor (11.0.0).
const minJamfProVersion = ""

// SelfServiceBrandingImageResource implements the Terraform resource for Self
// Service branding images.
type SelfServiceBrandingImageResource struct {
	client *pro.Client
}

var (
	_ resource.Resource                = &SelfServiceBrandingImageResource{}
	_ resource.ResourceWithImportState = &SelfServiceBrandingImageResource{}
	_ resource.ResourceWithIdentity    = &SelfServiceBrandingImageResource{}
	_ resource.ResourceWithModifyPlan  = &SelfServiceBrandingImageResource{}
)

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
)

// selfServiceBrandingImageIdentityModel is the identity struct for import.
type selfServiceBrandingImageIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// NewSelfServiceBrandingImageResource returns a new instance of the resource.
func NewSelfServiceBrandingImageResource() resource.Resource {
	return &SelfServiceBrandingImageResource{}
}

// Metadata sets the resource type name.
func (r *SelfServiceBrandingImageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_self_service_branding_image"
}

// IdentitySchema defines the identifier used for import.
func (r *SelfServiceBrandingImageResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro Self Service branding image ID used to uniquely reference the image.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the branding image resource.
func (r *SelfServiceBrandingImageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Uploads an image to Jamf Pro for use in Self Service branding. The resulting ` + "`id`" + ` is referenced by ` + "`jamfplatform_pro_self_service_branding_macos`" + ` (` + "`icon_id`" + ` / ` + "`banner_image_id`" + `) and ` + "`jamfplatform_pro_self_service_branding_ios`" + ` (` + "`icon_id`" + `).

The Self Service branding image store is **separate** from the general Jamf Pro icon store (` + "`jamfplatform_pro_icon`" + `): the same numeric ID refers to different images in each store, so a branding configuration must reference an ID minted by this resource, not a ` + "`jamfplatform_pro_icon`" + ` ID.

**Source-driven change detection**: the provider opens ` + "`image_file_source`" + ` during every plan, computes a SHA-256 of the bytes, and stores it as ` + "`source_hash`" + `. When the hash changes, Terraform replaces the resource — Jamf Pro has no branding-image update endpoint. When the hash is unchanged the resource is stable.

**Source types**:
- **Local file**: ` + "`image_file_source = \"./banner.png\"`" + `. Read on every plan; stable unless file content changes.
- **URL**: ` + "`image_file_source = \"https://cdn.example.com/banner.png\"`" + `. Downloaded on every plan; triggers replacement when remote content changes.

**Recommended dimensions** (from the Jamf Pro UI): icon 180×180, Home page banner image 1500×235. PNG or GIF.

**Destroy behaviour**: Jamf Pro has no API to delete a branding image. ` + "`terraform destroy`" + ` and replacements both remove the resource from Terraform state only; the image record persists on the tenant.

**Import** (` + "`terraform import jamfplatform_pro_self_service_branding_image.example 81`" + `): the provider downloads the image bytes via the API and stores their SHA-256. Because Jamf Pro may re-encode uploaded images, point ` + "`image_file_source`" + ` at the API-downloaded copy (not your original upload) to avoid a spurious replacement on the next plan.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Jamf Pro Self Service branding image ID, derived from the upload URL. Changes when the resource is replaced.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"image_file_source": schema.StringAttribute{
				MarkdownDescription: "Local filesystem path or `http(s)://` URL to the image. Read by the provider during every plan to compute `source_hash`. Required.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"source_hash": schema.StringAttribute{
				MarkdownDescription: "Provider-computed SHA-256 of the image bytes, prefixed `sha256:`. Replacement is triggered when this value changes.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Download URL returned by Jamf Pro after upload.",
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

// Configure wires the Jamf Pro client into the resource.
func (r *SelfServiceBrandingImageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_self_service_branding_image")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ModifyPlan implements provider-driven change detection: it opens
// image_file_source, computes a SHA-256, sets source_hash on Create plans, and
// triggers RequiresReplace when the hash differs from state (Jamf Pro has no
// branding-image update endpoint). The destroy path returns early.
func (r *SelfServiceBrandingImageResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan SelfServiceBrandingImageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ImageFileSource.IsNull() || plan.ImageFileSource.IsUnknown() || plan.ImageFileSource.ValueString() == "" {
		return
	}

	file, _, cleanup, openErr := files.OpenUploadSource(ctx, plan.ImageFileSource.ValueString(), files.DefaultMaxBytes)
	if openErr != nil {
		resp.Diagnostics.AddError("Error opening branding image source during plan", openErr.Error())
		return
	}
	defer cleanup()

	data, readErr := io.ReadAll(file)
	if readErr != nil {
		resp.Diagnostics.AddError("Error reading branding image source during plan", readErr.Error())
		return
	}
	newHash := files.ComputeContentSHA256(data)

	// Create case — set computed source_hash on the plan.
	if req.State.Raw.IsNull() {
		plan.SourceHash = types.StringValue(newHash)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
		return
	}

	var state SelfServiceBrandingImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.SourceHash.ValueString() == newHash {
		// Hashes match — UseStateForUnknown carried state forward. Do nothing.
		return
	}

	// Hashes differ — request replacement. Mark computed attrs Unknown so the
	// framework re-derives them on apply.
	plan.SourceHash = types.StringValue(newHash)
	plan.ID = types.StringUnknown()
	plan.URL = types.StringUnknown()
	resp.RequiresReplace = append(resp.RequiresReplace, path.Root("source_hash"))
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// ImportState handles import by the Jamf Pro branding image ID.
func (r *SelfServiceBrandingImageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
