// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_onboarding

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// onboardingTimeoutAttributeTypes defines the timeout attribute types for the resource.
var onboardingTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// selfServiceEntityType wire enum values. The OpenAPI spec documents seven values
// (the four below plus OS_X_EBOOK, OS_X_PATCH_POLICY, and the UNKNOWN fallback);
// only these four have a corresponding UI tab and eligible-* discovery endpoint, so
// the resource accepts only these four (wire-confirmed 2026-06-11).
const (
	entityTypePolicy        = "OS_X_POLICY"
	entityTypeConfigProfile = "OS_X_CONFIG_PROFILE"
	entityTypeMacApp        = "OS_X_MAC_APP"
	entityTypeAppInstaller  = "OS_X_APP_INSTALLER"
)

// validEntityTypes is the accepted self_service_entity_type vocabulary (OneOf).
var validEntityTypes = []string{
	entityTypePolicy,
	entityTypeConfigProfile,
	entityTypeMacApp,
	entityTypeAppInstaller,
}

// entityType options accepted by the eligible-items data source's entity_type
// argument, each mapping to one eligible-* SDK call.
const (
	eligiblePolicies              = "policies"
	eligibleConfigurationProfiles = "configuration_profiles"
	eligibleApps                  = "apps"
)

// validEligibleEntityTypes is the OneOf vocabulary for the eligible-items data
// source entity_type argument.
var validEligibleEntityTypes = []string{
	eligiblePolicies,
	eligibleConfigurationProfiles,
	eligibleApps,
}

// onboardingItemAttrTypes is the attr.Type map matching onboardingItemModel and the
// nested-object schema. Used to build a types.List of onboarding items.
func onboardingItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"entity_id":                types.StringType,
		"self_service_entity_type": types.StringType,
		"priority":                 types.Int64Type,
		"id":                       types.StringType,
		"entity_name":              types.StringType,
		"scope_description":        types.StringType,
		"site_description":         types.StringType,
	}
}

// onboardingItemObjectType is the types.ObjectType element type for a types.List of
// onboarding items.
func onboardingItemObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: onboardingItemAttrTypes()}
}
