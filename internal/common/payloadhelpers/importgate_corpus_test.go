// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
)

// macOSFixtures / mobileFixtures are the checked-in profile corpora the two
// configuration-profile resources test against.
const (
	macOSFixtures  = "../../resources/pro/macos_configuration_profile/testdata"
	mobileFixtures = "../../resources/pro/mobile_device_configuration_profile/testdata"
)

// divergentPaths is the set of plist paths the comparison reports between two
// payloads, used to compare the gate's *prediction* against what a live Jamf Pro
// actually did.
func divergentPaths(t *testing.T, authored, stored []byte) map[string]struct{} {
	t.Helper()
	findings, ok := diffPayloadStrings(authored, stored)
	if !ok {
		t.Fatalf("fixture pair did not parse")
	}
	out := map[string]struct{}{}
	for _, f := range findings {
		out[f.path] = struct{}{}
	}
	return out
}

// predictedPaths is the same set, derived without touching the network by
// applying the storage-category transform.
func predictedPaths(t *testing.T, payload []byte, platform ProfilePlatform) map[string]struct{} {
	t.Helper()
	tree, _, err := plisthelpers.ParsePlist(payload)
	if err != nil {
		t.Fatalf("payload did not parse: %v", err)
	}
	out := map[string]struct{}{}
	for _, f := range diffPayloadTrees(tree, predictStoredTree(tree, platform, false)) {
		out[f.path] = struct{}{}
	}
	return out
}

// TestImportGate_AgreesWithRecordedServerResponses is the false-positive guard
// with real data behind it. Each fixture pair is an authored mobileconfig plus
// the response a live Jamf Pro returned for it, so the recorded response is
// ground truth for which values that tenant mangled.
//
// The assertion is one-sided on purpose: the gate must never flag a value the
// real server round-tripped (a false positive blocks a working import), while
// flagging fewer values than the server mangled is tolerated here — the
// post-write checks on create and update still catch those, and a gap shows up
// as the diagnostic below rather than a failure.
func TestImportGate_AgreesWithRecordedServerResponses(t *testing.T) {
	entries, err := os.ReadDir(macOSFixtures)
	if err != nil {
		t.Fatalf("reading fixtures: %v", err)
	}

	pairs := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".mobileconfig") {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), ".mobileconfig")
		for _, kind := range []string{"create_response", "update_response"} {
			respPath := filepath.Join(macOSFixtures, stem+"."+kind+".xml")
			resp, rErr := os.ReadFile(respPath)
			if rErr != nil {
				continue
			}
			authored, aErr := os.ReadFile(filepath.Join(macOSFixtures, e.Name()))
			if aErr != nil {
				t.Fatalf("reading %s: %v", e.Name(), aErr)
			}
			stored, sErr := ExtractServerPayloadFromGeneral(resp)
			if sErr != nil {
				t.Fatalf("extracting payload from %s: %v", respPath, sErr)
			}
			pairs++

			actual := divergentPaths(t, authored, stored)
			predicted := predictedPaths(t, authored, PlatformMacOS)

			var falsePositives []string
			for p := range predicted {
				if _, mangled := actual[p]; !mangled {
					falsePositives = append(falsePositives, p)
				}
			}
			sort.Strings(falsePositives)
			if len(falsePositives) > 0 {
				t.Errorf("%s (%s): gate would refuse values this tenant stored faithfully: %v",
					stem, kind, falsePositives)
			}

			var missed []string
			for p := range actual {
				if _, flagged := predicted[p]; !flagged {
					missed = append(missed, p)
				}
			}
			sort.Strings(missed)
			if len(missed) > 0 {
				t.Logf("%s (%s): server mangled values the gate does not predict (post-write checks still cover these): %v",
					stem, kind, missed)
			}
		}
	}

	if pairs == 0 {
		t.Fatal("no authored/response fixture pairs found — the false-positive guard is not actually running")
	}
	t.Logf("checked %d authored/recorded-response fixture pairs", pairs)
}

// TestImportGate_KnownFixtureVerdicts pins the verdict for each reserved-character
// and line-break fixture the repo already keeps. The interesting pair is
// reserved_character_corpus.mobileconfig, which appears in both testdata
// directories with the same reserved characters in a
// com.apple.ManagedClient.preferences payload: refused for a mobile device
// profile and accepted for a macOS one, because Jamf Pro re-renders that payload
// type on macOS and stores it verbatim on mobile. A single global table would
// have to get one of those two wrong.
func TestImportGate_KnownFixtureVerdicts(t *testing.T) {
	cases := []struct {
		dir       string
		file      string
		platform  ProfilePlatform
		wantLossy bool
		why       string
	}{
		{macOSFixtures, "login_window_ampersand.mobileconfig", PlatformMacOS, true,
			"the ampersand sits in com.apple.MCX / com.apple.screensaver, both stored verbatim"},
		{macOSFixtures, "setup_manager_ampersand.mobileconfig", PlatformMacOS, false,
			"the ampersand sits inside Application & Custom Settings, which macOS re-renders"},
		{macOSFixtures, "reserved_character_corpus.mobileconfig", PlatformMacOS, false,
			"reserved characters sit inside Application & Custom Settings, which macOS re-renders"},
		{mobileFixtures, "reserved_character_corpus.mobileconfig", PlatformMobileDevice, true,
			"the same payload type is stored verbatim in a mobile device profile"},
		{macOSFixtures, "mscp_consent_cr_refs.mobileconfig", PlatformMacOS, false,
			"a carriage-return reference is the recommended line break and survives verbatim storage"},
	}

	for _, tc := range cases {
		t.Run(tc.file+"/"+platformName(tc.platform), func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(tc.dir, tc.file))
			if err != nil {
				t.Skipf("fixture absent: %v", err)
			}
			detail, lossy, ok := PayloadImportRisk(b, tc.platform)
			if !ok {
				t.Fatalf("fixture did not parse")
			}
			switch {
			case tc.wantLossy && !lossy:
				t.Fatalf("expected the gate to refuse this fixture — %s", tc.why)
			case !tc.wantLossy && lossy:
				t.Fatalf("false positive — %s:\n%s", tc.why, detail)
			}
			if lossy && !strings.Contains(detail, "Nothing has been imported") {
				t.Errorf("detail is missing the import-phase remediation:\n%s", detail)
			}
		})
	}
}

func platformName(p ProfilePlatform) string {
	if p == PlatformMobileDevice {
		return "mobile"
	}
	return "macos"
}
