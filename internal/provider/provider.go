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
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/api/api_client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/api/api_clients"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/api/api_role"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/api/api_role_privileges"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/api/api_roles"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/apps/app_installer"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/apps/app_installer_title"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/apps/app_installer_titles"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/apps/app_installers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/apps/mac_app_store_app"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/apps/mobile_device_app"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/configuration_profiles/macos_configuration_profile"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/configuration_profiles/mobile_device_configuration_profile"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/enrollment/automated_device_enrollment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/enrollment/automated_device_enrollment_public_key"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/enrollment/computer_prestage_enrollment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/enrollment/enrollment_customization"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/enrollment/mobile_device_prestage_enrollment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/building"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/buildings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/categories"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/category"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/department"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/departments"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/directory_binding"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/disk_encryption_configuration"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/dock_item"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/ibeacon"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/icon"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/network_segment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/network_segments"
	pkg "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/package"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/printer"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/site"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory/sites"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/policies/policy"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/policies/restricted_software"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/policies/script"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/policies/scripts"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/settings/cloud_distribution_point"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/settings/cloud_identity_provider"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/settings/ldap_server"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/settings/re_enrollment_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/settings/self_service_plus_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/settings/sso_failover_url"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/settings/sso_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/settings/user_initiated_enrollment_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/users/user_group"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/users/user_groups"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/volume_purchasing/location"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/webhooks/webhook"
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
		api_client.NewApiClientResource,
		api_role.NewApiRoleResource,
		automated_device_enrollment.NewAutomatedDeviceEnrollmentResource,
		benchmark.NewBenchmarkResource,
		blueprint.NewBlueprintResource,
		building.NewBuildingResource,
		category.NewCategoryResource,
		cloud_distribution_point.NewCloudDistributionPointResource,
		cloud_identity_provider.NewCloudIdentityProviderResource,
		department.NewDepartmentResource,
		device_group.NewDeviceGroupResource,
		directory_binding.NewDirectoryBindingResource,
		ldap_server.NewLdapServerResource,
		disk_encryption_configuration.NewDiskEncryptionConfigurationResource,
		computer_prestage_enrollment.NewComputerPrestageEnrollmentResource,
		mobile_device_prestage_enrollment.NewMobileDevicePrestageEnrollmentResource,
		dock_item.NewDockItemResource,
		enrollment_customization.NewEnrollmentCustomizationResource,
		ibeacon.NewIbeaconResource,
		icon.NewIconResource,
		app_installer.NewAppInstallerResource,
		mac_app_store_app.NewMacAppResource,
		mobile_device_app.NewMobileAppResource,
		macos_configuration_profile.NewResource,
		mobile_device_configuration_profile.NewResource,
		network_segment.NewNetworkSegmentResource,
		pkg.NewPackageResource,
		policy.NewPolicyResource,
		restricted_software.NewRestrictedSoftwareResource,
		printer.NewPrinterResource,
		re_enrollment_settings.NewReEnrollmentSettingsResource,
		user_initiated_enrollment_settings.NewUserInitiatedEnrollmentSettingsResource,
		script.NewScriptResource,
		self_service_plus_settings.NewSelfServicePlusSettingsResource,
		site.NewSiteResource,
		sso_failover_url.NewSsoFailoverURLResource,
		sso_settings.NewSsoSettingsResource,
		user_group.NewUserGroupResource,
		location.NewVolumePurchasingLocationResource,
		webhook.NewWebhookResource,
	}
}

