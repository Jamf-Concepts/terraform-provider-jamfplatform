// Copyright 2025 Jamf Software LLC.

package devices

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const defaultDevicesReadTimeout = 90 * time.Second

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DevicesDataSource{}

// NewDevicesDataSource returns a new instance of DevicesDataSource.
func NewDevicesDataSource() datasource.DataSource {
	return &DevicesDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *DevicesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_devices"
}

// Schema defines the data source schema.
func (d *DevicesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List Jamf devices via the Device Inventory API. Requires **Device Inventory API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal identifier for this data source read.",
				Computed:            true,
			},
			"devices": schema.ListNestedAttribute{
				MarkdownDescription: "Devices that matched the optional filter.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Device UUID (Jamf Pro Management ID).",
							Computed:            true,
						},
						"serial_number": schema.StringAttribute{
							MarkdownDescription: "Device serial number (case-sensitive).",
							Computed:            true,
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
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.ListNestedBlock{
				MarkdownDescription: "Declarative RSQL filter clauses. Each block represents one selector/operator/argument clause.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"selector": schema.StringAttribute{
							MarkdownDescription: "RSQL selector. Valid values are `id`, `name`, `model`, `modelIdentifier`, `serialNumber`, `lastInventoryUpdateTime`, `lastCheckInTime`, `operatingSystemVersion`, `userId`, `enrollmentType`, and `lastEnrollmentTime`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("id", "name", "model", "modelIdentifier", "serialNumber", "lastInventoryUpdateTime", "lastCheckInTime", "operatingSystemVersion", "userId", "enrollmentType", "lastEnrollmentTime"),
							},
						},
						"operator": schema.StringAttribute{
							MarkdownDescription: "RSQL comparison operator. Valid values are `==`, `!=`, `=in=`, `=out=`, `>`, `<`, `>=`, and `<=`. Defaults to `==` when omitted.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("==", "!=", "=in=", "=out=", ">", "<", ">=", "<="),
							},
						},
						"argument": schema.StringAttribute{
							MarkdownDescription: "RSQL argument portion for the selector/operator. Provide the value exactly as required by the API (the provider will escape double quotes automatically).",
							Required:            true,
						},
						"join_with": schema.StringAttribute{
							MarkdownDescription: "Logical operator used to join this clause with the previous one. Valid values are `and` and `or`. Defaults to `and` when omitted or for the first block.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("and", "or"),
							},
						},
					},
				},
			},
			"timeouts": timeouts.Block(ctx),
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *DevicesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches devices (optionally filtered) and populates the Terraform state.
func (d *DevicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DevicesDataSourceModel

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

	readTimeout := defaultDevicesReadTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Read(ctx, defaultDevicesReadTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		readTimeout = configuredTimeout
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	filterExpression := buildDevicesFilter(&data)

	devices, err := d.client.GetDevicesV1(readCtx, nil, filterExpression)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list devices", err.Error())
		return
	}

	deviceEntries := make([]DevicesListEntry, 0, len(devices))
	for _, device := range devices {
		deviceEntries = append(deviceEntries, DevicesListEntry{
			ID:                     types.StringValue(device.ID),
			SerialNumber:           stringValueOrNull(device.SerialNumber),
			Name:                   stringValueOrNull(device.Name),
			Model:                  stringValueOrNull(device.Model),
			ModelIdentifier:        stringValueOrNull(device.ModelIdentifier),
			LastInventoryUpdate:    stringValueOrNull(device.LastInventoryUpdateTime),
			LastCheckIn:            stringPointerValueOrNull(device.LastCheckInTime),
			OperatingSystemVersion: stringValueOrNull(device.OperatingSystemVersion),
			UserID:                 stringPointerValueOrNull(device.UserID),
			EnrollmentType:         stringValueOrNull(device.EnrollmentType),
			LastEnrollmentTime:     stringValueOrNull(device.LastEnrollmentTime),
		})
	}

	data.ID = types.StringValue("devices")
	data.Devices = deviceEntries

	tflog.Trace(ctx, "read devices data source", map[string]interface{}{
		"filter": filterExpression,
		"count":  len(deviceEntries),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
