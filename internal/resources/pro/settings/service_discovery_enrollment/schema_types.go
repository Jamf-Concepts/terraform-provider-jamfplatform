// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package service_discovery_enrollment

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// serviceDiscoveryEnrollmentTimeoutAttributeTypes defines the timeout attribute
// types for the resource operations.
var serviceDiscoveryEnrollmentTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// enrollmentType wire enum values (api/pro_api.json ServiceDiscoveryVersion). The
// SDK exposes ServiceDiscoveryVersion as a bare string alias with no constants, so
// the vocabulary is carried here and enforced with a OneOf validator.
const (
	enrollmentTypeNone    = "none"
	enrollmentTypeMDMBYOD = "mdm-byod"
	enrollmentTypeMDMADDE = "mdm-adde"
)

// validEnrollmentTypes is the accepted enrollment_type vocabulary (OneOf).
var validEnrollmentTypes = []string{
	enrollmentTypeNone,
	enrollmentTypeMDMBYOD,
	enrollmentTypeMDMADDE,
}

// wellKnownSettingAttrTypes is the attr.Type map matching wellKnownSettingModel and
// the nested-object schema. Used to build a types.List of well-known settings rows.
func wellKnownSettingAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"server_uuid":     types.StringType,
		"enrollment_type": types.StringType,
		"org_name":        types.StringType,
	}
}

// wellKnownSettingObjectType is the types.ObjectType element type for a types.List of
// well-known settings rows.
func wellKnownSettingObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: wellKnownSettingAttrTypes()}
}
