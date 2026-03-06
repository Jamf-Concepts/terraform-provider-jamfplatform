// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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

// BlueprintResource implements the Terraform resource for Jamf Blueprint.
type BlueprintResource struct {
	client *client.Client
}

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

// uuidRegex matches UUID strings used to validate device group IDs.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

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
		Version:             2,
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
						uuidRegex,
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
			"legacy_payloads": schema.DynamicAttribute{
				MarkdownDescription: "Legacy configuration profile payloads as a list of objects. Each object must have a `payload_type` key (Apple reverse-domain identifier, e.g. `com.apple.applicationaccess`) and an optional `settings` object containing the payload key-value pairs. The payload identifier is auto-generated and the display name uses the blueprint name.",
				Optional:            true,
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
			"raw_component": schema.SetNestedAttribute{
				MarkdownDescription: "Raw component configuration using key-value pairs.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
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
			"audio_accessory_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Audio accessory settings component for managing temporary pairing and unpairing policies.",
				Optional:            true,
				Attributes:          components.AudioAccessorySettingsComponentSchema(),
			},
			"custom_declarations": schema.SingleNestedAttribute{
				MarkdownDescription: "Custom declarations component for managing custom DDM declarations with system or user channel types.",
				Optional:            true,
				Attributes:          components.CustomDeclarationsComponentSchema(),
			},
			"disk_management_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Disk management settings component for controlling external and network storage restrictions.",
				Optional:            true,
				Attributes:          components.DiskManagementPolicyComponentSchema(),
			},
			"math_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Math settings component for managing calculator modes and system behavior.",
				Optional:            true,
				Attributes:          components.MathSettingsComponentSchema(),
			},
			"passcode_policy": schema.SingleNestedAttribute{
				MarkdownDescription: "Passcode policy component for managing device passcode requirements and restrictions.",
				Optional:            true,
				Attributes:          components.PasscodePolicyComponentSchema(),
			},
			"safari_bookmarks": schema.SingleNestedAttribute{
				MarkdownDescription: "Safari bookmarks component for managing Safari managed bookmarks and bookmark groups.",
				Optional:            true,
				Attributes:          components.SafariBookmarksComponentSchema(),
			},
			"safari_extensions": schema.SingleNestedAttribute{
				MarkdownDescription: "Safari extensions component for managing Safari extension permissions and states.",
				Optional:            true,
				Attributes:          components.SafariExtensionsComponentSchema(),
			},
			"safari_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Safari settings component for managing Safari browser behavior and security settings.",
				Optional:            true,
				Attributes:          components.SafariSettingsComponentSchema(),
			},
			"service_background_tasks": schema.SingleNestedAttribute{
				MarkdownDescription: "Service background tasks component for managing background service tasks and launchd configurations.",
				Optional:            true,
				Attributes:          components.ServiceBackgroundTasksComponentSchema(),
			},
			"service_configuration_files": schema.SingleNestedAttribute{
				MarkdownDescription: "Service configuration files component for managing configuration files for system services.",
				Optional:            true,
				Attributes:          components.ServiceConfigurationFilesComponentSchema(),
			},
			"software_update": schema.SingleNestedAttribute{
				MarkdownDescription: "Software update component for enforcing OS updates on devices.",
				Optional:            true,
				Attributes:          components.SoftwareUpdateComponentSchema(),
			},
			"software_update_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Software update settings component for configuring system update behavior and policies.",
				Optional:            true,
				Attributes:          components.SoftwareUpdateSettingsComponentSchema(),
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
