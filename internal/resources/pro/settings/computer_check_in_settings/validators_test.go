// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_check_in_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestComputerCheckInSettingsResource_CheckInFrequencyValidatorWired asserts that the
// check_in_frequency attribute carries the int64validator.OneOf validator. The exact
// allowed-value enforcement (5/15/30/60) is exercised end-to-end by the acceptance
// ExpectError test; this unit test just pins that a validator is wired at all so a
// refactor cannot silently drop it.
func TestComputerCheckInSettingsResource_CheckInFrequencyValidatorWired(t *testing.T) {
	r := NewComputerCheckInSettingsResource()
	var resp resource.SchemaResponse
	r.(*ComputerCheckInSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	attr, ok := resp.Schema.Attributes["check_in_frequency"].(schema.Int64Attribute)
	if !ok {
		t.Fatalf("check_in_frequency is not an Int64Attribute: %T", resp.Schema.Attributes["check_in_frequency"])
	}
	if len(attr.Validators) == 0 {
		t.Errorf("check_in_frequency must carry at least one validator (expected int64validator.OneOf)")
	}
}
