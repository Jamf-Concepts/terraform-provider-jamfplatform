// Copyright 2025 Jamf Software LLC.

package device

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stringValueOrNull converts a Go string into a Terraform string attribute, preserving null when empty.
func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

// stringPointerValueOrNull safely unwraps a *string and converts it to a Terraform string.
func stringPointerValueOrNull(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// escapeFilterValue escapes double quotes for use in API filter strings.
func escapeFilterValue(value string) string {
	return strings.ReplaceAll(value, "\"", `\\"`)
}
