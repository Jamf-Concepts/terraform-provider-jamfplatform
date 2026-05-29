// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package user_initiated_enrollment_settings implements the
// jamfplatform_pro_user_initiated_enrollment_settings singleton resource and
// data source. It wraps the Jamf Pro User-Initiated Enrollment settings page
// (General, Computers and Devices tabs), the embedded MDM-profile signing and
// QuickAdd developer signing certificates, and the directory-service Access
// Group collection.
package user_initiated_enrollment_settings

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required.
// Empty: the User-Initiated Enrollment settings endpoint ships at the
// provider's overall floor.
const minJamfProVersion = ""

// UserInitiatedEnrollmentSettingsResource implements the singleton Jamf Pro
// User-Initiated Enrollment settings resource.
//
// The resource is backed by an Update-only Jamf Pro API — one User-Initiated
// Enrollment settings object per tenant. Create funnels into a
// read-merge-write. Delete is state-only by design.
type UserInitiatedEnrollmentSettingsResource struct {
	client *pro.Client

	// enrollmentMu serializes writes to the shared enrollment-settings backing
	// store. The User-Initiated Enrollment settings object and the
	// Re-enrollment settings object are two views of ONE record, and this
	// resource's write is a read-merge-write that must round-trip fields it
	// does not own. Obtained by reference from the shared providerdata.Data at
	// Configure; the same *sync.Mutex instance is handed to every enrollment
	// resource.
	enrollmentMu *sync.Mutex
}

var _ resource.Resource = &UserInitiatedEnrollmentSettingsResource{}
var _ resource.ResourceWithImportState = &UserInitiatedEnrollmentSettingsResource{}
var _ resource.ResourceWithIdentity = &UserInitiatedEnrollmentSettingsResource{}
var _ resource.ResourceWithConfigValidators = &UserInitiatedEnrollmentSettingsResource{}

// Default timeouts.
const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 30 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 30 * time.Second
)

// NewUserInitiatedEnrollmentSettingsResource constructs the resource.
func NewUserInitiatedEnrollmentSettingsResource() resource.Resource {
	return &UserInitiatedEnrollmentSettingsResource{}
}

// Metadata sets the resource type name.
func (r *UserInitiatedEnrollmentSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_user_initiated_enrollment_settings"
}

