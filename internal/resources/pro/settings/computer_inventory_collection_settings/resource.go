// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package computer_inventory_collection_settings implements the
// jamfplatform_pro_computer_inventory_collection_settings singleton resource and data
// source backed by the Jamf Pro Computer Inventory Collection Settings V2 API.
package computer_inventory_collection_settings

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this resource.
// Empty: the computer-inventory-collection-settings V2 endpoint is present at the
// provider's overall floor, so no per-resource gate is needed.
const minJamfProVersion = ""

// ComputerInventoryCollectionSettingsResource implements the singleton resource for
// Jamf Pro computer inventory collection settings (V2). Backed by an Update-only
// settings API (PATCH; no Create/Delete on the remote): Create funnels into Update;
// Delete is a no-op that only removes the object from Terraform state. The custom
// application search-path collection is managed via dedicated create/delete endpoints
// (there is no path-update endpoint), reconciled by path string in Create/Update.
type ComputerInventoryCollectionSettingsResource struct {
	client *pro.Client
}

var _ resource.Resource = &ComputerInventoryCollectionSettingsResource{}
var _ resource.ResourceWithImportState = &ComputerInventoryCollectionSettingsResource{}
var _ resource.ResourceWithIdentity = &ComputerInventoryCollectionSettingsResource{}
var _ resource.ResourceWithValidateConfig = &ComputerInventoryCollectionSettingsResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewComputerInventoryCollectionSettingsResource returns a new instance of the resource.
func NewComputerInventoryCollectionSettingsResource() resource.Resource {
	return &ComputerInventoryCollectionSettingsResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *ComputerInventoryCollectionSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_computer_inventory_collection_settings"
}

// IdentitySchema defines the identifier used for import. Singleton resources accept
// only the fixed helpers.SingletonID value.
func (r *ComputerInventoryCollectionSettingsResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Fixed singleton identifier. Always \"singleton\" — computer inventory collection settings are one-per-tenant.",
				RequiredForImport: true,
			},
		},
	}
}

// optionalComputedBool returns the canonical Optional+Computed bool attribute used for
// every collection-preference toggle. These are plain (non-nested) scalars, so
// UseStateForUnknown is the correct plan modifier.
func optionalComputedBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

