// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package app_installer implements the jamfplatform_pro_app_installer resource,
// data source, and list resource backed by the Jamf Pro App Installer
// deployments API. An App Installer deployment manages an automatically-built,
// signed installer for a title published to the Jamf App Catalog; the catalog
// itself is exposed read-only via jamfplatform_pro_app_installer_title(s).
package app_installer

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty: the App Installer deployment endpoints predate the
// provider's overall floor (the App Catalog shipped well before 11.0.0). The
// provider-level advisory still fires through providerdata.ConfigurePro when the
// tenant is below the floor.
const minJamfProVersion = ""

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 90 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// AppInstallerResource implements the Terraform resource for Jamf Pro App
// Installer deployments.
type AppInstallerResource struct {
	client *pro.Client
}

var (
	_ resource.Resource                = &AppInstallerResource{}
	_ resource.ResourceWithImportState = &AppInstallerResource{}
	_ resource.ResourceWithIdentity    = &AppInstallerResource{}
	_ resource.ResourceWithModifyPlan  = &AppInstallerResource{}
)

// NewAppInstallerResource returns a new instance of AppInstallerResource.
func NewAppInstallerResource() resource.Resource {
	return &AppInstallerResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *AppInstallerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_app_installer"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *AppInstallerResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "App Installer deployment ID used to uniquely reference the deployment.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the App Installer deployment resource.
func (r *AppInstallerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro App Installer — an automatically-built, signed installer for a title published to the Jamf App Catalog. Choose the catalog title by name via `app_title_name` (list available titles with the `jamfplatform_pro_app_installer_titles` data source). " +
			"`update_behavior` controls when updates apply (`AUTOMATIC` tracks the latest catalog version, `MANUAL` updates on your schedule); the version itself is always the latest available and is reported in `selected_version`. " +
			"Setting `category_id`, `site_id`, or `smart_group_id` to `-1` means \"none\".",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Deployment ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name for the App Installer. Must not be blank.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"enabled": schema.BoolAttribute{
				// Jamf Pro rejects enabling a deployment that has no smart group
				// scope: a deployment can only be enabled when smart_group_id is
				// set to a real group (not -1). Not validated at plan time because
				// smart_group_id is Computed and reads back Unknown when omitted;
				// the server enforces it with a clear error.
				MarkdownDescription: "Whether the deployment is enabled. A deployment can only be enabled when `smart_group_id` is set to a real smart group (not `-1`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"app_title_name": schema.StringAttribute{
				MarkdownDescription: "Name of the App Catalog title to deploy, exactly as listed in the catalog (for example `Google Chrome`). List available titles with the `jamfplatform_pro_app_installer_titles` data source. The provider verifies the title exists at plan time and resolves it to `app_title_id`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"app_title_id": schema.StringAttribute{
				// Computed, resolved from app_title_name. No UseStateForUnknown:
				// it is derived from the mutable app_title_name, so it must go
				// Unknown (not pin the stale ID) when the name changes, or the
				// post-apply consistency check trips. Same rationale as
				// mac_app_store_app's derived *_name attributes.
				MarkdownDescription: "ID of the App Catalog title, resolved from `app_title_name`. Returned by Jamf Pro.",
				Computed:            true,
			},
			"deployment_type": schema.StringAttribute{
				MarkdownDescription: "How the app is delivered. One of `INSTALL_AUTOMATICALLY` (push to all in-scope devices) or `SELF_SERVICE` (offered in Self Service).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(deploymentTypeInstallAutomatically, deploymentTypeSelfService),
				},
			},
			"update_behavior": schema.StringAttribute{
				MarkdownDescription: "How updates are applied. One of `AUTOMATIC` (always track the latest catalog version; `selected_version` is forced empty) or `MANUAL` (stay on the pinned `selected_version`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(updateBehaviorAutomatic, updateBehaviorManual),
				},
			},
			"selected_version": schema.StringAttribute{
				// Computed-only, permanently. Pinning a specific version is not a
				// real feature: the server coerces any submitted value to the
				// actual available catalog version (wire-probed: "1.0.0" → "15.3"),
				// and the Jamf Pro UI exposes no version picker (only the
				// Automatic/Manual update-behavior choice). It reads back as the
				// latest available version, or "" while AUTOMATIC has not yet
				// recorded one. No UseStateForUnknown: it is derived from the
				// mutable app_title_name / update_behavior and must go Unknown when
				// those change, or the post-apply consistency check trips.
				MarkdownDescription: "Version Jamf Pro has selected for the deployment. Always the latest available version; the version is not user-pinnable. Returned by Jamf Pro.",
				Computed:            true,
			},
			"latest_available_version": schema.StringAttribute{
				// No UseStateForUnknown: derived from the mutable app_title_id /
				// selected_version, so it must go Unknown (not pin the stale
				// value) when those change, or the post-apply consistency check
				// trips. Same rationale as mac_app_store_app's category_name.
				MarkdownDescription: "Latest version available for the title in the catalog. Returned by Jamf Pro; not user-settable.",
				Computed:            true,
			},
			"title_available_in_ais": schema.BoolAttribute{
				MarkdownDescription: "Whether the title is available in the App Installers catalog. Returned by Jamf Pro; not user-settable.",
				Computed:            true,
			},
			"version_removed": schema.BoolAttribute{
				MarkdownDescription: "Whether the pinned version has been removed from the catalog. Returned by Jamf Pro; not user-settable.",
				Computed:            true,
			},
			"category_id": schema.StringAttribute{
				MarkdownDescription: "Jamf Pro category ID for the deployment. Use `-1` for no category.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"site_id": schema.StringAttribute{
				MarkdownDescription: "Jamf Pro site ID scoping the deployment. Use `-1` for no site.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"smart_group_id": schema.StringAttribute{
				MarkdownDescription: "Smart computer group ID scoping the deployment. Use `-1` for no smart group.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"install_predefined_config_profiles": schema.BoolAttribute{
				MarkdownDescription: "Whether Jamf installs the title's predefined configuration profiles alongside the app.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"trigger_admin_notifications": schema.BoolAttribute{
				MarkdownDescription: "Whether Jamf Pro logs event notifications for this app (raising administrator notifications). Defaults to off on create.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"notification_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "End-user notification presentation (the \"End user experience\" tab). Supply the block to manage notifications; omit it to leave Jamf's defaults. Each field is independent — omit a field to keep Jamf's default for it. Message fields must not be blank, and the interval/delay values must be positive, when set.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"notification_message": schema.StringAttribute{
						MarkdownDescription: "Message shown to the user when an install or update is available.",
						Optional:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"notification_interval": schema.Int64Attribute{
						MarkdownDescription: "Hours between repeat notifications.",
						Optional:            true,
						Validators:          []validator.Int64{int64validator.AtLeast(1)},
					},
					"deadline_message": schema.StringAttribute{
						MarkdownDescription: "Message shown as the install deadline approaches.",
						Optional:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"deadline": schema.Int64Attribute{
						MarkdownDescription: "Hours the user may defer before the install is forced.",
						Optional:            true,
						Validators:          []validator.Int64{int64validator.AtLeast(1)},
					},
					"quit_delay": schema.Int64Attribute{
						MarkdownDescription: "Seconds the user is given to quit the app before the install proceeds.",
						Optional:            true,
						Validators:          []validator.Int64{int64validator.AtLeast(1)},
					},
					"complete_message": schema.StringAttribute{
						MarkdownDescription: "Message shown once the install or update completes.",
						Optional:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"relaunch": schema.BoolAttribute{
						MarkdownDescription: "Whether to relaunch the app after the update.",
						Optional:            true,
					},
					"suppress": schema.BoolAttribute{
						MarkdownDescription: "Whether to suppress end-user notifications entirely.",
						Optional:            true,
					},
				},
			},
			"self_service_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Self Service presentation. Supply the block to manage how the deployment appears in Self Service; omit it to leave Jamf's defaults. Every field is replaced on each apply, so set all fields you care about. Note Jamf accepts a Self Service block even for `INSTALL_AUTOMATICALLY` deployments.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						MarkdownDescription: "Description shown in Self Service. Omit to leave it unset.",
						Optional:            true,
					},
					"force_view_description": schema.BoolAttribute{
						MarkdownDescription: "Force users to view the description before installing.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
					},
					"include_in_compliance_category": schema.BoolAttribute{
						MarkdownDescription: "Include the deployment in the Self Service compliance category.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
					},
					"include_in_featured_category": schema.BoolAttribute{
						MarkdownDescription: "Include the deployment in the Self Service featured category.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
					},
					"categories": schema.SetNestedAttribute{
						MarkdownDescription: "Self Service categories the deployment appears under, keyed by `category_id`.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"category_id": schema.StringAttribute{
									MarkdownDescription: "Self Service category ID.",
									Required:            true,
									Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
								},
								"featured": schema.BoolAttribute{
									MarkdownDescription: "Whether the deployment is featured within this category.",
									Optional:            true,
									Computed:            true,
									PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
								},
							},
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

// Configure wires the Jamf Pro client into the resource via the shared
// providerdata.ConfigurePro helper.
func (r *AppInstallerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_app_installer")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import by the Jamf Pro deployment ID.
func (r *AppInstallerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ModifyPlan validates app_title_name against the live App Catalog at plan time,
// surfacing an unknown title as a clear error instead of the apply-time
// resolution failure. No-op on destroy (null plan).
func (r *AppInstallerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan AppInstallerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateAppTitleName(ctx, r.client, plan.AppTitleName, path.Root("app_title_name"))...)
}
