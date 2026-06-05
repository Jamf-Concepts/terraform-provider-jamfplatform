// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package licensed_software

import (
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// licenseTermPerpetual / licenseTermAnnual are the two values of the
// purchasing.license_term enum, which collapses the wire's mutually-exclusive
// is_perpetual / is_annual bool pair (server-enforced exactly-one).
const (
	licenseTermPerpetual = "perpetual"
	licenseTermAnnual    = "annual"
)

// extractLicensedSoftwareID returns the assigned ID as a string from a
// Create/GET response. Create returns the ID at the top level
// (<licensed_software><id>); GET echoes it inside <general>. Prefer the
// top-level reading, fall back to general.
func extractLicensedSoftwareID(ls *proclassic.LicensedSoftware) string {
	if ls == nil {
		return ""
	}
	if ls.ID != nil {
		return strconv.Itoa(*ls.ID)
	}
	if ls.General != nil && ls.General.ID != nil {
		return strconv.Itoa(*ls.General.ID)
	}
	return ""
}

// stringIDPtr parses a TF String holding a numeric ID into *int. Returns nil for
// null/unknown/empty/un-parseable.
func stringIDPtr(value types.String) *int {
	if !helpers.IsConfiguredValue(value) {
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

// stringFromIntPtr renders an *int as an *string for the preferCurrent helpers.
func stringFromIntPtr(p *int) *string {
	if p == nil {
		return nil
	}
	s := strconv.Itoa(*p)
	return &s
}

// emptyToNil collapses a wire empty string to a nil pointer. The classic
// /licensedsoftware endpoint echoes "" for every unset optional string, so an
// adopted-from-wire value must be nulled to match an unset (null) config.
func emptyToNil(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

// preferCurrentStringPointer returns the caller's configured value when set,
// otherwise adopts the API value with "" collapsed to null. Protects
// user-authored strings in positionally-reconciled nested lists against the
// classic-API empty-string echo: a null config and a "" echo both resolve to
// null, while an explicitly configured "" is preserved. The standard ProClassic
// tradeoff is no server-drift detection on these fields (see restricted_software).
func preferCurrentStringPointer(api *string, current types.String) types.String {
	if helpers.IsConfiguredValue(current) {
		return current
	}
	return helpers.StringPointerValueOrNull(emptyToNil(api))
}

// preferCurrentBoolPointer is the bool sibling of preferCurrentStringPointer.
func preferCurrentBoolPointer(api *bool, current types.Bool) types.Bool {
	if helpers.IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*api)
}

// int64ValueOrZero renders an *int as a known Int64, defaulting nil to 0. Used
// for license_count, where 0 is a meaningful value (unlimited) and the schema
// default is 0 — never null.
func int64ValueOrZero(p *int) types.Int64 {
	if p == nil {
		return types.Int64Value(0)
	}
	return types.Int64Value(int64(*p))
}

// int64ValueOrNullZero renders an *int as an Int64, collapsing nil AND 0 to
// null. Used for the unset-sentinel fields (life_expectancy, the *_epoch
// echoes) where the server returns 0 to mean "not set".
func int64ValueOrNullZero(p *int) types.Int64 {
	if p == nil || *p == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}

// stringValueOrNullEmpty renders an *string as a String, collapsing nil AND ""
// to null. Used for Computed-only echo strings (the *_utc siblings) that have no
// configured counterpart to prefer.
func stringValueOrNullEmpty(p *string) types.String {
	return helpers.StringPointerValueOrNull(emptyToNil(p))
}

// derefString returns the underlying string for a non-nil *string, or "" for nil.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// licensedSoftwareListItemName is the name accessor passed to filters.ApplyClassicFilter.
func licensedSoftwareListItemName(item proclassic.LicensedSoftwareAllItemLicensedSoftware) string {
	return derefString(item.Name)
}
