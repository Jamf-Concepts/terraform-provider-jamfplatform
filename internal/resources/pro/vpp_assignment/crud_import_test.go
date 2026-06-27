// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_assignment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// TestImportModelRoundTripsThroughState reproduces the Read import path's final
// resp.State.Set. The import branch builds an identity-only model and the
// state-builder (includeUnmanaged=false) leaves an unmanaged content set
// untouched, so each *_adam_ids Set must already carry a concrete Int64 element
// type — a bare types.Set{} (DynamicPseudoType) fails conversion to the schema's
// Set[Int64] with "Set[!!! MISSING TYPE !!!]". Guards the typed-null seeding in
// Read against the schema staying Set[Int64].
func TestImportModelRoundTripsThroughState(t *testing.T) {
	ctx := context.Background()

	var sresp resource.SchemaResponse
	NewVPPAssignmentResource().Schema(ctx, resource.SchemaRequest{}, &sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", sresp.Diagnostics)
	}

	state := VPPAssignmentResourceModel{
		ID:            types.StringValue("2"),
		Timeouts:      helpers.NewResourceTimeoutsNullValue(vppAssignmentTimeoutAttributeTypes),
		IosAppAdamIDs: types.SetNull(types.Int64Type),
		MacAppAdamIDs: types.SetNull(types.Int64Type),
		EbookAdamIDs:  types.SetNull(types.Int64Type),
	}
	assignVPPAssignmentResourceModel(ctx, &state, sampleAPI(), false)

	tfState := tfsdk.State{Schema: sresp.Schema}
	if diags := tfState.Set(ctx, &state); diags.HasError() {
		t.Fatalf("State.Set on imported identity-only model failed: %v", diags)
	}
	for _, c := range []struct {
		name string
		set  types.Set
	}{
		{"ios_app_adam_ids", state.IosAppAdamIDs},
		{"mac_app_adam_ids", state.MacAppAdamIDs},
		{"ebook_adam_ids", state.EbookAdamIDs},
	} {
		if et := c.set.ElementType(ctx); !et.Equal(types.Int64Type) {
			t.Errorf("%s element type = %v, want Int64", c.name, et)
		}
	}
}
