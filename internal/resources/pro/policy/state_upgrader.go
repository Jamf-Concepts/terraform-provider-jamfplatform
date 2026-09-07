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
// Both migrations rewrite the raw prior-state JSON and re-emit it through the
// current schema type. Every attribute they do not name is carried across
// verbatim as raw JSON, so no value is reinterpreted or lost.
//
// v0 → v1: the script-assignment parameter attributes were renamed from
// `parameter4`…`parameter11` to snake_case `parameter_4`…`parameter_11` to
// align with the jamfplatform_pro_script resource and the provider-wide
// snake_case attribute convention. A nested-object attribute rename is
// something the framework cannot decode against the newer schema on its own.
//
// v1 → v2: `printers.leave_existing_default` was removed. It modelled a dead
// wire element — Jamf Pro answers `<leave_existing_default/>` however the
// field was written, on an admin-UI-configured policy as much as a
// Terraform-managed one, and the default-printer choice it appeared to express
// is actually persisted per printer as `printers[].make_default`. An attribute
// that is gone from the schema must be gone from prior state too, or the decode
// fails on the surplus key.
//
// The v0 upgrader applies both rewrites, because state written against v0 has
// never been through the v1 → v2 step either. Terraform runs exactly one
// upgrader — the one matching the stored version — rather than chaining them.
func (r *PolicyResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				r.upgradeRawState(ctx, req, resp, "v0", func(raw []byte) ([]byte, error) {
					renamed, err := renameScriptParameterKeys(raw)
					if err != nil {
						return nil, fmt.Errorf("renaming script parameter keys: %w", err)
					}
					return dropPrintersLeaveExistingDefault(renamed)
				})
			},
		},
		1: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				r.upgradeRawState(ctx, req, resp, "v1", dropPrintersLeaveExistingDefault)
			},
		},
	}
}

// upgradeRawState applies one raw-JSON rewrite and re-encodes the result
// against the current schema. from names the stored version for diagnostics.
func (r *PolicyResource) upgradeRawState(
	ctx context.Context,
	req resource.UpgradeStateRequest,
	resp *resource.UpgradeStateResponse,
	from string,
	rewrite func([]byte) ([]byte, error),
) {
	if req.RawState == nil {
		return
	}

	fail := func(what string, err error) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to upgrade jamfplatform_pro_policy state from %s", from),
			fmt.Sprintf("%s: %s", what, err),
		)
	}

	rewritten, err := rewrite(req.RawState.JSON)
	if err != nil {
		fail("Could not rewrite prior state", err)
		return
	}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	upgraded := tfprotov6.RawState{JSON: rewritten}
	value, err := upgraded.Unmarshal(schemaType)
	if err != nil {
		fail("Could not decode rewritten prior state against the current schema", err)
		return
	}

	dynamicValue, err := tfprotov6.NewDynamicValue(schemaType, value)
	if err != nil {
		fail("Could not encode upgraded state", err)
		return
	}

	resp.DynamicValue = &dynamicValue
}

// dropPrintersLeaveExistingDefault removes the printers.leave_existing_default
// key from prior-state JSON. Everything else is preserved as raw JSON bytes, so
// numeric IDs and other values keep their exact representation. A state that
// never carried the key, or carries a null printers block, is returned
// untouched.
func dropPrintersLeaveExistingDefault(rawJSON []byte) ([]byte, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &top); err != nil {
		return nil, err
	}

	printersRaw, ok := top["printers"]
	if !ok || string(printersRaw) == "null" {
		return rawJSON, nil
	}

	var printersBlock map[string]json.RawMessage
	if err := json.Unmarshal(printersRaw, &printersBlock); err != nil {
		return nil, err
	}
	if _, present := printersBlock["leave_existing_default"]; !present {
		return rawJSON, nil
	}
	delete(printersBlock, "leave_existing_default")

	newPrinters, err := json.Marshal(printersBlock)
	if err != nil {
		return nil, err
	}
	top["printers"] = newPrinters

	return json.Marshal(top)
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
