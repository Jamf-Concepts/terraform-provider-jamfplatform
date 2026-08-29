// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_hostname_mappings

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// writeMappingObject builds one mapping element with each field set independently.
//
// It exists alongside validators_test.go's mappingObject, which fixes both booleans
// to false because the cross-field validator has no opinion on them. The write-input
// tests do: a builder that transposed ztna and secureDns, or aRecords and
// aaaaRecords, sends a wrong payload that a fixture with matching values cannot
// distinguish. So every field here differs from every other.
func writeMappingObject(t *testing.T, hostname string, ipv4, ipv6 types.Set, ztna, secureDNS bool) types.Object {
	t.Helper()
	object, diags := types.ObjectValue(mappingAttributeTypes, map[string]attr.Value{
		"hostname":              types.StringValue(hostname),
		"ipv4_addresses":        ipv4,
		"ipv6_addresses":        ipv6,
		"connect_to_ztna":       types.BoolValue(ztna),
		"connect_to_secure_dns": types.BoolValue(secureDNS),
	})
	if diags.HasError() {
		t.Fatalf("building mapping object: %v", diags)
	}
	return object
}

// TestBuildMappingsWriteInput_CarriesEveryField pins that each planned field reaches
// the field of the payload it belongs in.
//
// The mapping is built directly rather than round-tripped through
// assignHostnameMappingsResourceModel, so the assertions have something to be wrong
// against: a round-trip fixture supplies both sides of the comparison and a
// transposition survives it. The IPv4 and IPv6 lists are both populated for the same
// reason — the endpoint refuses an IPv4 address in aaaaRecords with the same opaque
// 400 it uses for everything else, which is the failure this ordering mistake would
// produce.
func TestBuildMappingsWriteInput_CarriesEveryField(t *testing.T) {
	const hostname = "Internal-Host.example.com"

	input, diags := buildMappingsWriteInput(context.Background(), mappingSet(t,
		writeMappingObject(t, hostname, addressSet(t, "10.0.0.1"), addressSet(t, "2001:db8::1"), true, false),
	))
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if len(input) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(input))
	}
	got := input[0]

	if got.Hostname != hostname {
		t.Errorf("Hostname = %q, want %q", got.Hostname, hostname)
	}
	if got.ARecords == nil {
		t.Fatal("ARecords must be sent")
	}
	if want := []string{"10.0.0.1"}; !slices.Equal(*got.ARecords, want) {
		t.Errorf("ARecords = %v, want %v", *got.ARecords, want)
	}
	if got.AaaaRecords == nil {
		t.Fatal("AaaaRecords must be sent")
	}
	if want := []string{"2001:db8::1"}; !slices.Equal(*got.AaaaRecords, want) {
		t.Errorf("AaaaRecords = %v, want %v", *got.AaaaRecords, want)
	}
	if got.Ztna == nil || !*got.Ztna {
		t.Errorf("Ztna = %v, want a pointer to true", got.Ztna)
	}
	if got.SecureDns == nil || *got.SecureDns {
		t.Errorf("SecureDns = %v, want a pointer to false", got.SecureDns)
	}
}

