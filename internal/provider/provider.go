// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	deviceactions "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/actions/device"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprints"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/component"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/components"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/cbengine/baselines"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/cbengine/benchmark"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/cbengine/benchmarks"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/cbengine/rules"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/device"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/device_group"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/device_groups"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/devices"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/building"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/buildings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/categories"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/category"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/department"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/departments"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/network_segment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/network_segments"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/site"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/sites"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/policies/script"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/policies/scripts"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/settings/self_service_plus_settings"
)

// Constants for environment variable names.
const (
	envBaseURL      = "JAMFPLATFORM_BASE_URL"
	envClientID     = "JAMFPLATFORM_CLIENT_ID"
	envClientSecret = "JAMFPLATFORM_CLIENT_SECRET"
	envTenantID     = "JAMFPLATFORM_TENANT_ID"
)

// Ensure JamfPlatformProvider satisfies the various provider interfaces.
var _ provider.Provider = &JamfPlatformProvider{}
var _ provider.ProviderWithListResources = &JamfPlatformProvider{}
var _ provider.ProviderWithActions = &JamfPlatformProvider{}

// JamfPlatformProvider defines the provider implementation.
type JamfPlatformProvider struct {
	version string
}

// JamfPlatformProviderModel describes the provider data model.
type JamfPlatformProviderModel struct {
	BaseURL      types.String `tfsdk:"base_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	TenantID     types.String `tfsdk:"tenant_id"`
}

func (p *JamfPlatformProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jamfplatform"
	resp.Version = p.version
}

func (p *JamfPlatformProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(
			"Provider for [Jamf Platform API Services](https://developer.jamf.com/platform-api/reference/getting-started-with-platform-api). "+
				"Configure `base_url` and credentials via the provider block, environment variables, or Terraform variables.\n\n"+
				"**Supported Jamf products and tenant version targets**\n\n"+
				"| Product | Resource namespace | Built against API as of |\n"+
				"|---------|--------------------|--------------------------|\n"+
				"| Jamf Pro | `jamfplatform_pro_*` | %s |\n\n"+
				"Tenants below the listed version emit an advisory warning at apply time; individual resources that depend on newer endpoints declare their own per-resource floor and will error out explicitly on unsupported tenants. Resources outside the listed namespaces (Blueprints, Device Groups, Devices, Device Actions, Compliance Benchmarks) target continuously-deployed Jamf Platform microservices and have no tenant version requirement.",
			providerdata.ProviderMinJamfProVersion,
		),
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "The Jamf Platform base URL to use (e.g., https://us.apigw.jamf.com for production US region or https://us.stage.apigw.jamfnebula.com for internal staging US region). Can also be set via the JAMFPLATFORM_BASE_URL environment variable.",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Description: "OAuth client ID for Jamf Platform API. Can also be set via the JAMFPLATFORM_CLIENT_ID environment variable.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth client secret for Jamf Platform API. Can also be set via the JAMFPLATFORM_CLIENT_SECRET environment variable.",
			},
			"tenant_id": schema.StringAttribute{
				Optional:    true,
				Description: "Tenant UUID used to scope all API requests. Can also be set via the JAMFPLATFORM_TENANT_ID environment variable.",
			},
		},
	}
}

func (p *JamfPlatformProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data JamfPlatformProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := data.BaseURL.ValueString()
	if baseURL == "" {
		baseURL = getenv(envBaseURL)
	}
	if baseURL == "" {
		resp.Diagnostics.AddError(
			"Missing Required Provider Configuration",
			"base_url must be set either in the provider block or via the JAMFPLATFORM_BASE_URL environment variable.",
		)
		return
	}

	clientID := data.ClientID.ValueString()
	if clientID == "" {
		clientID = getenv(envClientID)
	}
	if clientID == "" {
		resp.Diagnostics.AddError(
			"Missing Required Provider Configuration",
			"client_id must be set either in the provider block or via the JAMFPLATFORM_CLIENT_ID environment variable.",
		)
		return
	}

	clientSecret := data.ClientSecret.ValueString()
	if clientSecret == "" {
		clientSecret = getenv(envClientSecret)
	}
	if clientSecret == "" {
		resp.Diagnostics.AddError(
			"Missing Required Provider Configuration",
			"client_secret must be set either in the provider block or via the JAMFPLATFORM_CLIENT_SECRET environment variable.",
		)
		return
	}

	tenantID := data.TenantID.ValueString()
	if tenantID == "" {
		tenantID = getenv(envTenantID)
	}
	if tenantID == "" {
		resp.Diagnostics.AddError(
			"Missing Required Provider Configuration",
			"tenant_id must be set either in the provider block or via the JAMFPLATFORM_TENANT_ID environment variable.",
		)
		return
	}

	opts := []jamfplatform.Option{
		jamfplatform.WithUserAgent("terraform-provider-jamfplatform/" + p.version),
		jamfplatform.WithTenantID(tenantID),
		jamfplatform.WithRetryOn4xx(true),
	}
	if shouldEnableHTTPLogging() {
		opts = append(opts, jamfplatform.WithLogger(NewTerraformLogger()))
	}
	apiClient := jamfplatform.NewClient(baseURL, clientID, clientSecret, opts...)

	if err := apiClient.ValidateCredentials(ctx); err != nil {
		resp.Diagnostics.AddError(
			"Authentication Failed",
			fmt.Sprintf("Unable to authenticate with Jamf Platform API. Please verify your credentials are correct.\n\nError: %s", err.Error()),
		)
		return
	}

	tflog.Info(ctx, "Jamf Platform provider configured", map[string]any{
		"provider_version":     p.version,
		"jamf_pro_api_version": jamfplatform.JamfProAPIVersion,
		"provider_pro_floor":   providerdata.ProviderMinJamfProVersion,
	})

	pd := providerdata.New(apiClient)
	resp.DataSourceData = pd
	resp.ResourceData = pd
	resp.ListResourceData = pd
	resp.ActionData = pd
}

func (p *JamfPlatformProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		benchmark.NewBenchmarkResource,
		blueprint.NewBlueprintResource,
		building.NewBuildingResource,
		category.NewCategoryResource,
		department.NewDepartmentResource,
		device_group.NewDeviceGroupResource,
		network_segment.NewNetworkSegmentResource,
		script.NewScriptResource,
		self_service_plus_settings.NewSelfServicePlusSettingsResource,
		site.NewSiteResource,
	}
}

func (p *JamfPlatformProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		blueprint.NewBlueprintDataSource,
		blueprints.NewBlueprintsDataSource,
		component.NewComponentDataSource,
		components.NewComponentsDataSource,
		baselines.NewBaselinesDataSource,
		rules.NewRulesDataSource,
		benchmark.NewBenchmarkDataSource,
		benchmarks.NewBenchmarksDataSource,
		building.NewBuildingDataSource,
		buildings.NewBuildingsDataSource,
		categories.NewCategoriesDataSource,
		category.NewCategoryDataSource,
		department.NewDepartmentDataSource,
		departments.NewDepartmentsDataSource,
		device_group.NewDeviceGroupDataSource,
		device_groups.NewDeviceGroupsDataSource,
		device.NewDeviceDataSource,
		devices.NewDevicesDataSource,
		network_segment.NewNetworkSegmentDataSource,
		network_segments.NewNetworkSegmentsDataSource,
		script.NewScriptDataSource,
		scripts.NewScriptsDataSource,
		self_service_plus_settings.NewSelfServicePlusSettingsDataSource,
		site.NewSiteDataSource,
		sites.NewSitesDataSource,
	}
}

func (p *JamfPlatformProvider) ListResources(ctx context.Context) []func() list.ListResource {
	return []func() list.ListResource{
		benchmark.NewBenchmarkListResource,
		blueprint.NewBlueprintListResource,
		building.NewBuildingListResource,
		category.NewCategoryListResource,
		department.NewDepartmentListResource,
		device_group.NewDeviceGroupListResource,
		network_segment.NewNetworkSegmentListResource,
		script.NewScriptListResource,
		site.NewSiteListResource,
	}
}

func (p *JamfPlatformProvider) Actions(ctx context.Context) []func() action.Action {
	return []func() action.Action{
		deviceactions.NewEraseAction,
		deviceactions.NewRestartAction,
		deviceactions.NewShutdownAction,
		deviceactions.NewUnmanageAction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &JamfPlatformProvider{
			version: version,
		}
	}
}

// getenv is a helper to get an environment variable, returns empty string if not set.
func getenv(key string) string {
	v, _ := os.LookupEnv(key)
	return v
}

// shouldEnableHTTPLogging checks TF_LOG to determine whether HTTP logging should be wired up.
func shouldEnableHTTPLogging() bool {
	level, ok := os.LookupEnv("TF_LOG")
	if !ok {
		return false
	}

	switch strings.ToLower(level) {
	case "debug", "trace":
		return true
	default:
		return false
	}
}
