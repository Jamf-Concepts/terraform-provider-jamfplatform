// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package accrequire

import (
	"slices"
	"testing"
)

// AccEnv is the only thing standing between a stale local .env and a suite that
// self-skips green, and it is the half of the rename that CI can never exercise:
// a GitHub secret is not an ambient variable, so the fallback branch is taken on
// developer machines and nowhere else. Untested, the one code path that only
// runs where nobody is watching.
//
// The mutation these were written against: inverting the `!ok` guard at the map
// lookup, which makes every MAPPED name return "" and sends only unmapped names
// to the legacy lookup. It survived the whole suite.

func TestAccEnvPrefersTheCurrentNameAndFallsBackToTheLegacyOne(t *testing.T) {
	// Not parallel: every case sets process environment through t.Setenv.
	const (
		current = "JAMFPLATFORM_ACC_PRO_VPP_TOKEN"
		legacy  = "JAMFPLATFORM_VPP_TOKEN"
	)
	if accLegacyEnvNames[current] != legacy {
		t.Fatalf("this test is pinned to the %s → %s mapping, which the map no longer declares (it says %q); repoint the test rather than deleting it", current, legacy, accLegacyEnvNames[current])
	}

	t.Run("current name only", func(t *testing.T) {
		t.Setenv(current, "new-value")
		t.Setenv(legacy, "")
		if got := AccEnv(current); got != "new-value" {
			t.Errorf("AccEnv(%s) = %q, want the current name's value", current, got)
		}
	})

	// The migration path, and the case the inverted-guard mutation breaks: a
	// contributor whose .env predates the rename must keep working.
	t.Run("legacy name only", func(t *testing.T) {
		t.Setenv(current, "")
		t.Setenv(legacy, "old-value")
		if got := AccEnv(current); got != "old-value" {
			t.Errorf("AccEnv(%s) = %q, want the legacy %s value — the dual-read is what keeps a pre-rename .env working, and without it the fixture silently drops and its tests self-skip green", current, got, legacy)
		}
	})

	t.Run("current name wins when both are set", func(t *testing.T) {
		t.Setenv(current, "new-value")
		t.Setenv(legacy, "old-value")
		if got := AccEnv(current); got != "new-value" {
			t.Errorf("AccEnv(%s) = %q, want the current name to win; deferring to the legacy value would make a completed rename un-completable", current, got)
		}
	})

	t.Run("neither set", func(t *testing.T) {
		t.Setenv(current, "")
		t.Setenv(legacy, "")
		if got := AccEnv(current); got != "" {
			t.Errorf("AccEnv(%s) = %q, want empty", current, got)
		}
	})

	// A name with no legacy spelling must read itself and not panic on the
	// missing map entry. JAMFPLATFORM_ACC_REQUIRE is the live example: AccEnv
	// reads it and it has no pre-rename name.
	t.Run("unmapped name reads itself", func(t *testing.T) {
		const unmapped = "JAMFPLATFORM_ACC_REQUIRE"
		if _, mapped := accLegacyEnvNames[unmapped]; mapped {
			t.Fatalf("%s has acquired a legacy mapping, so it is no longer the unmapped case this test covers", unmapped)
		}
		t.Setenv(unmapped, "platform")
		if got := AccEnv(unmapped); got != "platform" {
			t.Errorf("AccEnv(%s) = %q, want %q", unmapped, got, "platform")
		}
	})
}

// TestLegacyEnvNamesAreUnambiguous is the integrity assertion the map needs at
// entry 48 rather than at entry 47: the shape is correct today, and this is what
// keeps it correct.
//
// Two legacy names colliding is the interesting failure. Go rejects duplicate
// KEYS in a map literal at compile time, so the current names are safe by
// construction, but nothing stops two current names claiming one legacy name —
// and if that happened, one fixture would silently read the other's value, which
// is worse than reading nothing.
func TestLegacyEnvNamesAreUnambiguous(t *testing.T) {
	t.Parallel()

	byLegacy := map[string][]string{}
	for current, legacy := range accLegacyEnvNames {
		if current == "" || legacy == "" {
			t.Errorf("accLegacyEnvNames has an empty name in the pair %q → %q", current, legacy)
			continue
		}
		if current == legacy {
			t.Errorf("%s maps to itself, so the entry does nothing; drop it", current)
		}
		byLegacy[legacy] = append(byLegacy[legacy], current)
	}

	for legacy, currents := range byLegacy {
		if len(currents) > 1 {
			slices.Sort(currents)
			t.Errorf("legacy name %s is claimed by %v — one of those fixtures would silently read the other's value; give each its own pre-rename name or drop the duplicate", legacy, currents)
		}
	}

	// A legacy name that is also a current name would make AccEnv's fallback
	// resolve to a variable the rename is trying to retire, so the old name
	// would keep working forever and the deprecation log would never fire for
	// it.
	for _, legacy := range accLegacyEnvNames {
		if _, isCurrent := accLegacyEnvNames[legacy]; isCurrent {
			t.Errorf("%s appears as both a current name and a legacy name, so the rename chains through itself", legacy)
		}
	}
}
