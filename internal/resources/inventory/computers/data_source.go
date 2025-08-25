// Copyright 2025 Jamf Software LLC.

package computers

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure DataSourceComputers implements the datasource.DataSource interface.
var _ datasource.DataSource = &DataSourceComputers{}

// NewDataSourceComputers returns a new data source instance.
func NewDataSourceComputers() datasource.DataSource {
	return &DataSourceComputers{}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DataSourceComputers) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected ProviderData type",
			"Expected *client.Client, got something else.",
		)
		return
	}
	d.client = apiClient
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DataSourceComputers) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory_computers"
}

// Schema defines the schema for the computers data source.
func (d *DataSourceComputers) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"filter": schema.StringAttribute{
				Optional:    true,
				Description: "Optional filter string to limit results (e.g., 'general.name==\"MacBook*\"')",
			},
			"computers": schema.ListAttribute{
				ElementType: types.MapType{ElemType: types.StringType},
				Computed:    true,
				Description: "List of computers.",
			},
		},
	}
}

// Read fetches the list of computers and sets the state.
func (d *DataSourceComputers) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data computersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filter := ""
	if !data.Filter.IsNull() && !data.Filter.IsUnknown() {
		filter = data.Filter.ValueString()
	}

	computers, err := d.client.GetInventoryAllComputers(ctx, filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get computers",
			fmt.Sprintf("Error retrieving computers: %s", err),
		)
		return
	}

	var computerList []map[string]types.String
	for _, comp := range computers {
		compMap := map[string]types.String{
			"id":                 types.StringValue(comp.ID),
			"udid":               types.StringValue(comp.UDID),
			"name":               types.StringValue(comp.General.Name),
			"serial_number":      types.StringValue(comp.Hardware.SerialNumber),
			"os_version":         types.StringValue(comp.OperatingSystem.Version),
			"model":              types.StringValue(comp.Hardware.Model),
			"last_enrolled_date": types.StringValue(comp.General.LastEnrolledDate),
			"last_contact_time":  types.StringValue(comp.General.LastContactTime),
		}
		computerList = append(computerList, compMap)
	}

	computersListVal, diags := types.ListValueFrom(ctx, types.MapType{ElemType: types.StringType}, computerList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue("static-id")
	data.Computers = computersListVal
	resp.State.Set(ctx, &data)
}
