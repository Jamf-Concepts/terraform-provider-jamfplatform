// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_hostname_mappings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// HostnameMappingsResourceModel represents the Terraform resource model for the
// Jamf Security Cloud custom hostname mappings.
type HostnameMappingsResourceModel struct {
	ID       types.String           `tfsdk:"id"`
	Mappings types.Set              `tfsdk:"mappings"`
	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// MappingModel represents one hostname mapping.
//
// The two address lists are sets because the server dedupes them and their order
// carries no meaning: a list would diff forever against a configuration containing a
// duplicate. See STYLE_GUIDE §Sets vs Lists.
type MappingModel struct {
	Hostname           types.String `tfsdk:"hostname"`
	IPv4Addresses      types.Set    `tfsdk:"ipv4_addresses"`
	IPv6Addresses      types.Set    `tfsdk:"ipv6_addresses"`
	ConnectToZTNA      types.Bool   `tfsdk:"connect_to_ztna"`
	ConnectToSecureDNS types.Bool   `tfsdk:"connect_to_secure_dns"`
}

// hostnameMappingsIdentityModel represents the identity object for the hostname
// mappings resource. The ID is always helpers.SingletonID.
type hostnameMappingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// HostnameMappingsDataSourceModel represents the Terraform data source model for the
// Jamf Security Cloud custom hostname mappings. Collections are lists rather than
// sets because data source attributes returning API data are always read-only.
type HostnameMappingsDataSourceModel struct {
	ID       types.String                     `tfsdk:"id"`
	Mappings []HostnameMappingsDataSourceItem `tfsdk:"mappings"`
	Timeouts datasourceTimeouts.Value         `tfsdk:"timeouts"`
}

// HostnameMappingsDataSourceItem represents one hostname mapping in the data source
// results.
type HostnameMappingsDataSourceItem struct {
	Hostname           types.String `tfsdk:"hostname"`
	IPv4Addresses      types.List   `tfsdk:"ipv4_addresses"`
	IPv6Addresses      types.List   `tfsdk:"ipv6_addresses"`
	ConnectToZTNA      types.Bool   `tfsdk:"connect_to_ztna"`
	ConnectToSecureDNS types.Bool   `tfsdk:"connect_to_secure_dns"`
}