func (p *JamfPlatformProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		api_client.NewApiClientDataSource,
		api_clients.NewApiClientsDataSource,
		api_role.NewApiRoleDataSource,
		api_role_privileges.NewApiRolePrivilegesDataSource,
		api_roles.NewApiRolesDataSource,
		automated_device_enrollment.NewAutomatedDeviceEnrollmentDataSource,
		automated_device_enrollment_public_key.NewAutomatedDeviceEnrollmentPublicKeyDataSource,
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
		cloud_distribution_point.NewCloudDistributionPointDataSource,
		cloud_identity_provider.NewCloudIdentityProviderDataSource,
		cloud_identity_provider.NewCloudIdentityProvidersDataSource,
		department.NewDepartmentDataSource,
		departments.NewDepartmentsDataSource,
		device_group.NewDeviceGroupDataSource,
		device_groups.NewDeviceGroupsDataSource,
		device.NewDeviceDataSource,
		devices.NewDevicesDataSource,
		directory_binding.NewDirectoryBindingDataSource,
		ldap_server.NewLdapServerDataSource,
		disk_encryption_configuration.NewDiskEncryptionConfigurationDataSource,
		computer_prestage_enrollment.NewComputerPrestageEnrollmentDataSource,
		mobile_device_prestage_enrollment.NewMobileDevicePrestageEnrollmentDataSource,
		dock_item.NewDockItemDataSource,
		enrollment_customization.NewEnrollmentCustomizationDataSource,
		ibeacon.NewIbeaconDataSource,
		app_installer.NewAppInstallerDataSource,
		app_installers.NewAppInstallersDataSource,
		app_installer_title.NewAppInstallerTitleDataSource,
		app_installer_titles.NewAppInstallerTitlesDataSource,
		mac_app_store_app.NewMacAppDataSource,
		mobile_device_app.NewMobileAppDataSource,
		macos_configuration_profile.NewDataSource,
		mobile_device_configuration_profile.NewDataSource,
		network_segment.NewNetworkSegmentDataSource,
		network_segments.NewNetworkSegmentsDataSource,
		pkg.NewPackageDataSource,
		policy.NewPolicyDataSource,
		restricted_software.NewRestrictedSoftwareDataSource,
		printer.NewPrinterDataSource,
		re_enrollment_settings.NewReEnrollmentSettingsDataSource,
		user_initiated_enrollment_settings.NewUserInitiatedEnrollmentSettingsDataSource,
		script.NewScriptDataSource,
		scripts.NewScriptsDataSource,
		self_service_plus_settings.NewSelfServicePlusSettingsDataSource,
		site.NewSiteDataSource,
		sso_failover_url.NewSsoFailoverURLDataSource,
		sso_settings.NewSsoDependenciesDataSource,
		sso_settings.NewSsoSettingsDataSource,
		sso_settings.NewSsoSpMetadataDataSource,
		sites.NewSitesDataSource,
		user_group.NewUserGroupDataSource,
		user_groups.NewUserGroupsDataSource,
		location.NewVolumePurchasingLocationDataSource,
		webhook.NewWebhookDataSource,
	}
}

func (p *JamfPlatformProvider) ListResources(ctx context.Context) []func() list.ListResource {
	return []func() list.ListResource{
		api_client.NewApiClientListResource,
		api_role.NewApiRoleListResource,
		automated_device_enrollment.NewAutomatedDeviceEnrollmentListResource,
		benchmark.NewBenchmarkListResource,
		blueprint.NewBlueprintListResource,
		building.NewBuildingListResource,
		category.NewCategoryListResource,
		cloud_identity_provider.NewCloudIdentityProviderListResource,
		department.NewDepartmentListResource,
		device_group.NewDeviceGroupListResource,
		directory_binding.NewDirectoryBindingListResource,
		ldap_server.NewLdapServerListResource,
		disk_encryption_configuration.NewDiskEncryptionConfigurationListResource,
		computer_prestage_enrollment.NewComputerPrestageEnrollmentListResource,
		mobile_device_prestage_enrollment.NewMobileDevicePrestageEnrollmentListResource,
		dock_item.NewDockItemListResource,
		enrollment_customization.NewEnrollmentCustomizationListResource,
		ibeacon.NewIbeaconListResource,
		app_installer.NewAppInstallerListResource,
		mac_app_store_app.NewMacAppListResource,
		mobile_device_app.NewMobileAppListResource,
		macos_configuration_profile.NewListResource,
		mobile_device_configuration_profile.NewListResource,
		network_segment.NewNetworkSegmentListResource,
		pkg.NewPackageListResource,
		policy.NewPolicyListResource,
		restricted_software.NewRestrictedSoftwareListResource,
		printer.NewPrinterListResource,
		script.NewScriptListResource,
		site.NewSiteListResource,
		user_group.NewUserGroupListResource,
		location.NewVolumePurchasingLocationListResource,
		webhook.NewWebhookListResource,
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
