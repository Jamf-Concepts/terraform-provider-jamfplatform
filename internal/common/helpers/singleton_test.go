// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// TestImportSingletonState covers both import forms and every way the identifier
// can be wrong, driven through the framework server so the request looks the way
// Terraform builds it — including the null identity fwserver pre-populates when
// the practitioner supplied none.
//
// The identity cases are the regression the helper exists for: that form leaves
// req.ID empty, and every singleton resource used to refuse it outright while
// advertising an IdentitySchema that said it worked.
func TestImportSingletonState(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		identity *tfprotov6.ResourceIdentityData
		wantErr  string
	}{
		{
			name: "id form accepts the singleton",
			id:   helpers.SingletonID,
		},
		{
			name:     "identity form accepts the singleton",
			identity: identityData(t, helpers.SingletonID),
		},
		{
			name:    "id form refuses anything else",
			id:      "not-the-singleton",
			wantErr: `Got "not-the-singleton"`,
		},
		{
			name:     "identity form refuses anything else",
			identity: identityData(t, "not-the-singleton"),
			wantErr:  `Got "not-the-singleton"`,
		},
		{
			name:     "an empty identity is refused rather than treated as the singleton",
			identity: identityData(t, ""),
			wantErr:  `Got ""`,
		},
		{
			name:    "neither form supplied is refused",
			wantErr: `Got ""`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := providerserver.NewProtocol6(probeProvider{})()

			resp, err := srv.ImportResourceState(context.Background(), &tfprotov6.ImportResourceStateRequest{
				TypeName: probeTypeName,
				ID:       tc.id,
				Identity: tc.identity,
			})
			if err != nil {
				t.Fatalf("ImportResourceState: %v", err)
			}

			if tc.wantErr != "" {
				assertRefusal(t, resp.Diagnostics, tc.wantErr)
				return
			}

			if hasError(resp.Diagnostics) {
				t.Fatalf("unexpected diagnostics: %s", renderDiags(resp.Diagnostics))
			}
			if len(resp.ImportedResources) != 1 {
				t.Fatalf("expected 1 imported resource, got %d", len(resp.ImportedResources))
			}
			assertSingletonID(t, "state", resp.ImportedResources[0].State)
			if resp.ImportedResources[0].Identity == nil {
				t.Fatal("the imported resource carries no identity; the framework refuses the read that follows " +
					"with \"Missing Resource Identity After Read\", which tells the practitioner to report a " +
					"provider bug")
			}
			assertSingletonID(t, "identity", resp.ImportedResources[0].Identity.IdentityData)
		})
	}
}

// assertRefusal checks the helper's own diagnostic came back, naming the resource
// type that refused and the value it was given.
func assertRefusal(t *testing.T, diags []*tfprotov6.Diagnostic, wantGot string) {
	t.Helper()

	if !hasError(diags) {
		t.Fatal("expected an error diagnostic, got none")
	}
	for _, d := range diags {
		if d.Severity != tfprotov6.DiagnosticSeverityError {
			continue
		}
		if d.Summary != "Invalid singleton import identifier" {
			t.Errorf("unexpected diagnostic: %s", d.Summary)
		}
		if !strings.Contains(d.Detail, "jamfplatform_pro_example_settings") {
			t.Errorf("diagnostic must name the resource type, got %q", d.Detail)
		}
		if !strings.Contains(d.Detail, wantGot) {
			t.Errorf("diagnostic must report the value it was given (%s), got %q", wantGot, d.Detail)
		}
	}
}

// assertSingletonID decodes one of the import response's object values and checks
// its id is the singleton constant.
func assertSingletonID(t *testing.T, label string, dv *tfprotov6.DynamicValue) {
	t.Helper()

	if dv == nil {
		t.Fatalf("%s is absent from the import response", label)
	}
	val, err := dv.Unmarshal(probeObjType)
	if err != nil {
		t.Fatalf("unmarshalling %s: %v", label, err)
	}
	var attrs map[string]tftypes.Value
	if err := val.As(&attrs); err != nil {
		t.Fatalf("reading %s as an object: %v", label, err)
	}
	var got string
	if err := attrs["id"].As(&got); err != nil {
		t.Fatalf("reading %s id: %v", label, err)
	}
	if got != helpers.SingletonID {
		t.Errorf("%s id = %q, want %q", label, got, helpers.SingletonID)
	}
}
