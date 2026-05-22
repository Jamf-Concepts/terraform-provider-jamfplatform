// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dock_item

// derefString returns the underlying string for a non-nil *string, or "" for
// nil. Used by the list resource for display-name rendering and by the
// classic-filter name accessor.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
