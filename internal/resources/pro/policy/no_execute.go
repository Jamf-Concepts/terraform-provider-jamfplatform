// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// The daily no-execute window (`general.date_time_limitations.no_execute_start`
// and `no_execute_end`) cannot be written to the classic policies API at face
// value. Jamf Pro parses the value into minutes since midnight and then stores
// it ONLY if that total is strictly greater than 1440 — a full day — wrapping
// it `mod 1440` on the way in. Every real time of day is 0…1439 minutes, so
// every real time is accepted with HTTP 201 and silently dropped, while an
// impossible one such as `41:00 AM` (2460 minutes) is stored, as 2460 mod 1440
// = 1020 = `5:00 PM`.
//
// The read side is unaffected: the GET returns an ordinary 12-hour time, and
// the admin UI sets the window happily because it bypasses this API for a
// servlet form post.
//
// So the window IS writable, by sending the desired time offset a full day
// forward. That is what encodeNoExecuteTime does, unconditionally: the
// behaviour is not a bisected regression window but — as far as anything here
// can tell — simply how the endpoint has always worked, exposed only now that
// this resource reads the wire instead of trusting prior state. There is
// therefore nothing to gate on, and a canary test rather than a version
// constant is what will tell us if that ever changes.
//
// Wire-probed against Jamf Pro 11.31.1 on 2026-09-07 across 7,801 renderings,
// then narrowed by an hour sweep (0…29) and a boundary sweep at 1440/1441:
//
//	24:00 AM = 1440 -> discarded        24:01 AM = 1441 -> stored as 12:01 AM
//	23:60 AM = 1440 -> discarded        12:00 PM =  720 -> discarded
//	48:00 AM        -> stored 12:00 AM  71:59 AM        -> stored 11:59 PM
//
// The mapping is deterministic and stable — the same input written three times
// yields the same stored value — and the hour field is not length-capped
// (`100:15 AM` is accepted). A bare 24-hour value with no meridiem is refused
// with HTTP 409, so the meridiem is mandatory whatever the hour. The window
// does not depend on no_execute_on: it persists with the day list empty.
//
// One consequence worth knowing: a sub-1440 value does not merely fail to
// store, it CLEARS whatever was stored, while an empty element is ignored. So
// sending a practitioner's time unencoded would erase a window set in the admin
// UI — which makes the encoding the safe option rather than merely the useful
// one.
//
// Tracked as Jamf PI-1661; if it is ever fixed,
// TestAccPolicyResource_NoExecuteWindowEncoding fails and this whole file goes
// away. None of this is surfaced to practitioners: the
// schema describes the attribute as the ordinary 12-hour time it accepts and
// reads back, because from a configuration's point of view that is exactly what
// it is. See TestAccPolicyResource_NoExecuteWindowEncoding for the tripwire.

// noExecuteWireTimePattern matches the 12-hour time the classic GET returns and
// the schema accepts: `h:MM AM` / `h:MM PM`, hour 1-12 with no leading zero.
// It is deliberately stricter than what the write path parses.
var noExecuteWireTimePattern = regexp.MustCompile(`^(1[0-2]|[1-9]):([0-5]\d) (AM|PM)$`)

// encodeNoExecuteTime rewrites a 12-hour no-execute time into the form Jamf Pro
// actually stores: the same time offset 48 hours forward, expressed with an AM
// meridiem. `5:00 PM` becomes `65:00 AM` — 65×60 = 3900 minutes, which clears
// the 1440 threshold and wraps to 1020, which is 17:00, which renders as
// `5:00 PM`.
//
// The offset is 48 hours rather than 24 because the threshold is strictly
// greater than 1440: a 24-hour offset of midnight lands exactly on 1440 and is
// discarded (`24:00 AM` stores nothing, while `24:01 AM` stores `12:01 AM`).
// 48 hours clears it for every minute of the day, midnight included.
//
// A value that does not match noExecuteWireTimePattern is returned unchanged.
// The schema validator makes that unreachable from configuration; it matters
// for a value a Read adopted from the wire, which is already in this shape, and
// it keeps the function total rather than panicking on a surprise.
func encodeNoExecuteTime(value string) string {
	m := noExecuteWireTimePattern.FindStringSubmatch(value)
	if m == nil {
		return value
	}
	hour, err := strconv.Atoi(m[1])
	if err != nil {
		return value
	}
	// 12 AM is hour 0 and 12 PM is hour 12; every other PM hour adds 12.
	switch {
	case m[3] == "AM" && hour == 12:
		hour = 0
	case m[3] == "PM" && hour != 12:
		hour += 12
	}
	return fmt.Sprintf("%d:%s AM", hour+48, m[2])
}

// noExecuteTimePointer returns the wire value to send for one end of the
// window, or nil to omit the element entirely.
//
// nil for a null or unknown attribute, and that omission is load-bearing: the
// element is destructive when present and ineffective when absent, so a policy
// whose window was set outside Terraform keeps it as long as the provider stays
// quiet. Omitting is also what a Computed-but-unset attribute must do on create.
func noExecuteTimePointer(value types.String) *string {
	if !helpers.IsConfiguredValue(value) {
		return nil
	}
	s := value.ValueString()
	if s == "" {
		return nil
	}
	encoded := encodeNoExecuteTime(s)
	return &encoded
}
