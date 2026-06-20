// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// trimmedStringType is a custom String type whose semantic equality treats
// two values as equal when they differ only by trailing whitespace. Backs
// the `ppd_contents` schema attribute: the Jamf Pro server strips trailing
// whitespace from this field on every round-trip, so without semantic
// equality every `terraform plan` for a printer configured via
// `file("...")` would show drift (file always carries a trailing newline)
// and every `terraform apply` would surface "Provider produced inconsistent
// result after apply" because the applied value differs from the planned
// value.
//
// Pairs with `Optional: true, Computed: true` on the attribute so the
// framework is permitted to surface a computed value that differs from the
// user's config. Mirrors the JSON-policy pattern used by the AWS provider
// for IAM policies, where the wire format differs from the user's input
// format and semantic equality is the framework-blessed reconciliation.
type trimmedStringType struct {
	basetypes.StringType
}

// Equal returns whether the two String types are equivalent. The framework
// uses this when checking attribute type compatibility.
func (t trimmedStringType) Equal(o attr.Type) bool {
	other, ok := o.(trimmedStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

// String returns a human-readable representation of the type, used in error
// messages and debugging.
func (trimmedStringType) String() string {
	return "printer.trimmedStringType"
}

// ValueFromString promotes a basetypes.StringValue into a trimmedStringValue.
func (trimmedStringType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return trimmedStringValue{StringValue: in}, nil
}

// ValueFromTerraform converts a raw Terraform protocol value into the
// custom Value type. The framework calls this when reading state or
// receiving values from Terraform Core.
func (t trimmedStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type from StringType.ValueFromTerraform: expected basetypes.StringValue, got %T", attrValue)
	}
	return trimmedStringValue{StringValue: stringValue}, nil
}

// ValueType reports the underlying Value type produced by this Type.
func (trimmedStringType) ValueType(_ context.Context) attr.Value {
	return trimmedStringValue{}
}

// trimmedStringValue wraps a basetypes.StringValue with trailing-whitespace
// semantic equality.
type trimmedStringValue struct {
	basetypes.StringValue
}

// Type reports the Type associated with this Value.
func (trimmedStringValue) Type(_ context.Context) attr.Type {
	return trimmedStringType{}
}

// Equal returns byte-wise equality (used by the framework for raw value
// comparison; semantic equality lives in StringSemanticEquals).
func (v trimmedStringValue) Equal(o attr.Value) bool {
	other, ok := o.(trimmedStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals reports whether two trimmedStringValues are
// semantically equivalent. The framework calls this when reconciling
// planned vs applied values and when refining plans against prior state;
// returning true suppresses the difference and avoids spurious drift.
//
// Two non-null, non-unknown values are semantically equal when their
// trailing-whitespace-stripped forms match. Null and unknown compare
// strictly by null/unknown flag.
func (v trimmedStringValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newValue, ok := newValuable.(trimmedStringValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Type Mismatch",
			fmt.Sprintf("expected %T, got %T", v, newValuable),
		)
		return false, diags
	}
	if v.IsNull() != newValue.IsNull() {
		return false, diags
	}
	if v.IsUnknown() != newValue.IsUnknown() {
		return false, diags
	}
	if v.IsNull() || v.IsUnknown() {
		return true, diags
	}
	return strings.TrimRight(v.ValueString(), " \t\r\n") == strings.TrimRight(newValue.ValueString(), " \t\r\n"), diags
}

// newTrimmedStringNull constructs a null trimmedStringValue.
func newTrimmedStringNull() trimmedStringValue {
	return trimmedStringValue{StringValue: basetypes.NewStringNull()}
}

// newTrimmedStringValue constructs a non-null trimmedStringValue.
func newTrimmedStringValue(s string) trimmedStringValue {
	return trimmedStringValue{StringValue: basetypes.NewStringValue(s)}
}

// trimmedStringValueFromPtr converts an SDK `*string` into a
// trimmedStringValue, mapping nil to a null value. Used by the state
// builders so the model field stays the custom type that the schema
// declares.
func trimmedStringValueFromPtr(p *string) trimmedStringValue {
	if p == nil {
		return newTrimmedStringNull()
	}
	return newTrimmedStringValue(*p)
}

// Compile-time interface assertions.
var (
	_ basetypes.StringTypable                    = trimmedStringType{}
	_ basetypes.StringValuable                   = trimmedStringValue{}
	_ basetypes.StringValuableWithSemanticEquals = trimmedStringValue{}
)
