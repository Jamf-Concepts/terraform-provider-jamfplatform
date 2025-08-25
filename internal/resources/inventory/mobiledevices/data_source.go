// Copyright 2025 Jamf Software LLC.

package mobiledevices

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSourceMobileDevices{}

// NewDataSourceMobileDevices returns a new instance of DataSourceMobileDevices.
func NewDataSourceMobileDevices() datasource.DataSource {
	return &DataSourceMobileDevices{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DataSourceMobileDevices) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory_mobile_devices"
}

// Schema sets the data source schema for the mobile devices.
func (d *DataSourceMobileDevices) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

// Configure sets up the API client for the data source from the provider configuration.
func (d *DataSourceMobileDevices) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Read fetches the mobile devices and sets the data source state.
func (d *DataSourceMobileDevices) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MobileDevicesDataSourceModel

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

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
