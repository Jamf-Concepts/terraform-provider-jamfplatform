// Copyright 2025 Jamf Software LLC.

package device_group

import (
	"context"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/device_groups"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ list.ListResource = &DeviceGroupListResource{}
var _ list.ListResourceWithConfigure = &DeviceGroupListResource{}

// NewDeviceGroupListResource returns a list resource for device group queries.
func NewDeviceGroupListResource() list.ListResource {
	return &DeviceGroupListResource{}
}

// DeviceGroupListResource implements Terraform query list support for device groups.
type DeviceGroupListResource struct {
	client *client.Client
}

// Metadata sets the list resource type name.
func (r *DeviceGroupListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_group"
}

// Configure wires the provider client into the list resource.
func (r *DeviceGroupListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected List Configure Type",
			"Expected *client.Client. Please report this issue to the provider developers.",
		)
		return
	}

	r.client = client
}

// ListResourceConfigSchema describes the supported list filters.
func (r *DeviceGroupListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Searches for Jamf device groups using the same filter clauses as the device_groups data source.",
		Attributes: map[string]listschema.Attribute{
			"filter": filters.ListFilterAttribute(
				filters.SelectorDescription(device_groups.DeviceGroupFilterSelectors),
				device_groups.DeviceGroupFilterSelectors,
			),
		},
	}
}

// List executes the query and streams device group identities back to Terraform.
func (r *DeviceGroupListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config DeviceGroupListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	filterExpression := filters.BuildRSQLExpression(config.Filters, filters.AllowList(device_groups.DeviceGroupFilterSelectors))

	tflog.Debug(ctx, "device group list filters", map[string]any{
		"filter": filterExpression,
	})

	groups, err := r.client.GetDeviceGroupsV1(ctx, nil, filterExpression)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unable to list device groups",
				err.Error(),
			),
		})
		return
	}

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(groups)) {
		maxResults = int64(len(groups))
	}

	results := make([]list.ListResult, 0, int(maxResults))
	var emitted int64

	for _, grp := range groups {
		if maxResults > 0 && emitted >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = grp.Name

		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, deviceGroupIdentityModel{ID: types.StringValue(grp.ID)})...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			detail, err := r.client.GetDeviceGroupByIDV1(ctx, grp.ID)
			if err != nil {
				stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
					diag.NewErrorDiagnostic(
						"Unable to read device group",
						err.Error(),
					),
				})
				return
			}

			manageMembers := strings.EqualFold(detail.GroupType, "STATIC")
			var members []string
			if manageMembers {
				members, err = r.client.GetDeviceGroupMembersV1(ctx, detail.ID)
				if err != nil {
					stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
						diag.NewErrorDiagnostic(
							"Unable to read device group members",
							err.Error(),
						),
					})
					return
				}
			}

			state := DeviceGroupResourceModel{
				Timeouts: helpers.NewResourceTimeoutsNullValue(deviceGroupTimeoutAttributeTypes),
			}

			result.Diagnostics.Append(assignDeviceGroupModel(ctx, &state, detail, members, manageMembers, true)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}

			state.Timeouts = helpers.EnsureResourceTimeouts(state.Timeouts, deviceGroupTimeoutAttributeTypes)

			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
		emitted++
	}

	tflog.Debug(ctx, "Listed device groups", map[string]any{
		"filter":   filterExpression,
		"limit":    req.Limit,
		"returned": len(results),
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
