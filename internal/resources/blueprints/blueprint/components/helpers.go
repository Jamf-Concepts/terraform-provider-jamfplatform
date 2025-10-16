// Copyright 2025 Jamf Software LLC.

package components

import "github.com/hashicorp/terraform-plugin-framework/types"

// setBoolField sets a boolean field for the Jamf API request body.
// If the field is null or unknown, it sets the field to the provided default value and marks it as not included.
func setBoolField(field types.Bool, defaultValue bool) map[string]interface{} {
	if !field.IsNull() && !field.IsUnknown() {
		return map[string]interface{}{
			"Enabled":  field.ValueBool(),
			"Included": true,
		}
	}
	return map[string]interface{}{
		"Enabled":  defaultValue,
		"Included": false,
	}
}

// setStringField sets a string field for the Jamf API request body.
// If the field is null or unknown, it sets the field to the provided default value and marks it as not included.
func setStringField(field types.String, defaultValue string) map[string]interface{} {
	if !field.IsNull() && !field.IsUnknown() {
		return map[string]interface{}{
			"Value":    field.ValueString(),
			"Included": true,
		}
	}
	return map[string]interface{}{
		"Value":    defaultValue,
		"Included": false,
	}
}

// setBoolFieldWithKey sets a boolean field with a custom key for the Jamf API request body.
// If the field is null or unknown, it sets the field to the provided default value and marks it as not included.
func setBoolFieldWithKey(field types.Bool, key string, defaultValue bool) map[string]interface{} {
	if !field.IsNull() && !field.IsUnknown() {
		return map[string]interface{}{
			key:        field.ValueBool(),
			"Included": true,
		}
	}
	return map[string]interface{}{
		key:        defaultValue,
		"Included": false,
	}
}
