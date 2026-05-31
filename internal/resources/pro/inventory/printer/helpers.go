// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Sentinel that the classic /printers endpoint stores when a printer has no
// category assigned. Visible on every GET; the provider decodes it to null
// in state and rejects it as a literal user input to avoid confusion.
const categoryUnassignedSentinel = "No category assigned"

// derefString returns the underlying string for a non-nil *string, or "" for
// nil. Used by the list resource for display-name rendering and by the
// classic-filter name accessor.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// stringPtrEmitAlways returns a non-nil *string regardless of whether the TF
// String is null. Null/unknown produce a pointer to "", which the SDK encodes
// as `<element></element>`. The classic /printers endpoint distinguishes:
//   - omitted tag → server preserves the current stored value (no-op)
//   - empty tag   → server clears the value (sentinels populated for
//     category, defaults restored for use_generic-bound fields)
//
// `category` is the field that needs this treatment: TF state "null" must
// mean "no category" not "leave the current category alone."
func stringPtrEmitAlways(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		empty := ""
		return &empty
	}
	out := v.ValueString()
	return &out
}
