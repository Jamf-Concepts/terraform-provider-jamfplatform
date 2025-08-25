package mobiledevices

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
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
