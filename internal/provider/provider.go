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

// Ensure JamfPlatformProvider satisfies the provider.Provider interface.
var _ provider.Provider = &JamfPlatformProvider{}

// JamfPlatformProvider implements the Terraform provider for Jamf Platform.
type JamfPlatformProvider struct {
	version string
}

// JamfPlatformProviderModel describes the provider data model for configuration.
type JamfPlatformProviderModel struct {
	BaseURL      types.String `tfsdk:"base_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// Metadata sets the provider type name for the Terraform provider.
func (p *JamfPlatformProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jamfplatform"
	resp.Version = p.version
}

// Schema sets the Terraform schema for the provider.
func (p *JamfPlatformProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for Jamf Platform. https://developer.jamf.com/platform-api/docs/getting-started-with-the-platform-api Configure base_url and service-specific credentials. Values can be set via provider block, environment variables, or Terraform variables.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "The Jamf Platform base URL to use (e.g., https://us.apigw.jamf.com for production US region or https://us.stage.apigw.jamfnebula.com for internal staging US region). Can also be set via the JAMFPLATFORM_BASE_URL environment variable.",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth client ID for Jamf Platform API. Can also be set via the JAMFPLATFORM_CLIENT_ID environment variable.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth client secret for Jamf Platform API. Can also be set via the JAMFPLATFORM_CLIENT_SECRET environment variable.",
			},
		},
	}
}

// Configure sets up the API client for the provider from the provider configuration.
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
	clientSecret := data.ClientSecret.ValueString()
	if clientSecret == "" {
		clientSecret = getenv(envClientSecret)
	}

	client := client.NewClient(baseURL, clientID, clientSecret)
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources returns the list of resource constructors for the provider.
func (p *JamfPlatformProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		benchmark.NewBenchmarkResource,
		blueprint.NewBlueprintResource,
	}
}

func (p *JamfPlatformProvider) DataSources(_ context.Context) []func() datasource.DataSource {
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
