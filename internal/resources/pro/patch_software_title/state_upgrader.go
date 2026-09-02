// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// removedInV1 are the attributes dropped from the schema at v1. Both were
// display names the classic /patchsoftwaretitles payload carried inline
// alongside the id; the v3 configuration reports ids only, and resolving each
// name would cost a lookup per read for a value Terraform never writes.
var removedInV1 = []string{"category_name", "site_name"}

// UpgradeState migrates prior state versions to the current schema.
//
// v0 → v1: category_name and site_name were removed when read/update/delete
// moved from the classic /patchsoftwaretitles endpoints to Jamf Pro v3
// /patch-software-title-configurations, which does not report them. The
// framework cannot decode prior state carrying attributes the current schema
// has no home for, so the keys are stripped from the raw prior state JSON and
// the remainder re-emitted through the current schema type. Every other
// attribute is carried across verbatim as raw JSON, so no value is
// reinterpreted or lost.
func (r *PatchSoftwareTitleResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				if req.RawState == nil {
					return
				}

				rewritten, err := dropRemovedAttributes(req.RawState.JSON, removedInV1)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to upgrade jamfplatform_pro_patch_software_title state from v0 to v1",
						fmt.Sprintf("Could not remove the withdrawn category_name / site_name attributes from prior state: %s", err),
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
						"Unable to upgrade jamfplatform_pro_patch_software_title state from v0 to v1",
						fmt.Sprintf("Could not decode rewritten prior state against the current schema: %s", err),
					)
					return
				}

				dynamicValue, err := tfprotov6.NewDynamicValue(schemaType, value)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to upgrade jamfplatform_pro_patch_software_title state from v0 to v1",
						fmt.Sprintf("Could not re-encode the upgraded state: %s", err),
					)
					return
				}

				resp.DynamicValue = &dynamicValue
			},
		},
	}
}

// dropRemovedAttributes deletes the named top-level keys from a raw state JSON
// object. A key that is already absent is not an error — state written by a
// build that never set it is still valid v0 state.
func dropRemovedAttributes(raw []byte, names []string) ([]byte, error) {
	var state map[string]json.RawMessage
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("decode prior state: %w", err)
	}
	for _, name := range names {
		delete(state, name)
	}
	out, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("re-encode prior state: %w", err)
	}
	return out, nil
}
