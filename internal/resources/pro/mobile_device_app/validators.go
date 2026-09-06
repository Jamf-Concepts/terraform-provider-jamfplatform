// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// makeAvailableAfterInstallPath is the sibling toggle that gates
// self_service.after_install_button_text. It lives under a different top-level
// block, so the path is absolute rather than relative.
var makeAvailableAfterInstallPath = path.Root("general").AtName("make_available_after_install")

// requiresMakeAvailableAfterInstall is the attribute-level validator for
// self_service.after_install_button_text. Jamf Pro stores and echoes that label
// only while general.make_available_after_install is true; with the toggle off
// the write is accepted and then silently discarded, and the GET omits the
// element entirely (wire-probed against Jamf Pro 11.31.1 on 2026-09-06, in
// isolation: the same POST with the toggle on round-trips the label and with it
// off drops it, every other field unchanged).
//
// Without this guard the attribute is a config value that does nothing, and the
// only signal is its absence from the refreshed state — which the read
// deliberately hides, because helpers.WireWhenPresentString keeps the
// configured value rather than nulling it and tripping "inconsistent result
// after apply". Catching it at plan time is what makes that read safe to have.
//
// This is a value-specific "requires-sibling-true" rule, so it is a custom
// validator.String per STYLE_GUIDE.md §Cross-field validation: off-the-shelf
// stringvalidator.AlsoRequires only asserts that the sibling is *present*, and
// would pass for make_available_after_install = false, which is the exact case
// this exists to reject.
type requiresMakeAvailableAfterInstall struct{}

var _ validator.String = requiresMakeAvailableAfterInstall{}

func (v requiresMakeAvailableAfterInstall) Description(_ context.Context) string {
	return "may only be set when general.make_available_after_install is true"
}

func (v requiresMakeAvailableAfterInstall) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v requiresMakeAvailableAfterInstall) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var gate types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, makeAvailableAfterInstallPath, &gate)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Defer on unknown or absent: a variable- or resource-sourced toggle is not
	// resolvable here, and an omitted Optional+Computed toggle resolves to the
	// server's own default. Error only when it is known to be false.
	if gate.IsNull() || gate.IsUnknown() || gate.ValueBool() {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Attribute Combination",
		fmt.Sprintf("%s requires general.make_available_after_install to be true. Jamf Pro discards the label otherwise, and never returns it. Set general.make_available_after_install = true, or remove this attribute.", req.Path),
	)
}
