// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var _ resource.ResourceWithUpgradeState = &PolicyResource{}

// UpgradeState migrates prior state versions to the current schema.
//
// v0 → v1: the script-assignment parameter attributes were renamed from
// `parameter4`…`parameter11` to snake_case `parameter_4`…`parameter_11` to
// align with the jamfplatform_pro_script resource and the provider-wide
// snake_case attribute convention. The rename is a nested-object attribute
// name change, which the framework cannot decode against the v1 schema
// without an explicit migration, so we rewrite the affected keys in the raw
// prior state JSON and re-emit it through the current schema type. Every
// other attribute is carried across verbatim as raw JSON, so no value is
// reinterpreted or lost.
func (r *PolicyResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				if req.RawState == nil {
					return
				}

				rewritten, err := renameScriptParameterKeys(req.RawState.JSON)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to upgrade jamfplatform_pro_policy state from v0 to v1",
						fmt.Sprintf("Could not rewrite the renamed script parameter keys in prior state: %s", err),
					)
					return
				}

				var schemaResp resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
				schemaType := schemaResp.Schema.Type().TerraformType(ctx)

				upgraded := tfprotov6.RawState{JSON: rewritten}
				value, err := upgraded.Unmarshal(schemaType)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to upgrade jamfplatform_pro_policy state from v0 to v1",
						fmt.Sprintf("Could not decode rewritten prior state against the current schema: %s", err),
					)
					return
				}

				dynamicValue, err := tfprotov6.NewDynamicValue(schemaType, value)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to upgrade jamfplatform_pro_policy state from v0 to v1",
						fmt.Sprintf("Could not encode upgraded state: %s", err),
					)
					return
				}

				resp.DynamicValue = &dynamicValue
			},
		},
	}
}

// renameScriptParameterKeys rewrites the prior-state JSON, renaming each
// scripts.scripts[*].parameterN key to parameter_N (N = 4..11). All other
// state is preserved as raw JSON bytes — only the eight affected keys are
// touched — so numeric IDs and other values keep their exact representation.
func renameScriptParameterKeys(rawJSON []byte) ([]byte, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &top); err != nil {
		return nil, err
	}

	scriptsRaw, ok := top["scripts"]
	if !ok || string(scriptsRaw) == "null" {
		return rawJSON, nil
	}

	var scriptsBlock map[string]json.RawMessage
	if err := json.Unmarshal(scriptsRaw, &scriptsBlock); err != nil {
		return nil, err
	}

	itemsRaw, ok := scriptsBlock["scripts"]
	if !ok || string(itemsRaw) == "null" {
		return rawJSON, nil
	}

	var items []map[string]json.RawMessage
	if err := json.Unmarshal(itemsRaw, &items); err != nil {
		return nil, err
	}

	for _, item := range items {
		for n := 4; n <= 11; n++ {
			oldKey := fmt.Sprintf("parameter%d", n)
			value, present := item[oldKey]
			if !present {
				continue
			}
			item[fmt.Sprintf("parameter_%d", n)] = value
			delete(item, oldKey)
		}
	}

	newItems, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	scriptsBlock["scripts"] = newItems

	newScripts, err := json.Marshal(scriptsBlock)
	if err != nil {
		return nil, err
	}
	top["scripts"] = newScripts

	return json.Marshal(top)
}
