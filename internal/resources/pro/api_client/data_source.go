// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// ApiClientDataSource implements the Terraform data source for Jamf Pro API clients.
type ApiClientDataSource struct {
	client *pro.Client
}

var _ datasource.DataSource = &ApiClientDataSource{}

// NewApiClientDataSource returns a new instance of ApiClientDataSource.
func NewApiClientDataSource() datasource.DataSource {
	return &ApiClientDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *ApiClientDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_api_client"
}

// Schema returns the data source schema. The client secret is never exposed —
// Jamf Pro never returns it on read.
func (d *ApiClientDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a Jamf Pro API client by ID. The client secret is never exposed — Jamf Pro does not return it on read.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "API client ID to look up.",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "API client display name.",
				Computed:            true,
			},
			"api_roles": schema.SetAttribute{
				MarkdownDescription: "The set of API role display names assigned to this client.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the client may authenticate.",
				Computed:            true,
			},
			"access_token_lifetime_seconds": schema.Int64Attribute{
				MarkdownDescription: "The lifetime, in seconds, of access tokens issued to this client.",
				Computed:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The OAuth client identifier assigned by Jamf Pro.",
				Computed:            true,
			},
			"app_type": schema.StringAttribute{
				MarkdownDescription: "Returned by Jamf Pro; not user-settable (`NONE` or `CLIENT_CREDENTIALS`).",
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

// Configure wires the Jamf Pro client into the data source via the shared
// providerdata.ConfigurePro helper.
func (d *ApiClientDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_api_client")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

// Read fetches an API client by ID and populates Terraform state.
func (d *ApiClientDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	var data ApiClientDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() || data.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "The id attribute must be provided to read a Jamf Pro API client.")
		return
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, data.Timeouts.IsNull(), data.Timeouts.IsUnknown(), defaultReadTimeout, data.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	got, err := d.client.GetApiIntegrationV1(readCtx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to find Jamf Pro API client", err.Error())
		return
	}
	resp.Diagnostics.Append(assignApiClientDataSourceModel(ctx, &data, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read Jamf Pro API client data source", map[string]any{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
