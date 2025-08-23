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
func updateModelFromAPIResponse(model *blueprintResourceModel, blueprint *client.BlueprintDetail) {
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
		components := make([]componentModel, len(step.Components))
		for i, comp := range step.Components {
			configMap := make(map[string]string)
			if comp.Configuration != nil {
				var jsonObj map[string]interface{}
				if err := json.Unmarshal(comp.Configuration, &jsonObj); err == nil {
					flattenJSON(jsonObj, "", configMap)
				}
			}

			configMapValue, _ := types.MapValueFrom(context.Background(), types.StringType, configMap)
			components[i] = componentModel{
				Identifier:    types.StringValue(comp.Identifier),
				Configuration: configMapValue,
			}
		}
		model.Components = components
	} else {
		model.Components = []componentModel{}
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
