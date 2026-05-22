// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package user_group implements the jamfplatform_pro_user_group resource,
// data source, and list resource backed by the Jamf ProClassic usergroups API.
package user_group

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this resource.
// Empty: classic /usergroups predates the provider's overall floor.
const minJamfProVersion = ""

// UserGroupResource implements the Terraform resource for Jamf Pro user groups.
type UserGroupResource struct {
	client *proclassic.Client
}

var _ resource.Resource = &UserGroupResource{}
var _ resource.ResourceWithImportState = &UserGroupResource{}
var _ resource.ResourceWithIdentity = &UserGroupResource{}
var _ resource.ResourceWithConfigValidators = &UserGroupResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewUserGroupResource returns a new instance of UserGroupResource.
func NewUserGroupResource() resource.Resource {
	return &UserGroupResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *UserGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_user_group"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *UserGroupResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro user group ID used to uniquely reference the user group.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the user group resource.
func (r *UserGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro user group (smart or static). For smart groups, supply `criteria`; the user list is server-resolved. For static groups, supply `members` (user IDs as strings); `criteria` is forbidden. The `group_type` field acts as the discriminator and triggers a replace if changed — mirrors `jamfplatform_device_group`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "User group ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "User group display name. Must be unique within the tenant.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"group_type": schema.StringAttribute{
				MarkdownDescription: "Group implementation type. Changes require resource replacement. Valid values are `static` or `smart`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("static", "smart"),
				},
			},
			"notify_on_membership_change": schema.BoolAttribute{
				MarkdownDescription: "Whether Jamf Pro emits a notification when group membership changes. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"site_id": schema.StringAttribute{
				MarkdownDescription: "Optional Jamf Pro site ID to scope the user group. Omit to leave unscoped (server sets the `NONE` site, id `-1`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(noSiteID),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_name": schema.StringAttribute{
				MarkdownDescription: "Site name reported by Jamf Pro for the assigned `site_id`. Server-derived.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"members": schema.SetAttribute{
				MarkdownDescription: "User IDs (as strings) to assign as members of a static user group. Required when `group_type = \"static\"`. Forbidden when `group_type = \"smart\"` — smart-group membership is server-resolved from `criteria`.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"member_count": schema.Int64Attribute{
				MarkdownDescription: "Total members reported by Jamf Pro.",
				Computed:            true,
			},
			"criteria": schema.ListNestedAttribute{
				MarkdownDescription: "Ordered list of criteria evaluated by Jamf Pro to determine smart-group membership. Required when `group_type = \"smart\"`. Forbidden when `group_type = \"static\"`. Order is significant — Jamf evaluates left-to-right with the supplied `and_or` joins.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"priority": schema.Int64Attribute{
							MarkdownDescription: "Evaluation order for the criterion. Defaults to the element index (zero-based) if omitted.",
							Optional:            true,
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Inventory attribute to evaluate (e.g. `User Group`, `Username`, `Full Name`, `VPP Invitation Status`).",
							Required:            true,
						},
						"search_type": schema.StringAttribute{
							MarkdownDescription: operatorDescription(),
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf(ValidOperators...),
							},
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Comparison value for the operator.",
							Required:            true,
						},
						"and_or": schema.StringAttribute{
							MarkdownDescription: "How this criterion joins to the next. Valid values are `and` or `or`. Defaults to `and` if omitted.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("and"),
							Validators: []validator.String{
								stringvalidator.OneOf("and", "or"),
							},
						},
						"has_opening_parenthesis": schema.BoolAttribute{
							MarkdownDescription: "Whether the criterion begins a parenthetical grouping.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"has_closing_parenthesis": schema.BoolAttribute{
							MarkdownDescription: "Whether the criterion ends a parenthetical grouping.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
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

// Configure wires the Jamf ProClassic client into the resource via the shared
// providerdata.ConfigureProClassic helper.
func (r *UserGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_user_group")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import by the Jamf Pro user group ID.
func (r *UserGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ConfigValidators registers the plan-time cross-field validator. The
// apply-time helper validateUserGroupPlan remains as defence-in-depth.
func (r *UserGroupResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		smartStaticConfigValidator{},
	}
}
