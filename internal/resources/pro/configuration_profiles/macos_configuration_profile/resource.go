// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is the minimum Jamf Pro tenant version required. Empty:
// the classic /osxconfigurationprofiles endpoint predates the provider's
// declared floor.
const minJamfProVersion = ""

// Resource implements jamfplatform_pro_macos_configuration_profile.
type Resource struct {
	client *proclassic.Client
}

var (
	_ resource.Resource                = &Resource{}
	_ resource.ResourceWithImportState = &Resource{}
	_ resource.ResourceWithIdentity    = &Resource{}
)

const (
	defaultCreateTimeout = 90 * time.Second
	defaultReadTimeout   = 90 * time.Second
	defaultUpdateTimeout = 90 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewResource returns a new Resource instance.
func NewResource() resource.Resource {
	return &Resource{}
}

// Metadata sets the Terraform type name.
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_macos_configuration_profile"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *Resource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro macOS configuration profile ID.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a macOS configuration profile in Jamf Pro via the classic `/JSSResource/osxconfigurationprofiles` endpoint. The `general.payloads` attribute carries the raw `.mobileconfig` plist XML; the provider suppresses diffs produced by Jamf Pro's standard server-side normalisations while still surfacing drift on values the user has authored.\n\n**Payload diff suppression** — the comparison runs in three layers:\n\n1. **Unconditional skip** for fields Jamf Pro always re-derives on write: top-level `PayloadDisplayName` (set from `general.name`), top-level `PayloadIdentifier` and `PayloadUUID` (Jamf Pro assigns lowercase UUIDs), plus the same three keys on every entry of `PayloadContent` (Jamf Pro may assign inner UUIDs on Create; preserved on Update by the identifier injection step below).\n2. **Empty-string normalisation** — keys whose value is an empty string on either side are treated as absent. Jamf Pro substitutes a server default for an empty `PayloadOrganization`; this normalisation means `\"\"` and \"key omitted\" compare equal.\n3. **Intersection compare** for everything else — keys present on only one side are ignored. This is how server-defaulted fields stay quiet: when the user omits `PayloadEnabled`, `PayloadDescription`, `PayloadRemovalDisallowed`, `PayloadOrganization`, `AllowUserOverrides`, or when Jamf Pro removes `VendorConfig` from `com.apple.webcontent-filter` payloads, the asymmetric key falls out of the compare. **When the user explicitly authors any of those keys**, the value is present on both sides and any subsequent change is detected as real drift. Setting `PayloadEnabled=false` in the payload will produce a plan if state holds `true`; omitting `PayloadEnabled` lets Jamf Pro's `true` default ride.\n\nWhitespace inside string values is trimmed before comparison so the platform's whitespace handling (observed on `Rules[].Comment` and similar text fields) does not surface as drift.\n\nSee `PROFILE_ROUNDTRIP_REPORT.md` (developer-facing, gitignored) for the empirical 200-profile corpus the comparison is verified against, plus a per-fixture build-tagged regression test (`go test -tags profile_corpus`).\n\n**Scope** blocks mirror `jamfplatform_pro_policy`: targets / limitations / exclusions all carry flat sets of numeric Jamf Pro classic IDs (or directory-service names where the wire is name-keyed). `all_computers` and `all_jss_users` conflict with their per-ID siblings via attribute-level validators.\n\n**Update behaviour** — the provider re-applies the existing top-level `PayloadUUID` and `PayloadIdentifier` from state into every user-supplied payload before PUT, so Jamf Pro preserves them on update. This keeps the profile's identity stable across applies — otherwise connected macOS devices would treat each update as a fresh profile installation (\"ghost profile\"). Same mechanism as `profileconvert.InjectIdentifiers` in jamf-cli.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Profile ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"general": schema.SingleNestedAttribute{
				MarkdownDescription: "Profile general settings. `name` and `payloads` are required.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Profile ID under <general> — server-derived. Matches the top-level `id`.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Display name of the profile. Must be unique within the tenant. Jamf Pro also propagates this value into the top-level `PayloadDisplayName` of the mobileconfig payload, so any value supplied for that key inside `payloads` is superseded on the wire.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"description": optComputedString("Free-text description shown in the Jamf Pro admin UI."),
					"level": schema.StringAttribute{
						MarkdownDescription: "Profile delivery level. UI-canonical values: `Computer Level` (default) / `User Level`. Wire field `<level>`: the classic API accepts `Computer`/`User` on write and reports `System`/`User` on read. The provider translates at the boundary so the Terraform-facing value mirrors the admin UI dropdown.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf(validLevels...),
						},
					},
					"distribution_method": schema.StringAttribute{
						MarkdownDescription: "How the profile reaches devices. `Install Automatically` pushes via MDM; `Make Available in Self Service` lists the profile under the Self Service tab. Wire-symmetric — no translation.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf(validDistributionMethods...),
						},
					},
					"user_removable": schema.BoolAttribute{
						MarkdownDescription: "Whether the device user can remove the profile from System Settings. Defaults to false. Wire field `<user_removable>`. Note that this is independent of the Self Service `removal_disallowed` security setting — the two interact only for Self Service profiles.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
					"redeploy_on_update": schema.StringAttribute{
						MarkdownDescription: "Re-deploy behaviour when the profile is updated. Wire field `<redeploy_on_update>`. Valid values: `Newly Assigned` (push to newly-scoped devices only) or `All` (push to every scoped device on the next update). **Note**: Jamf Pro's classic API always returns `Newly Assigned` on read, even after a successful PUT with `All`. The provider treats this as a write-only field — once the user has authored a value the wire response is ignored so subsequent refreshes do not snap state back to `Newly Assigned`. Set this attribute explicitly to control redeployment behaviour on Update.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"uuid": schema.StringAttribute{
						MarkdownDescription: "Profile UUID — minted by Jamf Pro on creation and propagated as the top-level `PayloadUUID` inside the mobileconfig payload. Server-derived; cannot be set by the user.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"payloads": schema.StringAttribute{
						MarkdownDescription: "The mobileconfig plist XML carrying the configuration the profile delivers. Jamf Pro re-serialises the plist server-side (compact format, `plist version=\"1\"`, server-assigned top-level UUIDs, defaults for fields the user omits, etc.); the provider's diff suppression treats those normalisations as no-ops. See the resource description for the exact comparison rules.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							payloadsDiffSuppressor(),
						},
					},
					"category_id": schema.StringAttribute{
						MarkdownDescription: "Jamf Pro category ID. Use `-1` (default) for \"no category\". String form to mirror the rest of the provider's classic-API ID handling.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"category_name": schema.StringAttribute{
						MarkdownDescription: "Category display name returned by Jamf Pro. Server-derived.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"site_id": schema.StringAttribute{
						MarkdownDescription: "Jamf Pro site ID. Use `-1` (default) for \"no site\".",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"site_name": schema.StringAttribute{
						MarkdownDescription: "Site display name returned by Jamf Pro. Server-derived.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},
			"scope": schema.SingleNestedAttribute{
				MarkdownDescription: "Profile scope. `all_computers = true` forbids per-computer / per-group / per-building / per-department targets. `all_jss_users = true` forbids per-user / per-user-group targets. The wire elements are `<jss_users>` and `<jss_user_groups>`; the UI labels them \"Users\" / \"User Groups\", so the provider exposes them as `user_ids` / `user_group_ids`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"all_computers": schema.BoolAttribute{
						MarkdownDescription: "Scope to every computer in the tenant.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						Validators: []validator.Bool{
							scope.AllFlagConflictsWith(
								path.MatchRelative().AtParent().AtName("computer_ids"),
								path.MatchRelative().AtParent().AtName("computer_group_ids"),
								path.MatchRelative().AtParent().AtName("building_ids"),
								path.MatchRelative().AtParent().AtName("department_ids"),
							),
						},
					},
					"all_jss_users": schema.BoolAttribute{
						MarkdownDescription: "Scope to every Jamf Pro user in the tenant. Wire field `all_jss_users`; UI label \"All Users\".",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						Validators: []validator.Bool{
							scope.AllFlagConflictsWith(
								path.MatchRelative().AtParent().AtName("user_ids"),
								path.MatchRelative().AtParent().AtName("user_group_ids"),
							),
						},
					},
					"computer_ids":       scope.IDSetAttribute("computer"),
					"computer_group_ids": scope.IDSetAttribute("computer group"),
					"building_ids":       scope.IDSetAttribute("building"),
					"department_ids":     scope.IDSetAttribute("department"),
					"user_ids":           scope.IDSetAttribute("user"),
					"user_group_ids":     scope.IDSetAttribute("user group"),
					"limitations": schema.SingleNestedAttribute{
						MarkdownDescription: "Scope limitations narrow the audience after the targets resolve. `directory_service_or_local_user_names` carries names (no IDs) per wire shape; `directory_service_user_group_names` similarly.",
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
							"user_ids":                              scope.IDSetAttribute("user"),
							"user_group_ids":                        scope.IDSetAttribute("user group"),
							"network_segment_ids":                   scope.IDSetAttribute("network segment"),
							"ibeacon_ids":                           scope.IDSetAttribute("iBeacon"),
							"directory_service_or_local_user_names": scope.NameSetAttribute("directory service or local user"),
							"directory_service_user_group_names":    scope.NameSetAttribute("directory service user group"),
						},
					},
				},
			},
			"self_service": schema.SingleNestedAttribute{
				MarkdownDescription: "Self Service integration. Only meaningful when `general.distribution_method = \"Make Available in Self Service\"` — for `Install Automatically` profiles the server still emits a populated `<self_service>` block but the values are ignored by the Self Service tab.\n\nThe `display_notifications` boolean and `notification_location` string project into the wire's dual `<notification>` elements (the server emits one bool form and one string form per profile).",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"self_service_display_name":     optComputedString("Display name shown in Self Service. Defaults to the profile name when blank. Wire field `<self_service_display_name>` (Self Service 10.0.0+)."),
					"install_button_text":           optComputedString("Install-button label. Defaults to `Install`. Wire field `<install_button_text>`."),
					"self_service_description":      optComputedString("Description shown in Self Service. Markdown supported."),
					"ensure_users_view_description": optComputedBool("Force users to view the description before installing. Wire field `<force_users_to_view_description>`."),
					"feature_on_main_page":          optComputedBool("Feature the profile on the Self Service main page. Wire field `<feature_on_main_page>`."),
					"display_notifications":         optComputedBool("Whether Self Service surfaces a notification when the profile becomes available. Wire emits as `<notification>true|false</notification>` (first form). Pair with `notification_location` for the delivery target."),
					"notification_location": schema.StringAttribute{
						MarkdownDescription: "Notification delivery location. Wire emits as `<notification>Self Service|Self Service and Notification Center</notification>` (second form). Valid values: `Self Service`, `Self Service and Notification Center`.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf(validNotificationLocations...),
						},
					},
					"notification_subject": optComputedString("Notification subject line."),
					"notification_message": optComputedString("Notification body text. Displays in Self Service only."),
					"removal_disallowed": schema.StringAttribute{
						MarkdownDescription: "Removal-by-end-user policy for the Self Service profile. Valid values: `Never`, `Always`, `With Authorization`. Wire field `<security><removal_disallowed>`. `With Authorization` additionally requires a `<security><password>` companion that the provider does not currently surface — open an issue if you need this knob.",
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						Validators: []validator.String{
							stringvalidator.OneOf(validRemovalDisallowedValues...),
						},
					},
					"categories": schema.ListNestedAttribute{
						MarkdownDescription: "Categories under which the profile appears in Self Service. Each entry pairs a category ID with `display_in` / `feature_in` toggles matching the Self Service \"Display in\" / \"Feature in\" columns of the admin UI.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id":         schema.StringAttribute{MarkdownDescription: "Category ID.", Required: true},
								"name":       optComputedStringInList("Category display name (server-derived)."),
								"display_in": optComputedBoolInList("Display the profile in this category."),
								"feature_in": optComputedBoolInList("Feature the profile in this category."),
							},
						},
					},
				},
			},
			"timeouts": timeouts.Attributes(context.Background(), timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// Configure wires the Jamf ProClassic client into the resource.
func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_macos_configuration_profile")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import by Jamf Pro profile ID.
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// optComputedString and optComputedBool are the top-level / single-nested
// Optional+Computed field helpers. They use UseStateForUnknown so a state
// value of Null carries through to the plan as Null (instead of staying
// Unknown). Required for fields the user may omit in HCL — without this,
// every plan refresh would show null→Unknown as a planned change and
// produce a spurious [update] action.
//
// For Optional+Computed scalars that live INSIDE a ListNestedAttribute or
// SetNestedAttribute element, use optComputedStringInList / optComputedBoolInList
// — they apply UseNonNullStateForUnknown because an appended list element
// has prior state Null at its new index, and UseStateForUnknown there would
// trip the "Provider produced inconsistent result after apply" check.
// STYLE_GUIDE §"Optional+Computed scalars inside a ListNestedAttribute …"
// is the reference for that split.
func optComputedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func optComputedBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
	}
}

func optComputedStringInList(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
	}
}

func optComputedBoolInList(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
	}
}

// _ tracks the types import — the model fields use it implicitly.
var _ = types.StringNull
