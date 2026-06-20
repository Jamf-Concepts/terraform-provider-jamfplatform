// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_identity_provider

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// assignAzureState folds an Azure Cloud Identity Provider GET response into the
// resource model. There are no WriteOnly fields on Azure, so no prior-value
// threading is required.
func assignAzureState(state *CloudIdentityProviderResourceModel, resp *pro.AzureConfiguration) {
	if resp == nil {
		return
	}

	// `mappings` is Optional (not Computed): the server always returns
	// generated mappings, but if the user did not author the block we keep it
	// null in state to avoid a "planned null, got object" consistency error.
	var priorMappings *cloudAzureMappingsModel
	if state.Azure != nil {
		priorMappings = state.Azure.Mappings
	}

	if resp.CloudIDPCommon != nil {
		if resp.CloudIDPCommon.ID != "" {
			state.ID = types.StringValue(resp.CloudIDPCommon.ID)
		}
		state.DisplayName = types.StringValue(resp.CloudIDPCommon.DisplayName)
		state.ProviderName = types.StringValue(providerNameFromWire(resp.CloudIDPCommon.ProviderName))
	}

	if resp.Server != nil {
		s := resp.Server
		state.Azure = &cloudIdentityProviderAzureModel{
			TenantID:                                 types.StringValue(s.TenantID),
			SearchTimeout:                            types.Int64Value(int64(s.SearchTimeout)),
			Enabled:                                  types.BoolValue(s.Enabled),
			MembershipCalculationOptimizationEnabled: types.BoolValue(s.MembershipCalculationOptimizationEnabled),
			TransitiveMembershipEnabled:              types.BoolValue(s.TransitiveMembershipEnabled),
			TransitiveMembershipUserField:            types.StringValue(s.TransitiveMembershipUserField),
			TransitiveDirectoryMembershipEnabled:     types.BoolValue(s.TransitiveDirectoryMembershipEnabled),
			// Server-derived echoes (Computed-only).
			Type:              types.StringValue(s.Type),
			Migrated:          types.BoolValue(s.Migrated),
			DeprecatedConsent: types.BoolValue(s.DeprecatedConsent),
			Mappings:          assignAzureMappingsState(s.Mappings, priorMappings),
		}
	}
}

// assignAzureMappingsState builds the TF mappings model from the SDK response,
// scoped to whether the user authored the block (`prior`). `mappings` is
// Optional (not Computed), so surfacing server-generated mappings the user did
// not configure would trip a "planned null, got object" consistency error.
func assignAzureMappingsState(m *pro.AzureMappings, prior *cloudAzureMappingsModel) *cloudAzureMappingsModel {
	if prior == nil || m == nil {
		return nil
	}
	return &cloudAzureMappingsModel{
		UserID:     types.StringValue(m.UserID),
		UserName:   types.StringValue(m.UserName),
		RealName:   types.StringValue(m.RealName),
		Email:      types.StringValue(m.Email),
		Department: types.StringValue(m.Department),
		Building:   types.StringValue(m.Building),
		Room:       types.StringValue(m.Room),
		Phone:      types.StringValue(m.Phone),
		Position:   types.StringValue(m.Position),
		GroupID:    types.StringValue(m.GroupID),
		GroupName:  types.StringValue(m.GroupName),
	}
}
