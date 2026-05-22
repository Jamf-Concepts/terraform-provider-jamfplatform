// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package policy implements the jamfplatform_pro_policy resource, data
// source, and list resource backed by the Jamf ProClassic policies API. It
// is the canary consumer of internal/common/scope for the Phase 5 fan-out
// across every scope-bearing classic resource.
package policy

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty: classic /policies predates the provider's overall floor.
const minJamfProVersion = ""

// PolicyResource implements the Terraform resource for Jamf Pro classic policies.
type PolicyResource struct {
	client *proclassic.Client
}

var _ resource.Resource = &PolicyResource{}
var _ resource.ResourceWithImportState = &PolicyResource{}
var _ resource.ResourceWithIdentity = &PolicyResource{}

const (
	defaultCreateTimeout = 120 * time.Second
	defaultReadTimeout   = 90 * time.Second
	defaultUpdateTimeout = 120 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewPolicyResource returns a new instance of PolicyResource.
func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *PolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_policy"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *PolicyResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro policy ID used to uniquely reference the policy.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the policy resource. STYLE_GUIDE
// §Schema rules: keep inline and as flat as possible. The 13-section policy
// schema is large by necessity — every section mirrors the SDK Policy type.
func (r *PolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro classic policy. Supports the full 13-section policy payload (general, scope, self_service, package_configuration, scripts, printers, dock_items, account_maintenance, reboot, maintenance, files_processes, user_interaction, disk_encryption). Scope targets are flat sets of numeric Jamf Pro classic IDs — interpolate `jamfplatform_device_group.x.jamf_pro_id` to bridge from Platform Services. The `scope.limit_to_users` block is intentionally omitted in v1 pending an upstream SDK fix.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Policy ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"general": schema.SingleNestedAttribute{
				MarkdownDescription: "Policy general settings. `name` is required; every other field is optional. Server-derived fields (`category_name`, `site_name`, `id`) are Computed.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Policy ID nested under `general` as returned by Jamf Pro. Server-derived; matches the top-level `id` attribute.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Policy display name. Must be unique within the tenant.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"enabled":                       optComputedBool("Whether the policy is enabled."),
					"trigger":                       optComputedString("Aggregate legacy trigger label (`EVENT`, `USER_INITIATED`, etc.)."),
					"trigger_checkin":               optComputedBool("Fire on managed check-in."),
					"trigger_enrollment_complete":   optComputedBool("Fire when device enrollment completes."),
					"trigger_login":                 optComputedBool("Fire on user login."),
					"trigger_logout":                optComputedBool("Fire on user logout."),
					"trigger_network_state_changed": optComputedBool("Fire when the device's network state changes."),
					"trigger_startup":               optComputedBool("Fire on device startup."),
					"trigger_other":                 optComputedString("Custom event name to trigger the policy."),
					"frequency":                     optComputedString("How often the policy runs. Valid values include `Once per computer`, `Once per user per computer`, `Once per user`, `Once every day`, `Once every week`, `Once every month`, `Ongoing`."),
					"retry_event":                   optComputedString("Retry trigger: `none`, `trigger`, or `check-in`."),
					"retry_attempts":                optComputedInt("Maximum number of retry attempts (-1 means no retries)."),
					"notify_on_each_failed_retry":   optComputedBool("Notify the admin on each failed retry."),
					"location_user_only":            optComputedBool("Restrict the policy to location-bound users only."),
					"target_drive":                  optComputedString("Drive target (e.g. `/`)."),
					"offline":                       optComputedBool("Allow execution while the device is offline."),
					"network_requirements":          optComputedString("Network requirements label (`Any`, `Network Limitations`, etc.)."),
					"category_id":                   optComputedString("Jamf Pro category ID. Use `-1` to clear."),
					"category_name": schema.StringAttribute{
						MarkdownDescription: "Category display name reported by Jamf Pro. Server-derived.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"site_id": optComputedString("Jamf Pro site ID scoping the policy. `-1` means no site (`NONE`)."),
					"site_name": schema.StringAttribute{
						MarkdownDescription: "Site display name reported by Jamf Pro. Server-derived.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"date_time_limitations": schema.SingleNestedAttribute{
						MarkdownDescription: "Optional schedule limitations for when the policy may run.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"activation_date":       optComputedString("Activation date (`yyyy-mm-dd hh:mm:ss`)."),
							"activation_date_epoch": optComputedInt("Activation date as a Unix epoch in milliseconds."),
							"activation_date_utc":   optComputedString("Activation date in UTC ISO-8601."),
							"expiration_date":       optComputedString("Expiration date (`yyyy-mm-dd hh:mm:ss`)."),
							"expiration_date_epoch": optComputedString("Expiration date as a base-10 integer (string-encoded — values may exceed int64)."),
							"expiration_date_utc":   optComputedString("Expiration date in UTC ISO-8601."),
							"no_execute_on": schema.SetAttribute{
								MarkdownDescription: "Day-of-week labels on which the policy must not execute (e.g. `Sun`, `Mon`, …).",
								ElementType:         types.StringType,
								Optional:            true,
								Computed:            true,
							},
							"no_execute_start": optComputedString("Daily start of the no-execute window (e.g. `5:00 PM`)."),
							"no_execute_end":   optComputedString("Daily end of the no-execute window (e.g. `7:00 AM`)."),
						},
					},
					"network_limitations": schema.SingleNestedAttribute{
						MarkdownDescription: "Optional network limitations for when the policy may run. The Jamf classic `network_limitations` block uses `network_segments` under `general` independently of `scope.limitations.network_segment_ids` — both can carry network-segment IDs but apply to different policy stages.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"minimum_network_connection": optComputedString("Minimum network connection label (`Ethernet`, `Wireless`, `No Minimum`)."),
							"any_ip_address":             optComputedBool("Whether the policy applies on any IP address."),
							"network_segment_ids": schema.SetAttribute{
								MarkdownDescription: "Network segment IDs to allow the policy to run on. Numeric Jamf Pro classic IDs as strings.",
								ElementType:         types.StringType,
								Optional:            true,
								Computed:            true,
							},
						},
					},
					"override_default_settings": schema.SingleNestedAttribute{
						MarkdownDescription: "Optional per-policy overrides for tenant-wide defaults.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"target_drive":       optComputedString("Override the target drive."),
							"distribution_point": optComputedString("Override the distribution point."),
							"force_afp_smb":      optComputedBool("Force AFP/SMB protocol."),
							"sus":                optComputedString("Software update server URL."),
						},
					},
				},
			},
			"scope": schema.SingleNestedAttribute{
				MarkdownDescription: "Policy scope. Targets are flat sets of numeric Jamf Pro classic IDs; interpolate `jamfplatform_device_group.<x>.jamf_pro_id` to bridge from Platform Services UUIDs. Setting `all_computers = true` forbids `computer_ids`, `computer_group_ids`, `building_ids`, `department_ids`. An equivalent `all_jss_users` attribute is intentionally omitted in v1 — the underlying SDK does not expose the field, so a no-op would silently scope to zero users.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"all_computers": schema.BoolAttribute{
						MarkdownDescription: "Scope the policy to every computer in the tenant. Forbids per-computer / per-group / per-building / per-department targets when true.",
						Optional:            true,
						Validators: []validator.Bool{
							scope.AllFlagConflictsWith(
								path.MatchRelative().AtParent().AtName("computer_ids"),
								path.MatchRelative().AtParent().AtName("computer_group_ids"),
								path.MatchRelative().AtParent().AtName("building_ids"),
								path.MatchRelative().AtParent().AtName("department_ids"),
							),
						},
					},
					"computer_ids":       scope.IDSetAttribute("computer"),
					"computer_group_ids": scope.IDSetAttribute("computer group"),
					"building_ids":       scope.IDSetAttribute("building"),
					"department_ids":     scope.IDSetAttribute("department"),
					"jss_user_ids":       scope.IDSetAttribute("JSS user"),
					"jss_user_group_ids": scope.IDSetAttribute("JSS user group"),
					"limitations": schema.SingleNestedAttribute{
						MarkdownDescription: "Scope limitations narrow the audience after the targets are resolved.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"network_segment_ids":                   scope.IDSetAttribute("network segment"),
							"ibeacon_ids":                           scope.IDSetAttribute("iBeacon"),
							"directory_service_or_local_user_names": scope.NameSetAttribute("directory service or local user"),
							"directory_service_user_group_names":    scope.NameSetAttribute("directory service user group"),
						},
					},
					"exclusions": schema.SingleNestedAttribute{
						MarkdownDescription: "Scope exclusions remove items that would otherwise be included by targets or limitations.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"computer_ids":                          scope.IDSetAttribute("computer"),
							"computer_group_ids":                    scope.IDSetAttribute("computer group"),
							"building_ids":                          scope.IDSetAttribute("building"),
							"department_ids":                        scope.IDSetAttribute("department"),
							"jss_user_ids":                          scope.IDSetAttribute("JSS user"),
							"jss_user_group_ids":                    scope.IDSetAttribute("JSS user group"),
							"network_segment_ids":                   scope.IDSetAttribute("network segment"),
							"ibeacon_ids":                           scope.IDSetAttribute("iBeacon"),
							"directory_service_or_local_user_names": scope.NameSetAttribute("directory service or local user"),
							"directory_service_user_group_names":    scope.NameSetAttribute("directory service user group"),
						},
					},
				},
			},
			"self_service": schema.SingleNestedAttribute{
				MarkdownDescription: "Self Service integration. The classic wire carries the notification bool as `<notification>` and the delivery method as the sibling `<notification_type>` element — the provider models them as `notification_enabled` (bool) and `notification_type` (string).",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"use_for_self_service":            optComputedBool("Expose the policy in Self Service."),
					"self_service_display_name":       optComputedString("Self Service display name (defaults to the policy name)."),
					"install_button_text":             optComputedString("Install-button label."),
					"reinstall_button_text":           optComputedString("Re-install-button label."),
					"self_service_description":        optComputedString("Self Service description (Markdown supported)."),
					"force_users_to_view_description": optComputedBool("Require users to view the description before installing."),
					"feature_on_main_page":            optComputedBool("Feature the policy on the Self Service main page."),
					"notification_enabled":            optComputedBool("Whether Self Service surfaces a notification when the policy becomes available."),
					"notification_type": schema.StringAttribute{
						MarkdownDescription: "Notification delivery method. Valid values are `Self Service` and `Self Service and Notification Center`.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf("Self Service", "Self Service and Notification Center"),
						},
					},
					"notification_subject": optComputedString("Notification subject line."),
					"notification_message": optComputedString("Notification body text."),
					"self_service_icon": schema.SingleNestedAttribute{
						MarkdownDescription: "Self Service icon. The icon binary is uploaded out-of-band; the provider surfaces the resolved id, URI, and filename. The SDK does not currently expose a base64 `data` field — track upstream if required.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"id":       optComputedString("Icon ID assigned by Jamf Pro."),
							"uri":      optComputedString("Icon URI (Computed)."),
							"filename": optComputedString("Icon filename (Computed)."),
						},
					},
					"category": schema.SingleNestedAttribute{
						MarkdownDescription: "Self Service category. The classic wire form is `<self_service_categories><category>…</category></self_service_categories>`; the SDK models a single category — the provider mirrors that.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"id":         optComputedString("Category ID."),
							"name":       optComputedString("Category display name."),
							"display_in": optComputedBool("Display the policy in this category."),
							"feature_in": optComputedBool("Feature the policy in this category."),
						},
					},
				},
			},
			"package_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "Packages to install / cache / remove.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"packages": schema.SetNestedAttribute{
						MarkdownDescription: "Set of package assignments. Each item identifies the package by classic ID; `name` is server-derived. `action` is one of `Install`, `Cache`, `Install Cached`, `Uninstall`.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":             schema.StringAttribute{MarkdownDescription: "Package ID.", Required: true},
								"name":           optComputedString("Package name (server-derived)."),
								"action":         optComputedString("Package action."),
								"fut":            optComputedBool("Fill user template at install time."),
								"feu":            optComputedBool("Fill existing user accounts."),
								"update_autorun": optComputedBool("Update autorun data."),
							},
						},
					},
				},
			},
			"scripts": schema.SingleNestedAttribute{
				MarkdownDescription: "Scripts to run as part of the policy.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"scripts": schema.SetNestedAttribute{
						MarkdownDescription: "Set of script assignments. `priority` is one of `Before`, `After`, `At Reboot`.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":          schema.StringAttribute{MarkdownDescription: "Script ID.", Required: true},
								"name":        optComputedString("Script name (server-derived)."),
								"priority":    optComputedString("Run order."),
								"parameter4":  optComputedString("Parameter 4 passed to the script."),
								"parameter5":  optComputedString("Parameter 5 passed to the script."),
								"parameter6":  optComputedString("Parameter 6 passed to the script."),
								"parameter7":  optComputedString("Parameter 7 passed to the script."),
								"parameter8":  optComputedString("Parameter 8 passed to the script."),
								"parameter9":  optComputedString("Parameter 9 passed to the script."),
								"parameter10": optComputedString("Parameter 10 passed to the script."),
								"parameter11": optComputedString("Parameter 11 passed to the script."),
							},
						},
					},
				},
			},
			"printers": schema.SingleNestedAttribute{
				MarkdownDescription: "Printers to install or remove. The classic API returns a `size` field — Computed.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"size":                   schema.Int64Attribute{MarkdownDescription: "Number of printers reported by Jamf Pro.", Computed: true},
					"leave_existing_default": optComputedBool("Leave the device's existing default printer in place."),
					"printers": schema.SetNestedAttribute{
						MarkdownDescription: "Set of printer assignments.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":           schema.StringAttribute{MarkdownDescription: "Printer ID.", Required: true},
								"name":         optComputedString("Printer name (server-derived)."),
								"action":       optComputedString("Action (`install` or `uninstall`)."),
								"make_default": optComputedBool("Make this printer the device's default."),
							},
						},
					},
				},
			},
			"dock_items": schema.SingleNestedAttribute{
				MarkdownDescription: "Dock items to add or remove.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"dock_items": schema.SetNestedAttribute{
						MarkdownDescription: "Set of dock item assignments.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":     schema.StringAttribute{MarkdownDescription: "Dock item ID.", Required: true},
								"name":   optComputedString("Dock item name (server-derived)."),
								"action": optComputedString("Action (`Add To Beginning`, `Add To End`, `Remove`)."),
							},
						},
					},
				},
			},
			"account_maintenance": schema.SingleNestedAttribute{
				MarkdownDescription: "Account maintenance actions. Account secrets surface as `Optional+Sensitive` plaintext; the server returns SHA-256 hashes which the provider exposes as separate Computed attributes. WriteOnly adoption is tracked as a follow-up.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"accounts": schema.SetNestedAttribute{
						MarkdownDescription: "Set of local account operations.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"action":   optComputedString("Account action (`Create`, `Reset`, `Delete`, `DisableFileVault2`)."),
								"username": optComputedString("Account username."),
								"realname": optComputedString("Account real (full) name."),
								"password": schema.StringAttribute{
									MarkdownDescription: "Plaintext password. Sensitive — surfaces in state until WriteOnly support lands.",
									Optional:            true,
									Sensitive:           true,
								},
								"password_sha256": schema.StringAttribute{
									MarkdownDescription: "SHA-256 hash of the password reported by Jamf Pro.",
									Computed:            true,
									PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
								},
								"archive_home_directory":    optComputedBool("Archive the home directory on deletion."),
								"archive_home_directory_to": optComputedString("Destination for the archived home directory."),
								"home":                      optComputedString("Home directory path."),
								"hint":                      optComputedString("Password hint."),
								"picture":                   optComputedString("Account picture path."),
								"admin":                     optComputedBool("Whether the account is an admin."),
								"filevault_enabled":         optComputedBool("Whether FileVault 2 is enabled for the account."),
								"secure_token_allowed":      optComputedBool("Whether the account is allowed to hold a Secure Token."),
							},
						},
					},
					"directory_bindings": schema.SetNestedAttribute{
						MarkdownDescription: "Set of directory binding assignments.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":   schema.StringAttribute{MarkdownDescription: "Directory binding ID.", Required: true},
								"name": optComputedString("Directory binding name (server-derived)."),
							},
						},
					},
					"management_account": schema.SingleNestedAttribute{
						MarkdownDescription: "Management account configuration.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"action": optComputedString("Management account action (e.g. `doNotChange`, `rotate`)."),
							"managed_password": schema.StringAttribute{
								MarkdownDescription: "Plaintext managed password (Sensitive).",
								Optional:            true,
								Sensitive:           true,
							},
							"managed_password_length": optComputedInt("Length used when randomly generating the managed password."),
						},
					},
					"open_firmware_efi_password": schema.SingleNestedAttribute{
						MarkdownDescription: "Open Firmware / EFI password configuration.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"of_mode": optComputedString("Open Firmware mode (`command` or `full`)."),
							"of_password": schema.StringAttribute{
								MarkdownDescription: "Plaintext OF/EFI password (Sensitive).",
								Optional:            true,
								Sensitive:           true,
							},
							"of_password_sha256": schema.StringAttribute{
								MarkdownDescription: "SHA-256 hash of the OF/EFI password reported by Jamf Pro.",
								Computed:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
							},
						},
					},
				},
			},
			"reboot": schema.SingleNestedAttribute{
				MarkdownDescription: "Reboot configuration after the policy completes.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"message":                        optComputedString("Reboot prompt message."),
					"startup_disk":                   optComputedString("Startup disk label."),
					"specify_startup":                optComputedString("Specified startup volume."),
					"no_user_logged_in":              optComputedString("Action when no user is logged in."),
					"user_logged_in":                 optComputedString("Action when a user is logged in."),
					"minutes_until_reboot":           optComputedInt("Minutes to wait before forcing reboot."),
					"start_reboot_timer_immediately": optComputedBool("Start the reboot countdown immediately."),
					"file_vault_2_reboot":            optComputedBool("Trigger a FileVault 2 reboot."),
				},
			},
			"maintenance": schema.SingleNestedAttribute{
				MarkdownDescription: "Maintenance tasks to run as part of the policy.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"recon":                       optComputedBool("Update inventory."),
					"reset_name":                  optComputedBool("Reset the computer name."),
					"install_all_cached_packages": optComputedBool("Install all cached packages."),
					"heal":                        optComputedBool("Heal."),
					"prebindings":                 optComputedBool("Fix prebindings."),
					"permissions":                 optComputedBool("Fix permissions."),
					"byhost":                      optComputedBool("Fix ByHost files."),
					"system_cache":                optComputedBool("Flush system cache."),
					"user_cache":                  optComputedBool("Flush all users' caches."),
					"verify":                      optComputedBool("Verify startup disk."),
				},
			},
			"files_processes": schema.SingleNestedAttribute{
				MarkdownDescription: "File and process operations.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"search_by_path":         optComputedString("Path to search by."),
					"delete_file":            optComputedBool("Delete files matching the search criteria."),
					"locate_file":            optComputedString("File name to locate."),
					"update_locate_database": optComputedBool("Update the locate database before searching."),
					"spotlight_search":       optComputedString("Spotlight query."),
					"search_for_process":     optComputedString("Process name to search for."),
					"kill_process":           optComputedBool("Kill processes matching the search."),
					"run_command":            optComputedString("Command to run."),
				},
			},
			"user_interaction": schema.SingleNestedAttribute{
				MarkdownDescription: "User interaction prompts shown around policy execution.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"message_start":            optComputedString("Message displayed before the policy runs."),
					"allow_users_to_defer":     optComputedBool("Allow the user to defer the policy."),
					"allow_deferral_until_utc": optComputedString("Maximum deferral cut-off in UTC ISO-8601."),
					"allow_deferral_minutes":   optComputedInt("Maximum deferral duration in minutes."),
					"message_finish":           optComputedString("Message displayed after the policy completes."),
				},
			},
			"disk_encryption": schema.SingleNestedAttribute{
				MarkdownDescription: "Disk encryption configuration to apply.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"action":                           optComputedString("Disk encryption action (`apply`, `remediate`, `none`)."),
					"disk_encryption_configuration_id": optComputedInt("Disk encryption configuration ID to apply."),
					"auth_restart":                     optComputedBool("Use authenticated restart."),
					"remediate_key_type":               optComputedString("Key type for remediation (`Individual`, `Institutional`, `Individual And Institutional`)."),
					"remediate_disk_encryption_configuration_id": optComputedInt("Disk encryption configuration ID used to remediate."),
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

// Configure wires the Jamf ProClassic client into the resource via the shared
// providerdata.ConfigureProClassic helper.
func (r *PolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_policy")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import by the Jamf Pro policy ID.
func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// optComputedString returns an Optional+Computed StringAttribute with the
// standard UseStateForUnknown plan modifier for server-augmented fields.
// Field-level construction helper, not a schema-section decomposition —
// STYLE_GUIDE §Schema keeps the section bodies inline.
func optComputedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

// optComputedBool is the bool sibling of optComputedString.
func optComputedBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
	}
}

// optComputedInt is the int64 sibling of optComputedString.
func optComputedInt(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	}
}
