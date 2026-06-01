// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package api_client implements the jamfplatform_pro_api_client resource, data
// source, and list resource backed by the Jamf Pro API Integrations API
// (`/api/v1/api-integrations`). The Jamf Pro admin UI calls these "API clients"
// (Settings → System → API roles and clients), so the Terraform construct uses
// that name.
package api_client

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty string skips the version check — the API Integrations
// endpoints (Jamf Pro 10.49+) predate the provider's overall floor (11.0.0).
const minJamfProVersion = ""

// ApiClientResource implements the Terraform resource for Jamf Pro API clients.
type ApiClientResource struct {
	client *pro.Client
}

var _ resource.Resource = &ApiClientResource{}
var _ resource.ResourceWithImportState = &ApiClientResource{}
var _ resource.ResourceWithIdentity = &ApiClientResource{}
var _ resource.ResourceWithModifyPlan = &ApiClientResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewApiClientResource returns a new instance of ApiClientResource.
func NewApiClientResource() resource.Resource {
	return &ApiClientResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *ApiClientResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_api_client"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *ApiClientResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro API client ID used to uniquely reference the client.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the API client resource.
func (r *ApiClientResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro API client (Settings → System → API roles and clients). An API client authenticates to the Jamf Pro API using the OAuth client-credentials grant; its privileges come from the `jamfplatform_pro_api_role`s assigned in `api_roles`.\n\n" +
			"**Client secret lifecycle:** Jamf Pro generates the client secret and returns it only once, at generation time — it can never be read back afterwards. Set `credential_rotation` to generate a secret (the client must be `enabled`); change that value to rotate it. The generated `client_secret` is stored — `Sensitive` — in Terraform state so dependent resources can consume it. Disabling the client (`enabled = false`) revokes its credentials in Jamf Pro.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "API client ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "**\"Display name\"** in the Jamf Pro admin UI. API client display name. Must be unique across API clients; editable in place.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			// Wire element: authorizationScopes — renamed to mirror the "API roles" UI label.
			"api_roles": schema.SetAttribute{
				MarkdownDescription: "**\"API roles\"** in the Jamf Pro admin UI. The set of `jamfplatform_pro_api_role` **display names** assigned to this client; the client's effective privileges are the union of the assigned roles' privileges. Reference roles by their `display_name`, e.g. `[jamfplatform_pro_api_role.example.display_name]`. Unknown role names are rejected by Jamf Pro when you apply.",
				ElementType:         types.StringType,
				Required:            true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "**\"Enable/disable API client\"** in the Jamf Pro admin UI. Whether the client may authenticate. Defaults to `false` when omitted. Disabling the client revokes its client credentials in Jamf Pro (`app_type` reverts to `NONE` and the stored `client_secret` is cleared on the next read).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"access_token_lifetime_seconds": schema.Int64Attribute{
				MarkdownDescription: "**\"Access token lifetime\"** in the Jamf Pro admin UI. The lifetime, in seconds, of access tokens issued to this client. Defaults to `300` when omitted.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "**\"Client ID\"** in the Jamf Pro admin UI. The OAuth client identifier assigned by Jamf Pro at creation. Stable for the life of the client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_type": schema.StringAttribute{
				MarkdownDescription: "Returned by Jamf Pro; not user-settable. `NONE` until a client secret has been generated, then `CLIENT_CREDENTIALS`. Reverts to `NONE` when the client is disabled.",
				Computed:            true,
			},
			"credential_rotation": schema.StringAttribute{
				MarkdownDescription: "Rotation trigger for the OAuth client secret. Set this (any value) to mint a `client_secret` — the client must be `enabled`. Change the value to rotate the secret (Jamf Pro mints a fresh secret and invalidates the previous one). Leaving it unset means no secret is generated. Mirrors the admin UI's \"Rotate client secret\" action.",
				Optional:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The OAuth client secret. Generated by Jamf Pro when `credential_rotation` is set, and **never readable from Jamf Pro afterwards** — its value is therefore stored (`Sensitive`) in Terraform state. `null` until a secret is generated, and cleared if the client is disabled.",
				Computed:            true,
				Sensitive:           true,
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
func (r *ApiClientResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_api_client")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ModifyPlan enforces the credential_rotation⇒enabled invariant and manages the
// planned value of the sticky, server-minted client_secret:
//   - credential_rotation set while enabled=false → plan error (server would
//     reject the rotation).
//   - enabled planned false → client_secret planned null: disabling the client
//     revokes its credentials server-side, so predict the clear to avoid a
//     "provider produced inconsistent result after apply" error.
//   - credential_rotation changed to a new value → client_secret planned unknown
//     so Update re-mints it.
//   - otherwise the attribute-level UseStateForUnknown keeps the stored secret.
func (r *ApiClientResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan ApiClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateCredentialRotationRequiresEnabled(plan.CredentialRotation, plan.Enabled, path.Root("enabled"))...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create: no prior state — client_secret stays unknown and Create sets it.
	if req.State.Raw.IsNull() {
		return
	}
	var state ApiClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch {
	case !plan.Enabled.IsUnknown() && !plan.Enabled.ValueBool():
		// Disabling revokes the secret server-side.
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("client_secret"), types.StringNull())...)
	case shouldRotateCredentials(plan.CredentialRotation, state.CredentialRotation):
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("client_secret"), types.StringUnknown())...)
	}
}

// ImportState handles import by the Jamf Pro API client ID.
func (r *ApiClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
