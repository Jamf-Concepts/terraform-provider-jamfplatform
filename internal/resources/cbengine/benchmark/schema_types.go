// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

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

// sourceObjectType defines the attribute types for the computed source nested objects.
var sourceObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"branch":   types.StringType,
		"revision": types.StringType,
	},
}

// osVersionObjectType defines the attribute types for the operating-system
// version nested objects used by selected_os_versions and available_os_versions.
var osVersionObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"os_type":    types.StringType,
		"os_version": types.Int64Type,
	},
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
