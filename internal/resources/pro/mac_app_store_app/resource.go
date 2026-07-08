// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package mac_app_store_app implements the jamfplatform_pro_mac_app_store_app
// resource, data source, and list resource backed by the Jamf ProClassic
// /macapplications API. The construct name mirrors the Jamf Pro admin UI
// ("App Store App" under the "Mac Apps" sidebar) and disambiguates from the
// mobile App Store App equivalent. Scope is computer-only and omits iBeacon
// targets (the endpoint silently drops them) via the shared computer-scope
// helper with IncludeIbeacons=false.
package mac_app_store_app

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/ldapgroups"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty: classic /macapplications predates the provider's overall
// floor. The provider-level advisory still fires through
// providerdata.ConfigureProClassic when the tenant is below the floor.
const minJamfProVersion = ""

// MacAppResource implements the Terraform resource for Jamf Pro App Store Mac apps.
type MacAppResource struct {
	client *proclassic.Client
	// ldapSearcher backs the plan-time directory-service user-group preflight in
	// ModifyPlan. The LDAP group search is a Pro (v1) endpoint, so it is a
	// separate client from the ProClassic CRUD client. nil until Configure runs.
	ldapSearcher ldapgroups.Searcher
}

var _ resource.Resource = &MacAppResource{}
var _ resource.ResourceWithImportState = &MacAppResource{}
var _ resource.ResourceWithIdentity = &MacAppResource{}
var _ resource.ResourceWithModifyPlan = &MacAppResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 90 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewMacAppResource returns a new instance of MacAppResource.
func NewMacAppResource() resource.Resource {
	return &MacAppResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *MacAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_mac_app_store_app"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *MacAppResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro Mac App Store app ID used to uniquely reference the app.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the resource. Attribute names mirror
// the Jamf Pro admin UI labels; differing wire element names are noted in the
// attribute descriptions.
func (r *MacAppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro App Store Mac app (the classic `/macapplications` endpoint — the \"App Store App\" entry under the \"Mac Apps\" sidebar). `general.name`, `general.version`, `general.bundle_id`, and `general.url` are required on create and stored verbatim — there is **no** App Store metadata resolution from the URL. Scope targets are flat sets of Jamf Pro IDs; interpolate `jamfplatform_device_group.<x>.jamf_pro_id` to bridge from Platform Services. Scope omits iBeacon limitations/exclusions because the endpoint silently drops them." + resourcePrivileges,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "App ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"general": schema.SingleNestedAttribute{
				MarkdownDescription: "General settings. `name`, `version`, `bundle_id`, and `url` are required on create. Read-only fields (`category_name`, `site_name`, `id`) are returned by Jamf Pro.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "App ID under `general`. Matches the top-level `id`. Returned by Jamf Pro.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "App display name. Must be unique within the tenant.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"version": schema.StringAttribute{
						MarkdownDescription: "App version string. Stored verbatim — Jamf Pro does not resolve it from the App Store.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"bundle_id": schema.StringAttribute{
						MarkdownDescription: "App bundle identifier (e.g. `com.apple.iMovieApp`). Stored verbatim — Jamf Pro does not resolve it from the App Store.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"url": schema.StringAttribute{
						MarkdownDescription: "App Store (iTunes) URL. Required on create — the server rejects a POST without it. Stored verbatim as a string; it does not auto-populate name/version/bundle_id.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"is_free": optComputedBool("Whether the app is free. Server-defaults to false on create."),
					"deployment_type": schema.StringAttribute{
						// Optional+Computed and the server always echoes a value
						// (defaults to "Make Available in Self Service"), so an
						// unset deployment_type must stay Unknown on create rather
						// than copying the null prior state — otherwise the
						// server's default trips the post-apply consistency check.
						// UseNonNullStateForUnknown behaves like UseStateForUnknown
						// once a value is present.
						MarkdownDescription: "Install method. One of `Make Available in Self Service` or `Install Automatically/Prompt Users to Install`. Server-defaults to `Make Available in Self Service` on create.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf(deploymentTypeSelfService, deploymentTypeAutomatic),
						},
					},
					"category_id": optComputedString("Jamf Pro category ID. Use `-1` for \"No category\"."),
					"category_name": schema.StringAttribute{
						// No UseStateForUnknown: category_name is derived from
						// category_id, so it must go Unknown (not pin the stale
						// value) when category_id changes, or the post-apply
						// consistency check trips.
						MarkdownDescription: "Category display name. Returned by Jamf Pro; not user-settable.",
						Computed:            true,
					},
					"site_id": optComputedString("Jamf Pro site ID scoping the app. Use `-1` for \"No site\"."),
					"site_name": schema.StringAttribute{
						// No UseStateForUnknown: site_name is derived from site_id
						// (same rationale as category_name above).
						MarkdownDescription: "Site display name. Returned by Jamf Pro; not user-settable.",
						Computed:            true,
					},
				},
			},
			"scope": schema.SingleNestedAttribute{
				MarkdownDescription: "App scope. Each category is independently owned: declare it (including `[]`, which clears it) and Terraform manages its members; omit it and it is left as configured outside Terraform — updates preserve it. Targets are flat sets of Jamf Pro IDs; interpolate `jamfplatform_device_group.<x>.jamf_pro_id` to bridge from Platform Services. Setting `all_computers = true` forbids `computer_ids`, `computer_group_ids`, `building_ids`, `department_ids`. Setting `all_jss_users = true` forbids `user_ids` and `user_group_ids`. iBeacon limitations/exclusions are intentionally absent — the endpoint silently drops them.",
				Optional:            true,
				Attributes:          scope.ComputerScopeAttributes(scope.ComputerScopeOptions{IncludeIbeacons: false}),
			},
			"self_service": schema.SingleNestedAttribute{
				MarkdownDescription: "Self Service integration. Relevant when `general.deployment_type` is `Make Available in Self Service`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"install_button_text":             optComputedString("Install-button label."),
					"self_service_description":        optComputedString("Self Service description. Markdown supported."),
					"force_users_to_view_description": optComputedBool("Force users to view the description before installing."),
					"feature_on_main_page":            optComputedBool("Feature the app on the Self Service main page."),
					"notification_enabled":            optComputedBool("Whether Self Service surfaces a notification when the app becomes available. Pair with `notification_method`."),
					"notification_method":             optComputedString("Notification delivery method (e.g. `Self Service`). The server defaults a method when notifications are enabled."),
					"notification_subject":            optComputedString("Notification subject line."),
					"notification_message":            optComputedString("Notification body text."),
					"self_service_icon": schema.SingleNestedAttribute{
						MarkdownDescription: "Self Service icon. Set `id` to reference an already-uploaded icon; `uri` is returned by Jamf Pro. Uploading icon bytes inline is not supported (Jamf re-encodes PNGs server-side, which would permadiff) — open an issue if you need it.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"id":  optComputedString("Icon ID assigned by Jamf Pro."),
							"uri": optComputedString("Icon URI. Returned by Jamf Pro."),
						},
					},
					"self_service_categories": schema.SetNestedAttribute{
						MarkdownDescription: "Set of Self Service categories the app appears under. Each item identifies the category by `id`; `name` is returned by Jamf Pro.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":         schema.StringAttribute{MarkdownDescription: "Category ID.", Required: true},
								"name":       optComputedString("Category display name. Returned by Jamf Pro."),
								"display_in": optComputedBool("Display the app in this category."),
								"feature_in": optComputedBool("Feature the app in this category."),
							},
						},
					},
				},
			},
			"vpp": schema.SingleNestedAttribute{
				MarkdownDescription: "Volume Purchasing (VPP) assignment. `assign_vpp_device_based_licenses` and `vpp_admin_account_id` are writable only for a genuinely VPP-backed title — setting `assign_vpp_device_based_licenses = true` on a non-VPP app returns HTTP 409 \"App is not available for device assignment\". The license counts are server-computed.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"assign_vpp_device_based_licenses": optComputedBool("Assign VPP device-based licenses."),
					"vpp_admin_account_id":             optComputedString("VPP admin account ID. `-1` when the app is not VPP-backed."),
					"total_vpp_licenses":               computedInt64("Total VPP licenses. Returned by Jamf Pro."),
					"remaining_vpp_licenses":           computedInt64("Remaining VPP licenses. Returned by Jamf Pro."),
					"used_vpp_licenses":                computedInt64("Used VPP licenses. Returned by Jamf Pro."),
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

// Configure wires the Jamf ProClassic client into the resource via the shared
// providerdata.ConfigureProClassic helper.
func (r *MacAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_mac_app_store_app")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client

	// Also obtain a Pro (v1) client for the scope directory-service group
	// preflight. Same provider data and version contract; ConfigurePro returns
	// (nil, nil) during early lifecycle, leaving the preflight a no-op until the
	// provider is fully configured.
	proClient, proDiags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_mac_app_store_app")
	resp.Diagnostics.Append(proDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if proClient != nil {
		r.ldapSearcher = proClient
	}
}

// ImportState handles import by the Jamf Pro app ID.
func (r *MacAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ModifyPlan runs the plan-time directory-service user-group preflight on the
// scope limitations/exclusions: each directory_service_user_group_names entry
// is matched against the tenant's configured LDAP / cloud-IdP, surfacing an
// unknown group as a clear plan error instead of the opaque apply-time 409
// ("Problem matching limitation user group"). Best-effort: a search transport
// error or an unconfigured directory downgrades to a warning. No-op on destroy
// (null plan) and when no scope groups are declared.
func (r *MacAppResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.ldapSearcher == nil || req.Plan.Raw.IsNull() {
		return
	}

	var plan MacAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || plan.Scope == nil {
		return
	}

	scopeRoot := path.Root("scope")
	if plan.Scope.Limitations != nil {
		resp.Diagnostics.Append(scope.ValidateDirectoryServiceUserGroupNames(
			ctx, r.ldapSearcher, plan.Scope.Limitations.DirectoryServiceUserGroupNames,
			scopeRoot.AtName("limitations").AtName("directory_service_user_group_names"),
		)...)
	}
	if plan.Scope.Exclusions != nil {
		resp.Diagnostics.Append(scope.ValidateDirectoryServiceUserGroupNames(
			ctx, r.ldapSearcher, plan.Scope.Exclusions.DirectoryServiceUserGroupNames,
			scopeRoot.AtName("exclusions").AtName("directory_service_user_group_names"),
		)...)
	}

	// Granular-ownership visibility: undeclared scope categories are preserved
	// silently on apply (read-merge-write), so surface any that currently have
	// members configured outside Terraform. Update plans only (state exists),
	// best-effort — a read failure never blocks the plan.
	if r.client != nil && !req.State.Raw.IsNull() {
		var state MacAppResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if state.ID.IsNull() || state.ID.ValueString() == "" {
			return
		}
		current, err := r.client.GetMacApplicationByID(ctx, state.ID.ValueString())
		if err != nil || current == nil || current.Scope == nil {
			if err != nil {
				tflog.Debug(ctx, "skipping co-managed scope check: read failed", map[string]any{"error": err.Error()})
			}
			return
		}
		serverScope := &scope.ComputerScopeModelNoIbeacons{}
		flattenMacAppScope(ctx, current.Scope, serverScope, true)
		scope.WarnUnmanagedCategories(&resp.Diagnostics, scopeRoot,
			scope.UnmanagedComputerScopeNoIbeaconsCategories(plan.Scope, serverScope))
	}
}

// optComputedString returns an Optional+Computed StringAttribute with the
// UseNonNullStateForUnknown plan modifier for server-augmented fields. Used
// both at top level and inside SetNested list elements — see the policy
// resource doc comment for why UseNonNullStateForUnknown (not UseStateForUnknown)
// is required for nested-list growth.
func optComputedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
	}
}

// optComputedBool is the bool sibling of optComputedString.
func optComputedBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
	}
}

// computedInt64 returns a Computed-only Int64Attribute for server-derived
// values (VPP license counts).
func computedInt64(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{
		MarkdownDescription: desc,
		Computed:            true,
		PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	}
}
