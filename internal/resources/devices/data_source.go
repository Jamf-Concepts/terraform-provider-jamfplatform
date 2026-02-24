// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package devices

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const defaultReadTimeout = 90 * time.Second

var filterSelectors = []string{
	"id",
	"name",
	"model",
	"modelIdentifier",
	"serialNumber",
	"lastInventoryUpdateTime",
	"lastCheckInTime",
	"operatingSystemVersion",
	"userId",
	"enrollmentType",
	"lastEnrollmentTime",
}

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
			"timeouts": timeouts.Attributes(ctx),
			"filter": filters.FilterAttribute(
				filters.SelectorDescription(filterSelectors),
				filterSelectors,
			),
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

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, data.Timeouts.IsNull(), data.Timeouts.IsUnknown(), defaultReadTimeout, data.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	filterExpression := filters.BuildRSQLExpression(data.Filters, filters.AllowList(filterSelectors))

	tflog.Debug(ctx, "devices filter expression", map[string]any{
		"filter": filterExpression,
	})

	devices, err := d.client.GetDevicesV1(readCtx, nil, filterExpression)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list devices", err.Error())
		return
	}

	results := make([]DevicesListEntry, 0, len(devices))
	for _, device := range devices {
		results = append(results, DevicesListEntry{
			ID:                     types.StringValue(device.ID),
			SerialNumber:           helpers.StringValueOrNull(device.SerialNumber),
			Name:                   helpers.StringValueOrNull(device.Name),
			Model:                  helpers.StringValueOrNull(device.Model),
			ModelIdentifier:        helpers.StringValueOrNull(device.ModelIdentifier),
			LastInventoryUpdate:    helpers.StringValueOrNull(device.LastInventoryUpdateTime),
			LastCheckIn:            helpers.StringPointerValueOrNull(device.LastCheckInTime),
			OperatingSystemVersion: helpers.StringValueOrNull(device.OperatingSystemVersion),
			UserID:                 helpers.StringPointerValueOrNull(device.UserID),
			EnrollmentType:         helpers.StringValueOrNull(device.EnrollmentType),
			LastEnrollmentTime:     helpers.StringValueOrNull(device.LastEnrollmentTime),
		})
	}

	data.ID = types.StringValue("devices")
	data.Devices = results

	tflog.Trace(ctx, "read devices data source", map[string]any{
		"filter": filterExpression,
		"count":  len(results),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
