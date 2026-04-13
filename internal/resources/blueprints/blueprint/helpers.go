// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// generatePayloadIdentifier produces a deterministic UUID-formatted identifier from a payload type string.
func generatePayloadIdentifier(payloadType string) string {
	hash := sha256.Sum256([]byte(payloadType))
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

// setNestedValue sets a value in a nested map structure using underscore notation.
func setNestedValue(obj map[string]any, key string, value string) {
	parts := strings.Split(key, "_")
	current := obj

	for i := range len(parts) - 1 {
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
			result[fullKey] = strconv.FormatBool(v)
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

// desiredDeployedValue returns the desired deployed state based on the provided types.Bool value.
func desiredDeployedValue(v types.Bool) bool {
	if !helpers.IsConfiguredValue(v) {
		return true
	}
	return v.ValueBool()
}

// reconcileBlueprintDeployment ensures the blueprint's deployment state matches the desired state.
func (r *BlueprintResource) reconcileBlueprintDeployment(ctx context.Context, blueprintID string, desiredDeployed bool) (*jamfplatform.BlueprintDetail, error) {
	blueprint, err := r.client.GetBlueprint(ctx, blueprintID)
	if err != nil {
		return nil, err
	}

	deployedState := ""
	if blueprint.DeploymentState != nil {
		deployedState = blueprint.DeploymentState.State
	}

	if desiredDeployed {
		if !strings.EqualFold(deployedState, blueprintDeploymentStateDeployed) {
			if err := r.client.DeployBlueprint(ctx, blueprintID); err != nil {
				return blueprint, err
			}
			return r.client.GetBlueprint(ctx, blueprintID)
		}
		return blueprint, nil
	}

	if strings.EqualFold(deployedState, blueprintDeploymentStateNotDeployed) {
		return blueprint, nil
	}

	if err := r.client.UndeployBlueprint(ctx, blueprintID); err != nil {
		return blueprint, err
	}

	return r.client.GetBlueprint(ctx, blueprintID)
}

// scopeDeviceGroups safely extracts device group IDs from an optional blueprint scope.
func scopeDeviceGroups(scope *jamfplatform.BlueprintScope) []string {
	if scope == nil {
		return []string{}
	}
	return scope.DeviceGroups
}

// getBlueprintByName looks up a blueprint by name using the list API.
func getBlueprintByName(ctx context.Context, c *jamfplatform.Client, name string) (*jamfplatform.BlueprintDetail, error) {
	blueprints, err := c.ListBlueprints(ctx, nil, name)
	if err != nil {
		return nil, fmt.Errorf("failed to list blueprints: %w", err)
	}
	for _, bp := range blueprints {
		if bp.Name == name {
			return c.GetBlueprint(ctx, bp.ID)
		}
	}
	return nil, fmt.Errorf("blueprint with name %q not found", name)
}
