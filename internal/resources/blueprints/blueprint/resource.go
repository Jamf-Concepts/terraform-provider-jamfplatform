// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BlueprintResource{}
var _ resource.ResourceWithImportState = &BlueprintResource{}
var _ resource.ResourceWithIdentity = &BlueprintResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

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
		MarkdownDescription: "Resource schema for creating and managing Jamf Blueprints. Requires **Blueprints API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the blueprint.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Blueprint name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Blueprint description.",
				Optional:            true,
			},
			"deployed": schema.BoolAttribute{
				MarkdownDescription: "Whether the blueprint should be deployed. If set to `true`, the provider will deploy the blueprint (and redeploy if it becomes `OUT_OF_DATE`). If set to `false`, the provider will undeploy the blueprint.",
				Required:            true,
			},
			"device_groups": schema.SetAttribute{
				MarkdownDescription: "Set of device group Platform IDs to target. Specified as a set of strings in UUID format.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`),
						"Each device group ID must be a valid UUID",
					)),
				},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp.",
				Computed:            true,
			},
			"updated": schema.StringAttribute{
				MarkdownDescription: "Last updated timestamp.",
				Computed:            true,
			},
			"deployment_state": schema.StringAttribute{
				MarkdownDescription: "Current deployment state.",
				Computed:            true,
			},
			"legacy_payloads": schema.StringAttribute{
				MarkdownDescription: "JSON-encoded array of legacy configuration profile payload objects. Refer to https://github.com/apple/device-management/tree/release/mdm/profiles for individual payload schemas. Each payload must have `payloadType` and `payloadIdentifier` fields. The payload display name will automatically use the blueprint name.",
				Optional:            true,
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
		Blocks: map[string]schema.Block{
			"raw_component": schema.ListNestedBlock{
				MarkdownDescription: "Raw component configuration using key-value pairs.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"identifier": schema.StringAttribute{
							MarkdownDescription: "Component identifier (e.g., `com.jamf.ddm.disk-management`).",
							Required:            true,
						},
						"configuration": schema.MapAttribute{
							MarkdownDescription: "Component configuration as key-value pairs. Each component has its own unique configuration options.",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
			"audio_accessory_settings": schema.ListNestedBlock{
				MarkdownDescription: "Audio accessory settings component for managing temporary pairing and unpairing policies.",
				NestedObject:        components.AudioAccessorySettingsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"custom_declarations": schema.ListNestedBlock{
				MarkdownDescription: "Custom declarations component for managing custom DDM declarations with system or user channel types.",
				NestedObject:        components.CustomDeclarationsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"disk_management_settings": schema.ListNestedBlock{
				MarkdownDescription: "Disk management settings component for controlling external and network storage restrictions.",
				NestedObject:        components.DiskManagementPolicyComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"math_settings": schema.ListNestedBlock{
				MarkdownDescription: "Math settings component for managing calculator modes and system behavior.",
				NestedObject:        components.MathSettingsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"passcode_policy": schema.ListNestedBlock{
				MarkdownDescription: "Passcode policy component for managing device passcode requirements and restrictions.",
				NestedObject:        components.PasscodePolicyComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"safari_bookmarks": schema.ListNestedBlock{
				MarkdownDescription: "Safari bookmarks component for managing Safari managed bookmarks and bookmark groups.",
				NestedObject:        components.SafariBookmarksComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"safari_extensions": schema.ListNestedBlock{
				MarkdownDescription: "Safari extensions component for managing Safari extension permissions and states.",
				NestedObject:        components.SafariExtensionsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"safari_settings": schema.ListNestedBlock{
				MarkdownDescription: "Safari settings component for managing Safari browser behavior and security settings.",
				NestedObject:        components.SafariSettingsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"service_background_tasks": schema.ListNestedBlock{
				MarkdownDescription: "Service background tasks component for managing background service tasks and launchd configurations.",
				NestedObject:        components.ServiceBackgroundTasksComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"service_configuration_files": schema.ListNestedBlock{
				MarkdownDescription: "Service configuration files component for managing configuration files for system services.",
				NestedObject:        components.ServiceConfigurationFilesComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"software_update": schema.ListNestedBlock{
				MarkdownDescription: "Software update component for enforcing OS updates on devices.",
				NestedObject:        components.SoftwareUpdateComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"software_update_settings": schema.ListNestedBlock{
				MarkdownDescription: "Software update settings component for configuring system update behavior and policies.",
				NestedObject:        components.SoftwareUpdateSettingsComponentSchema(),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
		},
	}
}

// IdentitySchema defines the blueprint identity used across CRUD and list.
func (r *BlueprintResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Blueprint ID used to uniquely reference Jamf blueprints.",
				RequiredForImport: true,
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
