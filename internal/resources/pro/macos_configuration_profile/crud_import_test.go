// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// TestImportModelRoundTripsThroughState reproduces the Read import branch's final
// resp.State.Set. That branch seeds an identity-only model, so the timeouts value
// must carry the full create/read/update/delete object type — a zero
// resourcetimeouts.Value{} (empty Object[]) fails conversion to the schema's
// timeouts type. Guards the NewResourceTimeoutsNullValue seeding against the
// timeoutAttributeTypes map drifting from the schema's timeouts.Opts.
func TestImportModelRoundTripsThroughState(t *testing.T) {
	ctx := context.Background()

	var sresp resource.SchemaResponse
	NewResource().Schema(ctx, resource.SchemaRequest{}, &sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", sresp.Diagnostics)
	}

	state := tfsdk.State{Schema: sresp.Schema}
	model := ResourceModel{
		ID:       types.StringValue("123"),
		Timeouts: helpers.NewResourceTimeoutsNullValue(timeoutAttributeTypes),
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatalf("State.Set on imported identity-only model failed: %v", diags)
	}
}
