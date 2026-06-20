// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_settings

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildSsoSettingsInput converts the Terraform plan model into a /v3/sso PUT
// payload.
//
// Three load-bearing transforms:
//
//  1. OidcSettings is a value type on SsoSettingsV3 and its UserMapping field
//     has no omitempty, so the marshalled body always includes
//     `oidcSettings.userMapping`. Jamf Pro rejects an empty userMapping. In
//     pure SAML mode, when the user did not supply an oidc_settings block,
//     populate a stub UserMapping="EMAIL". The value is harmless — Jamf
//     ignores oidcSettings when configurationType=SAML.
//
//  2. metadata_source URL/FILE mutex is enforced on the wire regardless of
//     what the user wrote: URL mode zeroes federationMetadataFile and
//     metadataFileName; FILE mode zeroes idpUrl. metadataFileName must be
//     null (not empty-string) in URL mode.
//
//  3. groupRdnKey must serialise as "" rather than be omitted when SAML
//     mode is active. The SDK *string + omitempty would drop the field on
//     nil; populate a pointer-to-"" so the body carries the explicit
//     empty-string value.
func buildSsoSettingsInput(ctx context.Context, plan SsoSettingsResourceModel) (*pro.SsoSettingsV3, diag.Diagnostics) {
	var diags diag.Diagnostics

	out := &pro.SsoSettingsV3{
		ConfigurationType:             plan.ConfigurationType.ValueString(),
		SsoEnabled:                    plan.SsoEnabled.ValueBool(),
		SsoBypassAllowed:              plan.SsoBypassAllowed.ValueBool(),
		SsoForEnrollmentEnabled:       plan.SsoForEnrollmentEnabled.ValueBool(),
		SsoForMacOsSelfServiceEnabled: plan.SsoForMacOsSelfServiceEnabled.ValueBool(),
		EnrollmentSsoForAccountDrivenEnrollmentEnabled: plan.EnrollmentSsoForAccountDrivenEnrollmentEnabled.ValueBool(),
		GroupEnrollmentAccessEnabled:                   plan.GroupEnrollmentAccessEnabled.ValueBool(),
		GroupEnrollmentAccessName:                      helpers.OptionalStringPointer(plan.GroupEnrollmentAccessName),
	}

	// OidcSettings — see gotcha (1).
	if plan.OidcSettings != nil {
		out.OidcSettings = pro.OidcSettings{
			UserMapping:                   plan.OidcSettings.UserMapping.ValueString(),
			JamfIDAuthenticationEnabled:   helpers.OptionalBoolPointer(plan.OidcSettings.JamfIDAuthenticationEnabled),
			UsernameAttributeClaimMapping: helpers.OptionalStringPointer(plan.OidcSettings.UsernameAttributeClaimMapping),
		}
	} else {
		out.OidcSettings = pro.OidcSettings{UserMapping: "EMAIL"}
	}

	// SamlSettings — value type that always serialises. Build with content
	// when the user supplied a block, otherwise leave at zero value (Jamf
	// Pro tolerates an empty samlSettings object).
	if plan.SamlSettings != nil {
		s := plan.SamlSettings
		// Jamf Pro's SAML validator rejects JSON `null` on the bool
		// fields when SAML is active — the API expects concrete booleans
		// matching the Jamf Pro admin UI defaults. Populate when the
		// user did not author the attribute.
		tokenExpirationDisabled := helpers.OptionalBoolPointer(s.TokenExpirationDisabled)
		if tokenExpirationDisabled == nil {
			t := true
			tokenExpirationDisabled = &t
		}
		userAttributeEnabled := helpers.OptionalBoolPointer(s.UserAttributeEnabled)
		if userAttributeEnabled == nil {
			f := false
			userAttributeEnabled = &f
		}
		out.SamlSettings = pro.SamlSettings{
			IdpProviderType:         helpers.OptionalStringPointer(s.IdpProviderType),
			OtherProviderTypeName:   helpers.OptionalStringPointer(s.OtherProviderTypeName),
			EntityID:                helpers.OptionalStringPointer(s.EntityID),
			MetadataSource:          helpers.OptionalStringPointer(s.MetadataSource),
			SessionTimeout:          helpers.OptionalInt64Pointer(s.SessionTimeout),
			TokenExpirationDisabled: tokenExpirationDisabled,
			UserMapping:             helpers.OptionalStringPointer(s.UserMapping),
			UserAttributeEnabled:    userAttributeEnabled,
			UserAttributeName:       helpers.OptionalStringPointer(s.UserAttributeName),
			GroupAttributeName:      helpers.OptionalStringPointer(s.GroupAttributeName),
		}

		// URL/FILE mutex on the wire — gotcha (2).
		metadataSource := ""
		if !s.MetadataSource.IsNull() && !s.MetadataSource.IsUnknown() {
			metadataSource = s.MetadataSource.ValueString()
		}
		switch metadataSource {
		case metadataSourceURL:
			out.SamlSettings.IdpURL = helpers.OptionalStringPointer(s.IdpURL)
			// URL mode: federationMetadataFile and metadataFileName must
			// arrive as JSON null so the server clears any cached FILE
			// state. Nil pointer marshals to null (SDK omitempty removed
			// on SSO body types).
			out.SamlSettings.MetadataFileName = nil
			out.SamlSettings.FederationMetadataFile = nil
		case metadataSourceFILE:
			// FILE mode: idp_url must arrive as JSON null so the server
			// clears any cached URL state. Nil pointer marshals to
			// null.
			out.SamlSettings.IdpURL = nil
			out.SamlSettings.MetadataFileName = helpers.OptionalStringPointer(s.MetadataFileName)
			if isStringConfigured(s.FederationMetadataFile) {
				decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s.FederationMetadataFile.ValueString()))
				if err != nil {
					diags.AddAttributeError(
						path.Root("saml_settings").AtName("federation_metadata_file"),
						"Invalid federation_metadata_file base64",
						"The supplied federation_metadata_file is not valid RFC 4648 base64: "+err.Error(),
					)
					return nil, diags
				}
				out.SamlSettings.FederationMetadataFile = &decoded
			}
		default:
			// No metadata source declared — pass through the user's
			// values as-is; the server will surface a field-named error.
			out.SamlSettings.IdpURL = helpers.OptionalStringPointer(s.IdpURL)
			out.SamlSettings.MetadataFileName = helpers.OptionalStringPointer(s.MetadataFileName)
			if isStringConfigured(s.FederationMetadataFile) {
				decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s.FederationMetadataFile.ValueString()))
				if err != nil {
					diags.AddAttributeError(
						path.Root("saml_settings").AtName("federation_metadata_file"),
						"Invalid federation_metadata_file base64",
						"The supplied federation_metadata_file is not valid RFC 4648 base64: "+err.Error(),
					)
					return nil, diags
				}
				out.SamlSettings.FederationMetadataFile = &decoded
			}
		}

		// groupRdnKey — gotcha (3). Pointer-to-"" so JSON renders `""`,
		// not omitted-as-null.
		if isStringConfigured(s.GroupRdnKey) {
			v := s.GroupRdnKey.ValueString()
			out.SamlSettings.GroupRdnKey = &v
		} else {
			empty := ""
			out.SamlSettings.GroupRdnKey = &empty
		}
	}

	// EnrollmentSsoConfig — *EnrollmentSsoConfig + omitempty, so emit only
	// when the user supplied a block.
	if plan.EnrollmentSsoConfig != nil {
		ec := &pro.EnrollmentSsoConfig{
			ManagementHint: helpers.OptionalStringPointer(plan.EnrollmentSsoConfig.ManagementHint),
		}
		if !plan.EnrollmentSsoConfig.Hosts.IsNull() && !plan.EnrollmentSsoConfig.Hosts.IsUnknown() {
			hosts, d := helpers.SetToStringSlice(ctx, plan.EnrollmentSsoConfig.Hosts)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			ec.Hosts = &hosts
		}
		out.EnrollmentSsoConfig = ec
	}

	return out, diags
}