// IdentitySchema defines the import identifier — singleton id only.
func (r *UserInitiatedEnrollmentSettingsResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Fixed singleton identifier. Always \"singleton\".",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the resource schema. Per the style guide it is kept inline and
// as flat as possible — every attribute is declared in place rather than via
// attribute-returning helpers.
func (r *UserInitiatedEnrollmentSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the Jamf Pro **User-Initiated Enrollment** settings page (UI: Settings → Global → User-initiated enrollment) — the General, Computers and Devices tabs, the third-party MDM-profile signing certificate, the QuickAdd package signing certificate, and the directory-service Access Groups. Singleton — one record per tenant.\n\n" +
			"**Shared backing record** — the User-Initiated Enrollment settings and the Re-enrollment settings (`jamfplatform_pro_re_enrollment_settings`) are two views of one tenant record. This resource preserves the Re-enrollment options untouched on every apply; manage those with the dedicated re-enrollment resource. Within a single Terraform run the provider serializes writes to the shared record, but two separate `terraform apply` processes against the same tenant can still race.\n\n" +
			"**Third-party MDM signing certificate** — set `signing_mdm_profile_enabled = true` and supply the `mdm_signing_certificate` block to upload a keystore. Leaving the block absent on a later apply preserves the existing certificate. Setting `signing_mdm_profile_enabled = false` removes the stored certificate. When `signing_mdm_profile_enabled = true`, the `mdm_signing_certificate` block is required (plan-time validated).\n\n" +
			"**Destroy** — `terraform destroy` removes the resource from Terraform state only. The settings, certificates and Access Groups are left intact on the tenant. To reset options, set them explicitly and apply before destroy.\n\n" +
			"Import with `terraform import jamfplatform_pro_user_initiated_enrollment_settings.<name> singleton`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Fixed singleton identifier. Always `singleton`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			// ===== General tab =====
			"skip_certificate_installation": schema.BoolAttribute{
				MarkdownDescription: "Skip certificate installation during enrollment. Matches the \"Skip certificate installation during enrollment\" checkbox.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"restrict_reenrollment": schema.BoolAttribute{
				MarkdownDescription: "Restrict re-enrollment to authorized users only. Matches the \"Restrict re-enrollment to authorized users only\" checkbox.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"signing_mdm_profile_enabled": schema.BoolAttribute{
				MarkdownDescription: "Use a third-party signing certificate to sign the MDM profile. Matches the \"Use a third-party signing certificate\" checkbox. When `true`, supply the `mdm_signing_certificate` block (or rely on a previously-uploaded certificate).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},

			// ===== Computers tab =====
			"enable_computer_enrollment": schema.BoolAttribute{
				MarkdownDescription: "Enable user-initiated enrollment for computers. Matches the computers \"Enable user-initiated enrollment\" toggle.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"create_management_account": schema.BoolAttribute{
				MarkdownDescription: "Create a managed local administrator account on enrolled computers. Matches the \"Create managed local administrator account\" checkbox.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"management_username": schema.StringAttribute{
				MarkdownDescription: "Username for the managed local administrator account created on enrolled computers. Matches the \"Management Account\" username field.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"hide_management_account": schema.BoolAttribute{
				MarkdownDescription: "Hide the managed local administrator account on enrolled computers. Matches the \"Hide managed local administrator account\" checkbox.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"allow_ssh_only_management_account": schema.BoolAttribute{
				MarkdownDescription: "Allow the managed local administrator account SSH access only. Matches the \"Allow SSH access for the managed local administrator account only\" checkbox.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"ensure_ssh_running": schema.BoolAttribute{
				MarkdownDescription: "Ensure SSH (Remote Login) is enabled on enrolled computers. Matches the \"Ensure SSH is enabled\" checkbox.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"launch_self_service": schema.BoolAttribute{
				MarkdownDescription: "Launch Self Service after a computer completes enrollment. Matches the \"Launch Self Service when done\" checkbox.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sign_quickadd_package": schema.BoolAttribute{
				MarkdownDescription: "Sign the QuickAdd package with a developer certificate. Matches the \"Sign QuickAdd Package\" checkbox. Supply the `developer_certificate` block to upload a signing identity.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"account_driven_device_enrollment_macos": schema.BoolAttribute{
				MarkdownDescription: "Enable Account-Driven Device Enrollment for institutionally owned computers. Matches the computers Account-Driven Device Enrollment toggle.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},

			// ===== Devices tab =====
			"profile_driven_enrollment_via_url_institutional": schema.BoolAttribute{
				MarkdownDescription: "Enable Profile-Driven Enrollment via URL for institutionally owned mobile devices. Matches the institutional Profile-Driven Enrollment via URL toggle.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"profile_driven_enrollment_via_url_personal": schema.BoolAttribute{
				MarkdownDescription: "Enable Profile-Driven Enrollment via URL for personally owned mobile devices. Matches the personal Profile-Driven Enrollment via URL toggle.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"account_driven_user_enrollment": schema.BoolAttribute{
				MarkdownDescription: "Enable Account-Driven User Enrollment for mobile devices. Matches the Account-Driven User Enrollment toggle.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"account_driven_user_enrollment_visionos": schema.BoolAttribute{
				MarkdownDescription: "Enable Account-Driven User Enrollment for Apple Vision Pro. Matches the Account-Driven User Enrollment (Apple Vision Pro) toggle.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"merge_managed_apple_account_usernames": schema.BoolAttribute{
				MarkdownDescription: "Merge matching Managed Apple Account usernames during enrollment. Matches the \"Merge matching Managed Apple Account usernames\" checkbox.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"account_driven_device_enrollment_ios": schema.BoolAttribute{
				MarkdownDescription: "Enable Account-Driven Device Enrollment for institutionally owned mobile devices. Matches the device Account-Driven Device Enrollment toggle.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"account_driven_device_enrollment_visionos": schema.BoolAttribute{
				MarkdownDescription: "Enable Account-Driven Device Enrollment for Apple Vision Pro. Matches the Account-Driven Device Enrollment (Apple Vision Pro) toggle.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},

			// ===== Deprecated =====
			"personal_device_enrollment_type": schema.StringAttribute{
				MarkdownDescription: "Personal-device enrollment type. Deprecated as of Jamf Pro 11.25 — the server ignores any supplied value and always reports `USERENROLLMENT`. Read-only.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			// ===== Certificates =====
			"mdm_signing_certificate": schema.SingleNestedAttribute{
				MarkdownDescription: "Third-party signing certificate used to sign the MDM enrollment profile. Required when `signing_mdm_profile_enabled = true`. Supply `keystore_file` (raw base64 of a `.p12`) and `keystore_password`; both are `WriteOnly`. Removing the block while `signing_mdm_profile_enabled` stays `true` preserves the existing certificate; setting `signing_mdm_profile_enabled = false` removes it.\n\n`keystore_password` is `WriteOnly` — sent to Jamf Pro on writes but never persisted in Terraform state. Bump `keystore_password_wo_version` to force the next apply to re-send the keystore and password.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"keystore_file": schema.StringAttribute{
						MarkdownDescription: "Raw base64 of the keystore file (`.p12`). Idiomatic usage: `filebase64(\"signing.p12\")`. `WriteOnly` — never persisted in Terraform state.",
						Optional:            true,
						Sensitive:           true,
						WriteOnly:           true,
					},
					"keystore_file_name": schema.StringAttribute{
						MarkdownDescription: "Optional display filename for the uploaded keystore, for your own reference. Jamf Pro does not return a filename, so this is not server-derived — it round-trips from configuration only.",
						Optional:            true,
					},
					"keystore_password": schema.StringAttribute{
						MarkdownDescription: "Keystore password. `WriteOnly` — sent to Jamf Pro on writes but never persisted in Terraform state. Pair with `keystore_password_wo_version` (the rotation companion); bump that integer to re-send.",
						Optional:            true,
						Sensitive:           true,
						WriteOnly:           true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("keystore_password_wo_version")),
						},
					},
					"keystore_password_wo_version": schema.Int64Attribute{
						MarkdownDescription: "Rotation trigger for the `WriteOnly` `keystore_password`. Bump this integer to force re-sending the keystore and password on the next apply.",
						Optional:            true,
					},
					"subject": schema.StringAttribute{
						MarkdownDescription: "Certificate subject DN. Populated by Jamf Pro while the certificate is in use; may be empty when the matching toggle is disabled.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
					},
					"serial_number": schema.StringAttribute{
						MarkdownDescription: "Certificate serial number. Populated by Jamf Pro while the certificate is in use; may be empty when the matching toggle is disabled.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
					},
				},
			},
			"developer_certificate": schema.SingleNestedAttribute{
				MarkdownDescription: "Developer signing identity used to sign the QuickAdd package when `sign_quickadd_package = true`. Supply `keystore_file` (raw base64 of a `.p12`) and `keystore_password`; both are `WriteOnly`. This path expects an Apple Developer ID signing certificate.\n\n`keystore_password` is `WriteOnly` — sent to Jamf Pro on writes but never persisted in Terraform state. Bump `keystore_password_wo_version` to force the next apply to re-send the keystore and password.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"keystore_file": schema.StringAttribute{
						MarkdownDescription: "Raw base64 of the keystore file (`.p12`). Idiomatic usage: `filebase64(\"signing.p12\")`. `WriteOnly` — never persisted in Terraform state.",
						Optional:            true,
						Sensitive:           true,
						WriteOnly:           true,
					},
					"keystore_file_name": schema.StringAttribute{
						MarkdownDescription: "Optional display filename for the uploaded keystore, for your own reference. Jamf Pro does not return a filename, so this is not server-derived — it round-trips from configuration only.",
						Optional:            true,
					},
					"keystore_password": schema.StringAttribute{
						MarkdownDescription: "Keystore password. `WriteOnly` — sent to Jamf Pro on writes but never persisted in Terraform state. Pair with `keystore_password_wo_version` (the rotation companion); bump that integer to re-send.",
						Optional:            true,
						Sensitive:           true,
						WriteOnly:           true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("keystore_password_wo_version")),
						},
					},
					"keystore_password_wo_version": schema.Int64Attribute{
						MarkdownDescription: "Rotation trigger for the `WriteOnly` `keystore_password`. Bump this integer to force re-sending the keystore and password on the next apply.",
						Optional:            true,
					},
					"subject": schema.StringAttribute{
						MarkdownDescription: "Certificate subject DN. Populated by Jamf Pro while the certificate is in use; may be empty when the matching toggle is disabled.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
					},
					"serial_number": schema.StringAttribute{
						MarkdownDescription: "Certificate serial number. Populated by Jamf Pro while the certificate is in use; may be empty when the matching toggle is disabled.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
					},
				},
			},

			// ===== Access Groups =====
			"access_group": schema.SetNestedAttribute{
				MarkdownDescription: "Directory-service Access Groups permitted to perform user-initiated enrollment (UI: Access tab). Each group is identified by its `name` and `ldap_server_id`; the provider resolves the directory's canonical group id for you (like the UI's \"Resolve\" action). Omit the block entirely to leave the tenant's Access Groups unmanaged. The built-in \"All Directory Service Users\" group always exists and cannot be created or removed — declare it (with `ldap_server_id = \"-1\"`) to edit its toggles, or leave it out to keep it untouched.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Server-assigned Access Group identifier.",
							Computed:            true,
							PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Directory-service group name, exactly as it appears in the directory. Resolved against `ldap_server_id` to the directory's group id. For the built-in group use `All Directory Service Users`.",
							Required:            true,
						},
						"ldap_server_id": schema.StringAttribute{
							MarkdownDescription: "Identifier of the LDAP / directory-service server hosting the group. Use `-1` for the built-in \"All Directory Service Users\" group (no directory lookup is performed).",
							Required:            true,
						},
						"directory_service_group_id": schema.StringAttribute{
							MarkdownDescription: "Directory's canonical identifier for the group, resolved by the provider from `name` and `ldap_server_id`. Computed — do not set. For the built-in \"All Directory Service Users\" group this is `-1`.",
							Computed:            true,
							PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
						},
						"site_id": schema.StringAttribute{
							MarkdownDescription: "Site assigned to devices enrolled through this group, or `-1` for no site.",
							Optional:            true,
							Computed:            true,
							PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
						},
						"enterprise_enrollment_enabled": schema.BoolAttribute{
							MarkdownDescription: "Allow institutional (enterprise) enrollment for members of this group.",
							Optional:            true,
							Computed:            true,
							PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
						},
						"personal_enrollment_enabled": schema.BoolAttribute{
							MarkdownDescription: "Allow personal-device enrollment for members of this group.",
							Optional:            true,
							Computed:            true,
							PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
						},
						"account_driven_user_enrollment_enabled": schema.BoolAttribute{
							MarkdownDescription: "Allow Account-Driven User Enrollment for members of this group.",
							Optional:            true,
							Computed:            true,
							PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
						},
						"require_eula": schema.BoolAttribute{
							MarkdownDescription: "Require members of this group to accept the EULA during enrollment. **Known limitation:** Jamf Pro may override the requested value depending on the other enrollment toggles (it has been observed to force `true`). When the server overrides an explicitly-set value, Terraform will show a perpetual diff for this attribute — leave it unset to defer to the server, or align it with the value the server enforces.",
							Optional:            true,
							Computed:            true,
							PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
						},
					},
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

