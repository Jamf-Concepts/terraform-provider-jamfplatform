// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"
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

// StringPointerValueOrEmpty unwraps a *string to a Terraform string, returning an
// empty string (never null) when the pointer is nil. Use for a Required attribute
// the server may return absent or empty but which must round-trip as a present
// value — e.g. ssh_username on a DEP-created computer invitation, where the create
// endpoint requires the <ssh_username> element present but accepts it empty.
func StringPointerValueOrEmpty(value *string) types.String {
	if value == nil {
		return types.StringValue("")
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

// AlwaysEmitStringPointer converts a Terraform string into a *string for a
// classic XML payload whose PUT merges field by field and clears a field only
// when the element is present and empty (STYLE_GUIDE.md §Classic XML PUT merge
// where empty clears). Null yields a pointer to "" — encoded as
// `<element></element>`, which clears the stored value — where
// OptionalStringPointer would yield nil and omit the element, leaving the
// server's value in place and the plan's `-> null` unhonoured (issue #384).
// Unknown still yields nil: an attribute that is unknown at apply time is
// Computed, and the server owns it. Pair every use with
// ReconcileOptionalStringPointer in the state builder so the echoed "" folds
// back to null for an unconfigured attribute.
func AlwaysEmitStringPointer(value types.String) *string {
	if value.IsUnknown() {
		return nil
	}
	s := value.ValueString()
	return &s
}

// AlwaysEmitBoolPointer is the boolean form of AlwaysEmitStringPointer: null
// yields a pointer to false so the classic merge PUT carries
// `<element>false</element>` and a flag the user removed from config is turned
// off rather than retained (wire-probed on /networksegments
// override_buildings, 2026-09-06). Unknown yields nil for the same reason as
// the string form. Pair with ReconcileOptionalBoolPointer in the state builder.
func AlwaysEmitBoolPointer(value types.Bool) *bool {
	if value.IsUnknown() {
		return nil
	}
	b := value.ValueBool()
	return &b
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

// StringIDPtr parses a Terraform string holding a numeric ID into a *int for API
// payloads. Returns nil for null, unknown, empty, or un-parseable values so the
// SDK's omitempty tag drops the field from the wire. Jamf Pro exposes many
// referenced-object IDs as integers on the wire while Terraform carries them as
// strings; this performs the narrowing at the boundary.
func StringIDPtr(value types.String) *int {
	if !IsConfiguredValue(value) {
		return nil
	}
	s := value.ValueString()
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

// StringFromIntPtr converts a *int (the ProClassic SDK's integer wire shape) into
// a *string, returning nil for nil input. Use when round-tripping an integer wire
// ID through a string-typed helper such as PreferCurrentStringPointer.
func StringFromIntPtr(p *int) *string {
	if p == nil {
		return nil
	}
	s := strconv.Itoa(*p)
	return &s
}

// DerivedRefName maps a server-echoed reference name onto state for a Computed
// `*_name` field derived from a sibling reference id (site_name←site_id,
// category_name←category_id, etc.). The Jamf classic GET nondeterministically
// either echoes `<name>NONE</name>` or omits the name element entirely for the
// "none"/unassigned sentinel (id nil or <= 0), so the echoed name MUST NOT be
// trusted there: a sentinel id always yields a null name, making the derived
// field deterministic across reads. Trusting the echo lets the value flip
// between "NONE" and null between refreshes, tripping ImportStateVerify and
// "Provider produced inconsistent result after apply". For a real positive id
// the echoed name is authoritative. This is the derived-name analogue of the
// id sentinel-collapse rule — see STYLE_GUIDE.md §Server-derived computed fields
// & Optional+Computed attributes.
func DerivedRefName(id *int, name *string) types.String {
	if id == nil || *id <= 0 {
		return types.StringNull()
	}
	return StringPointerValueOrNull(name)
}

// Int64FromIntPtr converts a *int (the ProClassic SDK's integer wire shape) into a
// Terraform Int64, mapping nil to null.
func Int64FromIntPtr(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}

// PreferCurrentStringPointer returns the caller's configured value when it is set,
// otherwise adopts the API value. Use for Optional+Computed string scalars nested
// in a server-managed section where the classic API may echo a value the user did
// not author — unlike ReconcileOptionalStringPointer (which prefers the API value
// except for the explicit empty-string edge), this always preserves a configured
// current value.
func PreferCurrentStringPointer(api *string, current types.String) types.String {
	if IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.StringNull()
	}
	return types.StringValue(*api)
}

// PreferCurrentBoolPointer is the bool sibling of PreferCurrentStringPointer.
func PreferCurrentBoolPointer(api *bool, current types.Bool) types.Bool {
	if IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*api)
}

// ProviderNotConfiguredError returns the (summary, detail) pair every Pro CRUD
// handler uses to guard against a nil SDK client — the case where a CRUD method
// fires before Configure populated the provider data.
func ProviderNotConfiguredError() (string, string) {
	return "Provider not configured",
		"The Jamf Pro client was not configured before the CRUD operation fired. Verify the provider block, credentials, and that Configure ran without errors."
}

// InitialSingletonID returns the fixed Terraform state ID used by Pro singleton
// resources. See SingletonID for the rationale.
func InitialSingletonID() types.String {
	return types.StringValue(SingletonID)
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

// SetIdentity assigns the provided identity object when the target setter is
// available. Callers pass a framework field such as `resp.Identity`, which is a
// `*tfsdk.ResourceIdentity` and is nil whenever the resource declares no
// identity schema or the Terraform client does not support identity. A nil
// pointer of that type still satisfies IdentitySetter once it is boxed, so
// `target == nil` is false for exactly the case the guard exists to catch, and
// `Set` then dereferences nil and panics. isNilIdentitySetter looks through the
// interface instead.
func SetIdentity(ctx context.Context, target IdentitySetter, identity any) diag.Diagnostics {
	if isNilIdentitySetter(target) {
		return nil
	}
	return target.Set(ctx, identity)
}

// isNilIdentitySetter reports whether target holds nothing: either a nil
// interface, or a non-nil interface wrapping a nil pointer. Reflection covers
// every implementation rather than the two the provider happens to pass today,
// and SetIdentity runs once per CRUD call, so the cost does not matter.
func isNilIdentitySetter(target IdentitySetter) bool {
	if target == nil {
		return true
	}
	value := reflect.ValueOf(target)
	return value.Kind() == reflect.Pointer && value.IsNil()
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

// ReconcileOrAdoptBoolPointer applies the Optional+Computed reconcile rule for
// CRUD reads (adopt is false) and adopts the wire value verbatim for the list
// resource's config-generation path (adopt is true). The reconcile rule keeps
// an Optional+Computed bool null when the caller never authored it so a refresh
// does not snap it to the server default; config generation has no plan to
// reconcile against, so the server value is authoritative and a value-carrying
// flag (e.g. all_computers) must survive into the exported config.
func ReconcileOrAdoptBoolPointer(apiValue *bool, current types.Bool, adopt bool) types.Bool {
	if adopt {
		return BoolPointerValueOrNull(apiValue)
	}
	return ReconcileOptionalBoolPointer(apiValue, current)
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
// Used to detect missing permissions on cross-API bridging calls (e.g. resolving the Jamf Pro
// classic ID for a Platform Services device group when the integration lacks the Device groups
// Read permission).
func IsForbiddenError(err error) bool {
	if apiErr, ok := errors.AsType[*jamfplatform.APIResponseError](err); ok {
		return apiErr.HasStatus(http.StatusForbidden)
	}
	return false
}

// directoryGroupMatch bounds the retry-on-write that rides out a bootstrap apply.
const (
	directoryGroupMatchTimeout  = 90 * time.Second
	directoryGroupMatchInterval = 5 * time.Second
)

// IsDirectoryGroupMatchConflict reports whether err is the classic scope endpoints'
// rejection of a directory-service user-group name it cannot match against the
// configured directory ("Problem matching limitation user group"). During a
// bootstrap apply this is transient — the referenced LDAP / cloud-IdP directory is
// still being created in the same apply, or its bind has not finished coming up — so
// the write is retried (see RetryOnDirectoryGroupMatchConflict). The phrase is
// distinctive; match on it within any Jamf API error.
func IsDirectoryGroupMatchConflict(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := errors.AsType[*jamfplatform.APIResponseError](err); !ok {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "matching limitation user group")
}

// RetryOnDirectoryGroupMatchConflict calls write and, while it returns the classic
// "Problem matching limitation user group" rejection, retries it for a bounded
// window. This makes a from-scratch single apply work: a group-scoped resource and
// the jamfplatform_pro_ldap_server it references (by group NAME, which creates no
// dependency) are created in the same apply, so the scope write can land before the
// directory is queryable. Any other error — or a persistent conflict past the
// timeout (a genuinely wrong group name) — is returned unchanged so it surfaces
// loudly. Honours ctx cancellation. Use the err-only variant for Update writes.
func RetryOnDirectoryGroupMatchConflict(ctx context.Context, write func() error) error {
	deadline := time.Now().Add(directoryGroupMatchTimeout)
	for {
		err := write()
		if err == nil || !IsDirectoryGroupMatchConflict(err) {
			return err
		}
		if time.Now().After(deadline) {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(directoryGroupMatchInterval):
		}
	}
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