// buildSsoCertificateInput converts the signing_certificate sub-block to the
// /v2/sso/cert PUT payload. Only valid when setup_type=UPLOADED.
//
// `key` is the alias lookup field; `keys[]` is informational and tolerated
// empty, so we omit it. `keystoreSetupType` is left nil — the upload shape
// is identified by the presence of `password` + `key`, not by an explicit
// setupType marker.
func buildSsoCertificateInput(plan signingCertificateModel) (*pro.SsoKeystore, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.SetupType.ValueString() != setupTypeUploaded {
		diags.AddError(
			"Internal error: buildSsoCertificateInput called for non-UPLOADED setup_type",
			"This is a provider bug — the CRUD orchestrator should only call this builder when setup_type=UPLOADED.",
		)
		return nil, diags
	}

	keystoreBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(plan.KeystoreFile.ValueString()))
	if err != nil {
		diags.AddAttributeError(
			path.Root("signing_certificate").AtName("keystore_file"),
			"Invalid keystore_file base64",
			"The supplied keystore_file is not valid RFC 4648 base64: "+err.Error(),
		)
		return nil, diags
	}

	return &pro.SsoKeystore{
		Key:              plan.Key.ValueString(),
		KeystoreFile:     keystoreBytes,
		KeystoreFileName: plan.KeystoreFileName.ValueString(),
		KeystorePassword: plan.KeystorePassword.ValueString(),
		Password:         plan.Password.ValueString(),
		Type:             plan.Type.ValueString(),
	}, diags
}
