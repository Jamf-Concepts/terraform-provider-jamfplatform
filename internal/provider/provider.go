// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	deviceactions "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/actions/device"
	jamfprotectactions "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/actions/pro/jamf_protect"
	maintenanceactions "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/actions/pro/maintenance"
	msuactions "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/actions/pro/managed_software_updates"
	mdmactions "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/actions/pro/mdm"
	patchactions "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/actions/pro/patch"
	mcxforcedpayload "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/functions/mcx_forced_payload"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/functions/mobileconfig"
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
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/access_management_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/account"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/account_group"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/account_privileges"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/activation_code"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/advanced_computer_search"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/advanced_mobile_device_search"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/advanced_user_search"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/advanced_volume_purchasing_content_search"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/allowed_file_extension"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/api_client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/api_role"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/api_role_privileges"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/app_installer"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/app_installer_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/app_installer_title"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/app_request_form_field"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/app_request_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/app_store_country_codes"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/automated_device_enrollment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/automated_device_enrollment_public_key"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/building"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/category"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/class"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/cloud_distribution_point"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/cloud_identity_provider"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/computer_check_in_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/computer_extension_attribute"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/computer_inventory_collection_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/computer_invitation"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/computer_prestage_enrollment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/department"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/directory_binding"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/disk_encryption_configuration"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/dock_item"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/ebook"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/enrollment_customization"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/file_share_distribution_point"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/gsx_connection"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/ibeacon"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/icon"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/impact_alert_notification_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/inventory_preload_record"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/jamf_connect"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/jamf_parent_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/jamf_pro_server_url"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/jamf_protect"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/jamf_teacher_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/ldap_server"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/licensed_software"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/local_admin_password_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/location"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/login_page"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/mac_app_store_app"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/macos_configuration_profile"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/macos_onboarding"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/managed_software_updates"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/mdm_profile_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/mobile_device_app"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/mobile_device_configuration_profile"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/mobile_device_enrollment_profile"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/mobile_device_extension_attribute"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/mobile_device_invitation"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/mobile_device_prestage_enrollment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/mobile_device_provisioning_profile"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/network_segment"
	pkg "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/package"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/patch_external_source"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/patch_internal_source"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/patch_policy"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/patch_software_title"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/pki_adcs"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/pki_certificate_authority"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/pki_digicert"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/pki_json_web_token_configuration"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/pki_venafi"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/policy"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/printer"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/re_enrollment_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/removable_mac_address"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/restricted_software"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/return_to_service"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/script"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/self_service_branding_image"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/self_service_branding_ios"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/self_service_branding_macos"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/self_service_macos_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/self_service_plus_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/service_discovery_enrollment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/site"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/smtp_server"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/sso_failover_url"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/sso_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/supervision_identity"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/user"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/user_extension_attribute"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/user_group"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/user_initiated_enrollment_settings"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/volume_purchasing_notification"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/vpp_assignment"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/vpp_invitation"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/webhook"
)

