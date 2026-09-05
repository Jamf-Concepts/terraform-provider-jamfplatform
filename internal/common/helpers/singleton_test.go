// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// singletonImportFixture builds the request and response pair the framework hands
// ImportState, with an identity carrying the supplied value. A nil identityValue
// omits the identity entirely, which is the shape of a `terraform import` or an
// `id =` import block.
func singletonImportFixture(t *testing.T, id string, identityValue *string) (resource.ImportStateRequest, *resource.ImportStateResponse) {
	t.Helper()

	stateSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
		},
	}
	idSchema := identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}

	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String}}

	req := resource.ImportStateRequest{ID: id}
	if identityValue != nil {
		req.Identity = &tfsdk.ResourceIdentity{
			Schema: idSchema,
			Raw:    tftypes.NewValue(objType, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, *identityValue)}),
		}
	}

	// The framework seeds resp.Identity from req.Identity, so it already carries
	// the value on the identity path and is fully null on the req.ID path — which
	// is the asymmetry that made `terraform import <addr> singleton` produce an
	// identity-less resource. Mirrored here so the fixture fails the way production
	// did rather than the way a uniform null would.
	respIdentityRaw := tftypes.NewValue(objType, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, nil)})
	if identityValue != nil {
		respIdentityRaw = tftypes.NewValue(objType, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, *identityValue)})
	}

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: stateSchema,
			Raw:    tftypes.NewValue(objType, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, nil)}),
		},
		Identity: &tfsdk.ResourceIdentity{
			Schema: idSchema,
			Raw:    respIdentityRaw,
		},
	}
	return req, resp
}

// stateID reads the id the helper committed to state.
func stateID(t *testing.T, resp *resource.ImportStateResponse) types.String {
	t.Helper()
	var got types.String
	diags := resp.State.GetAttribute(context.Background(), path.Root("id"), &got)
	if diags.HasError() {
		t.Fatalf("reading id back from state: %v", diags.Errors())
	}
	return got
}

// identityID reads the id the helper mirrored into the resource identity.
//
// This has to hold on BOTH import forms, and the framework's own passthrough only
// manages it on one: on the req.ID path it writes state and returns, leaving the
// identity fully null. A singleton whose Read then finds nothing on the tenant
// returns with no error and no identity, and the framework answers that with
// "Missing Resource Identity After Read … report this to the provider
// developers". See ImportSingletonState.
func identityID(t *testing.T, resp *resource.ImportStateResponse) types.String {
	t.Helper()
	if resp.Identity == nil {
		t.Fatal("the framework always pre-populates an identity for a resource with an IdentitySchema")
	}
	var got types.String
	diags := resp.Identity.GetAttribute(context.Background(), path.Root("id"), &got)
	if diags.HasError() {
		t.Fatalf("reading id back from the identity: %v", diags.Errors())
	}
	return got
}

// TestImportSingletonState covers both import forms. The identity case is the
// regression the helper exists for: that form leaves req.ID empty, and every
// singleton resource used to refuse it outright while advertising an
// IdentitySchema that said it worked.
func TestImportSingletonState(t *testing.T) {
	singleton := SingletonID
	wrong := "not-the-singleton"
	empty := ""

	tests := []struct {
		name          string
		id            string
		identityValue *string
		wantErr       bool
	}{
		{
			name: "id form accepts the singleton",
			id:   SingletonID,
		},
		{
			name:          "identity form accepts the singleton",
			identityValue: &singleton,
		},
		{
			name:          "id form wins when both are supplied",
			id:            SingletonID,
			identityValue: &singleton,
		},
		{
			name:    "id form refuses anything else",
			id:      wrong,
			wantErr: true,
		},
		{
			name:          "identity form refuses anything else",
			identityValue: &wrong,
			wantErr:       true,
		},
		{
			name:          "an empty identity is refused rather than treated as the singleton",
			identityValue: &empty,
			wantErr:       true,
		},
		{
			name:    "neither form supplied is refused",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, resp := singletonImportFixture(t, tc.id, tc.identityValue)

			ImportSingletonState(context.Background(), req, resp, "jamfplatform_pro_example_settings")

			if tc.wantErr {
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected an error diagnostic, got none")
				}
				for _, d := range resp.Diagnostics.Errors() {
					if d.Summary() != "Invalid singleton import identifier" {
						t.Errorf("unexpected diagnostic: %s", d.Summary())
					}
					if !strings.Contains(d.Detail(), "jamfplatform_pro_example_settings") {
						t.Errorf("diagnostic must name the resource type, got %q", d.Detail())
					}
				}
				return
			}

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected error: %v", resp.Diagnostics.Errors())
			}
			if got := stateID(t, resp); got.ValueString() != SingletonID {
				t.Errorf("state id = %q, want %q", got.ValueString(), SingletonID)
			}
			if got := identityID(t, resp); got.ValueString() != SingletonID {
				t.Errorf("identity id = %q, want %q — an import that leaves the identity null makes the "+
					"framework refuse the read that follows with \"Missing Resource Identity After Read\", "+
					"which tells the practitioner to report a provider bug",
					got.ValueString(), SingletonID)
			}
		})
	}
}
