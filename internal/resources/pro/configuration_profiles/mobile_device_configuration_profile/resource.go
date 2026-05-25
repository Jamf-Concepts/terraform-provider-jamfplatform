// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

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
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required. Empty:
// the classic /mobiledeviceconfigurationprofiles endpoint predates the
// provider's declared floor.
const minJamfProVersion = ""

// Resource implements jamfplatform_pro_mobile_device_configuration_profile.
type Resource struct {
	client *proclassic.Client
}

var (
	_ resource.Resource                = &Resource{}
	_ resource.ResourceWithImportState = &Resource{}
	_ resource.ResourceWithIdentity    = &Resource{}
)

const (
	defaultCreateTimeout = 90 * time.Second
	defaultReadTimeout   = 90 * time.Second
	defaultUpdateTimeout = 90 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewResource returns a new Resource instance.
func NewResource() resource.Resource {
	return &Resource{}
}

// Metadata sets the Terraform type name.
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_mobile_device_configuration_profile"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *Resource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro mobile device configuration profile ID.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a mobile device configuration profile in Jamf Pro via the `/api/proclassic/tenant/{tenantId}/mobiledeviceconfigurationprofiles` endpoint. The `general.payloads` attribute carries the raw `.mobileconfig` plist XML; the provider suppresses diffs produced by Jamf Pro's standard server-side normalisations while still surfacing drift on values the user has authored.\n\nPayload diff suppression is identical to `jamfplatform_pro_macos_configuration_profile` — see that resource's documentation for the full diff-class catalogue.\n\n**Update behaviour** — the provider re-applies the existing top-level `PayloadUUID` and `PayloadIdentifier` from state into every user-supplied payload before PUT, preserving the profile's identity across updates so connected devices do not treat each update as a fresh profile installation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Profile ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"general": schema.SingleNestedAttribute{
				MarkdownDescription: "Profile general settings. `name` and `payloads` are required.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Profile ID under <general> — server-derived. Matches the top-level `id`.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Display name of the profile. Must be unique within the tenant.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"description": optComputedString("Free-text description shown in the Jamf Pro admin UI."),
					"level": schema.StringAttribute{
						MarkdownDescription: "Profile delivery level. UI-canonical values: `Device Level` (default) / `User Level`. Wire field `<level>`: the classic API accepts `Device`/`User` on write and reports `System`/`User` on read. The provider translates at the boundary so the Terraform-facing value mirrors the admin UI dropdown.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf(validLevels...),
						},
					},
					"distribution_method": schema.StringAttribute{
						MarkdownDescription: "How the profile reaches devices. `Install Automatically` pushes via MDM; `Make Available in Self Service` lists the profile in Self Service.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf(validDistributionMethods...),
						},
					},
					"redeploy_on_update": schema.StringAttribute{
						MarkdownDescription: "Re-deploy behaviour when the profile is updated. Valid values: `Newly Assigned` / `All`. **Note**: Jamf Pro's classic API always returns `Newly Assigned` on read, even after a successful PUT with `All`. The provider treats this as write-only — once the user has authored a value the wire response is ignored so subsequent refreshes do not snap state back to `Newly Assigned`.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"redeploy_days_before_certificate_expires": schema.Int64Attribute{
						MarkdownDescription: "Number of days before a certificate in the profile expires to trigger redeployment. `0` disables certificate-expiry redeployment.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
					},
					"uuid": schema.StringAttribute{
						MarkdownDescription: "Profile UUID — minted by Jamf Pro on creation and propagated as the top-level `PayloadUUID`. Server-derived; cannot be set by the user.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"payloads": schema.StringAttribute{
						MarkdownDescription: "The mobileconfig plist XML carrying the configuration the profile delivers. The provider's diff suppression treats Jamf Pro's server-side normalisations as no-ops.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							payloadsDiffSuppressor(),
						},
					},
					"category_id": schema.StringAttribute{
						MarkdownDescription: "Jamf Pro category ID. Use `-1` (default) for \"no category\".",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"category_name": schema.StringAttribute{
						MarkdownDescription: "Category display name returned by Jamf Pro. Server-derived.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"site_id": schema.StringAttribute{
						MarkdownDescription: "Jamf Pro site ID. Use `-1` (default) for \"no site\".",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"site_name": schema.StringAttribute{
						MarkdownDescription: "Site display name returned by Jamf Pro. Server-derived.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},
			"scope": schema.SingleNestedAttribute{
				MarkdownDescription: "Profile scope. `all_mobile_devices = true` forbids per-device / per-group / per-building / per-department targets. `all_jss_users = true` forbids per-user / per-user-group targets. Wire elements are `<jss_users>` / `<jss_user_groups>`; the provider exposes them as `user_ids` / `user_group_ids`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"all_mobile_devices": schema.BoolAttribute{
						MarkdownDescription: "Scope to every mobile device in the tenant.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						Validators: []validator.Bool{
							scope.AllFlagConflictsWith(
								path.MatchRelative().AtParent().AtName("mobile_device_ids"),
								path.MatchRelative().AtParent().AtName("mobile_device_group_ids"),
								path.MatchRelative().AtParent().AtName("building_ids"),
								path.MatchRelative().AtParent().AtName("department_ids"),
							),
						},
					},
					"all_jss_users": schema.BoolAttribute{
						MarkdownDescription: "Scope to every Jamf Pro user in the tenant.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						Validators: []validator.Bool{
							scope.AllFlagConflictsWith(
								path.MatchRelative().AtParent().AtName("user_ids"),
								path.MatchRelative().AtParent().AtName("user_group_ids"),
							),
						},
					},
					"mobile_device_ids":       scope.IDSetAttribute("mobile device"),
					"mobile_device_group_ids": scope.IDSetAttribute("mobile device group"),
					"building_ids":            scope.IDSetAttribute("building"),
					"department_ids":          scope.IDSetAttribute("department"),
					"user_ids":                scope.IDSetAttribute("user"),
					"user_group_ids":          scope.IDSetAttribute("user group"),
					"limitations": schema.SingleNestedAttribute{
						MarkdownDescription: "Scope limitations narrow the audience after the targets resolve.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"network_segment_ids":                   scope.IDSetAttribute("network segment"),
							"ibeacon_ids":                           scope.IDSetAttribute("iBeacon"),
							"directory_service_or_local_user_names": scope.NameSetAttribute("directory service or local user"),
							"directory_service_user_group_names":    scope.NameSetAttribute("directory service user group"),
						},
					},
					"exclusions": schema.SingleNestedAttribute{
						MarkdownDescription: "Scope exclusions remove items that would otherwise be included by targets or limitations.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"mobile_device_ids":                     scope.IDSetAttribute("mobile device"),
							"mobile_device_group_ids":               scope.IDSetAttribute("mobile device group"),
							"building_ids":                          scope.IDSetAttribute("building"),
							"department_ids":                        scope.IDSetAttribute("department"),
							"user_ids":                              scope.IDSetAttribute("user"),
							"user_group_ids":                        scope.IDSetAttribute("user group"),
							"network_segment_ids":                   scope.IDSetAttribute("network segment"),
							"ibeacon_ids":                           scope.IDSetAttribute("iBeacon"),
							"directory_service_or_local_user_names": scope.NameSetAttribute("directory service or local user"),
							"directory_service_user_group_names":    scope.NameSetAttribute("directory service user group"),
						},
					},
				},
			},
			"self_service": schema.SingleNestedAttribute{
				MarkdownDescription: "Self Service integration. Only meaningful when `general.distribution_method = \"Make Available in Self Service\"`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"self_service_description": optComputedString("Description shown in Self Service."),
					"feature_on_main_page":     optComputedBool("Feature the profile on the Self Service main page."),
					"removal_disallowed": schema.StringAttribute{
						MarkdownDescription: "Removal-by-end-user policy. Valid values: `Never`, `Always`, `With Authorization`. Wire field `<security><removal_disallowed>`.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf(validRemovalDisallowedValues...),
						},
					},
					"authorization_password": schema.StringAttribute{
						MarkdownDescription: "Authorization password required to remove the profile. Only effective when `removal_disallowed = \"With Authorization\"`. Jamf Pro returns the value in plaintext on read — stored in Terraform state and masked in plan/apply output.",
						Optional:            true,
						Sensitive:           true,
					},
					"categories": schema.ListNestedAttribute{
						MarkdownDescription: "Categories under which the profile appears in Self Service.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":   schema.StringAttribute{MarkdownDescription: "Category ID.", Required: true},
								"name": optComputedStringInList("Category display name (server-derived)."),
							},
						},
					},
				},
			},
			"timeouts": timeouts.Attributes(context.Background(), timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// Configure wires the Jamf ProClassic client into the resource.
func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_mobile_device_configuration_profile")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import by Jamf Pro profile ID.
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func optComputedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func optComputedBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
	}
}

func optComputedStringInList(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
	}
}

// _ tracks the types import — the model fields use it implicitly.
var _ = types.StringNull