// Constants for environment variable names.
const (
	envBaseURL              = "JAMFPLATFORM_BASE_URL"
	envClientID             = "JAMFPLATFORM_CLIENT_ID"
	envClientSecret         = "JAMFPLATFORM_CLIENT_SECRET"
	envTenantID             = "JAMFPLATFORM_TENANT_ID"
	envMinRequestIntervalMs = "JAMFPLATFORM_MIN_REQUEST_INTERVAL_MS"
	envImpactAlerts         = "JAMFPLATFORM_IMPACT_ALERTS"
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
	BaseURL              types.String `tfsdk:"base_url"`
	ClientID             types.String `tfsdk:"client_id"`
	ClientSecret         types.String `tfsdk:"client_secret"`
	TenantID             types.String `tfsdk:"tenant_id"`
	MinRequestIntervalMs types.Int64  `tfsdk:"min_request_interval_ms"`
	ImpactAlerts         types.Bool   `tfsdk:"impact_alerts"`
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
				"**📘 New here? Start with the getting-started guide:** [Managing the Jamf Platform with Terraform: the Jamf Platform provider](https://concepts.jamf.com/en/guides/infrastructure-as-code/managing-the-jamf-platform-with-terraform-the-jamf-platform-provider/) on Jamf Concepts walks through installing Terraform, creating API credentials, configuring the provider, writing your first device groups, compliance benchmarks and blueprints, applying a configuration, and bringing an existing tenant under management.\n\n"+
				"> **Note:** The Jamf Platform API is currently in public beta. Provider stability, functionality, and schemas are subject to change without notice.\n\n"+
				"**Supported Jamf products and tenant version targets**\n\n"+
				"| Product | Resource namespace | Built against API as of |\n"+
				"|---------|--------------------|--------------------------|\n"+
				"| Jamf Pro | `jamfplatform_pro_*` | %s |\n\n"+
				"Tenants below the listed version emit an advisory warning at apply time; individual resources that depend on newer endpoints declare their own per-resource floor and will error out explicitly on unsupported tenants. Resources outside the listed namespaces (Blueprints, Device Groups, Devices, Device Actions, Compliance Benchmarks) target continuously-deployed Jamf Platform microservices and have no tenant version requirement.\n\n"+
				"This provider builds on the work of [Deployment Theory](https://github.com/deploymenttheory)'s [terraform-provider-jamfpro](https://github.com/deploymenttheory/terraform-provider-jamfpro) — first released in early 2024, the most widely adopted community Terraform provider for Jamf.",
			providerdata.ProviderMinJamfProVersion,
		),
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Required. The Jamf Platform base URL to use (e.g., https://us.apigw.jamf.com for production US region or https://us.stage.apigw.jamfnebula.com for internal staging US region). Must be set either here or via the JAMFPLATFORM_BASE_URL environment variable. Marked Optional in the schema so it can be sourced from the environment; the provider errors at configure time if it is set in neither place.",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Description: "Required. OAuth client ID for Jamf Platform API. Must be set either here or via the JAMFPLATFORM_CLIENT_ID environment variable. Marked Optional in the schema so it can be sourced from the environment; the provider errors at configure time if it is set in neither place.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Required. OAuth client secret for Jamf Platform API. Must be set either here or via the JAMFPLATFORM_CLIENT_SECRET environment variable. Marked Optional in the schema so it can be sourced from the environment; the provider errors at configure time if it is set in neither place.",
			},
			"tenant_id": schema.StringAttribute{
				Optional:    true,
				Description: "Required. Tenant UUID used to scope all API requests. Must be set either here or via the JAMFPLATFORM_TENANT_ID environment variable. Marked Optional in the schema so it can be sourced from the environment; the provider errors at configure time if it is set in neither place.",
			},
			"min_request_interval_ms": schema.Int64Attribute{
				Optional:    true,
				Description: "Minimum elapsed time, in milliseconds, between the start of consecutive outbound API requests. Paces all traffic through the shared client (which Terraform fans out across parallel resource operations), giving the server breathing room and reducing rate-limit responses. Defaults to 100. Set to 0 to disable. Raising it slows large parallel applies; lowering it increases the chance of 429s. Can also be set via the JAMFPLATFORM_MIN_REQUEST_INTERVAL_MS environment variable.",
			},
			"impact_alerts": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "Show **impact alerts** during `terraform plan`: an advisory warning on each scopeable or deployable object whose scope is changing, reporting how many computers or mobile devices the change affects. " +
					"Mirrors the impact alert Jamf Pro shows on Save, and reads the same group membership counts the admin UI does.\n\n" +
					"Alerts are advisory only — they never block a plan, and a tenant that cannot be read simply produces one notice. Off by default, because enabling it reads group membership counts and device totals once per plan. " +
					"Figures are a snapshot: group membership is re-evaluated continuously, so the number affected can change before or during apply.\n\n" +
					"This does not change any Jamf Pro setting. To configure the impact alerts Jamf Pro shows in its own web interface, use `jamfplatform_pro_impact_alert_notification_settings`. " +
					"Can also be set via the JAMFPLATFORM_IMPACT_ALERTS environment variable.",
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
	}
	// Inter-request pacing. The SDK defaults to 100ms when the option is not
	// passed; only override when the operator sets it explicitly (including 0 to
	// disable) via the attribute or the env var, attribute taking precedence.
	// Eventual-consistency retries are NOT done by the transport — resources that
	// need them poll explicitly (see the apps and device_group Delete paths).
	minIntervalSet := false
	var minIntervalMs int64
	switch {
	case !data.MinRequestIntervalMs.IsNull() && !data.MinRequestIntervalMs.IsUnknown():
		minIntervalSet = true
		minIntervalMs = data.MinRequestIntervalMs.ValueInt64()
	case getenv(envMinRequestIntervalMs) != "":
		parsed, err := strconv.ParseInt(getenv(envMinRequestIntervalMs), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid JAMFPLATFORM_MIN_REQUEST_INTERVAL_MS",
				fmt.Sprintf("Expected an integer number of milliseconds, got %q: %s", getenv(envMinRequestIntervalMs), err),
			)
			return
		}
		minIntervalSet = true
		minIntervalMs = parsed
	}
	if minIntervalSet {
		opts = append(opts, jamfplatform.WithMinRequestInterval(time.Duration(minIntervalMs)*time.Millisecond))
	}
	if shouldEnableHTTPLogging() {
		opts = append(opts, jamfplatform.WithLogger(NewTerraformLogger()))
	}
	apiClient := jamfplatform.NewClient(baseURL, clientID, clientSecret, opts...)

	if err := apiClient.ValidateCredentials(ctx); err != nil {
		summary, detail := authFailureDiagnostic(baseURL, err)
		resp.Diagnostics.AddError(summary, detail)
		return
	}

	tflog.Info(ctx, "Jamf Platform provider configured", map[string]any{
		"provider_version":     p.version,
		"jamf_pro_api_version": jamfplatform.JamfProAPIVersion,
		"provider_pro_floor":   providerdata.ProviderMinJamfProVersion,
	})

	pd := providerdata.New(apiClient)
	if impactAlertsEnabled(data.ImpactAlerts) {
		pd.EnableImpactAlerts()
	}
	resp.DataSourceData = pd
	resp.ResourceData = pd
	resp.ListResourceData = pd
	resp.ActionData = pd
}

