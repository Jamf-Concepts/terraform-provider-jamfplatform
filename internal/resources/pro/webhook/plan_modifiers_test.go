// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestHashAlgorithmPlanDecision(t *testing.T) {
	hash := types.StringValue(authTypeHashSignature)
	none := types.StringValue(authTypeNone)

	tests := []struct {
		name       string
		configNull bool
		stateNull  bool
		planAuth   types.String
		stateAuth  types.String
		want       hashAlgoPlanAction
	}{
		{"explicit config honoured", false, false, none, hash, hashAlgoHonorConfig},
		{"create leaves proposal", true, true, hash, types.StringNull(), hashAlgoLeaveCreate},
		{"auth changing -> unknown (HASH->NONE)", true, false, none, hash, hashAlgoUnknownOnAuthChange},
		{"auth changing -> unknown (NONE->HASH)", true, false, hash, none, hashAlgoUnknownOnAuthChange},
		{"auth unknown -> unknown", true, false, types.StringUnknown(), none, hashAlgoUnknownOnAuthChange},
		{"auth stable -> reuse state", true, false, hash, hash, hashAlgoReuseState},
		{"auth stable none -> reuse state", true, false, none, none, hashAlgoReuseState},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hashAlgorithmPlanDecision(tt.configNull, tt.stateNull, tt.planAuth, tt.stateAuth)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
