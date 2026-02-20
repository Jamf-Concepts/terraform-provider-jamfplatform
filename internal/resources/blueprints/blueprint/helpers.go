// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// normalizeJSON takes a JSON string and returns it with sorted keys to ensure consistent comparison.
func normalizeJSON(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}

	var obj any
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return jsonStr
	}

	normalized, err := json.Marshal(obj)
	if err != nil {
		return jsonStr
	}

	return string(normalized)
}

// setNestedValue sets a value in a nested map structure using underscore notation.
func setNestedValue(obj map[string]any, key string, value string) {
	parts := strings.Split(key, "_")
	current := obj

	for i := 0; i < len(parts)-1; i++ {
		if current[parts[i]] == nil {
			current[parts[i]] = make(map[string]any)
		}
		if nested, ok := current[parts[i]].(map[string]any); ok {
			current = nested
		} else {
			current[parts[i]] = make(map[string]any)
			current = current[parts[i]].(map[string]any)
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
			var jsonValue any
			if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
				current[finalKey] = jsonValue
				return
			}
		}
		current[finalKey] = value
	}
}

// flattenJSON flattens a nested JSON object into a flat map with underscore notation keys.
func flattenJSON(obj map[string]any, prefix string, result map[string]string) {
	for key, value := range obj {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "_" + key
		}

		switch v := value.(type) {
		case map[string]any:
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

// isServerError checks if the error is a server error (500).
func isServerError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return strings.Contains(errorStr, "status 500") ||
		strings.Contains(errorStr, "Internal Server Error")
}

// desiredDeployedValue returns the desired deployed state based on the provided types.Bool value.
func desiredDeployedValue(v types.Bool) bool {
	if !helpers.IsConfiguredValue(v) {
		return true
	}
	return v.ValueBool()
}

// reconcileBlueprintDeployment ensures the blueprint's deployment state matches the desired state.
func (r *BlueprintResource) reconcileBlueprintDeployment(ctx context.Context, blueprintID string, desiredDeployed bool) (*client.BlueprintDetailV1, error) {
	blueprint, err := r.client.GetBlueprintByIDV1(ctx, blueprintID)
	if err != nil {
		return nil, err
	}

	if desiredDeployed {
		if !strings.EqualFold(blueprint.DeploymentState.State, blueprintDeploymentStateDeployed) {
			if err := r.client.DeployBlueprintV1(ctx, blueprintID); err != nil {
				return blueprint, err
			}
			return r.client.GetBlueprintByIDV1(ctx, blueprintID)
		}
		return blueprint, nil
	}

	if strings.EqualFold(blueprint.DeploymentState.State, blueprintDeploymentStateNotDeployed) {
		return blueprint, nil
	}

	if err := r.client.UndeployBlueprintV1(ctx, blueprintID); err != nil {
		return blueprint, err
	}

	return r.client.GetBlueprintByIDV1(ctx, blueprintID)
}
