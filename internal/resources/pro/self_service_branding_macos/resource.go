// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package self_service_branding_macos implements the
// jamfplatform_pro_self_service_branding_macos singleton resource and data
// source backed by the Jamf Pro Self Service macOS branding API
// (/api/pro/v1/self-service/branding/macos).
package self_service_branding_macos

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty: the Self Service branding endpoints predate the provider's
// overall floor (11.0.0).
const minJamfProVersion = ""

// SelfServiceBrandingMacosResource implements the presence-optional singleton
// resource for the Jamf Pro Self Service macOS branding configuration.
//
// One configuration per tenant (POST a second ⇒ 409 CREATE_FAILED), but the
// object is presence-optional — it can be created (POST) and deleted (DELETE),
// and the API is id-addressed. Because there is exactly one, the numeric API id
// is discovered via List for Read / Update / Delete; the Terraform state id is
// the fixed helpers.SingletonID.
//
//	Create = POST   (201; 409 ⇒ import-guidance error). GET-after for state.
//	Read   = List   (empty ⇒ RemoveResource; one ⇒ adopt).
//	Update = PUT     (full-replace; omitted field cleared to null).
//	Delete = DELETE  (204; already-absent treated as success).
type SelfServiceBrandingMacosResource struct {
	client *pro.Client
}

var (
	_ resource.Resource                = &SelfServiceBrandingMacosResource{}
	_ resource.ResourceWithImportState = &SelfServiceBrandingMacosResource{}
	_ resource.ResourceWithIdentity    = &SelfServiceBrandingMacosResource{}
)

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// selfServiceBrandingMacosIdentityModel is the identity struct for import.
type selfServiceBrandingMacosIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// NewSelfServiceBrandingMacosResource returns a new instance of the resource.
func NewSelfServiceBrandingMacosResource() resource.Resource {
	return &SelfServiceBrandingMacosResource{}
}

// Metadata sets the resource type name.
func (r *SelfServiceBrandingMacosResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_self_service_branding_macos"
}

// IdentitySchema defines the import identity. Singleton — only the fixed
// helpers.SingletonID value is accepted.
func (r *SelfServiceBrandingMacosResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Fixed singleton identifier. Always \"singleton\" — Self Service macOS branding is one-per-tenant.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the macOS branding resource.
func (r *SelfServiceBrandingMacosResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the Jamf Pro Self Service macOS branding configuration (Settings > Self Service > Branding > macOS Branding). " +
			"Singleton — one configuration per tenant. Creating this resource adds the macOS branding; destroying it removes it (the Self Service app reverts to default branding). " +
			"All fields are optional; removing a field from configuration clears it on the tenant. " +
			"Reference branding images by ID from `jamfplatform_pro_self_service_branding_image` (the branding image store is separate from `jamfplatform_pro_icon`). " +
			"Import with `terraform import jamfplatform_pro_self_service_branding_macos.<name> singleton`." + resourcePrivileges,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Fixed singleton identifier. Always `singleton`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_header": schema.StringAttribute{
				MarkdownDescription: "UI: **Application Header**. Name displayed for the Self Service application in Finder, the Dock, and the menu bar. Omit to leave unset.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"sidebar_heading": schema.StringAttribute{
				MarkdownDescription: "UI: **Sidebar - Heading**. Heading shown in the Self Service sidebar. Omit to leave unset.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"sidebar_subheading": schema.StringAttribute{
				MarkdownDescription: "UI: **Sidebar - Subheading**. Subheading shown in the Self Service sidebar. Omit to leave unset.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"home_page_heading": schema.StringAttribute{
				MarkdownDescription: "UI: **Home page - Heading**. Heading shown on the Self Service home page. Omit to leave unset.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"home_page_subheading": schema.StringAttribute{
				MarkdownDescription: "UI: **Home page - Subheading**. Subheading shown on the Self Service home page. Omit to leave unset.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"icon_id": schema.Int64Attribute{
				MarkdownDescription: "UI: **Icon**. ID of the branding image shown on the Login screen and in Self Service (recommended 180×180 PNG/GIF). Use a `jamfplatform_pro_self_service_branding_image` ID — **not** a `jamfplatform_pro_icon` ID (separate stores). Omit to leave unset.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"banner_image_id": schema.Int64Attribute{
				MarkdownDescription: "UI: **Home page - Banner Image**. ID of the banner image shown across the top of Self Service (recommended 1500×235 PNG/GIF). Use a `jamfplatform_pro_self_service_branding_image` ID. Omit to leave unset.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
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
func (r *SelfServiceBrandingMacosResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_self_service_branding_macos")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import for the singleton. Only the fixed
// helpers.SingletonID value is accepted.
func (r *SelfServiceBrandingMacosResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != helpers.SingletonID {
		resp.Diagnostics.AddError(
			"Invalid singleton import identifier",
			fmt.Sprintf(
				"jamfplatform_pro_self_service_branding_macos is a singleton resource and must be imported with id %q. Got %q.",
				helpers.SingletonID, req.ID,
			),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
