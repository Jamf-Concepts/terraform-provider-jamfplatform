// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BlueprintResource{}
var _ resource.ResourceWithImportState = &BlueprintResource{}

// NewBlueprintResource returns a new instance of BlueprintResource.
func NewBlueprintResource() resource.Resource {
	return &BlueprintResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *BlueprintResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints_blueprint"
}

// Schema returns the Terraform schema for the blueprint resource.
func (r *BlueprintResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource schema for creating and managing Jamf Blueprints. Blueprints are automatically deployed after successful creation or update.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the blueprint.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Blueprint name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Blueprint description.",
				Optional:    true,
			},
			"device_groups": schema.ListAttribute{
				Description: "List of device group Platform IDs to target. Specified as a list of strings in UUID format. The Platform ID can be sourced from the response body of the /api/v1/groups Jamf Pro API endpoint.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`),
						"Each device group ID must be a valid UUID",
					)),
				},
			},
			"created": schema.StringAttribute{
				Description: "Creation timestamp.",
				Computed:    true,
			},
			"updated": schema.StringAttribute{
				Description: "Last updated timestamp.",
				Computed:    true,
			},
			"deployment_state": schema.StringAttribute{
				Description: "Current deployment state.",
				Computed:    true,
			},
			"legacy_payloads": schema.StringAttribute{
				Description: "JSON-encoded array of legacy configuration profile payload objects. Refer to https://github.com/apple/device-management/tree/release/mdm/profiles for individual payload schemas. Each payload must have payloadType and payloadIdentifier fields. The payload display name will automatically use the blueprint name.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"raw_component": schema.ListNestedBlock{
				Description: "Raw component configuration using key-value pairs.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"identifier": schema.StringAttribute{
							Description: "Component identifier (e.g., com.jamf.ddm.disk-management).",
							Required:    true,
						},
						"configuration": schema.MapAttribute{
							Description: "Component configuration as key-value pairs. Each component has its own unique configuration options.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"audio_accessory_settings": schema.ListNestedBlock{
				Description:  "Audio accessory settings component for managing temporary pairing and unpairing policies.",
				NestedObject: components.AudioAccessorySettingsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"disk_management_settings": schema.ListNestedBlock{
				Description:  "Disk management settings component for controlling external and network storage restrictions.",
				NestedObject: components.DiskManagementPolicyComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"math_settings": schema.ListNestedBlock{
				Description:  "Math settings component for managing calculator modes and system behavior.",
				NestedObject: components.MathSettingsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"passcode_policy": schema.ListNestedBlock{
				Description:  "Passcode policy component for managing device passcode requirements and restrictions.",
				NestedObject: components.PasscodePolicyComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"safari_bookmarks": schema.ListNestedBlock{
				Description:  "Safari bookmarks component for managing Safari managed bookmarks and bookmark groups.",
				NestedObject: components.SafariBookmarksComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"safari_extensions": schema.ListNestedBlock{
				Description:  "Safari extensions component for managing Safari extension permissions and states.",
				NestedObject: components.SafariExtensionsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"safari_settings": schema.ListNestedBlock{
				Description:  "Safari settings component for managing Safari browser behavior and security settings.",
				NestedObject: components.SafariSettingsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"service_background_tasks": schema.ListNestedBlock{
				Description:  "Service background tasks component for managing background service tasks and launchd configurations.",
				NestedObject: components.ServiceBackgroundTasksComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"service_configuration_files": schema.ListNestedBlock{
				Description:  "Service configuration files component for managing configuration files for system services.",
				NestedObject: components.ServiceConfigurationFilesComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"software_update": schema.ListNestedBlock{
				Description:  "Software update component for enforcing OS updates on devices.",
				NestedObject: components.SoftwareUpdateComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"software_update_settings": schema.ListNestedBlock{
				Description:  "Software update settings component for configuring system update behavior and policies.",
				NestedObject: components.SoftwareUpdateSettingsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
		},
	}
}

// Configure sets up the API client for the resource from the provider configuration.
func (r *BlueprintResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// ImportState handles the import of existing Blueprint resources.
func (r *BlueprintResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
