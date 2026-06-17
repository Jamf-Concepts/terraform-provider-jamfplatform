// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devicegroups"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DeviceGroupResource implements the Terraform resource for Jamf device groups.
//
// Although this resource targets the Platform Services Device Group Inventory
// API, it also issues a single Pro `/v2/groups/{id}` call after every successful
// Get/Create/Update so the computed `jamf_pro_id` attribute can be populated.
// That bridges Platform UUIDs to the numeric Jamf Pro classic ID required by
// classic-API scope blocks (policies, configuration profiles, restricted
// software). The Pro lookup is best-effort: 403 nulls the attribute and emits a
// one-shot warning; 404/Pro-less tenants null the attribute silently.
type DeviceGroupResource struct {
	client    *devicegroups.Client
	proClient *pro.Client
	pd        *providerdata.Data
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DeviceGroupResource{}
var _ resource.ResourceWithImportState = &DeviceGroupResource{}
var _ resource.ResourceWithIdentity = &DeviceGroupResource{}
var _ resource.ResourceWithModifyPlan = &DeviceGroupResource{}

const (
	defaultCreateTimeout = 120 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewDeviceGroupResource returns a new instance of DeviceGroupResource.
func NewDeviceGroupResource() resource.Resource {
	return &DeviceGroupResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *DeviceGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_group"
}

// IdentitySchema defines the unique identifier for device group resources.
func (r *DeviceGroupResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Device group ID used to uniquely reference Jamf device groups.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the device group resource.
func (r *DeviceGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Manages Jamf device groups (static or smart) via the Platform API. Requires **Device Group Inventory API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier assigned by the API.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"jamf_pro_id": schema.StringAttribute{
				MarkdownDescription: "Numeric Jamf Pro classic ID for the group, resolved from the `/api/pro/v2/tenant/{tenantId}/groups` endpoint. Used to reference the group from classic-API scope blocks (policies, configuration profiles, restricted software). Null when the Platform API client lacks the `Read Groups` privilege (a single missing-privilege warning surfaces during plan), when the group cannot be located in Jamf Pro, or when the bridging call transiently fails.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name for the device group.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional Description for the device group.",
				Optional:            true,
			},
			"device_type": schema.StringAttribute{
				MarkdownDescription: "Jamf device type. Changes require resource replacement. Valid values are `computer` or `mobile`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("computer", "mobile"),
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
			"members": schema.SetAttribute{
				MarkdownDescription: "Optional device IDs to manage for static groups. When omitted, the provider leaves membership unchanged. Ignored for smart groups.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"member_count": schema.Int64Attribute{
				MarkdownDescription: "Total members reported by the API.",
				Computed:            true,
			},
			"criteria": schema.ListNestedAttribute{
				MarkdownDescription: "Smart-group criteria evaluated by the Jamf inventory service. Ordered: criteria are evaluated and rendered in the order listed.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"order": schema.Int64Attribute{
							MarkdownDescription: "Execution order for the criterion. Defaults to the element index if omitted.",
							Optional:            true,
						},
						"criteria": schema.StringAttribute{
							MarkdownDescription: "Inventory attribute to evaluate.",
							Required:            true,
						},
						"operator": schema.StringAttribute{
							MarkdownDescription: criteria.Description(criteria.Operators),
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf(criteria.Operators...),
							},
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Optional comparison value used by the operator.",
							Required:            true,
						},
						"and_or": schema.StringAttribute{
							MarkdownDescription: "How this criterion joins to the next. Valid values are `and` or `or`. Defaults to `and` if omitted.",
							Computed:            true,
							Optional:            true,
							Default:             stringdefault.StaticString("and"),
							Validators: []validator.String{
								stringvalidator.OneOf("and", "or"),
							},
						},
						"has_opening_parenthesis": schema.BoolAttribute{
							MarkdownDescription: "Whether the criterion begins a parenthetical grouping.",
							Optional:            true,
						},
						"has_closing_parenthesis": schema.BoolAttribute{
							MarkdownDescription: "Whether the criterion ends a parenthetical grouping.",
							Optional:            true,
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

// Configure sets up the API client for the resource from the provider configuration.
func (r *DeviceGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pd, ok := req.ProviderData.(*providerdata.Data)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providerdata.Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = devicegroups.New(pd.Client)
	r.proClient = pro.New(pd.Client)
	r.pd = pd
}

// ImportState handles the import of existing Device Group resources.
func (r *DeviceGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
