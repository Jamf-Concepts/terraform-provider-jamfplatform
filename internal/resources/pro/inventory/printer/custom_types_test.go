// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestTrimmedStringValue_SemanticEquals_TrailingWhitespace(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"identical", "foo", "foo", true},
		{"a-trails-newline", "foo\n", "foo", true},
		{"b-trails-newline", "foo", "foo\n", true},
		{"both-trail-different-whitespace", "foo  \n", "foo\t", true},
		{"different-content", "foo", "bar", false},
		{"leading-whitespace-not-stripped", " foo", "foo", false},
		{"internal-whitespace-not-stripped", "foo bar", "foobar", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := newTrimmedStringValue(c.a)
			b := newTrimmedStringValue(c.b)
			got, diags := a.StringSemanticEquals(context.Background(), b)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got != c.want {
				t.Errorf("StringSemanticEquals(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestTrimmedStringValue_SemanticEquals_NullAndUnknown(t *testing.T) {
	null := newTrimmedStringNull()
	unknown := trimmedStringValue{StringValue: basetypes.NewStringUnknown()}
	value := newTrimmedStringValue("foo")

	if eq, _ := null.StringSemanticEquals(context.Background(), null); !eq {
		t.Errorf("null vs null must be semantically equal")
	}
	if eq, _ := unknown.StringSemanticEquals(context.Background(), unknown); !eq {
		t.Errorf("unknown vs unknown must be semantically equal")
	}
	if eq, _ := null.StringSemanticEquals(context.Background(), value); eq {
		t.Errorf("null vs value must NOT be semantically equal")
	}
	if eq, _ := unknown.StringSemanticEquals(context.Background(), value); eq {
		t.Errorf("unknown vs value must NOT be semantically equal")
	}
}

func TestTrimmedStringValue_SemanticEquals_WrongTypeYieldsError(t *testing.T) {
	v := newTrimmedStringValue("foo")
	_, diags := v.StringSemanticEquals(context.Background(), basetypes.NewStringValue("foo"))
	if !diags.HasError() {
		t.Errorf("expected diagnostics on type mismatch, got none")
	}
}

func TestTrimmedStringValueFromPtr(t *testing.T) {
	if !trimmedStringValueFromPtr(nil).IsNull() {
		t.Errorf("nil pointer must yield null trimmedStringValue")
	}
	s := "abc"
	if got := trimmedStringValueFromPtr(&s); got.ValueString() != "abc" {
		t.Errorf("expected abc, got %q", got.ValueString())
	}
}

func TestTrimmedStringType_String(t *testing.T) {
	tt := trimmedStringType{}
	if tt.String() == "" {
		t.Errorf("String() must be non-empty for debugging output")
	}
}
