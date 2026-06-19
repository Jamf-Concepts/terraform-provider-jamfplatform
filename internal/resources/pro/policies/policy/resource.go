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
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/ldapgroups"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required by this
// resource. Empty: classic /policies predates the provider's overall floor.
const minJamfProVersion = ""

// PolicyResource implements the Terraform resource for Jamf Pro classic policies.
type PolicyResource struct {
	client *proclassic.Client
	// ldapSearcher backs the plan-time scope directory-service user-group
	// preflight (ModifyPlan). The LDAP group search is a Pro (v1) endpoint, so
	// it is a separate client from the ProClassic CRUD client. nil until Configure.
	ldapSearcher ldapgroups.Searcher
}

var _ resource.Resource = &PolicyResource{}
var _ resource.ResourceWithImportState = &PolicyResource{}
var _ resource.ResourceWithIdentity = &PolicyResource{}
var _ resource.ResourceWithModifyPlan = &PolicyResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 90 * time.Second
	defaultUpdateTimeout = 60 * time.Second
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
// Attribute names mirror the Jamf Pro admin UI labels (STYLE_GUIDE
// §"Attribute names mirror the Jamf Pro admin UI"); the underlying wire
// element names are noted in attribute descriptions where they differ.
func (r *PolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro policy. Top-level blocks mirror the admin UI's tabs and Options sidebar: `general`, `scope`, `self_service`, `user_interaction`, and the Options payloads `packages`, `scripts`, `printers`, `disk_encryption`, `dock_items`, `local_accounts`, `management_account`, `directory_bindings`, `efi_password`, `restart_options`, `maintenance`, `files_and_processes`. Scope targets are flat sets of Jamf Pro IDs — interpolate `jamfplatform_device_group.x.jamf_pro_id` to bridge from Platform Services. The four account-maintenance payloads (`local_accounts`, `management_account`, `directory_bindings`, `efi_password`) are flattened peers of the UI sections; internally Jamf Pro stores them as a single `account_maintenance` object. The legacy Software Update and Conditional Access policy sections are **intentionally not modelled** — both are obsolete in Jamf Pro, superseded by MDM-driven app installs / OS update scheduling and the patch-management surface. If you need to drive OS or app updates from Terraform, reach for the patch / DDM resources instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Policy ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"general": schema.SingleNestedAttribute{
				MarkdownDescription: "Policy general settings. `name` is required; every other field is optional. Read-only fields (`category_name`, `site_name`, `id`) are returned by Jamf Pro.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Policy ID under `general`. Matches the top-level `id`. Returned by Jamf Pro.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Policy display name. Must be unique within the tenant.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"enabled":                         optComputedBool("Whether the policy is enabled."),
					"trigger":                         optComputedString("Aggregate trigger label (`EVENT`, `USER_INITIATED`, etc.)."),
					"trigger_checkin":                 optComputedBool("Fire on managed check-in."),
					"trigger_enrollment_complete":     optComputedBool("Fire when device enrollment completes."),
					"trigger_login":                   optComputedBool("Fire on user login."),
					"trigger_network_state_changed":   optComputedBool("Fire when the device's network state changes."),
					"trigger_startup":                 optComputedBool("Fire on device startup."),
					"trigger_other":                   optComputedString("Custom event name to trigger the policy."),
					"frequency":                       optComputedString("How often the policy runs. Valid values include `Once per computer`, `Once per user per computer`, `Once per user`, `Once every day`, `Once every week`, `Once every month`, `Ongoing`."),
					"retry_event":                     optComputedString("Retry trigger: `none`, `trigger`, or `check-in`."),
					"retry_attempts":                  optComputedInt("Maximum number of retry attempts (-1 means no retries)."),
					"notify_on_each_failed_retry":     optComputedBool("Notify the admin on each failed retry."),
					"limit_to_jamf_pro_assigned_user": optComputedBool("Restrict the policy to the Jamf Pro-assigned user only. Mirrors Options > General > Client-Side Limitations > Limit to Jamf Pro-assigned user."),
					"target_drive":                    optComputedString("Drive target (e.g. `/`)."),
					"offline":                         optComputedBool("Allow execution while the device is offline."),
					"network_requirements":            optComputedString("Network requirements label (`Any`, `Network Limitations`, etc.)."),
					"category_id":                     optComputedString("Jamf Pro category ID. Use `-1` to clear."),
					"category_name": schema.StringAttribute{
						// No UseStateForUnknown: derived from the mutable category_id, so it
						// must go Unknown when category_id changes. See STYLE_GUIDE §886.
						MarkdownDescription: "Category display name. Returned by Jamf Pro; not user-settable.",
						Computed:            true,
					},
					"site_id": optComputedString("Jamf Pro site ID scoping the policy. Use `-1` for \"no site\"."),
					"site_name": schema.StringAttribute{
						// No UseStateForUnknown: derived from the mutable site_id, so it
						// must go Unknown when site_id changes. See STYLE_GUIDE §886.
						MarkdownDescription: "Site display name. Returned by Jamf Pro; not user-settable.",
						Computed:            true,
					},
					"date_time_limitations": schema.SingleNestedAttribute{
						MarkdownDescription: "Optional schedule limitations for when the policy may run. Only the user-authored date/time inputs are surfaced — the derived epoch / UTC siblings Jamf Pro also stores are deterministic transforms of `activation_date` / `expiration_date` and can be reproduced client-side with Terraform stdlib (`formatdate`, etc.).",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"activation_date": schema.StringAttribute{
								MarkdownDescription: "Activation date in 24-hour `YYYY-MM-DD HH:MM:SS` form (e.g. `2027-06-01 14:30:00`).",
								Optional:            true,
								Computed:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										activationExpirationDatePattern,
										"Value must be 24-hour YYYY-MM-DD HH:MM:SS (e.g. 2027-06-01 14:30:00)",
									),
								},
							},
							"expiration_date": schema.StringAttribute{
								MarkdownDescription: "Expiration date in 24-hour `YYYY-MM-DD HH:MM:SS` form (e.g. `2027-12-31 23:59:59`).",
								Optional:            true,
								Computed:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										activationExpirationDatePattern,
										"Value must be 24-hour YYYY-MM-DD HH:MM:SS (e.g. 2027-12-31 23:59:59)",
									),
								},
							},
							"no_execute_on": schema.SetAttribute{
								MarkdownDescription: "Day-of-week labels on which the policy must not execute. Three-letter abbreviations: `Sun`, `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat`.",
								ElementType:         types.StringType,
								Optional:            true,
								Computed:            true,
								Validators: []validator.Set{
									setvalidator.ValueStringsAre(
										stringvalidator.OneOf("Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"),
									),
								},
							},
							"no_execute_start": schema.StringAttribute{
								MarkdownDescription: "Daily start of the no-execute window in 12-hour `h:MM AM` / `h:MM PM` form, hour 1-12 with no leading zero (e.g. `5:00 PM`, `12:30 AM`).",
								Optional:            true,
								Computed:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										noExecuteTimePattern,
										"Value must be 12-hour h:MM AM / h:MM PM (e.g. 5:00 PM)",
									),
								},
							},
							"no_execute_end": schema.StringAttribute{
								MarkdownDescription: "Daily end of the no-execute window in 12-hour `h:MM AM` / `h:MM PM` form, hour 1-12 with no leading zero (e.g. `7:00 AM`, `12:30 PM`).",
								Optional:            true,
								Computed:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										noExecuteTimePattern,
										"Value must be 12-hour h:MM AM / h:MM PM (e.g. 7:00 AM)",
									),
								},
							},
						},
					},
					"network_limitations": schema.SingleNestedAttribute{
						MarkdownDescription: "Optional network limitations for when the policy may run. This `network_limitations.network_segment_ids` list applies independently of `scope.limitations.network_segment_ids` — both can carry network-segment IDs but apply to different policy stages.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"minimum_network_connection": optComputedString("Minimum network connection label (`Ethernet`, `Wireless`, `No Minimum`)."),
							"any_ip_address":             optComputedBool("Whether the policy applies on any IP address."),
							"network_segment_ids": schema.SetAttribute{
								MarkdownDescription: "Network segment IDs the policy may run on. Jamf Pro IDs as strings.",
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
				MarkdownDescription: "Policy scope. Targets are flat sets of Jamf Pro IDs; interpolate `jamfplatform_device_group.<x>.jamf_pro_id` to bridge from Platform Services UUIDs. Setting `all_computers = true` forbids `computer_ids`, `computer_group_ids`, `building_ids`, `department_ids`. Setting `all_jss_users = true` forbids `user_ids` and `user_group_ids`. `user_ids` / `user_group_ids` map to the admin UI's \"Users\" / \"User Groups\" lists.",
				Optional:            true,
				Attributes:          scope.ComputerScopeAttributes(scope.ComputerScopeOptions{IncludeIbeacons: true}),
			},
			"self_service": schema.SingleNestedAttribute{
				MarkdownDescription: "Self Service integration. Pair `display_notifications` with `notification_location` to control whether and where Self Service surfaces a notification when the policy becomes available.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"use_for_self_service":          optComputedBool("Expose the policy in Self Service."),
					"self_service_display_name":     optComputedString("Self Service display name (defaults to the policy name)."),
					"install_button_text":           optComputedString("Install-button label. Defaults to `Install`."),
					"reinstall_button_text":         optComputedString("Re-install-button label. Defaults to `Reinstall`."),
					"self_service_description":      optComputedString("Self Service description. Markdown supported."),
					"ensure_users_view_description": optComputedBool("Force users to view the description before installing."),
					"include_in_featured_category":  optComputedBool("Feature the policy on the Self Service main page."),
					"display_notifications":         optComputedBool("Whether Self Service surfaces a notification when the policy becomes available. Pair with `notification_location` to set the delivery target."),
					"notification_location": schema.StringAttribute{
						MarkdownDescription: "Notification delivery location. Valid values: `Self Service`, `Self Service and Notification Center`.",
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
						MarkdownDescription: "Self Service icon. The icon binary is uploaded out-of-band; the provider surfaces the resolved id, URI, and filename. Uploading the icon bytes inline is not currently supported — open an issue if you need it.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"id":       optComputedString("Icon ID assigned by Jamf Pro."),
							"uri":      optComputedString("Icon URI. Returned by Jamf Pro."),
							"filename": optComputedString("Icon filename. Returned by Jamf Pro."),
						},
					},
					"categories": schema.SetNestedAttribute{
						MarkdownDescription: "Self Service categories under which the policy appears. Each entry carries its own `display_in` / `feature_in` flags, mirroring the admin UI's parallel \"Display in\" / \"Feature in\" columns. A policy may appear in multiple categories.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{MarkdownDescription: "Category ID.", Required: true},
								// `name` is server-derived from the (mutable) `id`. It MUST be
								// plain Computed — NOT Optional+Computed+UseNonNullStateForUnknown.
								// In a Set, flipping a sibling element's flag changes that
								// element's hash, so the framework can't pair it to prior state;
								// a USFU-carried known `name` then mis-correlates and trips
								// "produced an inconsistent result after apply … does not correlate".
								// Leaving `name` Unknown at plan defers Set correlation cleanly.
								"name":       schema.StringAttribute{MarkdownDescription: "Category display name. Returned by Jamf Pro.", Computed: true},
								"display_in": optComputedBool("Display the policy in this category."),
								"feature_in": optComputedBool("Feature the policy in this category."),
							},
						},
					},
				},
			},
			"packages": schema.SingleNestedAttribute{
				MarkdownDescription: "Packages to install / cache / remove. Mirrors the admin UI's Options ▸ Packages section.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"distribution_point": schema.StringAttribute{
						MarkdownDescription: "Name of the file share distribution point used for the policy. Omit to inherit the tenant default.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"packages": schema.SetNestedAttribute{
						MarkdownDescription: "Set of package assignments. Each item identifies the package by ID; `name` is returned by Jamf Pro. `action` is one of `Install`, `Cache`, `Install Cached`, `Uninstall`.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":             schema.StringAttribute{MarkdownDescription: "Package ID.", Required: true},
								"name":           optComputedString("Package display name. Returned by Jamf Pro."),
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
								"name":        optComputedString("Script display name. Returned by Jamf Pro."),
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
				MarkdownDescription: "Printers to install or remove.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"leave_existing_default": optComputedBool("Leave the device's existing default printer in place."),
					"printers": schema.SetNestedAttribute{
						MarkdownDescription: "Set of printer assignments.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":   schema.StringAttribute{MarkdownDescription: "Printer ID.", Required: true},
								"name": optComputedString("Printer display name. Returned by Jamf Pro."),
								"action": schema.StringAttribute{
									MarkdownDescription: "Action. Mirrors the admin UI dropdown: `Map` (install printer) or `Unmap` (uninstall printer).",
									Optional:            true,
									Computed:            true,
									PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
									Validators: []validator.String{
										stringvalidator.OneOf("Map", "Unmap"),
									},
								},
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
								"name":   optComputedString("Dock item display name. Returned by Jamf Pro."),
								"action": optComputedString("Action (`Add To Beginning`, `Add To End`, `Remove`)."),
							},
						},
					},
				},
			},
			// account_maintenance is flattened into four UI-aligned peer blocks
			// to mirror the admin UI's Options sidebar (Local Accounts /
			// Management Accounts / Directory Bindings / EFI Password). The
			// classic wire still nests all four under a single
			// <account_maintenance> object — the input/state builders join
			// these four fields on write and split them on read.
			"local_accounts": schema.ListNestedAttribute{
				MarkdownDescription: "Local account operations (admin UI: Options ▸ Local Accounts). Each `password` is a Terraform `WriteOnly` attribute — sent to Jamf Pro on writes but never persisted in state; pair it with `password_wo_version` to rotate. Modelled as a List (rather than a Set) so the `WriteOnly` attribute is permitted inside each element; Jamf Pro matches accounts by `username` server-side and the order has no semantic effect.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							MarkdownDescription: "Account action. Valid values: `Create`, `Reset`, `Delete`, `DisableFileVault` (the admin UI labels the last action \"Disable FileVault\" — supply `DisableFileVault` here, with no trailing `2`). **Note:** the Jamf Pro classic `/policies` API silently strips `DisableFileVault` account entries on both create and update — they do not round-trip and the policy will report no such account afterwards (confirmed against the live API regardless of username existence, sibling accounts, or extra fields). Only the Jamf Pro web UI can persist this action. Avoid `DisableFileVault` in Terraform-managed policies; it will produce a perpetual diff.",
							Optional:            true,
							Computed:            true,
							PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
							Validators: []validator.String{
								stringvalidator.OneOf("Create", "Reset", "Delete", "DisableFileVault"),
							},
						},
						"username": optComputedString("Account username."),
						"realname": optComputedString("Account real (full) name."),
						"password": schema.StringAttribute{
							MarkdownDescription: "Plaintext password used by `Create` and `Reset` actions. `WriteOnly` — sent to Jamf Pro on writes but **never persisted in Terraform state**. Pair with `password_wo_version` to rotate the stored password. `local_accounts` is a List, so bumping `password_wo_version` surfaces in `terraform plan` as an in-place change to the list element at the matching index (Jamf matches accounts by `username` server-side).",
							Optional:            true,
							Sensitive:           true,
							WriteOnly:           true,
						},
						"password_wo_version": schema.Int64Attribute{
							MarkdownDescription: "Rotation trigger for the `WriteOnly` `password`. Bump this integer (any change) to force a new apply that re-sends `password` to Jamf Pro for this account. Initial Create should set `password_wo_version = 1`. Leaving it unset or unchanged signals \"leave the stored password alone\" — the provider omits the password from the next update so Jamf Pro retains the existing value.",
							Optional:            true,
						},
						"permanently_delete_home_directory": optComputedBool("Permanently delete the home directory when `action = \"Delete\"`. When true, the home is removed; when false (or unset), the home is archived to `archive_home_directory_to`. Mirrors the admin UI checkbox \"Permanently delete home directory\"."),
						"archive_home_directory_to":         optComputedString("Destination for the archived home directory. Only meaningful when `permanently_delete_home_directory = false`."),
						"home":                              optComputedString("Home directory path."),
						"hint":                              optComputedString("Password hint."),
						"picture":                           optComputedString("Account picture path."),
						"admin":                             optComputedBool("Whether the account is an admin."),
						"filevault_enabled":                 optComputedBool("Whether FileVault 2 is enabled for the account."),
						"secure_token_allowed":              optComputedBool("Whether the account is allowed to hold a Secure Token."),
					},
				},
			},
			"management_account": schema.SingleNestedAttribute{
				MarkdownDescription: "Management account configuration (admin UI: Options ▸ Management Accounts).",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"action": optComputedString("Management account action (e.g. `doNotChange`, `rotate`)."),
					"managed_password": schema.StringAttribute{
						MarkdownDescription: "Plaintext managed password. `WriteOnly` — sent to Jamf Pro on writes but **never persisted in Terraform state**. Pair with `managed_password_wo_version` to rotate the stored password.",
						Optional:            true,
						Sensitive:           true,
						WriteOnly:           true,
					},
					"managed_password_wo_version": schema.Int64Attribute{
						MarkdownDescription: "Rotation trigger for the `WriteOnly` `managed_password`. Bump this integer (any change) to force a new apply that re-sends `managed_password` to Jamf Pro. Initial Create should set `managed_password_wo_version = 1`. Leaving it unset or unchanged signals \"leave the stored password alone\" — the provider omits the password from the next update so Jamf Pro retains the existing value.",
						Optional:            true,
					},
					"managed_password_length": optComputedInt("Length used when randomly generating the managed password."),
				},
			},
			"directory_bindings": schema.SetNestedAttribute{
				MarkdownDescription: "Directory binding assignments (admin UI: Options ▸ Directory Bindings).",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{MarkdownDescription: "Directory binding ID.", Required: true},
						"name": optComputedString("Directory binding display name. Returned by Jamf Pro."),
					},
				},
			},
			"efi_password": schema.SingleNestedAttribute{
				MarkdownDescription: "Open Firmware / EFI password configuration (admin UI: Options ▸ EFI Password).",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"of_mode": optComputedString("Open Firmware mode (`command` or `full`)."),
					"of_password": schema.StringAttribute{
						MarkdownDescription: "Plaintext Open Firmware / EFI password. `WriteOnly` — sent to Jamf Pro on writes but **never persisted in Terraform state**. Pair with `of_password_wo_version` to rotate the stored password.",
						Optional:            true,
						Sensitive:           true,
						WriteOnly:           true,
					},
					"of_password_wo_version": schema.Int64Attribute{
						MarkdownDescription: "Rotation trigger for the `WriteOnly` `of_password`. Bump this integer (any change) to force a new apply that re-sends `of_password` to Jamf Pro. Initial Create should set `of_password_wo_version = 1`. Leaving it unset or unchanged signals \"leave the stored password alone\" — the provider omits the password from the next update so Jamf Pro retains the existing value.",
						Optional:            true,
					},
				},
			},
			"restart_options": schema.SingleNestedAttribute{
				MarkdownDescription: "Reboot configuration after the policy completes. Mirrors the admin UI's Options ▸ Restart Options section.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"message":      optComputedString("Reboot prompt message."),
					"startup_disk": optComputedString("Startup disk label."),
					"specify_startup": schema.StringAttribute{
						MarkdownDescription: "Reboot-method discriminator. Empty string is the default (standard reboot, no explicit method). `Standard Restart` matches the admin UI radio option. `MDM Restart with Kernel Cache Rebuild` issues an MDM-driven restart that rebuilds the kernel cache. Note: the admin UI surfaces a separate \"KEXT PATH\" text input alongside the radio, but Jamf Pro does not echo that value back, so it is not currently exposed here.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf("", "Standard Restart", "MDM Restart with Kernel Cache Rebuild"),
						},
					},
					"no_user_logged_in":              optComputedString("Action when no user is logged in."),
					"user_logged_in":                 optComputedString("Action when a user is logged in."),
					"delay_minutes":                  optComputedInt("Minutes to wait before forcing reboot. Mirrors the admin UI \"Delay\" input."),
					"start_reboot_timer_immediately": optComputedBool("Start the reboot countdown immediately."),
					"file_vault_2_reboot":            optComputedBool("Trigger a FileVault 2 reboot."),
				},
			},
			"maintenance": schema.SingleNestedAttribute{
				MarkdownDescription: "Maintenance tasks to run as part of the policy. Attribute names mirror the Jamf Pro admin UI checkbox labels.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"update_inventory":        optComputedBool("Update inventory."),
					"reset_computer_names":    optComputedBool("Reset computer names."),
					"install_cached_packages": optComputedBool("Install cached packages."),
					"fix_disk_permissions":    optComputedBool("Fix disk permissions."),
					"fix_byhost_files":        optComputedBool("Fix ByHost files."),
					"flush_system_caches":     optComputedBool("Flush system caches."),
					"flush_user_caches":       optComputedBool("Flush user caches."),
					"verify_startup_disk":     optComputedBool("Verify startup disk."),
				},
			},
			"files_and_processes": schema.SingleNestedAttribute{
				MarkdownDescription: "File and process operations (admin UI: Options ▸ Files and Processes). Attribute names mirror the Jamf Pro admin UI labels.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"search_by_path":         optComputedString("Path to search for. Mirrors the admin UI \"Search for File by Path\" input."),
					"delete_file_if_found":   optComputedBool("Delete files matching the search criteria if found."),
					"search_by_filename":     optComputedString("File name to search for. Mirrors the admin UI \"Search for File by Filename\" input."),
					"update_locate_database": optComputedBool("Update the locate database before searching."),
					"search_by_spotlight":    optComputedString("Spotlight query. Mirrors the admin UI \"Search for File Using Spotlight\" input."),
					"search_for_process":     optComputedString("Process name to search for."),
					"kill_process_if_found":  optComputedBool("Kill processes matching the search if found."),
					"execute_command":        optComputedString("Command to execute. Mirrors the admin UI \"Execute Command\" input."),
				},
			},
			"user_interaction": schema.SingleNestedAttribute{
				MarkdownDescription: "User interaction prompts shown around policy execution. The \"Deferral Type\" dropdown (None / Date / Duration) is modelled as `deferral_type`, with `deferral_until_utc` (Date form) and `deferral_days` (Duration form) as type-specific siblings. Switching between deferral types is an in-place change.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"start_message": optComputedString("Message displayed before the policy runs. Mirrors the admin UI \"Start Message\" input."),
					"deferral_type": schema.StringAttribute{
						MarkdownDescription: "User deferral mode. Mirrors the admin UI \"Deferral Type\" dropdown. One of:\n  - `none` — no deferral allowed (the policy runs without prompting).\n  - `date` — deferral allowed until `deferral_until_utc`; the policy runs after that cut-off regardless.\n  - `duration` — deferral allowed for `deferral_days` days from first prompt.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf("none", "date", "duration"),
							DeferralTypeCompanionsValidator(),
						},
					},
					// deferral_until_utc and deferral_days are plain Optional (no
					// Computed, no UseNonNullStateForUnknown) — value-discriminated
					// siblings of deferral_type. The companions must clear to null
					// in the plan on a type transition (e.g. Date→Duration removes
					// `deferral_until_utc`); UseNonNullStateForUnknown would
					// resurrect the prior-step value and trip the cross-field
					// validator. The server never independently surfaces these
					// (their wire values are always derived from deferral_type),
					// so Computed buys nothing.
					"deferral_until_utc": schema.StringAttribute{
						MarkdownDescription: "Date/time at which deferrals are prohibited and the policy runs. ISO-8601 with millisecond precision and a four-digit numeric offset (e.g. `2027-01-01T01:00:00.000+0000`). Required when `deferral_type = \"date\"`; forbidden otherwise.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								deferralUntilUtcPattern,
								"Value must be ISO-8601 with millisecond precision and a four-digit numeric offset (e.g. 2027-01-01T01:00:00.000+0000)",
							),
						},
					},
					"deferral_days": schema.Int64Attribute{
						MarkdownDescription: "Number of days the user may defer the policy after the first prompt. Mirrors the admin UI \"Duration\" input (in days). Required when `deferral_type = \"duration\"`; forbidden otherwise.",
						Optional:            true,
					},
					"complete_message": optComputedString("Message displayed after the policy completes. Mirrors the admin UI \"Complete Message\" input."),
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

	// Pro (v1) client for the scope directory-service group preflight.
	proClient, proDiags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_policy")
	resp.Diagnostics.Append(proDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if proClient != nil {
		r.ldapSearcher = proClient
	}
}

