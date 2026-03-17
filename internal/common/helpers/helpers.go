// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	resourcetimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringValueOrNull converts a Go string into a Terraform string attribute, preserving null when empty.
func StringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

// StringPointerValueOrNull safely unwraps a *string and converts it to a Terraform string.
func StringPointerValueOrNull(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// BoolPointerValueOrNull safely unwraps a *bool and converts it to a Terraform bool.
func BoolPointerValueOrNull(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

// IdentitySetter matches the identity setter interface exposed by Terraform resources and list results.
type IdentitySetter interface {
	Set(context.Context, any) diag.Diagnostics
}

// SetIdentity assigns the provided identity object when the target setter is available.
func SetIdentity(ctx context.Context, target IdentitySetter, identity any) diag.Diagnostics {
	if target == nil {
		return nil
	}
	return target.Set(ctx, identity)
}

// SetToStringSlice converts a Terraform set of strings into a Go slice, preserving diagnostics.
func SetToStringSlice(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	var values []string
	diags := set.ElementsAs(ctx, &values, false)
	return values, diags
}

// ResolveTimeout returns either the configured timeout or the provided default when unset.
// The resolver argument should be a method such as timeouts.Value.Read/Create/Update/Delete.
func ResolveTimeout(
	ctx context.Context,
	isNull bool,
	isUnknown bool,
	defaultDuration time.Duration,
	resolver func(context.Context, time.Duration) (time.Duration, diag.Diagnostics),
) (time.Duration, diag.Diagnostics) {
	if isNull || isUnknown || resolver == nil {
		return defaultDuration, nil
	}
	return resolver(ctx, defaultDuration)
}

// IsConfiguredValue reports whether Terraform has a non-null, non-unknown value.
func IsConfiguredValue(value interface {
	IsNull() bool
	IsUnknown() bool
}) bool {
	return !value.IsNull() && !value.IsUnknown()
}

// StringPointerValue returns a *string for configured Terraform strings, preserving empty strings.
func StringPointerValue(v types.String) *string {
	if !IsConfiguredValue(v) {
		return nil
	}
	return new(v.ValueString())
}

// BoolPointerValue returns a *bool for configured Terraform bools, preserving nulls when unset.
func BoolPointerValue(v types.Bool) *bool {
	if !IsConfiguredValue(v) {
		return nil
	}
	return new(v.ValueBool())
}

// ReconcileOptionalBool keeps the current value if not managed, otherwise sets to the API value.
func ReconcileOptionalBool(apiValue bool, current types.Bool) types.Bool {
	if IsConfiguredValue(current) {
		return types.BoolValue(apiValue)
	}
	return types.BoolNull()
}

// ReconcileOptionalInt keeps the current value if not managed, otherwise sets to the API value.
func ReconcileOptionalInt(apiValue int, current types.Int64) types.Int64 {
	if IsConfiguredValue(current) {
		return types.Int64Value(int64(apiValue))
	}
	return types.Int64Null()
}

// ReconcileOptionalString keeps explicit empty strings set by the user while allowing nulls when unset.
func ReconcileOptionalString(apiValue string, current types.String) types.String {
	if apiValue == "" {
		if IsConfiguredValue(current) && current.ValueString() == "" {
			return current
		}
		return types.StringNull()
	}

	return types.StringValue(apiValue)
}

// IsNotFoundError reports whether an error represents a 404/not found response from the Jamf API.
func IsNotFoundError(err error) bool {
	if apiErr, ok := errors.AsType[*jamfplatform.APIResponseError](err); ok {
		return apiErr.HasStatus(http.StatusNotFound)
	}
	return false
}

// IsServerError reports whether an error represents a 500/internal server error from the Jamf API.
func IsServerError(err error) bool {
	if apiErr, ok := errors.AsType[*jamfplatform.APIResponseError](err); ok {
		return apiErr.HasStatus(http.StatusInternalServerError)
	}
	return false
}

// EnsureResourceTimeouts guarantees the timeout object has the expected shape for resource timeouts.
func EnsureResourceTimeouts(value resourcetimeouts.Value, attrTypes map[string]attr.Type) resourcetimeouts.Value {
	if value.IsNull() && !value.IsUnknown() {
		value.Object = types.ObjectNull(attrTypes)
	}
	return value
}

// NewResourceTimeoutsNullValue builds a null timeout object with the provided attribute types.
func NewResourceTimeoutsNullValue(attrTypes map[string]attr.Type) resourcetimeouts.Value {
	return EnsureResourceTimeouts(resourcetimeouts.Value{}, attrTypes)
}

// NormalizedFilterString trims whitespace and reports whether a string attribute is usable as a filter.
func NormalizedFilterString(value types.String) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}

	trimmed := strings.TrimSpace(value.ValueString())
	if trimmed == "" {
		return "", false
	}

	return trimmed, true
}