// Schema returns the Terraform schema for the computer inventory collection settings resource.
func (r *ComputerInventoryCollectionSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Jamf Pro computer inventory collection settings (Settings → Computer Management → Inventory Collection). " +
			"Singleton — one record per tenant. Backed by the V2 API. " +
			"`application_search_paths` covers custom **application** search paths only; the Jamf Pro V2 API does not expose Fonts or Plug-ins custom paths (scope is fixed to `APP`). " +
			"Import with `terraform import jamfplatform_pro_computer_inventory_collection_settings.<name> singleton`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Fixed singleton identifier. Always `singleton`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Inventory Collection (General tab)
			"collect_local_user_accounts":                      optionalComputedBool("Collect local user accounts."),
			"include_home_directory_sizes":                     optionalComputedBool("Include home directory sizes when collecting local user accounts. Sub-option of `collect_local_user_accounts`: may only be `true` when `collect_local_user_accounts` is `true`."),
			"include_hidden_accounts":                          optionalComputedBool("Include hidden accounts when collecting local user accounts. Sub-option of `collect_local_user_accounts`: may only be `true` when `collect_local_user_accounts` is `true`."),
			"collect_printers":                                 optionalComputedBool("Collect printers."),
			"collect_active_services":                          optionalComputedBool("Collect active services."),
			"collect_synced_mobile_device_backup_dates":        optionalComputedBool("Collect last backup date/time for managed mobile devices that are synced to computers."),
			"collect_user_and_location_from_directory_service": optionalComputedBool("Collect user and location information from Directory Service."),
			"collect_package_receipts":                         optionalComputedBool("Collect package receipts."),
			"collect_available_software_updates":               optionalComputedBool("Collect available software updates."),
			"collect_unmanaged_certificates":                   optionalComputedBool("Collect unmanaged certificates."),
			"monitor_beacon_regions":                           optionalComputedBool("Monitor Beacon regions."),
			"allow_jamf_binary_user_and_location_changes":      optionalComputedBool("Allow local administrators to use the jamf binary recon verb to change User and Location inventory information in Jamf Pro (Advanced)."),

			// Software tab → Applications
			"collect_application_usage_information": optionalComputedBool("Collect Application Usage Information."),
			"use_unix_user_paths":                   optionalComputedBool("Enable inventory collection of applications in user (UNIX) home-directory paths."),

			// Read-only server-managed preference (no admin-UI control)
			"include_software_id": schema.BoolAttribute{
				MarkdownDescription: "Whether the inventory submission includes a software identifier. Server-managed; exposed read-only.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},

			"application_search_paths": schema.SetAttribute{
				MarkdownDescription: "Custom **application** search paths used when collecting applications (Software → Applications → Custom Search Paths). " +
					"Built-in paths (e.g. `/Applications/`, `/System/Applications/`) are managed by Jamf Pro and are not included here. " +
					"Changing an entry replaces it (the API has no path-update endpoint). " +
					"Omit the attribute to leave the tenant's custom paths unmanaged; set it to `[]` to remove all custom application paths.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},

			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// ValidateConfig rejects the one attribute combination the Jamf Pro admin UI greys out
// and the server refuses: include_home_directory_sizes and include_hidden_accounts are
// sub-options nested under collect_local_user_accounts, so they cannot be true while
// account collection is off. The server silently forces them off in that case, which
// would otherwise surface as a confusing "inconsistent result after apply" — catching it
// at config-validation time gives the user an actionable error instead. (A plan
// modifier cannot fix this: the framework forbids overriding an explicitly-configured
// value, and a sub-option left unset already resolves to false harmlessly.)
func (r *ComputerInventoryCollectionSettingsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg ComputerInventoryCollectionSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only enforce when account collection is known to be false; if it is unknown or
	// unset we cannot (and need not) judge the sub-options.
	if cfg.CollectLocalUserAccounts.IsNull() || cfg.CollectLocalUserAccounts.IsUnknown() || cfg.CollectLocalUserAccounts.ValueBool() {
		return
	}

	const detail = "This sub-option requires collect_local_user_accounts to be true. Set collect_local_user_accounts = true, or remove this attribute (or set it to false)."
	if !cfg.IncludeHomeDirectorySizes.IsNull() && !cfg.IncludeHomeDirectorySizes.IsUnknown() && cfg.IncludeHomeDirectorySizes.ValueBool() {
		resp.Diagnostics.AddAttributeError(path.Root("include_home_directory_sizes"), "Invalid Attribute Combination", detail)
	}
	if !cfg.IncludeHiddenAccounts.IsNull() && !cfg.IncludeHiddenAccounts.IsUnknown() && cfg.IncludeHiddenAccounts.ValueBool() {
		resp.Diagnostics.AddAttributeError(path.Root("include_hidden_accounts"), "Invalid Attribute Combination", detail)
	}
}

// Configure wires the Jamf Pro client into the resource via the shared
// providerdata.ConfigurePro helper.
func (r *ComputerInventoryCollectionSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_computer_inventory_collection_settings")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import for the singleton. Only the fixed helpers.SingletonID
// value is accepted; any other identifier is rejected with a clear error.
//
//	terraform import jamfplatform_pro_computer_inventory_collection_settings.<name> singleton
func (r *ComputerInventoryCollectionSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != helpers.SingletonID {
		resp.Diagnostics.AddError(
			"Invalid singleton import identifier",
			fmt.Sprintf(
				"jamfplatform_pro_computer_inventory_collection_settings is a singleton resource and must be imported with id %q. Got %q.",
				helpers.SingletonID, req.ID,
			),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
