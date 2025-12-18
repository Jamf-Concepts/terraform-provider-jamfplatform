// Copyright 2025 Jamf Software LLC.

package device

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DeviceDataSource{}

// NewDeviceDataSource returns a new instance of DeviceDataSource.
func NewDeviceDataSource() datasource.DataSource {
	return &DeviceDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DeviceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

// Schema defines the data source schema.
func (d *DeviceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lookup a Jamf device by ID or serial number via the Device Inventory API. Requires **Device Inventory API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Optional device UUID (Jamf Pro Management ID) to query.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("serial_number"),
					),
				},
			},
			"serial_number": schema.StringAttribute{
				MarkdownDescription: "Optional device serial number to query (case-sensitive).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("id"),
					),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Device name reported by inventory.",
				Computed:            true,
			},
			"model": schema.StringAttribute{
				MarkdownDescription: "Marketing model string.",
				Computed:            true,
			},
			"model_identifier": schema.StringAttribute{
				MarkdownDescription: "Model identifier (e.g., Mac14,6).",
				Computed:            true,
			},
			"last_inventory_update_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last inventory update in ISO 8601 format.",
				Computed:            true,
			},
			"last_check_in_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last check-in in ISO 8601 format.",
				Computed:            true,
			},
			"operating_system_version": schema.StringAttribute{
				MarkdownDescription: "Operating system version string.",
				Computed:            true,
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "User ID associated with the device, if any.",
				Computed:            true,
			},
			"enrollment_type": schema.StringAttribute{
				MarkdownDescription: "Enrollment type reported by the API.",
				Computed:            true,
			},
			"last_enrollment_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last enrollment in ISO 8601 format.",
				Computed:            true,
			},
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DeviceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches a device by ID or serial number and populates the Terraform state.
func (d *DeviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeviceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure the provider block is set up correctly.",
		)
		return
	}

	lookupID := ""
	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		lookupID = data.ID.ValueString()
	}

	lookupSerial := ""
	if !data.SerialNumber.IsNull() && !data.SerialNumber.IsUnknown() {
		lookupSerial = data.SerialNumber.ValueString()
	}

	if lookupID == "" && lookupSerial == "" {
		resp.Diagnostics.AddError(
			"Missing query arguments",
			"Either id or serial_number must be provided to read a device.",
		)
		return
	}

	var filter string
	if lookupID != "" {
		filter = fmt.Sprintf(`id=="%s"`, escapeFilterValue(lookupID))
	} else {
		filter = fmt.Sprintf(`serialNumber=="%s"`, escapeFilterValue(lookupSerial))
	}

	devices, err := d.client.GetDevicesV1(ctx, nil, filter)
	if err != nil {
		resp.Diagnostics.AddError("Unable to search devices", err.Error())
		return
	}

	switch len(devices) {
	case 0:
		resp.Diagnostics.AddError("Device not found", fmt.Sprintf("No devices matched filter %q", filter))
		return
	case 1:
	default:
		resp.Diagnostics.AddError("Multiple devices matched", fmt.Sprintf("Filter %q returned %d devices; please refine your query", filter, len(devices)))
		return
	}

	device := devices[0]
	data.ID = types.StringValue(device.ID)
	data.SerialNumber = stringValueOrNull(device.SerialNumber)
	data.Name = stringValueOrNull(device.Name)
	data.Model = stringValueOrNull(device.Model)
	data.ModelIdentifier = stringValueOrNull(device.ModelIdentifier)
	data.LastInventoryUpdate = stringValueOrNull(device.LastInventoryUpdateTime)
	data.LastCheckIn = stringPointerValueOrNull(device.LastCheckInTime)
	data.OperatingSystemVersion = stringValueOrNull(device.OperatingSystemVersion)
	data.UserID = stringPointerValueOrNull(device.UserID)
	data.EnrollmentType = stringValueOrNull(device.EnrollmentType)
	data.LastEnrollmentTime = stringValueOrNull(device.LastEnrollmentTime)

	tflog.Trace(ctx, "read device data source", map[string]interface{}{
		"id": device.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
