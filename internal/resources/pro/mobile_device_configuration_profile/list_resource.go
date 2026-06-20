// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

const defaultListTimeout = 120 * time.Second

var _ list.ListResource = &ListResource{}
var _ list.ListResourceWithConfigure = &ListResource{}

// NewListResource returns a list resource for mobile device configuration profile queries.
func NewListResource() list.ListResource {
	return &ListResource{}
}

// ListResource queries mobile device configuration profiles.
type ListResource struct {
	client *proclassic.Client
}

// ListResourceConfigModel is the config model for list queries.
type ListResourceConfigModel struct {
	Filter *filters.ClassicFilterModel `tfsdk:"filter"`
}

// Metadata sets the list resource type name.
func (r *ListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_mobile_device_configuration_profile"
}

// Configure wires the SDK client.
func (r *ListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_mobile_device_configuration_profile")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *ListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Lists mobile device configuration profiles in the tenant. Supply an optional case-insensitive `name_substring` filter — filtering is applied client-side, so all profiles are fetched before the filter runs. List entries return identity only; use the `jamfplatform_pro_mobile_device_configuration_profile` data source to fetch per-profile detail.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ClassicListFilterAttribute(),
		},
	}
}

// List executes the query and streams profile identities back to Terraform.
func (r *ListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run after `terraform init` completes.",
			),
		})
		return
	}

	var config ListResourceConfigModel
	if diags := req.Config.Get(ctx, &config); diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	resp, err := r.client.ListMobileDeviceConfigurationProfiles(listCtx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list mobile device configuration profiles", err.Error()),
		})
		return
	}

	items := []proclassic.MobileDeviceConfigurationProfilesItemConfigurationProfile{}
	if resp != nil {
		items = resp.ConfigurationProfiles
	}

	filter := filters.ClassicFilterModel{}
	if config.Filter != nil {
		filter = *config.Filter
	}
	items = filters.ApplyClassicFilter(items, filter, itemName)

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(items)) {
		maxResults = int64(len(items))
	}

	results := make([]list.ListResult, 0, maxResults)
	for _, it := range items {
		if int64(len(results)) >= maxResults {
			break
		}
		result := req.NewListResult(ctx)
		result.DisplayName = helpers.DerefString(it.Name)
		id := helpers.StringValueFromIntPtr(it.ID)
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, identityModel{ID: id})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}
		results = append(results, result)
	}

	tflog.Debug(ctx, "Listed Jamf Pro mobile device configuration profiles", map[string]any{
		"name_substring": filter.NameSubstring.ValueString(),
		"limit":          req.Limit,
		"returned":       len(results),
	})

	if len(results) == 0 {
		stream.Results = list.NoListResults
		return
	}
	stream.Results = func(push func(list.ListResult) bool) {
		for _, result := range results {
			if !push(result) {
				return
			}
		}
	}
}

func itemName(p proclassic.MobileDeviceConfigurationProfilesItemConfigurationProfile) string {
	return helpers.DerefString(p.Name)
}