// Functions registers the provider-defined functions exposed under the
// jamfplatform:: namespace.
func (p *JamfPlatformProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{
		mcxforcedpayload.NewFunction,
		mobileconfig.NewFunction,
	}
}

func (p *JamfPlatformProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		account.NewAccountResource,
		account_group.NewAccountGroupResource,
		api_client.NewApiClientResource,
		api_role.NewApiRoleResource,
		automated_device_enrollment.NewAutomatedDeviceEnrollmentResource,
		benchmark.NewBenchmarkResource,
		blueprint.NewBlueprintResource,
		allowed_file_extension.NewAllowedFileExtensionResource,
		building.NewBuildingResource,
		category.NewCategoryResource,
		computer_extension_attribute.NewComputerExtensionAttributeResource,
		mobile_device_extension_attribute.NewMobileDeviceExtensionAttributeResource,
		user_extension_attribute.NewUserExtensionAttributeResource,
		cloud_distribution_point.NewCloudDistributionPointResource,
		file_share_distribution_point.NewFileShareDistributionPointResource,
		cloud_identity_provider.NewCloudIdentityProviderResource,
		department.NewDepartmentResource,
		device_group.NewDeviceGroupResource,
		directory_binding.NewDirectoryBindingResource,
		ldap_server.NewLdapServerResource,
		local_admin_password_settings.NewLocalAdminPasswordSettingsResource,
		disk_encryption_configuration.NewDiskEncryptionConfigurationResource,
		computer_invitation.NewComputerInvitationResource,
		mobile_device_invitation.NewMobileDeviceInvitationResource,
		computer_prestage_enrollment.NewComputerPrestageEnrollmentResource,
		mobile_device_prestage_enrollment.NewMobileDevicePrestageEnrollmentResource,
		return_to_service.NewReturnToServiceResource,
		dock_item.NewDockItemResource,
		enrollment_customization.NewEnrollmentCustomizationResource,
		mobile_device_enrollment_profile.NewEnrollmentProfileResource,
		supervision_identity.NewSupervisionIdentityResource,
		ibeacon.NewIbeaconResource,
		icon.NewIconResource,
		inventory_preload_record.NewInventoryPreloadRecordResource,
		licensed_software.NewLicensedSoftwareResource,
		app_installer.NewAppInstallerResource,
		app_request_form_field.NewAppRequestFormFieldResource,
		app_request_settings.NewAppRequestSettingsResource,
		ebook.NewEbookResource,
		mac_app_store_app.NewMacAppResource,
		mobile_device_app.NewMobileAppResource,
		mobile_device_provisioning_profile.NewProvisioningProfileResource,
		macos_configuration_profile.NewResource,
		mobile_device_configuration_profile.NewResource,
		network_segment.NewNetworkSegmentResource,
		pkg.NewPackageResource,
		patch_external_source.NewPatchExternalSourceResource,
		patch_policy.NewPatchPolicyResource,
		patch_software_title.NewPatchSoftwareTitleResource,
		policy.NewPolicyResource,
		restricted_software.NewRestrictedSoftwareResource,
		printer.NewPrinterResource,
		removable_mac_address.NewRemovableMacAddressResource,
		re_enrollment_settings.NewReEnrollmentSettingsResource,
		access_management_settings.NewAccessManagementSettingsResource,
		user_initiated_enrollment_settings.NewUserInitiatedEnrollmentSettingsResource,
		script.NewScriptResource,
		advanced_computer_search.NewAdvancedComputerSearchResource,
		advanced_mobile_device_search.NewAdvancedMobileDeviceSearchResource,
		advanced_user_search.NewAdvancedUserSearchResource,
		advanced_volume_purchasing_content_search.NewAdvancedVolumePurchasingContentSearchResource,
		app_installer_settings.NewAppInstallerSettingsResource,
		self_service_plus_settings.NewSelfServicePlusSettingsResource,
		activation_code.NewActivationCodeResource,
		computer_check_in_settings.NewComputerCheckInSettingsResource,
		computer_inventory_collection_settings.NewComputerInventoryCollectionSettingsResource,
		gsx_connection.NewGsxConnectionSettingsResource,
		pki_venafi.NewPkiVenafiResource,
		pki_digicert.NewDigicertResource,
		pki_adcs.NewAdcsResource,
		pki_json_web_token_configuration.NewJSONWebTokenConfigurationResource,
		impact_alert_notification_settings.NewImpactAlertNotificationSettingsResource,
		managed_software_updates.NewManagedSoftwareUpdateResource,
		jamf_connect.NewJamfConnectResource,
		jamf_parent_settings.NewJamfParentSettingsResource,
		jamf_protect.NewJamfProtectResource,
		jamf_teacher_settings.NewJamfTeacherSettingsResource,
		login_page.NewLoginPageSettingsResource,
		macos_onboarding.NewOnboardingResource,
		mdm_profile_settings.NewMDMProfileSettingsResource,
		self_service_branding_image.NewSelfServiceBrandingImageResource,
		self_service_branding_ios.NewSelfServiceBrandingIosResource,
		self_service_branding_macos.NewSelfServiceBrandingMacosResource,
		self_service_macos_settings.NewSelfServiceMacosSettingsResource,
		service_discovery_enrollment.NewServiceDiscoveryEnrollmentResource,
		smtp_server.NewSmtpServerResource,
		site.NewSiteResource,
		sso_failover_url.NewSsoFailoverURLResource,
		sso_settings.NewSsoSettingsResource,
		class.NewClassResource,
		user_group.NewUserGroupResource,
		location.NewVolumePurchasingLocationResource,
		volume_purchasing_notification.NewVolumePurchasingNotificationResource,
		vpp_assignment.NewVPPAssignmentResource,
		vpp_invitation.NewVPPInvitationResource,
		webhook.NewWebhookResource,
	}
}

