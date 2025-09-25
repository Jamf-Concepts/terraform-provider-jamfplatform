// Copyright 2025 Jamf Software LLC.

package computers

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataSourceComputers defines the data source implementation.
type DataSourceComputers struct {
	client *client.Client
}

// ComputersDataSourceModel maps the data source schema data.
type ComputersDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Filter    types.String `tfsdk:"filter"`
	Computers types.List   `tfsdk:"computers"`
}
