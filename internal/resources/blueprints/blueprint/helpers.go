package blueprint

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// updateModelFromAPIResponse updates the Terraform model with data from the API response.
func updateModelFromAPIResponse(model *BlueprintResourceModel, blueprint *client.BlueprintDetail) {
	model.ID = types.StringValue(blueprint.ID)
	model.Name = types.StringValue(blueprint.Name)

	if model.Description.IsNull() && blueprint.Description == "" {
		model.Description = types.StringNull()
	} else {
		model.Description = types.StringValue(blueprint.Description)
	}

	model.Created = types.StringValue(blueprint.Created)
	model.Updated = types.StringValue(blueprint.Updated)
	model.DeploymentState = types.StringValue(blueprint.DeploymentState.State)

	deviceGroups := make([]types.String, len(blueprint.Scope.DeviceGroups))
	for i, dg := range blueprint.Scope.DeviceGroups {
		deviceGroups[i] = types.StringValue(dg)
	}
	model.DeviceGroups = deviceGroups

	if len(blueprint.Steps) > 0 {
		step := blueprint.Steps[0]

		apiComponentsByID := make(map[string]client.BlueprintComponent)
		for _, comp := range step.Components {
			apiComponentsByID[comp.Identifier] = comp
		}

		components := make([]ComponentModel, len(model.Components))
		for i, modelComp := range model.Components {
			identifier := modelComp.Identifier.ValueString()

			if apiComp, exists := apiComponentsByID[identifier]; exists {
				configMap := make(map[string]string)
				if apiComp.Configuration != nil {
					var jsonObj map[string]interface{}
					if err := json.Unmarshal(apiComp.Configuration, &jsonObj); err == nil {
						flattenJSON(jsonObj, "", configMap)
					}
				}

				configMapValue, _ := types.MapValueFrom(context.Background(), types.StringType, configMap)
				components[i] = ComponentModel{
					Identifier:    types.StringValue(apiComp.Identifier),
					Configuration: configMapValue,
				}
			} else {
				components[i] = modelComp
			}
		}
		model.Components = components
	} else {
		model.Components = []ComponentModel{}
	}
}

// normalizeJSON takes a JSON string and returns it with sorted keys to ensure consistent comparison
func normalizeJSON(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}

	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return jsonStr
	}

	normalized, err := json.Marshal(obj)
	if err != nil {
		return jsonStr
	}

	return string(normalized)
}

// setNestedValue sets a value in a nested map structure using underscore notation
func setNestedValue(obj map[string]interface{}, key string, value string) {
	parts := strings.Split(key, "_")
	current := obj

	for i := 0; i < len(parts)-1; i++ {
		if current[parts[i]] == nil {
			current[parts[i]] = make(map[string]interface{})
		}
		if nested, ok := current[parts[i]].(map[string]interface{}); ok {
			current = nested
		} else {
			current[parts[i]] = make(map[string]interface{})
			current = current[parts[i]].(map[string]interface{})
		}
	}

	finalKey := parts[len(parts)-1]
	if value == "" {
		current[finalKey] = nil
	} else if value == "true" {
		current[finalKey] = true
	} else if value == "false" {
		current[finalKey] = false
	} else if num, err := strconv.Atoi(value); err == nil {
		current[finalKey] = num
	} else {
		if strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
			var jsonValue interface{}
			if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
				current[finalKey] = jsonValue
				return
			}
		}
		current[finalKey] = value
	}
}

// flattenJSON flattens a nested JSON object into a flat map with underscore notation keys
func flattenJSON(obj map[string]interface{}, prefix string, result map[string]string) {
	for key, value := range obj {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "_" + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			flattenJSON(v, fullKey, result)
		case nil:
			result[fullKey] = ""
		case bool:
			if v {
				result[fullKey] = "true"
			} else {
				result[fullKey] = "false"
			}
		case float64:
			result[fullKey] = strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			result[fullKey] = strconv.Itoa(v)
		case string:
			result[fullKey] = v
		default:
			if jsonBytes, err := json.Marshal(v); err == nil {
				result[fullKey] = string(jsonBytes)
			} else {
				result[fullKey] = ""
			}
		}
	}
}

// isNotFoundError checks if the error is a 404 not found error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return strings.Contains(errorStr, "status 404") ||
		strings.Contains(errorStr, "was not found") ||
		strings.Contains(errorStr, "NOT_FOUND")
}
