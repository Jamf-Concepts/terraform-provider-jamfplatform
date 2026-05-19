// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package category implements the jamfplatform_pro_category resource, data source, and
// list resource backed by the Jamf Pro categories API.
package category

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

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this resource.
// Empty string skips the version check — the categories endpoint has been stable since
// well before the provider's overall floor (11.0.0), so no per-resource gate is needed.
const minJamfProVersion = ""

// CategoryResource implements the Terraform resource for Jamf Pro categories.
type CategoryResource struct {
	client *pro.Client
}

var _ resource.Resource = &CategoryResource{}
var _ resource.ResourceWithImportState = &CategoryResource{}
var _ resource.ResourceWithIdentity = &CategoryResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewCategoryResource returns a new instance of CategoryResource.
func NewCategoryResource() resource.Resource {
	return &CategoryResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *CategoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_category"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *CategoryResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro category ID used to uniquely reference the category.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the category resource.
func (r *CategoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro category. Categories group policies, scripts, packages, and other Jamf Pro objects.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Category ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Category display name. Must not be empty.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "Sort priority, 1–20. Lower numbers sort first.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 20),
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

// Configure wires the Jamf Pro client into the resource, fetches the tenant Jamf Pro version
// (cached via sync.Once on providerdata.Data so it fires at most once per terraform invocation),
// runs the per-resource version gate when minJamfProVersion is set, and surfaces the
// provider-floor advisory warning when applicable.
func (r *CategoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = pro.New(pd.Client)

	version, err := pd.GetJamfProVersion(ctx)
	if err != nil {
		if minJamfProVersion == "" {
			return
		}
		resp.Diagnostics.AddError(
			"Failed to read Jamf Pro tenant version",
			fmt.Sprintf("jamfplatform_pro_category requires Jamf Pro; could not read version: %s", err),
		)
		return
	}
	if minJamfProVersion != "" {
		resp.Diagnostics.Append(
			helpers.RequireMinJamfProVersion(version, minJamfProVersion, "jamfplatform_pro_category")...,
		)
	}
	if warn := pd.MaybeProviderFloorWarning(); warn != nil {
		resp.Diagnostics.Append(warn)
	}
}

// ImportState handles import by the Jamf Pro category ID.
func (r *CategoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
