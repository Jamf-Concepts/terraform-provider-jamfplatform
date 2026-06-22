// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// RequireMinJamfProVersion compares the tenant's reported Jamf Pro version against the
// minimum required by a Pro resource. Empty required → no-op. Build suffixes like
// "11.5.0-t1700000000" are stripped before parsing. Returns an error diagnostic on
// mismatch or parse failure.
func RequireMinJamfProVersion(actual, required, resourceType string) diag.Diagnostics {
	var diags diag.Diagnostics
	if required == "" {
		return diags
	}

	actualParsed, err := parseSemverPrefix(actual)
	if err != nil {
		diags.AddError(
			"Unparseable Jamf Pro tenant version",
			fmt.Sprintf("%s requires Jamf Pro >= %s but the tenant reported %q which could not be parsed: %s", resourceType, required, actual, err),
		)
		return diags
	}

	requiredParsed, err := parseSemverPrefix(required)
	if err != nil {
		diags.AddError(
			"Invalid resource minimum Jamf Pro version",
			fmt.Sprintf("%s declared minJamfProVersion %q which could not be parsed: %s", resourceType, required, err),
		)
		return diags
	}

	if compareSemver(actualParsed, requiredParsed) < 0 {
		diags.AddError(
			"Jamf Pro tenant version below resource minimum",
			fmt.Sprintf("%s requires Jamf Pro >= %s; tenant reports %s.", resourceType, required, actual),
		)
	}
	return diags
}

// WarnIfBelowProviderFloor returns a warning diagnostic when the tenant version is below
// the provider-wide recommended floor. Returns nil-equivalent (no severity) when at/above.
func WarnIfBelowProviderFloor(actual, floor string) diag.Diagnostic {
	if floor == "" {
		return nil
	}

	actualParsed, err := parseSemverPrefix(actual)
	if err != nil {
		return diag.NewWarningDiagnostic(
			"Unparseable Jamf Pro tenant version",
			fmt.Sprintf("Provider recommends Jamf Pro >= %s but the tenant reported %q which could not be parsed: %s", floor, actual, err),
		)
	}

	floorParsed, err := parseSemverPrefix(floor)
	if err != nil {
		return nil
	}

	if compareSemver(actualParsed, floorParsed) < 0 {
		return diag.NewWarningDiagnostic(
			"Jamf Pro tenant older than provider build target",
			fmt.Sprintf(
				"This provider release was built against the Jamf Pro API as of version %s. The tenant reports %s. Some Pro resources may rely on endpoints or fields that did not exist in the tenant's version and could fail at apply time. Upgrade Jamf Pro or pin an older provider release.",
				floor, actual,
			),
		)
	}
	return nil
}

// AtLeastJamfProVersion reports whether actual is >= min (semver-prefix compare,
// build suffixes stripped). FAIL-OPEN: on either parse failure it returns true.
// Callers use this to gate version-specific WORKAROUNDS for behaviour that exists
// only at/after some Jamf Pro version; the provider's support floor is the modern
// version (ProviderMinJamfProVersion), so an unknown/unparseable tenant version is
// treated as modern and the workaround stays engaged.
func AtLeastJamfProVersion(actual, min string) bool {
	a, err := parseSemverPrefix(actual)
	if err != nil {
		return true
	}
	b, err := parseSemverPrefix(min)
	if err != nil {
		return true
	}
	return compareSemver(a, b) >= 0
}

type semver struct {
	major, minor, patch int
}

func parseSemverPrefix(v string) (semver, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return semver{}, fmt.Errorf("empty version")
	}
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("expected MAJOR.MINOR.PATCH, got %q", v)
	}
	out := semver{}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return semver{}, fmt.Errorf("segment %d (%q) is not numeric", i, p)
		}
		switch i {
		case 0:
			out.major = n
		case 1:
			out.minor = n
		case 2:
			out.patch = n
		}
	}
	return out, nil
}

func compareSemver(a, b semver) int {
	switch {
	case a.major != b.major:
		return a.major - b.major
	case a.minor != b.minor:
		return a.minor - b.minor
	case a.patch != b.patch:
		return a.patch - b.patch
	}
	return 0
}
