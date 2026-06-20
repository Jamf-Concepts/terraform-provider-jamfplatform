// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateCredentialRotationRequiresEnabled(t *testing.T) {
	p := path.Root("enabled")

	tests := []struct {
		name      string
		rotation  types.String
		enabled   types.Bool
		wantError bool
	}{
		{"rotation set, enabled false -> error", types.StringValue("1"), types.BoolValue(false), true},
		{"rotation set, enabled true -> ok", types.StringValue("1"), types.BoolValue(true), false},
		{"rotation set, enabled unknown -> skip", types.StringValue("1"), types.BoolUnknown(), false},
		{"rotation null -> skip", types.StringNull(), types.BoolValue(false), false},
		{"rotation unknown -> skip", types.StringUnknown(), types.BoolValue(false), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateCredentialRotationRequiresEnabled(tc.rotation, tc.enabled, p)
			if diags.HasError() != tc.wantError {
				t.Errorf("HasError = %v, want %v (diags: %v)", diags.HasError(), tc.wantError, diags)
			}
		})
	}
}
