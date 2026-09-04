// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// TestJSONSemanticEquality is the test that matters most in this package: without it every plan for
// a policy authored with jsonencode drifts, because the platform returns the settings with the keys
// in the order they were first written and jsonencode sorts them.
func TestJSONSemanticEquality(t *testing.T) {
	cases := []struct {
		name  string
		left  string
		right string
		equal bool
	}{
		{"identical", `{"a":1}`, `{"a":1}`, true},
		{"key order differs", `{"b":2,"a":1}`, `{"a":1,"b":2}`, true},
		{"indentation differs", "{\n  \"a\": 1\n}", `{"a":1}`, true},
		{"nested key order differs", `{"x":{"b":2,"a":1}}`, `{"x":{"a":1,"b":2}}`, true},
		{"integer and float spelling of the same number", `{"a":1}`, `{"a":1.0}`, true},
		{"array with identical order", `{"a":[1,2]}`, `{"a":[1,2]}`, true},
		{"array of objects with differing key order", `{"a":[{"x":1,"y":2}]}`, `{"a":[{"y":2,"x":1}]}`, true},
		{"permissions allow list reindented", `{"permissions":{"allow":["Bash(git:*)","Read"]}}`, "{\n  \"permissions\": {\n    \"allow\": [\n      \"Bash(git:*)\",\n      \"Read\"\n    ]\n  }\n}", true},
		{"array order differs", `{"a":[1,2]}`, `{"a":[2,1]}`, false},
		{"array length differs", `{"a":[1]}`, `{"a":[1,2]}`, false},
		{"array against scalar", `{"a":[1]}`, `{"a":1}`, false},
		{"permissions allow list gains an entry", `{"permissions":{"allow":["Bash(git:*)","Read"]}}`, `{"permissions":{"allow":["Bash(git:*)","Read","Write"]}}`, false},
		{"different value", `{"a":1}`, `{"a":2}`, false},
		{"extra key", `{"a":1}`, `{"a":1,"b":2}`, false},
		{"empty against populated", `{}`, `{"a":1}`, false},
		{"both unparseable and identical", `{oops`, `{oops`, true},
		{"both unparseable and different", `{oops`, `{nope`, false},
		{"one unparseable", `{"a":1}`, `{oops`, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, diags := newJSONObjectValue(c.left).StringSemanticEquals(context.Background(), newJSONObjectValue(c.right))
			if diags.HasError() {
				t.Fatalf("diagnostics: %v", diags)
			}
			if got != c.equal {
				t.Errorf("semantic equality of %s and %s = %v, want %v", c.left, c.right, got, c.equal)
			}
		})
	}
}

func TestJSONSemanticEqualityNullAndUnknown(t *testing.T) {
	null := newJSONObjectNull()
	unknown := jsonObjectValue{StringValue: basetypes.NewStringUnknown()}
	value := newJSONObjectValue(`{}`)

	for _, c := range []struct {
		name  string
		left  jsonObjectValue
		right jsonObjectValue
		equal bool
	}{
		{"null equals null", null, null, true},
		{"unknown equals unknown", unknown, unknown, true},
		{"null differs from value", null, value, false},
		{"unknown differs from value", unknown, value, false},
		{"null differs from unknown", null, unknown, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, diags := c.left.StringSemanticEquals(context.Background(), c.right)
			if diags.HasError() {
				t.Fatalf("diagnostics: %v", diags)
			}
			if got != c.equal {
				t.Errorf("got %v, want %v", got, c.equal)
			}
		})
	}
}

func TestJSONObjectTypeIdentity(t *testing.T) {
	if !(jsonObjectType{}).Equal(jsonObjectType{}) {
		t.Error("jsonObjectType must equal itself")
	}
	if (jsonObjectType{}).Equal(basetypes.StringType{}) {
		t.Error("jsonObjectType must not equal a plain StringType")
	}
	if (jsonObjectType{}).String() == "" {
		t.Error("jsonObjectType must name itself for error messages")
	}
	if _, ok := (jsonObjectType{}).ValueType(context.Background()).(jsonObjectValue); !ok {
		t.Error("jsonObjectType must produce a jsonObjectValue")
	}
}

func TestSettingsValidator(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr string
	}{
		{"object", `{"model":"sonnet"}`, ""},
		{"empty object", `{}`, ""},
		{"array", `[]`, "must be a JSON object"},
		{"string", `"nope"`, "must be a JSON object"},
		{"number", `1`, "must be a JSON object"},
		{"null literal", `null`, "must be a JSON object"},
		{"unparseable", `{oops`, "not valid JSON"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("settings_json"),
				ConfigValue: basetypes.NewStringValue(c.value),
			}
			var resp validator.StringResponse
			jsonObjectValidator().ValidateString(context.Background(), req, &resp)

			if c.wantErr == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected an error containing %q", c.wantErr)
			}
			joined := resp.Diagnostics.Errors()[0].Summary() + " " + resp.Diagnostics.Errors()[0].Detail()
			if !strings.Contains(joined, c.wantErr) {
				t.Errorf("diagnostic %q does not mention %q", joined, c.wantErr)
			}
		})
	}
}

// TestSettingsValidatorSkipsUnsetValues pins that an interpolated value still unknown at plan time is
// passed over rather than reported as malformed JSON.
func TestSettingsValidatorSkipsUnsetValues(t *testing.T) {
	for _, value := range []basetypes.StringValue{basetypes.NewStringNull(), basetypes.NewStringUnknown()} {
		var resp validator.StringResponse
		jsonObjectValidator().ValidateString(context.Background(),
			validator.StringRequest{Path: path.Root("settings_json"), ConfigValue: value}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("unexpected diagnostics for %v: %v", value, resp.Diagnostics)
		}
	}
}
