// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"fmt"
	"time"
	// Embed the IANA tz database so timezone validation is deterministic
	// regardless of whether the host (or Terraform Cloud runner) ships system
	// zoneinfo. Without this, a host missing zoneinfo would make
	// time.LoadLocation fail for EVERY zone and false-reject valid configs.
	_ "time/tzdata"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// IANATimeZone checks that a string attribute is a real IANA time-zone
// identifier, validating against Go's embedded IANA tz database
// (time.LoadLocation).
//
// Why this source: the Jamf Pro endpoints that accept a timezone validate with
// java ZoneId semantics. Wire-probed on the mobile device prestage endpoints
// and again on `/v1/teacher-app` (2026-06-10): the server accepts "UTC" and
// "Etc/UTC" even though both are ABSENT from the `/api/pro/v1/time-zones`
// UI-dropdown list (a curated 476-entry, 8-region subset — also missing "GMT"),
// and rejects "PST", garbage, and null. Gating on that list would false-reject
// values the server accepts, so it cannot serve as the authoritative gate. Go's
// embedded tzdata accepts all 476/476 list entries (verified programmatically)
// and matches the server on the rejects ("PST" and bogus zones fail
// time.LoadLocation) — zero false-rejects in either direction on everything
// probed. That is why tzdata, not the API list, is the gate.
//
// Caveat: Java accepts a handful of legacy aliases the IANA database omits
// (e.g. "PST"); those would be rejected here. The Jamf UI steers users to
// region IDs ("America/Los_Angeles"), so this is a narrow edge. LoadLocation
// also falsely accepts "" (→ UTC) and "Local" (→ host local zone); "Local" is
// rejected explicitly and empty is left to LengthAtLeast(1) for a clearer
// error. Null/unknown values are deferred per STYLE_GUIDE §Config-time
// validators.
func IANATimeZone() validator.String {
	return ianaTimeZoneValidator{}
}

type ianaTimeZoneValidator struct{}

// Description returns a plain-text description of the validator.
func (ianaTimeZoneValidator) Description(_ context.Context) string {
	return `value must be a valid IANA time-zone identifier (e.g. "America/Chicago", "UTC").`
}

// MarkdownDescription returns the markdown description of the validator.
func (v ianaTimeZoneValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (ianaTimeZoneValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	zone := req.ConfigValue.ValueString()
	if zone == "" {
		return // LengthAtLeast(1) reports empty with a clearer message.
	}
	if zone == "Local" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid time zone",
			`"Local" is a Go-specific alias for the host's zone, not a valid Jamf Pro time zone; supply a specific IANA identifier such as "America/Chicago" or "UTC".`,
		)
		return
	}
	if _, err := time.LoadLocation(zone); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid time zone",
			fmt.Sprintf("%q is not a valid IANA time-zone identifier. Use a value such as \"America/Chicago\" or \"UTC\".", zone),
		)
	}
}
