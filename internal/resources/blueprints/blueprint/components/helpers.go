// Copyright 2025 Jamf Software LLC.

package components

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// setBoolField sets a boolean field for the Jamf API request body.
// If the field is null or unknown, it sets the field to the provided default value and marks it as not included.
func setBoolField(field types.Bool, defaultValue bool) map[string]any {
	return setBoolFieldWithKey(field, "Enabled", defaultValue)
}

// setStringField sets a string field for the Jamf API request body.
// If the field is null or unknown, it sets the field to the provided default value and marks it as not included.
func setStringField(field types.String, defaultValue string) map[string]any {
	if helpers.IsConfiguredValue(field) {
		return map[string]any{
			"Value":    field.ValueString(),
			"Included": true,
		}
	}
	return map[string]any{
		"Value":    defaultValue,
		"Included": false,
	}
}

// setBoolFieldWithKey sets a boolean field with a custom key for the Jamf API request body.
// If the field is null or unknown, it sets the field to the provided default value and marks it as not included.
func setBoolFieldWithKey(field types.Bool, key string, defaultValue bool) map[string]any {
	if helpers.IsConfiguredValue(field) {
		return map[string]any{
			key:        field.ValueBool(),
			"Included": true,
		}
	}
	return map[string]any{
		key:        defaultValue,
		"Included": false,
	}
}

// setValueField wraps a provided value in the Value/Included envelope expected by Jamf payloads.
func setValueField(value any, included bool) map[string]any {
	return map[string]any{
		"Value":    value,
		"Included": included,
	}
}
