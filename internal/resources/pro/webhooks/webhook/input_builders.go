// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildWebhookInput projects a plan model into an SDK *proclassic.Webhook for
// Create / Update. The classic PUT is a partial-merge, but the provider always
// sends the full plan so in-place edits converge.
//
// Two deliberate behaviours:
//
//   - `password` is sourced separately (it is a WriteOnly attribute, so the
//     plan exposes it as null). The caller threads a non-nil plaintext only
//     when `password_wo_version` changed; otherwise nil omits <password> and
//     Classic's merge retains the stored value (mirrors directory_binding).
//   - `smart_group_id` is ALWAYS emitted: the configured value, or the -1
//     sentinel ("none") when unset. This is load-bearing on transitions —
//     Classic's PUT merges, so omitting the element retains the stored group;
//     a smart→non-smart event change must therefore send -1 to clear the old
//     group, otherwise the server 409s (non-smart event + a real group is
//     illegal). The cross-field validator already forbids a configured
//     smart_group_id on a non-smart event, so the only value reaching a
//     non-smart event here is -1, which the server accepts and treats as
//     "none" (wire-probed — WEBHOOK_SPIKE.md §5.5). `display_fields` is never
//     written (Computed-only; the server 409s).
func buildWebhookInput(plan WebhookResourceModel, password *string) *proclassic.Webhook {
	in := &proclassic.Webhook{
		Name:                              helpers.OptionalStringPointer(plan.Name),
		Enabled:                           helpers.OptionalBoolPointer(plan.Enabled),
		URL:                               helpers.OptionalStringPointer(plan.URL),
		Event:                             helpers.OptionalStringPointer(plan.Event),
		ContentType:                       helpers.OptionalStringPointer(plan.ContentType),
		AuthenticationType:                helpers.OptionalStringPointer(plan.AuthenticationType),
		ConnectionTimeout:                 helpers.OptionalInt64Pointer(plan.ConnectionTimeout),
		ReadTimeout:                       helpers.OptionalInt64Pointer(plan.ReadTimeout),
		Username:                          helpers.OptionalStringPointer(plan.Username),
		Password:                          password,
		Header:                            helpers.OptionalStringPointer(plan.Header),
		HashAlgorithm:                     helpers.OptionalStringPointer(plan.HashAlgorithm),
		EnableDisplayFieldsForGroupObject: helpers.OptionalBoolPointer(plan.EnableDisplayFieldsForGroupObject),
	}

	if gid := helpers.OptionalInt64Pointer(plan.SmartGroupID); gid != nil {
		in.SmartGroupID = gid
	} else {
		none := -1
		in.SmartGroupID = &none
	}

	return in
}
