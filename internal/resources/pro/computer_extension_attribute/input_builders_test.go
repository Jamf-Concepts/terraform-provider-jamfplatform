// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_extension_attribute

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildComputerExtensionAttributeInput_ManageExistingDataCreateVsUpdate
// guards against the regression that produced
// "[INVALID_CONTENT] manageExistingData: This field should be blank for
// first time CEA creation." on live SCRIPT-EA create: Jamf Pro 400s if
// manageExistingData is present on create, and requires it on update.
func TestBuildComputerExtensionAttributeInput_ManageExistingDataCreateVsUpdate(t *testing.T) {
	plan := ComputerExtensionAttributeResourceModel{
		Name:      types.StringValue("zz-script"),
		DataType:  types.StringValue("STRING"),
		InputType: types.StringValue(inputTypeScript),
		Script:    types.StringValue("#!/bin/sh\necho 1"),
	}

	created, diags := buildComputerExtensionAttributeInput(context.Background(), plan, types.StringNull(), true)
	if diags.HasError() {
		t.Fatalf("create diags: %v", diags)
	}
	if created.ManageExistingData != nil {
		t.Fatalf("Create must omit manageExistingData, got %q", *created.ManageExistingData)
	}

	updated, diags := buildComputerExtensionAttributeInput(context.Background(), plan, types.StringNull(), false)
	if diags.HasError() {
		t.Fatalf("update diags: %v", diags)
	}
	if updated.ManageExistingData == nil || *updated.ManageExistingData != manageExistingDataDefault {
		t.Fatalf("Update must default manageExistingData to %q, got %v", manageExistingDataDefault, updated.ManageExistingData)
	}
}
