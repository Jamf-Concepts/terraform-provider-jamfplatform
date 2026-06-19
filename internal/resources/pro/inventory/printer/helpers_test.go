// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringPtrEmitAlways(t *testing.T) {
	cases := []struct {
		name string
		in   types.String
		want string
	}{
		{"value", types.StringValue("Printers"), "Printers"},
		{"empty", types.StringValue(""), ""},
		{"null", types.StringNull(), ""},
		{"unknown", types.StringUnknown(), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stringPtrEmitAlways(c.in)
			if got == nil {
				t.Fatalf("expected non-nil pointer regardless of null state")
			}
			if *got != c.want {
				t.Errorf("expected %q, got %q", c.want, *got)
			}
		})
	}
}

func TestDecodeCategory(t *testing.T) {
	sentinel := categoryUnassignedSentinel
	empty := ""
	value := "Printers"

	cases := []struct {
		name     string
		in       *string
		wantNull bool
		wantVal  string
	}{
		{"nil", nil, true, ""},
		{"empty", &empty, true, ""},
		{"sentinel", &sentinel, true, ""},
		{"value", &value, false, "Printers"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decodeCategory(c.in)
			if got.IsNull() != c.wantNull {
				t.Errorf("null-ness: expected %v, got %v (value=%q)", c.wantNull, got.IsNull(), got.ValueString())
				return
			}
			if !got.IsNull() && got.ValueString() != c.wantVal {
				t.Errorf("expected %q, got %q", c.wantVal, got.ValueString())
			}
		})
	}
}

func TestDerefString(t *testing.T) {
	if helpers.DerefString(nil) != "" {
		t.Errorf("nil deref must be empty string")
	}
	v := "hello"
	if helpers.DerefString(&v) != "hello" {
		t.Errorf("expected hello, got %q", helpers.DerefString(&v))
	}
}