// ImportState handles import by the Jamf Pro policy ID.
func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ModifyPlan runs the plan-time directory-service user-group preflight on the
// policy scope limitations/exclusions — surfacing an unknown group as a clear
// plan error instead of the apply-time 409 ("Problem matching limitation user
// group"). Best-effort: search errors / unconfigured LDAP downgrade to a
// warning. No-op on destroy and when no scope groups are declared.
func (r *PolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.ldapSearcher == nil || req.Plan.Raw.IsNull() {
		return
	}
	var plan PolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || plan.Scope == nil {
		return
	}
	scopeRoot := path.Root("scope")
	if plan.Scope.Limitations != nil {
		resp.Diagnostics.Append(scope.ValidateDirectoryServiceUserGroupNames(
			ctx, r.ldapSearcher, plan.Scope.Limitations.DirectoryServiceUserGroupNames,
			scopeRoot.AtName("limitations").AtName("directory_service_user_group_names"),
		)...)
	}
	if plan.Scope.Exclusions != nil {
		resp.Diagnostics.Append(scope.ValidateDirectoryServiceUserGroupNames(
			ctx, r.ldapSearcher, plan.Scope.Exclusions.DirectoryServiceUserGroupNames,
			scopeRoot.AtName("exclusions").AtName("directory_service_user_group_names"),
		)...)
	}
}

