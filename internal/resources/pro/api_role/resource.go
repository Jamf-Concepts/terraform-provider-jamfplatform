// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package api_role implements the jamfplatform_pro_api_role resource, data
// source, and list resource backed by the Jamf Pro API Roles API.
package api_role

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty string skips the version check — the API Roles endpoints
// (Jamf Pro 10.49+) predate the provider's overall floor (11.0.0), so no
// per-resource gate is needed.
const minJamfProVersion = ""

// ApiRoleResource implements the Terraform resource for Jamf Pro API roles.
type ApiRoleResource struct {
	client *pro.Client
}

var _ resource.Resource = &ApiRoleResource{}
var _ resource.ResourceWithImportState = &ApiRoleResource{}
var _ resource.ResourceWithIdentity = &ApiRoleResource{}
var _ resource.ResourceWithModifyPlan = &ApiRoleResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewApiRoleResource returns a new instance of ApiRoleResource.
func NewApiRoleResource() resource.Resource {
	return &ApiRoleResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *ApiRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_api_role"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *ApiRoleResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro API role ID used to uniquely reference the role.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the API role resource.
func (r *ApiRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro API role (Settings → System → API roles and clients). An API role is a named set of privileges; assign roles to a `jamfplatform_pro_api_client` (by `display_name`) to grant it those privileges.\n\nJamf Pro refuses to delete a role while it is still assigned to an API client. If you are both removing a role from a client and deleting the role, remove it from the client's `api_roles` first (a separate apply) so the role is no longer in use when it is destroyed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "API role ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "**\"Display name\"** in the Jamf Pro admin UI. Role display name. Must be unique across API roles; editable in place.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"privileges": schema.SetAttribute{
				MarkdownDescription: "**\"Privileges\"** in the Jamf Pro admin UI. The set of Jamf Pro privilege strings granted by this role (e.g. `Read Computers`, `Create API Roles`). Validated at plan time against the tenant's live privilege list (use the `jamfplatform_pro_api_role_privileges` data source to discover valid values); the valid set varies by Jamf Pro version. Order is not significant.",
				ElementType:         types.StringType,
				Required:            true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
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
func (r *ApiRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_api_role")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ModifyPlan runs the plan-time privilege preflight, surfacing an unknown
// privilege as a clear plan error instead of waiting for the apply-time 400
// ("privilege(s) are not valid [...]"). Best-effort: a transport error or
// unconfigured client downgrades to a warning. No-op on destroy.
func (r *ApiRoleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil || req.Plan.Raw.IsNull() {
		return
	}
	var plan ApiRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validatePrivileges(ctx, r.client, plan.Privileges, path.Root("privileges"))...)
}

// ImportState handles import by the Jamf Pro API role ID.
func (r *ApiRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
