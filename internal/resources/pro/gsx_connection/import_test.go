// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package gsx_connection

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestImportStateGuard_RejectsNonSingleton exercises the singleton import guard directly —
// no tenant or credentials needed. A non-singleton id raises a clear diagnostic. The valid
// (singleton) path is covered by the acceptance import round-trip.
func TestImportStateGuard_RejectsNonSingleton(t *testing.T) {
	r := NewGsxConnectionSettingsResource().(*GsxConnectionSettingsResource)

	var resp resource.ImportStateResponse
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "not-singleton"}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Errorf("expected an error for a non-singleton import id")
	}
}
