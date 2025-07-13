// Copyright 2025 Jamf Software LLC.

package mobiledevices

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/shared"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataSourceMobileDevices defines the data source implementation.
type DataSourceMobileDevices struct {
	client *client.Client
}

// mobileDevicesDataSourceModel maps the data source schema data.
type mobileDevicesDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Section types.List   `tfsdk:"section"`
	Devices types.List   `tfsdk:"devices"`
}

// Ensure DataSourceMobileDevices implements the datasource.DataSource interface.
var _ datasource.DataSource = &DataSourceMobileDevices{}

// NewDataSourceMobileDevices returns a new data source instance.
func NewDataSourceMobileDevices() datasource.DataSource {
	return &DataSourceMobileDevices{}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DataSourceMobileDevices) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerClients, ok := req.ProviderData.(*shared.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected ProviderData type",
			"Expected *shared.ProviderClients, got something else.",
		)
		return
	}
	if providerClients.Inventory == nil {
		resp.Diagnostics.AddError(
			"Inventory API client not configured",
			"The provider's inventory block is missing or incomplete. Please provide valid credentials.",
		)
		return
	}
	d.client = providerClients.Inventory
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DataSourceMobileDevices) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory_mobile_devices"
}

// Schema sets the data source schema for the mobile devices.
func (d *DataSourceMobileDevices) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"section": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of sections to include in the response (e.g., GENERAL, HARDWARE, etc.)",
			},
			"devices": schema.ListAttribute{
				ElementType: types.MapType{ElemType: types.StringType},
				Computed:    true,
				Description: "List of mobile devices.",
			},
		},
	}
}

// Read fetches the mobile devices and sets the data source state.
func (d *DataSourceMobileDevices) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data mobileDevicesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sections []string
	if !data.Section.IsNull() && !data.Section.IsUnknown() {
		var sectionVals []types.String
		data.Section.ElementsAs(ctx, &sectionVals, false)
		for _, s := range sectionVals {
			if !s.IsNull() && !s.IsUnknown() {
				sections = append(sections, s.ValueString())
			}
		}
	}

	devices, err := d.client.GetInventoryAllMobileDevices(ctx, sections)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get mobile devices",
			fmt.Sprintf("Error retrieving mobile devices: %s", err),
		)
		return
	}

	var deviceList []map[string]types.String
	for _, dev := range devices {
		devMap := map[string]types.String{
			"mobile_device_id":           types.StringValue(dev.MobileDeviceId),
			"device_type":                types.StringValue(dev.DeviceType),
			"udid":                       types.StringValue(dev.General.Udid),
			"display_name":               types.StringValue(dev.General.DisplayName),
			"serial_number":              types.StringValue(dev.Hardware.SerialNumber),
			"os_version":                 types.StringValue(dev.General.OsVersion),
			"last_inventory_update_date": types.StringValue(dev.General.LastInventoryUpdateDate),
			"last_enrolled_date":         types.StringValue(dev.General.LastEnrolledDate),
			"model":                      types.StringValue(dev.Hardware.Model),
		}
		deviceList = append(deviceList, devMap)
	}

	devicesListVal, diags := types.ListValueFrom(ctx, types.MapType{ElemType: types.StringType}, deviceList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue("static-id")
	data.Devices = devicesListVal
	resp.State.Set(ctx, &data)
}
