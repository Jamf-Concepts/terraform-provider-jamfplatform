// Copyright 2025 Jamf Software LLC.

package provider

import (
	"context"

	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/component"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/components"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/cbengine/baselines"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/cbengine/benchmark"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/cbengine/rules"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/inventory/computer"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/inventory/computers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/inventory/mobiledevice"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/inventory/mobiledevices"
)

// Constants for environment variable names.
const (
	envBaseURL      = "JAMFPLATFORM_BASE_URL"
	envClientID     = "JAMFPLATFORM_CLIENT_ID"
	envClientSecret = "JAMFPLATFORM_CLIENT_SECRET"
)

// providerModel describes the provider data model for configuration.
type providerModel struct {
	BaseURL      types.String `tfsdk:"base_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// jamfPlatformProvider implements the Terraform provider for Jamf Platform.
type jamfPlatformProvider struct {
	apiClient *client.Client
}

// Ensure jamfPlatformProvider satisfies the provider.Provider interface.
var _ provider.Provider = &jamfPlatformProvider{}

// Metadata sets the provider type name for the Terraform provider.
func (p *jamfPlatformProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jamfplatform"
}

// Schema sets the Terraform schema for the provider.
func (p *jamfPlatformProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for Jamf Platform. https://developer.jamf.com/platform-api/docs/getting-started-with-the-platform-api Configure base_url and service-specific credentials. Values can be set via provider block, environment variables, or Terraform variables.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Required:    true,
				Description: "The Jamf Platform base URL to use (e.g., https://us.apigw.jamf.com or https://us.stage.apigw.jamfnebula.com). Can also be set via the JAMFPLATFORM_BASE_URL environment variable.",
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "OAuth client ID for Jamf Platform API. Can also be set via the JAMFPLATFORM_CLIENT_ID environment variable.",
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "OAuth client secret for Jamf Platform API. Can also be set via the JAMFPLATFORM_CLIENT_SECRET environment variable.",
			},
		},
	}
}

// Configure sets up the API client for the provider from the provider configuration.
func (p *jamfPlatformProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := config.BaseURL.ValueString()
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

	clientID := config.ClientID.ValueString()
	if clientID == "" {
		clientID = getenv(envClientID)
	}
	clientSecret := config.ClientSecret.ValueString()
	if clientSecret == "" {
		clientSecret = getenv(envClientSecret)
	}

	if clientID != "" && clientSecret != "" {
		p.apiClient = client.NewClient(baseURL, clientID, clientSecret)
	} else {
		p.apiClient = nil
	}

	resp.DataSourceData = p.apiClient
	resp.ResourceData = p.apiClient
}

// getenv is a helper to get an environment variable, returns empty string if not set.
func getenv(key string) string {
	v, _ := os.LookupEnv(key)
	return v
}

// Resources returns the list of resource constructors for the provider.
func (p *jamfPlatformProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		benchmark.NewBenchmarkResource,
	}
}

func (p *jamfPlatformProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		blueprint.NewBlueprintDataSource,
		component.NewComponentDataSource,
		components.NewComponentsDataSource,
		baselines.NewBaselinesDataSource,
		rules.NewRulesDataSource,
		benchmark.NewBenchmarkDataSource,
		mobiledevices.NewDataSourceMobileDevices,
		computers.NewDataSourceComputers,
		computer.NewDataSourceComputer,
		mobiledevice.NewDataSourceMobileDevice,
	}
}

// New creates a new instance of the Jamf Platform provider.
func New() provider.Provider {
	return &jamfPlatformProvider{}
}
