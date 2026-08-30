// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// jsonObjectType is a String type whose semantic equality compares two values as JSON rather than as
// bytes. It backs `settings_json`.
//
// Normalisation is not optional here. The platform stores the settings object verbatim, keys in the
// order they were first written, and re-serialises it on the way out; `jsonencode` emits keys sorted
// and unindented, and a policy authored from a file or from a configuration exported out of the admin
// UI is formatted however its author left it. Without comparing as JSON, any of those differences
// would drift on every refresh even though the two values describe the same settings.
//
// Semantic equality is the whole mechanism, deliberately. A plan modifier that also held the state
// value when the configuration was only reformatted was tried and removed: it stopped state ever
// adopting the new formatting, so the configuration and state stayed apart and every later plan
// proposed the same update again. Letting the apply write the author's formatting, and letting
// semantic equality absorb what the platform hands back, converges instead.
type jsonObjectType struct {
	basetypes.StringType
}

// Equal reports whether two types are equivalent. The framework uses this for type compatibility.
func (t jsonObjectType) Equal(o attr.Type) bool {
	other, ok := o.(jsonObjectType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

// String returns a human-readable representation of the type for error messages.
func (jsonObjectType) String() string {
	return "policy.jsonObjectType"
}

// ValueFromString promotes a basetypes.StringValue into a jsonObjectValue.
func (jsonObjectType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return jsonObjectValue{StringValue: in}, nil
}

// ValueFromTerraform converts a raw protocol value into the custom Value type.
func (t jsonObjectType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type from StringType.ValueFromTerraform: expected basetypes.StringValue, got %T", attrValue)
	}
	return jsonObjectValue{StringValue: stringValue}, nil
}

// ValueType reports the Value type this Type produces.
func (jsonObjectType) ValueType(_ context.Context) attr.Value {
	return jsonObjectValue{}
}

// jsonObjectValue wraps a basetypes.StringValue with JSON semantic equality.
type jsonObjectValue struct {
	basetypes.StringValue
}

// Type reports the Type associated with this Value.
func (jsonObjectValue) Type(_ context.Context) attr.Type {
	return jsonObjectType{}
}

// Equal returns byte-wise equality. Semantic equality lives in StringSemanticEquals.
func (v jsonObjectValue) Equal(o attr.Value) bool {
	other, ok := o.(jsonObjectValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals reports whether two values describe the same JSON. The framework calls it
// when reconciling a planned value against the applied one and when refining a plan against prior
// state, so returning true suppresses a difference that is only formatting.
//
// A value that will not parse is compared byte-wise rather than reported here: the attribute's
// validator already reports unparseable JSON against the right path, and a second diagnostic from
// semantic equality would say the same thing without a path.
func (v jsonObjectValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newValue, ok := newValuable.(jsonObjectValue)
	if !ok {
		diags.AddError("Semantic Equality Type Mismatch", fmt.Sprintf("expected %T, got %T", v, newValuable))
		return false, diags
	}
	if v.IsNull() != newValue.IsNull() || v.IsUnknown() != newValue.IsUnknown() {
		return false, diags
	}
	if v.IsNull() || v.IsUnknown() {
		return true, diags
	}

	left, leftErr := decodeJSON(v.ValueString())
	right, rightErr := decodeJSON(newValue.ValueString())
	if leftErr != nil || rightErr != nil {
		return v.ValueString() == newValue.ValueString(), diags
	}
	return sameJSON(left, right), diags
}

// newJSONObjectNull constructs a null jsonObjectValue.
func newJSONObjectNull() jsonObjectValue {
	return jsonObjectValue{StringValue: basetypes.NewStringNull()}
}

// newJSONObjectValue constructs a jsonObjectValue from a string.
func newJSONObjectValue(value string) jsonObjectValue {
	return jsonObjectValue{StringValue: basetypes.NewStringValue(value)}
}

// decodeJSON parses a settings string into the generic value shape the schema walker expects.
func decodeJSON(raw string) (any, error) {
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

// sameJSON compares two decoded JSON values structurally, treating numbers by value so 1 and 1.0
// agree.
func sameJSON(a, b any) bool {
	switch typedA := a.(type) {
	case map[string]any:
		typedB, ok := b.(map[string]any)
		if !ok || len(typedA) != len(typedB) {
			return false
		}
		for name, valueA := range typedA {
			valueB, present := typedB[name]
			if !present || !sameJSON(valueA, valueB) {
				return false
			}
		}
		return true
	case []any:
		typedB, ok := b.([]any)
		if !ok || len(typedA) != len(typedB) {
			return false
		}
		for i := range typedA {
			if !sameJSON(typedA[i], typedB[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// settingsValidator reports a settings value that is not a JSON object. The platform refuses
// anything else: a JSON array is rejected as "array found, object expected" and a null as
// "must not be null", both mid-apply, so both are caught at plan time instead.
type settingsValidator struct{}

// jsonObjectValidator returns the validator for the settings attribute.
func jsonObjectValidator() validator.String {
	return settingsValidator{}
}

func (v settingsValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (settingsValidator) MarkdownDescription(context.Context) string {
	return "must be a JSON object"
}

// ValidateString checks that the value parses as a JSON object.
func (settingsValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !helpers.IsConfiguredValue(req.ConfigValue) {
		return
	}

	decoded, err := decodeJSON(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Settings are not valid JSON",
			fmt.Sprintf("The settings could not be parsed as JSON: %s. Author this attribute with jsonencode({ ... }) or from a JSON file.", err))
		return
	}
	if _, ok := decoded.(map[string]any); !ok {
		resp.Diagnostics.AddAttributeError(req.Path, "Settings must be a JSON object",
			"The settings must be a JSON object of setting names to values. An array, string, number or null is rejected.")
	}
}
