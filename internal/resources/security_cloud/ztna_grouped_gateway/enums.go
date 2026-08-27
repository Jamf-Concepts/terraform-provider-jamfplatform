// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_grouped_gateway

import (
	"strconv"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
)

// recoveryDelaySeconds is the set of stability durations Jamf Security Cloud
// accepts, quoted from its own rejection message (wire-probed 2026-08-27):
// "Required gateway stability (recoveryDelayInSec) must be one of the supported
// durations in seconds: 300 (5 min), 1800 (30 min), 3600 (1 h), 10800 (3 h),
// 28800 (8 h)." The SDK documents the same set but exposes no generated helper
// for it, so unlike the string enums this one is restated here.
var recoveryDelaySeconds = []int64{300, 1800, 3600, 10800, 28800}

// routingStrategyValues returns the accepted routing strategies, from the SDK's
// generated enum helper so the validator and the documented list cannot drift.
//
// Validating at plan time is not cosmetic: an unknown strategy is rejected with
// `400 [INVALID_FIELD] Request body is missing or malformed.` — no field, no
// value (wire-probed 2026-08-27 with `ROUND_ROBIN`, a plausible guess given the
// UI's "Random" option).
func routingStrategyValues() []string {
	return securitycloud.RoutingStrategyValues()
}

// recoveryDelayValues returns the accepted stability durations.
func recoveryDelayValues() []int64 {
	out := make([]int64, len(recoveryDelaySeconds))
	copy(out, recoveryDelaySeconds)
	return out
}

// markdownIntList renders an integer value set as a comma-separated list of
// backticked literals, so a description and its validator are generated from one
// slice.
func markdownIntList(values []int64) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, "`"+strconv.FormatInt(v, 10)+"`")
	}
	return strings.Join(quoted, ", ")
}
