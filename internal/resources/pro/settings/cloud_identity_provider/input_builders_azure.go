// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_identity_provider

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// azureConsentCodePlaceholder is sent as the OAuth consent `code` on Azure
// create. The Entra `code` is a single-use artifact obtained interactively in
// the Jamf admin UI; it is not exposed as a Terraform attribute. The server
// REQUIRES `code` to be non-blank (an empty value is rejected with
// 400 INVALID_FIELD "must not be blank"), but it does NOT validate the consent
// at create time — create returns 201 with an inactive connection. The admin
// then completes the manual "refresh consent" step in the Jamf UI to activate
// it. This placeholder satisfies the non-blank requirement.
const azureConsentCodePlaceholder = "terraform-managed"

// buildAzureCreateRequest assembles the Azure Cloud Identity Provider create
// body from the plan. `Code` is the non-blank placeholder (see
// azureConsentCodePlaceholder); `ID` and `Type` are omitempty pointers left
// nil on create.
func buildAzureCreateRequest(plan CloudIdentityProviderResourceModel) *pro.AzureConfigurationRequest {
	az := plan.Azure
	return &pro.AzureConfigurationRequest{
		CloudIDPCommon: pro.CloudIDPCommonRequest{
			DisplayName:  plan.DisplayName.ValueString(),
			ProviderName: wireProviderAzure,
		},
		Server: pro.AzureServerConfigurationRequest{
			Code:                                     azureConsentCodePlaceholder,
			TenantID:                                 az.TenantID.ValueString(),
			SearchTimeout:                            int(az.SearchTimeout.ValueInt64()),
			Enabled:                                  az.Enabled.ValueBool(),
			MembershipCalculationOptimizationEnabled: helpers.OptionalBoolPointer(az.MembershipCalculationOptimizationEnabled),
			TransitiveMembershipEnabled:              az.TransitiveMembershipEnabled.ValueBool(),
			TransitiveMembershipUserField:            az.TransitiveMembershipUserField.ValueString(),
			TransitiveDirectoryMembershipEnabled:     az.TransitiveDirectoryMembershipEnabled.ValueBool(),
			Mappings:                                 buildAzureMappings(az.Mappings),
			// ID and Type are omitempty *string — left nil on create.
		},
	}
}

// buildAzureUpdateRequest assembles the Azure Cloud Identity Provider update
// body from the plan. There is no Code, TenantID, or Type on the update shape.
// Server.ID is the same as the CloudIdP id (the TF state id).
func buildAzureUpdateRequest(plan CloudIdentityProviderResourceModel) *pro.AzureConfigurationUpdate {
	az := plan.Azure
	return &pro.AzureConfigurationUpdate{
		CloudIDPCommon: pro.CloudIDPCommon{
			ID:           plan.ID.ValueString(),
			DisplayName:  plan.DisplayName.ValueString(),
			ProviderName: wireProviderAzure,
		},
		Server: pro.AzureServerConfigurationUpdate{
			ID:                                       plan.ID.ValueString(),
			Enabled:                                  az.Enabled.ValueBool(),
			SearchTimeout:                            int(az.SearchTimeout.ValueInt64()),
			MembershipCalculationOptimizationEnabled: helpers.OptionalBoolPointer(az.MembershipCalculationOptimizationEnabled),
			TransitiveMembershipEnabled:              az.TransitiveMembershipEnabled.ValueBool(),
			TransitiveMembershipUserField:            az.TransitiveMembershipUserField.ValueString(),
			TransitiveDirectoryMembershipEnabled:     az.TransitiveDirectoryMembershipEnabled.ValueBool(),
			Mappings:                                 buildAzureMappings(az.Mappings),
		},
	}
}

// buildAzureMappings converts the TF mappings model to the SDK struct. When m
// is nil a zero AzureMappings is returned so the non-pointer field is always
// present on the wire (the server generates defaults for empty values).
func buildAzureMappings(m *cloudAzureMappingsModel) pro.AzureMappings {
	if m == nil {
		return pro.AzureMappings{}
	}
	return pro.AzureMappings{
		UserID:     m.UserID.ValueString(),
		UserName:   m.UserName.ValueString(),
		RealName:   m.RealName.ValueString(),
		Email:      m.Email.ValueString(),
		Department: m.Department.ValueString(),
		Building:   m.Building.ValueString(),
		Room:       m.Room.ValueString(),
		Phone:      m.Phone.ValueString(),
		Position:   m.Position.ValueString(),
		GroupID:    m.GroupID.ValueString(),
		GroupName:  m.GroupName.ValueString(),
	}
}
