// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// JSONToTerraformDynamic converts an arbitrary Go value (from json.Unmarshal) to a types.Dynamic.
func JSONToTerraformDynamic(v any) (types.Dynamic, error) {
	val, err := jsonToAttrValue(v)
	if err != nil {
		return types.DynamicNull(), err
	}
	return types.DynamicValue(val), nil
}

// TerraformDynamicToJSON converts a types.Dynamic value to a Go value suitable for json.Marshal.
func TerraformDynamicToJSON(d types.Dynamic) (any, error) {
	if d.IsNull() || d.IsUnknown() {
		return nil, nil
	}
	return attrValueToJSON(d.UnderlyingValue())
}

// jsonToAttrValue converts a Go value to a Terraform attr.Value.
func jsonToAttrValue(v any) (attr.Value, error) {
	switch val := v.(type) {
	case nil:
		return types.StringNull(), nil
	case bool:
		return types.BoolValue(val), nil
	case float64:
		return types.NumberValue(big.NewFloat(val)), nil
	case string:
		return types.StringValue(val), nil
	case []any:
		return jsonArrayToTuple(val)
	case map[string]any:
		return jsonObjectToTerraformObject(val)
	default:
		return nil, fmt.Errorf("unsupported JSON type: %T", v)
	}
}

// jsonArrayToTuple converts a JSON array to a Terraform tuple value.
func jsonArrayToTuple(arr []any) (basetypes.TupleValue, error) {
	if len(arr) == 0 {
		return types.TupleValueMust([]attr.Type{}, []attr.Value{}), nil
	}

	elemTypes := make([]attr.Type, len(arr))
	elemValues := make([]attr.Value, len(arr))

	for i, item := range arr {
		val, err := jsonToAttrValue(item)
		if err != nil {
			return types.TupleUnknown(nil), fmt.Errorf("array index %d: %w", i, err)
		}
		elemTypes[i] = val.Type(nil)
		elemValues[i] = val
	}

	return types.TupleValueMust(elemTypes, elemValues), nil
}

// jsonObjectToTerraformObject converts a JSON object to a Terraform object value.
func jsonObjectToTerraformObject(obj map[string]any) (basetypes.ObjectValue, error) {
	attrTypes := make(map[string]attr.Type, len(obj))
	attrValues := make(map[string]attr.Value, len(obj))

	for key, val := range obj {
		tfVal, err := jsonToAttrValue(val)
		if err != nil {
			return types.ObjectUnknown(nil), fmt.Errorf("key %q: %w", key, err)
		}
		attrTypes[key] = tfVal.Type(nil)
		attrValues[key] = tfVal
	}

	return types.ObjectValueMust(attrTypes, attrValues), nil
}

// attrValueToJSON converts a Terraform attr.Value to a Go value for JSON marshaling.
func attrValueToJSON(v attr.Value) (any, error) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}

	switch val := v.(type) {
	case basetypes.BoolValue:
		return val.ValueBool(), nil
	case basetypes.StringValue:
		return val.ValueString(), nil
	case basetypes.NumberValue:
		f, _ := val.ValueBigFloat().Float64()
		return f, nil
	case basetypes.Int64Value:
		return float64(val.ValueInt64()), nil
	case basetypes.Float64Value:
		return val.ValueFloat64(), nil
	case basetypes.TupleValue:
		return tupleToJSONArray(val)
	case basetypes.ListValue:
		return listToJSONArray(val)
	case basetypes.SetValue:
		return setToJSONArray(val)
	case basetypes.ObjectValue:
		return objectToJSONMap(val)
	case basetypes.MapValue:
		return mapToJSONMap(val)
	case basetypes.DynamicValue:
		return attrValueToJSON(val.UnderlyingValue())
	default:
		return nil, fmt.Errorf("unsupported Terraform type: %T", v)
	}
}

// tupleToJSONArray converts a Terraform tuple to a JSON array.
func tupleToJSONArray(t basetypes.TupleValue) ([]any, error) {
	elements := t.Elements()
	result := make([]any, len(elements))
	for i, elem := range elements {
		val, err := attrValueToJSON(elem)
		if err != nil {
			return nil, fmt.Errorf("tuple index %d: %w", i, err)
		}
		result[i] = val
	}
	return result, nil
}

// listToJSONArray converts a Terraform list to a JSON array.
func listToJSONArray(l basetypes.ListValue) ([]any, error) {
	elements := l.Elements()
	result := make([]any, len(elements))
	for i, elem := range elements {
		val, err := attrValueToJSON(elem)
		if err != nil {
			return nil, fmt.Errorf("list index %d: %w", i, err)
		}
		result[i] = val
	}
	return result, nil
}

// setToJSONArray converts a Terraform set to a JSON array.
func setToJSONArray(s basetypes.SetValue) ([]any, error) {
	elements := s.Elements()
	result := make([]any, len(elements))
	for i, elem := range elements {
		val, err := attrValueToJSON(elem)
		if err != nil {
			return nil, fmt.Errorf("set index %d: %w", i, err)
		}
		result[i] = val
	}
	return result, nil
}

// objectToJSONMap converts a Terraform object to a JSON map.
func objectToJSONMap(o basetypes.ObjectValue) (map[string]any, error) {
	attrs := o.Attributes()
	result := make(map[string]any, len(attrs))
	for key, val := range attrs {
		jsonVal, err := attrValueToJSON(val)
		if err != nil {
			return nil, fmt.Errorf("object key %q: %w", key, err)
		}
		result[key] = jsonVal
	}
	return result, nil
}

// mapToJSONMap converts a Terraform map to a JSON map.
func mapToJSONMap(m basetypes.MapValue) (map[string]any, error) {
	elements := m.Elements()
	result := make(map[string]any, len(elements))
	for key, val := range elements {
		jsonVal, err := attrValueToJSON(val)
		if err != nil {
			return nil, fmt.Errorf("map key %q: %w", key, err)
		}
		result[key] = jsonVal
	}
	return result, nil
}
