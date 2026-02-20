// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

var _ list.ListResource = &BenchmarkListResource{}
var _ list.ListResourceWithConfigure = &BenchmarkListResource{}

// NewBenchmarkListResource creates a new list resource for compliance benchmarks.
func NewBenchmarkListResource() list.ListResource {
	return &BenchmarkListResource{}
}

// BenchmarkListResource implements Terraform list support for compliance benchmarks.
type BenchmarkListResource struct {
	client *client.Client
}

// Metadata configures the resource type name for Terraform.
func (r *BenchmarkListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_benchmark"
}

// Configure wires the provider client into the list resource.
func (r *BenchmarkListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ListResourceConfigSchema declares the supported filter attributes.
func (r *BenchmarkListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		Description: "Searches for Jamf Compliance Benchmarks with an optional case-insensitive substring filter.",
		Attributes: map[string]listschema.Attribute{
			"search": listschema.StringAttribute{
				Optional:    true,
				Description: "Optional substring to match (case-insensitive) against benchmark title or description.",
			},
		},
	}
}

// List fetches Jamf Compliance Benchmarks and streams them back to Terraform.
func (r *BenchmarkListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if r.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unconfigured Provider",
				"The provider has not been configured yet. Re-run the command after `terraform init` completes successfully.",
			),
		})
		return
	}

	var config BenchmarkListResourceModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	searchTerm, hasSearch := helpers.NormalizedFilterString(config.Search)
	searchLower := strings.ToLower(searchTerm)

	resp, err := r.client.GetCBEngineBenchmarksV2(ctx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unable to list benchmarks",
				err.Error(),
			),
		})
		return
	}

	if resp == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unable to list benchmarks",
				"The Jamf Platform API returned an empty response.",
			),
		})
		return
	}

	benchmarks := resp.Benchmarks
	if len(benchmarks) == 0 {
		stream.Results = list.NoListResults
		return
	}

	maxResults := req.Limit
	if maxResults <= 0 || maxResults > int64(len(benchmarks)) {
		maxResults = int64(len(benchmarks))
	}

	results := make([]list.ListResult, 0, int(maxResults))
	var emitted int64

	for _, bench := range benchmarks {
		if hasSearch {
			titleMatch := strings.Contains(strings.ToLower(bench.Title), searchLower)
			descMatch := strings.Contains(strings.ToLower(bench.Description), searchLower)
			if !titleMatch && !descMatch {
				continue
			}
		}

		if maxResults > 0 && emitted >= maxResults {
			break
		}

		result := req.NewListResult(ctx)
		result.DisplayName = bench.Title

		identity := benchmarkIdentityModel{ID: types.StringValue(bench.ID)}
		result.Diagnostics.Append(helpers.SetIdentity(ctx, result.Identity, identity)...)
		if result.Diagnostics.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
			return
		}

		if req.IncludeResource {
			detail, err := r.client.GetCBEngineBenchmarkByIDV2(ctx, bench.ID)
			if err != nil {
				stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
					diag.NewErrorDiagnostic(
						"Unable to read benchmark",
						err.Error(),
					),
				})
				return
			}

			state := BenchmarkResourceModel{
				Timeouts: helpers.NewResourceTimeoutsNullValue(benchmarkTimeoutAttributeTypes),
			}

			assignBenchmarkModelFromResponse(&state, detail)
			state.Timeouts = helpers.EnsureResourceTimeouts(state.Timeouts, benchmarkTimeoutAttributeTypes)

			result.Diagnostics.Append(result.Resource.Set(ctx, &state)...)
			if result.Diagnostics.HasError() {
				stream.Results = list.ListResultsStreamDiagnostics(result.Diagnostics)
				return
			}
		}

		results = append(results, result)
		emitted++
	}

	tflog.Debug(ctx, "Listed compliance benchmarks", map[string]any{
		"search":   searchTerm,
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