func (p *JamfPlatformProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		account.NewAccountDataSource,
		account_group.NewAccountGroupDataSource,
		account_privileges.NewAccountPrivilegesDataSource,
		api_client.NewApiClientDataSource,
		api_client.NewApiClientsDataSource,
		api_role.NewApiRoleDataSource,
		api_role_privileges.NewApiRolePrivilegesDataSource,
		api_role.NewApiRolesDataSource,
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
		allowed_file_extension.NewAllowedFileExtensionDataSource,
		building.NewBuildingDataSource,
		building.NewBuildingsDataSource,
		category.NewCategoriesDataSource,
		category.NewCategoryDataSource,
		computer_extension_attribute.NewComputerExtensionAttributeDataSource,
		mobile_device_extension_attribute.NewMobileDeviceExtensionAttributeDataSource,
		user_extension_attribute.NewUserExtensionAttributeDataSource,
		cloud_distribution_point.NewCloudDistributionPointDataSource,
		file_share_distribution_point.NewFileShareDistributionPointDataSource,
		cloud_identity_provider.NewCloudIdentityProviderDataSource,
		cloud_identity_provider.NewCloudIdentityProvidersDataSource,
		department.NewDepartmentDataSource,
		department.NewDepartmentsDataSource,
		device_group.NewDeviceGroupDataSource,
		device_groups.NewDeviceGroupsDataSource,
		device.NewDeviceDataSource,
		devices.NewDevicesDataSource,
		directory_binding.NewDirectoryBindingDataSource,
		ldap_server.NewLdapServerDataSource,
		local_admin_password_settings.NewLocalAdminPasswordSettingsDataSource,
		disk_encryption_configuration.NewDiskEncryptionConfigurationDataSource,
		computer_invitation.NewComputerInvitationDataSource,
		mobile_device_invitation.NewMobileDeviceInvitationDataSource,
		computer_prestage_enrollment.NewComputerPrestageEnrollmentDataSource,
		mobile_device_prestage_enrollment.NewMobileDevicePrestageEnrollmentDataSource,
		return_to_service.NewReturnToServiceDataSource,
		dock_item.NewDockItemDataSource,
		enrollment_customization.NewEnrollmentCustomizationDataSource,
		mobile_device_enrollment_profile.NewEnrollmentProfileDataSource,
		supervision_identity.NewSupervisionIdentityDataSource,
		ibeacon.NewIbeaconDataSource,
		inventory_preload_record.NewInventoryPreloadRecordDataSource,
		licensed_software.NewLicensedSoftwareDataSource,
		app_installer.NewAppInstallerDataSource,
		app_installer.NewAppInstallersDataSource,
		app_installer_title.NewAppInstallerTitleDataSource,
		app_installer_title.NewAppInstallerTitlesDataSource,
		app_request_form_field.NewAppRequestFormFieldDataSource,
		app_store_country_codes.NewAppStoreCountryCodesDataSource,
		ebook.NewEbookDataSource,
		mac_app_store_app.NewMacAppDataSource,
		mobile_device_app.NewMobileAppDataSource,
		mobile_device_provisioning_profile.NewProvisioningProfileDataSource,
		macos_configuration_profile.NewDataSource,
		mobile_device_configuration_profile.NewDataSource,
		network_segment.NewNetworkSegmentDataSource,
		network_segment.NewNetworkSegmentsDataSource,
		pkg.NewPackageDataSource,
		policy.NewPolicyDataSource,
		restricted_software.NewRestrictedSoftwareDataSource,
		printer.NewPrinterDataSource,
		removable_mac_address.NewRemovableMacAddressDataSource,
		re_enrollment_settings.NewReEnrollmentSettingsDataSource,
		access_management_settings.NewAccessManagementSettingsDataSource,
		user_initiated_enrollment_settings.NewUserInitiatedEnrollmentSettingsDataSource,
		script.NewScriptDataSource,
		script.NewScriptsDataSource,
		app_installer_settings.NewAppInstallerSettingsDataSource,
		self_service_plus_settings.NewSelfServicePlusSettingsDataSource,
		activation_code.NewActivationCodeDataSource,
		computer_check_in_settings.NewComputerCheckInSettingsDataSource,
		computer_inventory_collection_settings.NewComputerInventoryCollectionSettingsDataSource,
		gsx_connection.NewGsxConnectionSettingsDataSource,
		pki_certificate_authority.NewCertificateAuthorityDataSource,
		pki_venafi.NewPkiVenafiDataSource,
		pki_digicert.NewDigicertDataSource,
		pki_adcs.NewAdcsDataSource,
		pki_json_web_token_configuration.NewJSONWebTokenConfigurationDataSource,
		impact_alert_notification_settings.NewImpactAlertNotificationSettingsDataSource,
		jamf_connect.NewJamfConnectDataSource,
		jamf_pro_server_url.NewJamfProServerURLDataSource,
		jamf_protect.NewJamfProtectPlansDataSource,
		login_page.NewLoginPageSettingsDataSource,
		macos_onboarding.NewOnboardingDataSource,
		macos_onboarding.NewOnboardingEligibleItemsDataSource,
		mdm_profile_settings.NewMDMProfileSettingsDataSource,
		self_service_branding_ios.NewSelfServiceBrandingIosDataSource,
		self_service_branding_macos.NewSelfServiceBrandingMacosDataSource,
		self_service_macos_settings.NewSelfServiceMacosSettingsDataSource,
		service_discovery_enrollment.NewServiceDiscoveryEnrollmentDataSource,
		smtp_server.NewSmtpServerDataSource,
		patch_external_source.NewPatchExternalSourceDataSource,
		patch_internal_source.NewPatchInternalSourceDataSource,
		patch_policy.NewPatchPolicyDataSource,
		patch_software_title.NewPatchSoftwareTitleDataSource,
		site.NewSiteDataSource,
		sso_failover_url.NewSsoFailoverURLDataSource,
		sso_settings.NewSsoDependenciesDataSource,
		sso_settings.NewSsoSettingsDataSource,
		sso_settings.NewSsoSpMetadataDataSource,
		site.NewSitesDataSource,
		class.NewClassDataSource,
		user_group.NewUserGroupDataSource,
		user_group.NewUserGroupsDataSource,
		user.NewUserDataSource,
		user.NewUsersDataSource,
		advanced_computer_search.NewAdvancedComputerSearchDataSource,
		advanced_mobile_device_search.NewAdvancedMobileDeviceSearchDataSource,
		advanced_user_search.NewAdvancedUserSearchDataSource,
		advanced_volume_purchasing_content_search.NewAdvancedVolumePurchasingContentSearchDataSource,
		location.NewVolumePurchasingLocationDataSource,
		volume_purchasing_notification.NewVolumePurchasingNotificationDataSource,
		vpp_assignment.NewVPPAssignmentDataSource,
		vpp_invitation.NewVPPInvitationDataSource,
		webhook.NewWebhookDataSource,
	}
}

