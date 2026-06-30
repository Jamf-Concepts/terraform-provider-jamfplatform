// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package mobile_device_enrollment_profile implements the
// jamfplatform_pro_mobile_device_enrollment_profile resource, data source, and
// list resource backed by the Jamf ProClassic mobile device enrollment profiles
// API.
//
// Server semantics (wire-probed): writes are a MERGE — omitting a field or block
// retains the server value, sending an empty element clears it. Input builders
// therefore always-emit managed scalar fields (empty when null) so a removed
// value is explicitly cleared. invitation and uuid are server-minted and stable.
// Attachments are read-only: the upload endpoint refuses OAuth bearer auth for
// this resource (both via the gateway and direct), so the provider cannot create
// them — only list what the server returns.
package mobile_device_enrollment_profile

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

const minJamfProVersion = ""

// EnrollmentProfileResource implements the Terraform resource.
type EnrollmentProfileResource struct {
	client *proclassic.Client
}

var _ resource.Resource = &EnrollmentProfileResource{}
var _ resource.ResourceWithImportState = &EnrollmentProfileResource{}
var _ resource.ResourceWithIdentity = &EnrollmentProfileResource{}

const (
	defaultCreateTimeout = 60 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 60 * time.Second
	defaultDeleteTimeout = 60 * time.Second
)

// NewEnrollmentProfileResource returns a new instance.
func NewEnrollmentProfileResource() resource.Resource {
	return &EnrollmentProfileResource{}
}

func (r *EnrollmentProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_mobile_device_enrollment_profile"
}

func (r *EnrollmentProfileResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Jamf Pro mobile device enrollment profile ID.",
				RequiredForImport: true,
			},
		},
	}
}

func (r *EnrollmentProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jamf Pro mobile device enrollment profile — the Apple Configurator / over-the-air enrollment profile that devices use to enrol into Jamf Pro. Carries enrolment metadata plus user/location and purchasing information.\n\n" +
			"Attachments are read-only: Jamf Pro's attachment-upload endpoint does not accept this provider's authentication for enrollment profiles, so attachments can be listed but not managed here — add or remove them in the Jamf Pro admin UI." + resourcePrivileges,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Enrollment profile ID assigned by Jamf Pro.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name for the enrollment profile. Must not be empty.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description shown on the enrollment profile's General tab.",
				Optional:            true,
			},
			"site_id": schema.StringAttribute{
				MarkdownDescription: "ID of the site mobile devices enrolled with this profile are added to. Defaults to `-1` (no site).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"site_name": schema.StringAttribute{
				MarkdownDescription: "Name of the site, derived from `site_id`.",
				Computed:            true,
			},
			"invitation": schema.StringAttribute{
				MarkdownDescription: "Enrollment invitation code minted by Jamf Pro. Stable for the life of the profile.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Profile UUID minted by Jamf Pro.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"location": schema.SingleNestedAttribute{
				MarkdownDescription: "User and Location Information for devices enrolled with this profile.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"username":      optString("Username."),
					"real_name":     optString("Full name."),
					"email_address": optString("Email address."),
					"phone_number":  optString("Phone number."),
					"department":    optString("Department."),
					"building":      optString("Building."),
					"room":          optString("Room."),
					"position":      optString("Position."),
				},
			},
			"purchasing": schema.SingleNestedAttribute{
				MarkdownDescription: "Purchasing Information for devices enrolled with this profile.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"is_purchased": schema.BoolAttribute{
						MarkdownDescription: "Whether the device is purchased. Defaults to true.",
						Optional:            true,
						Computed:            true,
					},
					"is_leased": schema.BoolAttribute{
						MarkdownDescription: "Whether the device is leased. Defaults to false.",
						Optional:            true,
						Computed:            true,
					},
					"po_number":              optString("Purchase order number."),
					"po_date":                optString("Purchase order date (YYYY-MM-DD)."),
					"po_date_epoch":          computedString("Purchase order date as epoch milliseconds."),
					"po_date_utc":            computedString("Purchase order date in UTC."),
					"vendor":                 optString("Vendor."),
					"warranty_expires":       optString("Warranty expiration date (YYYY-MM-DD)."),
					"warranty_expires_epoch": computedString("Warranty expiration as epoch milliseconds."),
					"warranty_expires_utc":   computedString("Warranty expiration in UTC."),
					"applecare_id":           optString("AppleCare ID."),
					"lease_expires":          optString("Lease expiration date (YYYY-MM-DD)."),
					"lease_expires_epoch":    computedString("Lease expiration as epoch milliseconds."),
					"lease_expires_utc":      computedString("Lease expiration in UTC."),
					"purchase_price":         optString("Purchase price."),
					"life_expectancy": schema.Int64Attribute{
						MarkdownDescription: "Life expectancy in years.",
						Optional:            true,
						Computed:            true,
					},
					"purchasing_account": optString("Purchasing account."),
					"purchasing_contact": optString("Purchasing contact."),
				},
			},
			"attachments": schema.ListNestedAttribute{
				MarkdownDescription: "Read-only list of attachments on the profile. Manage attachments in the Jamf Pro admin UI.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       computedString("Attachment ID."),
						"filename": computedString("Attachment filename."),
						"uri":      computedString("Attachment download URI."),
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
	}
}

func optString(desc string) schema.StringAttribute {
	return schema.StringAttribute{MarkdownDescription: desc, Optional: true}
}

func computedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{MarkdownDescription: desc, Computed: true}
}

func (r *EnrollmentProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_mobile_device_enrollment_profile")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

func (r *EnrollmentProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
