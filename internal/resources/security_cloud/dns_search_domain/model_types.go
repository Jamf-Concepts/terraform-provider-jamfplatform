// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_search_domain

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SearchDomainResourceModel represents the Terraform resource model for the Jamf
// Security Cloud search domain.
type SearchDomainResourceModel struct {
	ID         types.String           `tfsdk:"id"`
	DomainName types.String           `tfsdk:"domain_name"`
	Timeouts   resourceTimeouts.Value `tfsdk:"timeouts"`
}

// searchDomainIdentityModel represents the identity object for the search domain
// resource. The ID is always helpers.SingletonID.
type searchDomainIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// SearchDomainDataSourceModel represents the Terraform data source model for the
// Jamf Security Cloud search domain.
type SearchDomainDataSourceModel struct {
	ID         types.String             `tfsdk:"id"`
	DomainName types.String             `tfsdk:"domain_name"`
	Timeouts   datasourceTimeouts.Value `tfsdk:"timeouts"`
}