func (p *JamfPlatformProvider) ListResources(ctx context.Context) []func() list.ListResource {
	return []func() list.ListResource{
		account.NewAccountListResource,
		account_group.NewAccountGroupListResource,
		api_client.NewApiClientListResource,
		api_role.NewApiRoleListResource,
		automated_device_enrollment.NewAutomatedDeviceEnrollmentListResource,
		benchmark.NewBenchmarkListResource,
		blueprint.NewBlueprintListResource,
		allowed_file_extension.NewAllowedFileExtensionListResource,
		building.NewBuildingListResource,
		category.NewCategoryListResource,
		file_share_distribution_point.NewFileShareDistributionPointListResource,
		computer_extension_attribute.NewComputerExtensionAttributeListResource,
		mobile_device_extension_attribute.NewMobileDeviceExtensionAttributeListResource,
		user_extension_attribute.NewUserExtensionAttributeListResource,
		cloud_identity_provider.NewCloudIdentityProviderListResource,
		department.NewDepartmentListResource,
		device_group.NewDeviceGroupListResource,
		directory_binding.NewDirectoryBindingListResource,
		ldap_server.NewLdapServerListResource,
		disk_encryption_configuration.NewDiskEncryptionConfigurationListResource,
		computer_invitation.NewComputerInvitationListResource,
		mobile_device_invitation.NewMobileDeviceInvitationListResource,
		computer_prestage_enrollment.NewComputerPrestageEnrollmentListResource,
		mobile_device_prestage_enrollment.NewMobileDevicePrestageEnrollmentListResource,
		return_to_service.NewReturnToServiceListResource,
		dock_item.NewDockItemListResource,
		enrollment_customization.NewEnrollmentCustomizationListResource,
		mobile_device_enrollment_profile.NewEnrollmentProfileListResource,
		supervision_identity.NewSupervisionIdentityListResource,
		ibeacon.NewIbeaconListResource,
		inventory_preload_record.NewInventoryPreloadRecordListResource,
		pki_json_web_token_configuration.NewJSONWebTokenConfigurationListResource,
		licensed_software.NewLicensedSoftwareListResource,
		app_installer.NewAppInstallerListResource,
		app_request_form_field.NewAppRequestFormFieldListResource,
		ebook.NewEbookListResource,
		mac_app_store_app.NewMacAppListResource,
		mobile_device_app.NewMobileAppListResource,
		mobile_device_provisioning_profile.NewProvisioningProfileListResource,
		macos_configuration_profile.NewListResource,
		mobile_device_configuration_profile.NewListResource,
		network_segment.NewNetworkSegmentListResource,
		pkg.NewPackageListResource,
		policy.NewPolicyListResource,
		restricted_software.NewRestrictedSoftwareListResource,
		printer.NewPrinterListResource,
		removable_mac_address.NewRemovableMacAddressListResource,
		script.NewScriptListResource,
		patch_external_source.NewPatchExternalSourceListResource,
		patch_policy.NewPatchPolicyListResource,
		patch_software_title.NewPatchSoftwareTitleListResource,
		site.NewSiteListResource,
		class.NewClassListResource,
		user_group.NewUserGroupListResource,
		advanced_computer_search.NewAdvancedComputerSearchListResource,
		advanced_mobile_device_search.NewAdvancedMobileDeviceSearchListResource,
		advanced_user_search.NewAdvancedUserSearchListResource,
		advanced_volume_purchasing_content_search.NewAdvancedVolumePurchasingContentSearchListResource,
		location.NewVolumePurchasingLocationListResource,
		volume_purchasing_notification.NewVolumePurchasingNotificationListResource,
		vpp_assignment.NewVPPAssignmentListResource,
		vpp_invitation.NewVPPInvitationListResource,
		webhook.NewWebhookListResource,
	}
}

