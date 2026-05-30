// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_prestage_enrollment

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// listUseStateForUnknown is a List plan modifier with the same semantics as
// stringplanmodifier.UseStateForUnknown — the framework only exposes
// list-level modifiers via the listplanmodifier subpackage which doesn't yet
// ship UseStateForUnknown for List<String>.
func listUseStateForUnknown() planmodifier.List {
	return listUseStateForUnknownModifier{}
}

type listUseStateForUnknownModifier struct{}

func (listUseStateForUnknownModifier) Description(_ context.Context) string {
	return "Copies the prior state value into the plan when the plan is Unknown."
}

func (listUseStateForUnknownModifier) MarkdownDescription(ctx context.Context) string {
	return "Copies the prior state value into the plan when the plan is Unknown."
}

func (listUseStateForUnknownModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.StateValue.IsNull() {
		return
	}
	if !req.PlanValue.IsUnknown() {
		return
	}
	if req.ConfigValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}
