// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// benchmarkTimeoutAttributeTypes defines the timeout attribute types for benchmark resource operations.
var benchmarkTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"delete": types.StringType,
}

// osInfoObjectType defines the attribute types for OS info nested objects.
var osInfoObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"os_type":         types.StringType,
		"os_version":      types.Int64Type,
		"management_type": types.StringType,
	},
}

// osSpecificDefaultObjectType defines the attribute types for OS-specific default nested objects.
var osSpecificDefaultObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"title":       types.StringType,
		"description": types.StringType,
		"odv_value":   types.StringType,
		"odv_hint":    types.StringType,
	},
}
