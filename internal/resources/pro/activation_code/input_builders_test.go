// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package activation_code

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildActivationCodeInput verifies both fields are always sent (never a partial
// PUT) and that the code is whitespace-trimmed defensively.
func TestBuildActivationCodeInput(t *testing.T) {
	plan := ActivationCodeResourceModel{
		OrganizationName: types.StringValue("Acme Inc"),
		Code:             types.StringValue("  ABCD-EFGH-IJKL\n"),
	}
	in := buildActivationCodeInput(plan)

	if in.OrganizationName == nil || *in.OrganizationName != "Acme Inc" {
		t.Errorf("OrganizationName: want \"Acme Inc\", got %v", in.OrganizationName)
	}
	if in.Code == nil {
		t.Fatalf("Code must always be sent (never a partial PUT)")
	}
	if *in.Code != "ABCD-EFGH-IJKL" {
		t.Errorf("Code: want whitespace-trimmed \"ABCD-EFGH-IJKL\", got %q", *in.Code)
	}
}
