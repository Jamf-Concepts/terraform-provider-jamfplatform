// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_domain

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// domainTimeoutAttributeTypes defines the timeout attribute types for the SSO
// domain resource operations.
var domainTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// assignedConnectionAttributeTypes defines the object attribute types for one
// SSO connection an SSO domain is assigned to.
var assignedConnectionAttributeTypes = map[string]attr.Type{
	"connection_id":              types.StringType,
	"connection_organization_id": types.StringType,
	"region":                     types.StringType,
}

// assignedConnectionObjectType is the element type of the assigned_connections
// collection.
var assignedConnectionObjectType = types.ObjectType{AttrTypes: assignedConnectionAttributeTypes}
