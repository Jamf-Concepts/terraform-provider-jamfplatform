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

// OptionalStringPointer converts a Terraform string into a *string for API payloads.
// Returns nil for both Null and Unknown values — the Null/Unknown distinction matters
// for Optional+Computed attributes, where the framework reports Unknown until the
// server-derived value is known. types.String.ValueStringPointer returns a pointer to
// "" for Unknown, which Jamf Pro endpoints often reject with HTTP 500. Prefer this
// helper for every optional payload field; see STYLE_GUIDE.md §Server-derived
// computed fields & Optional+Computed attributes for the broader pattern.
func OptionalStringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	s := value.ValueString()
	return &s
}

// BoolPointerValueOrNull safely unwraps a *bool and converts it to a Terraform bool.
func BoolPointerValueOrNull(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

// OptionalBoolPointer converts a Terraform Bool into a *bool for API payloads.
// Mirrors OptionalStringPointer's contract for booleans: returns nil for both
// Null and Unknown values so the SDK's omitempty tag drops the field from the
// wire. Use this instead of types.Bool.ValueBoolPointer for Optional and
// Optional+Computed bool attributes — the latter returns a pointer to false
// for Unknown values, which Jamf Pro endpoints commonly treat as an explicit
// "set this to false" instead of "leave unchanged."
func OptionalBoolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	b := value.ValueBool()
	return &b
}

// OptionalInt64Pointer converts a Terraform Int64 into a *int for API payloads.
// Returns nil for both Null and Unknown values so the SDK's omitempty tag
// drops the field from the wire. The Jamf ProClassic SDK uses *int for
// integer wire fields (e.g. Priority, CachedCredentials); Terraform exposes
// Int64 so the narrowing happens here at the boundary.
func OptionalInt64Pointer(value types.Int64) *int {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	i := int(value.ValueInt64())
	return &i
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

// DerefString safely unwraps a *string, returning an empty string if nil.
func DerefString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// ReconcileOptionalBoolPointer behaves like ReconcileOptionalBool but accepts a *bool API value.
func ReconcileOptionalBoolPointer(apiValue *bool, current types.Bool) types.Bool {
	if apiValue == nil {
		return types.BoolNull()
	}
	return ReconcileOptionalBool(*apiValue, current)
}

// ReconcileOptionalStringPointer behaves like ReconcileOptionalString but accepts a *string API value.
func ReconcileOptionalStringPointer(apiValue *string, current types.String) types.String {
	return ReconcileOptionalString(DerefString(apiValue), current)
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

// PreserveStringWhenWireEmpty returns the server value when non-empty; otherwise
// keeps whatever was already in state. Use for user-controlled string fields
// where the Jamf Pro Classic API has been observed to echo an empty value after
// a successful write — `ReconcileOptionalStringPointer` would collapse that to
// Null and trip Terraform Core's "Provider produced inconsistent result after
// apply" check when plan holds a non-empty user-authored string. The error path
// is masked to the nearest non-sensitive parent block when the block contains a
// Sensitive sibling (e.g. `.self_service` instead of `.self_service.self_service_description`),
// which makes the root cause hard to spot — prefer this helper for any
// user-authored string under a block that also carries a Sensitive attribute.
// Observed at: macos / mobile_device configuration profile self-service blocks.
func PreserveStringWhenWireEmpty(wire *string, current types.String) types.String {
	if wire != nil && *wire != "" {
		return types.StringValue(*wire)
	}
	return current
}

// IsNotFoundError reports whether an error represents a "resource is gone"
// response from the Jamf API. Matches HTTP 404 (the conventional shape) AND
// HTTP 400 with an `INVALID_ID` error detail.
//
// Some Pro v1 endpoints — confirmed for `/device-enrollments/{id}` and
// `/volume-purchasing-locations/{id}` — return `400 Bad Request` with an
// `errorCode: INVALID_ID` detail (e.g. `Device Enrollment Instance with id
// 84 does not exist`) when the supplied ID does not exist, instead of the
// conventional `404 Not Found`. Treating both as "not found" keeps Read
// drift detection and acceptance-test CheckDestroy verification correct
// across the whole Pro v1 surface without per-resource overrides.
//
// The `Code == "INVALID_ID"` match is narrow enough to exclude legitimate
// schema-validation 400s (which use codes like `REQUIRED_FIELD`,
// `INVALID_PATTERN`, etc.) — only the documented "this ID does not exist"
// response flips to true.
func IsNotFoundError(err error) bool {
	apiErr, ok := errors.AsType[*jamfplatform.APIResponseError](err)
	if !ok {
		return false
	}
	if apiErr.HasStatus(http.StatusNotFound) {
		return true
	}
	if apiErr.HasStatus(http.StatusBadRequest) {
		for _, d := range apiErr.Details() {
			if d.Code == "INVALID_ID" {
				return true
			}
		}
	}
	return false
}

// IsClientError reports whether err is an *APIResponseError carrying a 4xx
// status (i.e. the server received and responded to the request with a client
// error). Used to distinguish an accepted-but-misleading 4xx — e.g. the classic
// /ebooks DELETE that returns HTTP 400 on an accepted async delete — from a
// transport error or a 5xx that should surface as a genuine failure.
func IsClientError(err error) bool {
	apiErr, ok := errors.AsType[*jamfplatform.APIResponseError](err)
	if !ok {
		return false
	}
	return apiErr.StatusCode >= 400 && apiErr.StatusCode < 500
}

// IsServerError reports whether an error represents a 500/internal server error from the Jamf API.
func IsServerError(err error) bool {
	if apiErr, ok := errors.AsType[*jamfplatform.APIResponseError](err); ok {
		return apiErr.HasStatus(http.StatusInternalServerError)
	}
	return false
}

// IsForbiddenError reports whether an error represents a 403/forbidden response from the Jamf API.
// Used to detect missing privileges on cross-API bridging calls (e.g. resolving the Jamf Pro
// classic ID for a Platform Services device group when the client lacks "Read Groups").
func IsForbiddenError(err error) bool {
	if apiErr, ok := errors.AsType[*jamfplatform.APIResponseError](err); ok {
		return apiErr.HasStatus(http.StatusForbidden)
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
