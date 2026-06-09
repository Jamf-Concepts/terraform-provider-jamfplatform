// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package login_page implements the
// jamfplatform_pro_login_page_settings singleton resource and data source backed
// by the Jamf Pro login page (login-customization) settings API.
package login_page

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this resource.
// Empty: the login-customization endpoint is present at the provider's overall floor, so
// no per-resource gate is needed.
const minJamfProVersion = ""

// LoginPageSettingsResource implements the singleton resource for the Jamf Pro
// login page settings. Backed by an Update-only API (no Create/Delete on the remote):
// Create funnels into Update; Delete is a no-op that only removes the object from
// Terraform state.
type LoginPageSettingsResource struct {
	client *pro.Client
}

var _ resource.Resource = &LoginPageSettingsResource{}
var _ resource.ResourceWithImportState = &LoginPageSettingsResource{}
var _ resource.ResourceWithIdentity = &LoginPageSettingsResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewLoginPageSettingsResource returns a new instance of the resource.
func NewLoginPageSettingsResource() resource.Resource {
	return &LoginPageSettingsResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *LoginPageSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_login_page_settings"
}

// IdentitySchema defines the identifier used for import. Singleton resources accept only
// the fixed helpers.SingletonID value.
func (r *LoginPageSettingsResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Fixed singleton identifier. Always \"singleton\" — login page settings are one-per-tenant.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the login page settings resource.
func (r *LoginPageSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the Jamf Pro login page disclaimer (Settings > System > Login page). " +
			"Singleton — one record per tenant. " +
			"**The three disclaimer text fields (`disclaimer_heading`, `disclaimer_main_text`, `action_text`) are required on every write, regardless of `include_custom_disclaimer`** — Jamf Pro rejects a write that omits any of them or sends an empty string (wire-probed 2026-06-09). The custom disclaimer is only *shown* to users when `include_custom_disclaimer = true`, but the text must always be present. " +
			"**Omit = preserve** — a field you omit keeps its current Jamf Pro value, including on the first apply: this resource adopts the existing settings and only changes the fields you declare. " +
			"Import with `terraform import jamfplatform_pro_login_page_settings.<name> singleton`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Fixed singleton identifier. Always `singleton`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"include_custom_disclaimer": schema.BoolAttribute{
				MarkdownDescription: "Whether the custom disclaimer message is shown on the Jamf Pro login page (\"Include a disclaimer message\"). " +
					"The disclaimer text fields must be populated whether or not this is enabled. " +
					"Omit to leave the current value untouched; set `true`/`false` to change it.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"disclaimer_heading": schema.StringAttribute{
				MarkdownDescription: "Text used for the title of the disclaimer dialog (\"Heading\"). Maximum 20 characters; must not be empty. " +
					"Omit to leave the current value untouched.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 20),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disclaimer_main_text": schema.StringAttribute{
				MarkdownDescription: "Text used for the body of the disclaimer dialog (\"Main\"). Maximum 2,500 characters; must not be empty. " +
					"Omit to leave the current value untouched.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 2500),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"action_text": schema.StringAttribute{
				MarkdownDescription: "Text used for the button that acknowledges the disclaimer dialog (\"Action\"). Maximum 20 characters; must not be empty. " +
					"Omit to leave the current value untouched.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 20),
				},
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
func (r *LoginPageSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_login_page_settings")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import for the singleton. Only the fixed helpers.SingletonID value is
// accepted; any other identifier is rejected with a clear error so users do not accidentally
// end up with mis-keyed state that the resource silently normalizes on the next Read.
//
//	terraform import jamfplatform_pro_login_page_settings.<name> singleton
func (r *LoginPageSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != helpers.SingletonID {
		resp.Diagnostics.AddError(
			"Invalid singleton import identifier",
			fmt.Sprintf(
				"jamfplatform_pro_login_page_settings is a singleton resource and must be imported with id %q. Got %q.",
				helpers.SingletonID, req.ID,
			),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
