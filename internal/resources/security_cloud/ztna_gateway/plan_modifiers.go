// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// requiresReplaceOnIpsecPresenceChange forces replacement when the `ipsec` block
// appears or disappears, and only then.
//
// Presence of the block is the gateway's form, and the form is immutable:
// wire-probed 2026-08-27, adding `ipsec` to an existing dedicated internet
// gateway is refused with `400 GATEWAY_TYPE_CHANGE_NOT_SUPPORTED`, and patching
// the dedicated-egress-IP flag in either direction returns `204` and then
// silently leaves the value alone — the worse of the two failures, because
// nothing tells the caller the write did not land. Editing fields *inside* an
// existing block is a normal in-place update, so the check is on null-ness only,
// not on the block's contents.
func requiresReplaceOnIpsecPresenceChange(_ context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = req.ConfigValue.IsNull() != req.StateValue.IsNull()
}