// TestBuildMappingsWriteInput_PreservesEveryMapping pins that a multi-mapping plan
// reaches the payload whole, with each mapping's fields still its own.
func TestBuildMappingsWriteInput_PreservesEveryMapping(t *testing.T) {
	nullSet := types.SetNull(types.StringType)

	input, diags := buildMappingsWriteInput(context.Background(), mappingSet(t,
		writeMappingObject(t, "one.example.com", addressSet(t, "10.0.0.1"), nullSet, true, false),
		writeMappingObject(t, "two.example.com", nullSet, addressSet(t, "2001:db8::2"), false, true),
	))
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if len(input) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(input))
	}

	byHostname := map[string]int{}
	for i, mapping := range input {
		byHostname[mapping.Hostname] = i
	}
	first, ok := byHostname["one.example.com"]
	if !ok {
		t.Fatalf("one.example.com is missing from the payload: %+v", input)
	}
	second, ok := byHostname["two.example.com"]
	if !ok {
		t.Fatalf("two.example.com is missing from the payload: %+v", input)
	}

	if !slices.Equal(*input[first].ARecords, []string{"10.0.0.1"}) || len(*input[first].AaaaRecords) != 0 {
		t.Errorf("one.example.com carries the wrong addresses: %+v", input[first])
	}
	if !slices.Equal(*input[second].AaaaRecords, []string{"2001:db8::2"}) || len(*input[second].ARecords) != 0 {
		t.Errorf("two.example.com carries the wrong addresses: %+v", input[second])
	}
	if input[first].Ztna == nil || !*input[first].Ztna || input[first].SecureDns == nil || *input[first].SecureDns {
		t.Errorf("one.example.com carries the wrong flags: %+v", input[first])
	}
	if input[second].Ztna == nil || *input[second].Ztna || input[second].SecureDns == nil || !*input[second].SecureDns {
		t.Errorf("two.example.com carries the wrong flags: %+v", input[second])
	}
}

// TestBuildMappingsWriteInput_SendsEmptyArraysNotNull pins the write side of the
// absent-equals-empty rule. The SDK types both address lists as *[]string with
// omitempty, which invites sending null; a mapping whose aRecords is absent is refused
// exactly as one whose aRecords is empty is, so there is nothing to gain and a nil
// pointer to get wrong.
//
// The assertion is on the marshalled payload rather than on the pointer, because the
// pointer cannot tell the two apart: a non-nil pointer to a nil slice is non-nil and
// has length zero, so a nil-ness check passes on exactly the payload this test exists
// to prevent. encoding/json does distinguish them — pointer-to-nil marshals as
// `"aRecords":null`, pointer-to-empty as `"aRecords":[]` — and the wire form is what
// the endpoint refuses.
func TestBuildMappingsWriteInput_SendsEmptyArraysNotNull(t *testing.T) {
	nullSet := types.SetNull(types.StringType)

	cases := map[string]struct {
		mapping     types.Object
		wantPresent []string
		wantAbsent  []string
	}{
		"ipv6 omitted": {
			mapping:     writeMappingObject(t, "a.example.com", addressSet(t, "10.0.0.1"), nullSet, false, false),
			wantPresent: []string{`"aRecords":["10.0.0.1"]`, `"aaaaRecords":[]`, `"ztna":false`, `"secureDns":false`},
			wantAbsent:  []string{`"aaaaRecords":null`},
		},
		"ipv4 omitted": {
			mapping:     writeMappingObject(t, "b.example.com", nullSet, addressSet(t, "2001:db8::1"), true, true),
			wantPresent: []string{`"aRecords":[]`, `"aaaaRecords":["2001:db8::1"]`, `"ztna":true`, `"secureDns":true`},
			wantAbsent:  []string{`"aRecords":null`},
		},
		"unknown ipv6 sent as empty": {
			mapping:     writeMappingObject(t, "c.example.com", addressSet(t, "10.0.0.1"), types.SetUnknown(types.StringType), false, false),
			wantPresent: []string{`"aaaaRecords":[]`},
			wantAbsent:  []string{`"aaaaRecords":null`},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			input, diags := buildMappingsWriteInput(context.Background(), mappingSet(t, tc.mapping))
			if diags.HasError() {
				t.Fatalf("diagnostics: %v", diags)
			}
			if len(input) != 1 {
				t.Fatalf("expected 1 mapping, got %d", len(input))
			}

			payload, err := json.Marshal(input[0])
			if err != nil {
				t.Fatalf("marshalling the payload: %v", err)
			}
			body := string(payload)

			for _, want := range tc.wantPresent {
				if !strings.Contains(body, want) {
					t.Errorf("payload must carry %s, got %s", want, body)
				}
			}
			for _, unwanted := range tc.wantAbsent {
				if strings.Contains(body, unwanted) {
					t.Errorf("payload must not carry %s, got %s", unwanted, body)
				}
			}
		})
	}
}