// ConfigValidators registers the plan-time cross-field invariants.
func (r *UserInitiatedEnrollmentSettingsResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		mdmSigningCertificateRequiredValidator{},
	}
}

// Configure wires the Jamf Pro client and the shared enrollment write lock.
func (r *UserInitiatedEnrollmentSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_user_initiated_enrollment_settings")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client

	// The shared enrollment write lock comes from the same providerdata.Data
	// value. The comma-ok type assertion is nil-safe: during the early-lifecycle
	// Configure call ProviderData is nil, so ok is false and the mutex stays nil
	// alongside the (also nil) client.
	if pd, ok := req.ProviderData.(*providerdata.Data); ok {
		r.enrollmentMu = pd.EnrollmentWriteLock()
	}
}

// ImportState handles import for the singleton.
func (r *UserInitiatedEnrollmentSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != helpers.SingletonID {
		resp.Diagnostics.AddError(
			"Invalid singleton import identifier",
			fmt.Sprintf(
				"jamfplatform_pro_user_initiated_enrollment_settings is a singleton resource and must be imported with id %q. Got %q.",
				helpers.SingletonID, req.ID,
			),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// initialID returns the canonical singleton id for use in CRUD handlers.
func initialID() types.String { return types.StringValue(helpers.SingletonID) }
