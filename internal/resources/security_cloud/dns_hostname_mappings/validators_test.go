// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_hostname_mappings

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestIPv6Address pins the forms the endpoint rejects with one indistinguishable
// message, plus the ones it accepts.
func TestIPv6Address(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"documentation prefix", "2001:db8::1", false},
		{"full form", "2001:db8:3333:4444:5555:6666:7777:8888", false},
		{"loopback", "::1", false},
		{"all zeros", "::", false},
		{"uppercase hex", "2001:DB8::ABCD", false},

		{"dotted quad", "203.0.113.53", true},
		{"ipv4-mapped", "::ffff:203.0.113.53", true},
		{"not an address", "not-an-address", true},
		{"empty", "", true},
		{"trailing junk", "2001:db8::1 ", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			IPv6Address().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("ipv6_addresses"),
				ConfigValue: types.StringValue(tc.value),
			}, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Fatalf("HasError() = %t, want %t (diags: %v)", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestIPv6AddressDefersOnNullAndUnknown pins STYLE_GUIDE §Config-time validators.
func TestIPv6AddressDefersOnNullAndUnknown(t *testing.T) {
	for name, value := range map[string]types.String{
		"null":    types.StringNull(),
		"unknown": types.StringUnknown(),
	} {
		t.Run(name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			IPv6Address().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("ipv6_addresses"),
				ConfigValue: value,
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no diagnostics, got %v", resp.Diagnostics)
			}
		})
	}
}

// mappingObject builds one mapping element for the cross-field validator tests.
func mappingObject(t *testing.T, hostname string, ipv4, ipv6 types.Set) types.Object {
	t.Helper()
	return mappingObjectNamed(t, types.StringValue(hostname), ipv4, ipv6)
}

// mappingObjectNamed builds one mapping element whose host name may be null or
// unknown, which mappingObject cannot express. A host name computed from another
// resource is a real configuration — and the only one that reaches the diagnostic's
// fallback wording.
func mappingObjectNamed(t *testing.T, hostname types.String, ipv4, ipv6 types.Set) types.Object {
	t.Helper()
	object, diags := types.ObjectValue(mappingAttributeTypes, map[string]attr.Value{
		"hostname":              hostname,
		"ipv4_addresses":        ipv4,
		"ipv6_addresses":        ipv6,
		"connect_to_ztna":       types.BoolValue(false),
		"connect_to_secure_dns": types.BoolValue(false),
	})
	if diags.HasError() {
		t.Fatalf("building mapping object: %v", diags)
	}
	return object
}

// addressSet builds a populated address set for the tests.
func addressSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	elements := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elements = append(elements, types.StringValue(v))
	}
	set, diags := types.SetValue(types.StringType, elements)
	if diags.HasError() {
		t.Fatalf("building address set: %v", diags)
	}
	return set
}

// mappingSet builds the mappings collection for the tests.
func mappingSet(t *testing.T, objects ...types.Object) types.Set {
	t.Helper()
	elements := make([]attr.Value, 0, len(objects))
	for _, o := range objects {
		elements = append(elements, o)
	}
	set, diags := types.SetValue(mappingObjectType, elements)
	if diags.HasError() {
		t.Fatalf("building mapping set: %v", diags)
	}
	return set
}