// optComputedString returns an Optional+Computed StringAttribute with the
// UseNonNullStateForUnknown plan modifier for server-augmented fields.
// Field-level construction helper, not a schema-section decomposition —
// STYLE_GUIDE §Schema keeps the section bodies inline.
//
// Why UseNonNullStateForUnknown and not UseStateForUnknown: these helpers
// are used both at top level and inside nested list elements. When a list
// element is appended (list-length growth), the new index has no prior
// state at its path — prior StateValue is Null. UseStateForUnknown copies
// that Null into the plan; if the server returns a real value for the
// field, the post-apply consistency check trips ("Provider produced
// inconsistent result after apply"). When a Sensitive sibling lives on
// the same nested element (e.g. a WriteOnly password), the error path
// is redacted up to the nearest non-sensitive ancestor, masking the
// real attribute. UseNonNullStateForUnknown leaves the plan Unknown when
// prior StateValue is Null, so the framework accepts whatever the
// server returns. Behavior is identical to UseStateForUnknown for the
// non-Null prior-state case (singletons, already-set values).
func optComputedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
	}
}

// optComputedBool is the bool sibling of optComputedString. Uses
// UseNonNullStateForUnknown for the same nested-list-growth reason —
// see the optComputedString doc comment.
func optComputedBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
	}
}

// optComputedInt is the int64 sibling of optComputedString. Uses
// UseNonNullStateForUnknown for the same nested-list-growth reason —
// see the optComputedString doc comment.
func optComputedInt(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()},
	}
}
