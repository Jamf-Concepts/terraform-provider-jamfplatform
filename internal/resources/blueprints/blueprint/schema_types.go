// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// blueprintTimeoutAttributeTypes defines the attribute types for blueprint resource timeouts.
var blueprintTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}