// TestEachMappingHasAnAddress covers the rule the server enforces but misattributes.
func TestEachMappingHasAnAddress(t *testing.T) {
	nullSet := types.SetNull(types.StringType)
	emptySet := addressSet(t)

	cases := map[string]struct {
		value   types.Set
		wantErr bool
	}{
		"ipv4 only": {
			value: mappingSet(t, mappingObject(t, "a.example.com", addressSet(t, "10.0.0.1"), nullSet)),
		},
		"ipv6 only": {
			value: mappingSet(t, mappingObject(t, "b.example.com", nullSet, addressSet(t, "2001:db8::1"))),
		},
		"both": {
			value: mappingSet(t, mappingObject(t, "c.example.com", addressSet(t, "10.0.0.1"), addressSet(t, "2001:db8::1"))),
		},
		"neither, both null": {
			value:   mappingSet(t, mappingObject(t, "d.example.com", nullSet, nullSet)),
			wantErr: true,
		},
		"neither, both empty": {
			value:   mappingSet(t, mappingObject(t, "e.example.com", emptySet, emptySet)),
			wantErr: true,
		},
		"one good one bad": {
			value: mappingSet(t,
				mappingObject(t, "f.example.com", addressSet(t, "10.0.0.1"), nullSet),
				mappingObject(t, "g.example.com", nullSet, nullSet),
			),
			wantErr: true,
		},
		"unknown addresses defer": {
			value: mappingSet(t, mappingObject(t, "h.example.com", types.SetUnknown(types.StringType), nullSet)),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp := &validator.SetResponse{}
			EachMappingHasAnAddress().ValidateSet(context.Background(), validator.SetRequest{
				Path:        path.Root("mappings"),
				ConfigValue: tc.value,
			}, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Fatalf("HasError() = %t, want %t (diags: %v)", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestEachMappingHasAnAddressNamesBothFields is the point of having the validator at
// all: the server blames aRecords whichever list was omitted, so a diagnostic that
// only named ipv4_addresses would repeat the server's own misdirection.
func TestEachMappingHasAnAddressNamesBothFields(t *testing.T) {
	nullSet := types.SetNull(types.StringType)
	resp := &validator.SetResponse{}
	EachMappingHasAnAddress().ValidateSet(context.Background(), validator.SetRequest{
		Path:        path.Root("mappings"),
		ConfigValue: mappingSet(t, mappingObject(t, "d.example.com", nullSet, nullSet)),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a diagnostic")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	for _, want := range []string{"ipv4_addresses", "ipv6_addresses", "d.example.com"} {
		if !strings.Contains(detail, want) {
			t.Errorf("diagnostic must mention %q, got: %s", want, detail)
		}
	}
}

// TestEachMappingHasAnAddressNamesAnUnnamedMapping covers the diagnostic's fallback
// wording, which a fixture with a known host name never reaches.
//
// The case is real rather than defensive: a host name interpolated from another
// resource is unknown at plan time while both address lists are known, so
// hasUnknownAddresses does not defer and the mapping is judged with nothing to call
// it. Printing an empty pair of quotes there would read as a mapping whose host name
// is the empty string.
func TestEachMappingHasAnAddressNamesAnUnnamedMapping(t *testing.T) {
	nullSet := types.SetNull(types.StringType)

	for name, hostname := range map[string]types.String{
		"unknown": types.StringUnknown(),
		"null":    types.StringNull(),
	} {
		t.Run(name, func(t *testing.T) {
			resp := &validator.SetResponse{}
			EachMappingHasAnAddress().ValidateSet(context.Background(), validator.SetRequest{
				Path:        path.Root("mappings"),
				ConfigValue: mappingSet(t, mappingObjectNamed(t, hostname, nullSet, nullSet)),
			}, resp)

			if !resp.Diagnostics.HasError() {
				t.Fatal("expected a diagnostic: an unnamed mapping with no address is still invalid")
			}
			detail := resp.Diagnostics.Errors()[0].Detail()
			if !strings.Contains(detail, "one of the mappings") {
				t.Errorf("diagnostic must fall back to naming the collection, got: %s", detail)
			}
			if strings.Contains(detail, "\"\"") {
				t.Errorf("diagnostic must not print an empty quoted host name, got: %s", detail)
			}
		})
	}
}

// TestEachMappingHasAnAddressDefersOnNullAndUnknownSet pins that the collection
// itself being absent is not this validator's business.
func TestEachMappingHasAnAddressDefersOnNullAndUnknownSet(t *testing.T) {
	for name, value := range map[string]types.Set{
		"null":    types.SetNull(mappingObjectType),
		"unknown": types.SetUnknown(mappingObjectType),
	} {
		t.Run(name, func(t *testing.T) {
			resp := &validator.SetResponse{}
			EachMappingHasAnAddress().ValidateSet(context.Background(), validator.SetRequest{
				Path:        path.Root("mappings"),
				ConfigValue: value,
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no diagnostics, got %v", resp.Diagnostics)
			}
		})
	}
}
