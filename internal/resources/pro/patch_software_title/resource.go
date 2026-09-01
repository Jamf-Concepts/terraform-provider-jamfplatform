// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package patch_software_title implements the jamfplatform_pro_patch_software_title
// resource, data source, and list resource backed by the Jamf ProClassic patch
// software titles API.
package patch_software_title

import (
	"context"
	"regexp"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
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
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this resource.
// Empty: the classic /patchsoftwaretitles endpoint predates the provider's overall
// floor (11.0.0). The provider-level advisory still fires through
// providerdata.ConfigureProClassic when the tenant is below ProviderMinJamfProVersion.
const minJamfProVersion = ""

// packageIDPattern matches a positive integer string (Jamf package ID). Used to
// validate version_packages map values at plan time.
var packageIDPattern = regexp.MustCompile(`^[1-9]\d*$`)

// PatchSoftwareTitleResource implements the Terraform resource for Jamf ProClassic
// patch software titles. CRUD runs over the classic /patchsoftwaretitles endpoint
// (client); extension-attribute read + accept use the modern v3
// /patch-software-title-configurations endpoint (proClient), keyed by the same
// id (the classic title id equals the configuration id — wire-verified).
type PatchSoftwareTitleResource struct {
	client    *proclassic.Client
	proClient *pro.Client
}

var _ resource.Resource = &PatchSoftwareTitleResource{}
var _ resource.ResourceWithImportState = &PatchSoftwareTitleResource{}
var _ resource.ResourceWithIdentity = &PatchSoftwareTitleResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewPatchSoftwareTitleResource returns a new instance of PatchSoftwareTitleResource.
func NewPatchSoftwareTitleResource() resource.Resource {
	return &PatchSoftwareTitleResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *PatchSoftwareTitleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_patch_software_title"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *PatchSoftwareTitleResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro patch software title ID used to uniquely reference the title.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the patch software title resource.
func (r *PatchSoftwareTitleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro patch software title, found in the UI under **Computers → Patch management**. A configured title spans the tabs of that interface: the **Software Title Settings** tab (`name`, `category_id`, `site_id`, notifications), the **Definition** tab (per-version package assignments), and the **Extension Attribute** tab (`extension_attributes` / `accept_extension_attributes`). A title is defined by its `name_id` (catalog key) and `source_id` (patch source); the server populates the full catalog of `available_versions`. Assign packages to specific versions via `version_packages` (the **Definition** tab's per-version **Package** column) so patch policies can target them.\n\n" +
			"> **Deprecation notice:** this resource is backed by the Jamf ProClassic `/patchsoftwaretitles` endpoints, which the Jamf API spec flags as deprecated in favour of `/patch-software-title-configurations`. The classic endpoints remain the only functional create surface: the configurations `POST` requires a `softwareTitleId`, which is the classic title id, and the only call that mints one also creates the configuration — so there is nothing for the provider to migrate create onto. Behaviour may change if Jamf removes the classic endpoints." +
			resourcePrivileges,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Patch software title ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the patch software title (UI \"Display Name\"). Must not be empty.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name_id": schema.StringAttribute{
				MarkdownDescription: "Patch catalog key that defines which software title this is (e.g. `285`). Immutable — changing it forces replacement.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_id": schema.Int64Attribute{
				MarkdownDescription: "Patch source ID this title is sourced from. Immutable — changing it forces replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"category_id": schema.StringAttribute{
				MarkdownDescription: "Jamf Pro category ID (UI \"Category\"). Use `-1` (default) for \"No category assigned\".",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"category_name": schema.StringAttribute{
				MarkdownDescription: "Category display name. Returned by Jamf Pro; not user-settable.",
				Computed:            true,
			},
			"site_id": schema.StringAttribute{
				MarkdownDescription: "Jamf Pro site ID. Use `-1` (default) for \"NONE\".",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_name": schema.StringAttribute{
				MarkdownDescription: "Site display name. Returned by Jamf Pro; not user-settable.",
				Computed:            true,
			},
			"web_notification": schema.BoolAttribute{
				MarkdownDescription: "Whether a Jamf Pro notification is raised for new versions (UI \"Jamf Pro Notification\"). Server-defaulted when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"email_notification": schema.BoolAttribute{
				MarkdownDescription: "Whether an email notification is sent for new versions (UI \"Email\"). Server-defaulted when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"version_packages": schema.MapAttribute{
				MarkdownDescription: "Managed map of version→package assignments. Keys are `software_version` strings drawn from `available_versions`; values are Jamf Pro package ID strings. A patch policy can only target a version that has a package assigned here. Only the keys you declare are managed: other server-side assignments are left untouched, and removing a key clears that version's package on the next apply.",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.Map{
					mapvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(packageIDPattern, "must be a positive integer package ID"),
					),
				},
			},
			"available_versions": schema.ListAttribute{
				MarkdownDescription: "All `software_version` strings the patch source publishes for this title, newest first. Returned by Jamf Pro; use these as keys for `version_packages`.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"accept_extension_attributes": schema.BoolAttribute{
				MarkdownDescription: "Accept the extension attribute(s) Jamf attaches to this title (UI \"Extension Attribute\" tab, **Accept**). For some titles Jamf supplies a script that runs on managed computers to collect the installed version; inventory is not gathered until it is accepted. Set to `true` to accept any pending extension attributes on the next apply. **Accepting is one-way** — it cannot be reverted, so setting this back to `false` (or removing it) does not un-accept; it simply stops accepting new ones. Leave unset for titles that have no extension attribute.",
				Optional:            true,
			},
			"extension_attributes": schema.ListNestedAttribute{
				MarkdownDescription: "Extension attributes Jamf has attached to this title, with their acceptance status. Read-only — use `accept_extension_attributes` to accept pending ones. Empty for titles with no extension attribute.",
				Computed:            true,
				// Plain Computed (no plan modifier): when accept_extension_attributes
				// flips a pending EA from accepted=false to true in the same apply,
				// the list must go Unknown so the post-apply read fills the new
				// value — UseStateForUnknown would copy the stale false and trip the
				// "inconsistent result after apply" check. Mirrors the target_version
				// -derived fields on jamfplatform_pro_patch_policy.
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ea_id": schema.StringAttribute{
							MarkdownDescription: "Stable identifier of the extension attribute (e.g. `jamf-patch-adobe-air`).",
							Computed:            true,
						},
						"display_name": schema.StringAttribute{
							MarkdownDescription: "Display name of the extension attribute (e.g. `Adobe AIR Bundle Version`).",
							Computed:            true,
						},
						"accepted": schema.BoolAttribute{
							MarkdownDescription: "Whether the extension attribute has been accepted. Once `true`, it cannot return to `false`.",
							Computed:            true,
						},
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

// Configure wires both Jamf clients into the resource: the ProClassic client
// (classic /patchsoftwaretitles CRUD) and the Pro client (v3 extension-attribute
// read + accept). Both are built from the same provider data.
func (r *PatchSoftwareTitleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_patch_software_title")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client

	proClient, proDiags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_patch_software_title")
	resp.Diagnostics.Append(proDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.proClient = proClient
}

// ImportState handles import by the Jamf Pro patch software title ID.
func (r *PatchSoftwareTitleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