func (p *JamfPlatformProvider) Actions(ctx context.Context) []func() action.Action {
	return []func() action.Action{
		deviceactions.NewEraseAction,
		deviceactions.NewRestartAction,
		deviceactions.NewShutdownAction,
		deviceactions.NewUnmanageAction,
		msuactions.NewPlanAction,
		msuactions.NewAbandonFeatureToggleAction,
		mdmactions.NewDeviceLockAction,
		mdmactions.NewEnableLostModeAction,
		mdmactions.NewDisableLostModeAction,
		mdmactions.NewPlayLostModeSoundAction,
		mdmactions.NewEnableRemoteDesktopAction,
		mdmactions.NewDisableRemoteDesktopAction,
		mdmactions.NewClearRestrictionsPasswordAction,
		mdmactions.NewClearPasscodeAction,
		mdmactions.NewDeleteUserAction,
		mdmactions.NewLogOutUserAction,
		mdmactions.NewUnlockUserAccountAction,
		mdmactions.NewSetAutoAdminPasswordAction,
		mdmactions.NewTriggerEnhancedLogCollectionAction,
		mdmactions.NewCancelEnhancedLogCollectionAction,
		mdmactions.NewSendBlankPushAction,
		mdmactions.NewRenewMdmProfileAction,
		mdmactions.NewFlushMdmCommandsAction,
		maintenanceactions.NewRedeployManagementFrameworkAction,
		maintenanceactions.NewFlushPolicyLogsAction,
		patchactions.NewRetryPatchPolicyLogsAction,
		jamfprotectactions.NewSyncPlansAction,
		jamfprotectactions.NewRetryDeploymentAction,
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
// impactAlertsEnabled resolves the impact_alerts setting, attribute taking
// precedence over the environment variable. Unparseable env values are treated
// as off rather than erroring: impact alerts are advisory, so a typo here must
// not stop a plan that would otherwise succeed.
func impactAlertsEnabled(attr types.Bool) bool {
	if !attr.IsNull() && !attr.IsUnknown() {
		return attr.ValueBool()
	}
	v, ok := os.LookupEnv(envImpactAlerts)
	if !ok {
		return false
	}
	enabled, err := strconv.ParseBool(strings.TrimSpace(v))
	return err == nil && enabled
}

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
