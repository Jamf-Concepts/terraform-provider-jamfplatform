// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestShouldRotateCredentials(t *testing.T) {
	tests := []struct {
		name        string
		planTrigger types.String
		state       types.String
		want        bool
	}{
		{"plan null -> no rotate", types.StringNull(), types.StringValue("1"), false},
		{"plan unknown -> no rotate", types.StringUnknown(), types.StringValue("1"), false},
		{"state null, plan set -> rotate", types.StringValue("1"), types.StringNull(), true},
		{"value changed -> rotate", types.StringValue("2"), types.StringValue("1"), true},
		{"value unchanged -> no rotate", types.StringValue("1"), types.StringValue("1"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRotateCredentials(tc.planTrigger, tc.state); got != tc.want {
				t.Errorf("shouldRotateCredentials = %v, want %v", got, tc.want)
			}
		})
	}
}
