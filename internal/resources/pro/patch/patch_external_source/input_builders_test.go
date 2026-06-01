// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_external_source

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildPatchExternalSourceInput_FullRoundTrip(t *testing.T) {
	plan := PatchExternalSourceResourceModel{
		Name:                         types.StringValue("Jamf Auto Update"),
		Enabled:                      types.BoolValue(true),
		HostName:                     types.StringValue("definitions.datajar.mobi/v2/"),
		Port:                         types.Int64Value(8443),
		SslEnabled:                   types.BoolValue(true),
		CertificateValidationEnabled: types.BoolValue(false),
	}
	got := buildPatchExternalSourceInput(plan)

	if got.Name == nil || *got.Name != "Jamf Auto Update" {
		t.Errorf("Name not round-tripped: %v", got.Name)
	}
	if got.Enabled == nil || *got.Enabled != true {
		t.Errorf("Enabled not round-tripped: %v", got.Enabled)
	}
	if got.HostName == nil || *got.HostName != "definitions.datajar.mobi/v2/" {
		t.Errorf("HostName not round-tripped: %v", got.HostName)
	}
	if got.Port == nil || *got.Port != 8443 {
		t.Errorf("Port not round-tripped: %v", got.Port)
	}
	if got.SslEnabled == nil || *got.SslEnabled != true {
		t.Errorf("SslEnabled not round-tripped: %v", got.SslEnabled)
	}
	if got.CertificateValidationEnabled == nil || *got.CertificateValidationEnabled != false {
		t.Errorf("CertificateValidationEnabled not round-tripped: %v", got.CertificateValidationEnabled)
	}
	if got.ID != nil {
		t.Errorf("expected nil ID on write payload, got %d", *got.ID)
	}
}

// TestBuildPatchExternalSourceInput_FalseBoolsAreSentExplicitly guards the
// OptionalBoolPointer contract: a configured false must serialise as an explicit
// false pointer (not be dropped), so the server records the toggle.
func TestBuildPatchExternalSourceInput_FalseBoolsAreSentExplicitly(t *testing.T) {
	plan := PatchExternalSourceResourceModel{
		Name:                         types.StringValue("X"),
		Enabled:                      types.BoolValue(false),
		SslEnabled:                   types.BoolValue(false),
		CertificateValidationEnabled: types.BoolValue(false),
	}
	got := buildPatchExternalSourceInput(plan)
	for label, b := range map[string]*bool{
		"Enabled":                      got.Enabled,
		"SslEnabled":                   got.SslEnabled,
		"CertificateValidationEnabled": got.CertificateValidationEnabled,
	} {
		if b == nil {
			t.Errorf("%s: expected explicit false pointer, got nil", label)
			continue
		}
		if *b != false {
			t.Errorf("%s: expected false, got %v", label, *b)
		}
	}
}

// TestBuildPatchExternalSourceInput_NullUnknownBecomeNilPointers verifies that
// null/unknown optional fields map to nil pointers so the SDK omitempty tag
// drops them from the wire (server keeps / defaults them).
func TestBuildPatchExternalSourceInput_NullUnknownBecomeNilPointers(t *testing.T) {
	cases := []struct {
		name string
		plan PatchExternalSourceResourceModel
	}{
		{
			name: "null",
			plan: PatchExternalSourceResourceModel{
				Name:                         types.StringNull(),
				Enabled:                      types.BoolNull(),
				HostName:                     types.StringNull(),
				Port:                         types.Int64Null(),
				SslEnabled:                   types.BoolNull(),
				CertificateValidationEnabled: types.BoolNull(),
			},
		},
		{
			name: "unknown",
			plan: PatchExternalSourceResourceModel{
				Name:                         types.StringUnknown(),
				Enabled:                      types.BoolUnknown(),
				HostName:                     types.StringUnknown(),
				Port:                         types.Int64Unknown(),
				SslEnabled:                   types.BoolUnknown(),
				CertificateValidationEnabled: types.BoolUnknown(),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildPatchExternalSourceInput(tc.plan)
			if got.Name != nil {
				t.Errorf("Name: expected nil, got %q", *got.Name)
			}
			if got.Enabled != nil {
				t.Errorf("Enabled: expected nil, got %v", *got.Enabled)
			}
			if got.HostName != nil {
				t.Errorf("HostName: expected nil, got %q", *got.HostName)
			}
			if got.Port != nil {
				t.Errorf("Port: expected nil, got %d", *got.Port)
			}
			if got.SslEnabled != nil {
				t.Errorf("SslEnabled: expected nil, got %v", *got.SslEnabled)
			}
			if got.CertificateValidationEnabled != nil {
				t.Errorf("CertificateValidationEnabled: expected nil, got %v", *got.CertificateValidationEnabled)
			}
		})
	}
}
