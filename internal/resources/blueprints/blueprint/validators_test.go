// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/jamf/terraform-provider-jamfplatform/internal/common/appleprofiles"
)

// payloadProblemCase pins one appleprofiles.ProblemKind to the diagnostic appendPayloadProblems
// raises for it: which attribute it is reported against, and whether it names the embedded snapshot
// and the raw_component escape hatch.
type payloadProblemCase struct {
	name          string
	payloadType   string
	settings      string
	kind          appleprofiles.ProblemKind
	target        path.Path
	summary       string
	namesSnapshot bool
}

// payloadProblemCases covers every ProblemKind appleprofiles can report for a legacy payload.
// Severity is the point of the table: each finding is an error because Jamf refuses or silently
// discards the write, and nothing else in this package fails if one is downgraded to a warning —
// an unrecognised key or payload type would become advisory again and the payload would apply as a
// no-op with a green plan. The snapshot column pins the other half: only a finding an older table
// could explain offers the escape hatch.
func payloadProblemCases() []payloadProblemCase {
	settingsPath := path.Root("legacy_payloads").AtListIndex(0).AtName("settings")
	typePath := path.Root("legacy_payloads").AtListIndex(0).AtName("payload_type")
	payloadPath := path.Root("legacy_payloads").AtListIndex(0)

	return []payloadProblemCase{
		{
			name:          "UnknownPayloadType",
			payloadType:   "com.example.notarealpayload",
			settings:      `{}`,
			kind:          appleprofiles.UnknownPayloadType,
			target:        typePath,
			summary:       "Unrecognised legacy payload type",
			namesSnapshot: true,
		},
		{
			name:          "MiscasedPayloadType",
			payloadType:   "com.apple.managedclient.preferences",
			settings:      `{}`,
			kind:          appleprofiles.MiscasedPayloadType,
			target:        typePath,
			summary:       "Unrecognised legacy payload type",
			namesSnapshot: false,
		},
		{
			name:          "UnknownKey",
			payloadType:   "com.apple.applicationaccess",
			settings:      `{"bogusKeyOneTwoThree":"x"}`,
			kind:          appleprofiles.UnknownKey,
			target:        settingsPath,
			summary:       "Legacy payload setting does not match Apple's schema",
			namesSnapshot: true,
		},
		{
			name:          "MiscasedKey",
			payloadType:   "com.apple.applicationaccess",
			settings:      `{"AllowCamera":true}`,
			kind:          appleprofiles.MiscasedKey,
			target:        settingsPath,
			summary:       "Legacy payload setting does not match Apple's schema",
			namesSnapshot: false,
		},
		{
			name:          "WrongType",
			payloadType:   "com.apple.applicationaccess",
			settings:      `{"allowScreenShot":"yes"}`,
			kind:          appleprofiles.WrongType,
			target:        settingsPath,
			summary:       "Legacy payload setting does not match Apple's schema",
			namesSnapshot: false,
		},
		{
			name:          "MissingRequiredKey",
			payloadType:   "com.apple.notificationsettings",
			settings:      `{}`,
			kind:          appleprofiles.MissingRequiredKey,
			target:        payloadPath,
			summary:       "Legacy payload is missing a required setting",
			namesSnapshot: true,
		},
		{
			name:          "IntegerOutOfRange",
			payloadType:   "com.apple.AssetCache.managed",
			settings:      `{"CacheLimit":2147483648}`,
			kind:          appleprofiles.IntegerOutOfRange,
			target:        settingsPath,
			summary:       "Legacy payload setting does not match Apple's schema",
			namesSnapshot: false,
		},
	}
}

func TestAppendPayloadProblems_EveryFindingIsAnError(t *testing.T) {
	for _, tc := range payloadProblemCases() {
		t.Run(tc.name, func(t *testing.T) {
			var decoded map[string]any
			if err := json.Unmarshal([]byte(tc.settings), &decoded); err != nil {
				t.Fatalf("failed to decode settings: %v", err)
			}

			if problems := appleprofiles.Validate(tc.payloadType, decoded); len(problems) != 1 || problems[0].Kind != tc.kind {
				t.Fatalf("fixture no longer produces exactly one %v: got %v", tc.kind, problems)
			}

			entry := path.Root("legacy_payloads").AtListIndex(0)
			var diags diag.Diagnostics
			appendPayloadProblems(&diags, tc.payloadType, decoded, entry, entry.AtName("payload_type"), entry.AtName("settings"))

			if len(diags.Warnings()) != 0 {
				t.Errorf("a finding Jamf refuses or silently discards must be an error, not a warning: %v", diags.Warnings())
			}
			if !diags.HasError() {
				t.Fatalf("expected an error diagnostic, got %v", diags)
			}
			if len(diags.Errors()) != 1 {
				t.Fatalf("expected exactly 1 error, got %v", diags.Errors())
			}

			d := diags.Errors()[0]
			if d.Summary() != tc.summary {
				t.Errorf("expected summary %q, got %q", tc.summary, d.Summary())
			}
			withPath, ok := d.(diag.DiagnosticWithPath)
			if !ok {
				t.Fatalf("expected an attribute error carrying a path, got %T", d)
			}
			if !withPath.Path().Equal(tc.target) {
				t.Errorf("expected the error against %s, got %s", tc.target, withPath.Path())
			}
		})
	}
}

func TestAppendPayloadProblems_SnapshotEscapeHatchIsPerBlock(t *testing.T) {
	for _, tc := range payloadProblemCases() {
		t.Run(tc.name, func(t *testing.T) {
			var decoded map[string]any
			if err := json.Unmarshal([]byte(tc.settings), &decoded); err != nil {
				t.Fatalf("failed to decode settings: %v", err)
			}

			entry := path.Root("legacy_payloads").AtListIndex(0)
			var diags diag.Diagnostics
			appendPayloadProblems(&diags, tc.payloadType, decoded, entry, entry.AtName("payload_type"), entry.AtName("settings"))

			if !diags.HasError() {
				t.Fatalf("expected an error diagnostic, got %v", diags)
			}
			detail := diags.Errors()[0].Detail()

			if tc.namesSnapshot {
				if !strings.Contains(detail, "apple/device-management") || !strings.Contains(detail, appleprofiles.ProvenanceSummary()) {
					t.Errorf("expected the snapshot named, got %q", detail)
				}
				if !strings.Contains(detail, "raw_component") {
					t.Errorf("expected the escape hatch offered, got %q", detail)
				}
				if !strings.Contains(detail, "every legacy payload in the same block") || !strings.Contains(detail, "com.jamf.ddm-configuration-profile") {
					t.Errorf("the escape hatch is per block, not per payload — the platform stores a block's payloads as one component; got %q", detail)
				}
				return
			}

			if strings.Contains(detail, "raw_component") || strings.Contains(detail, "apple/device-management") {
				t.Errorf("a finding the snapshot cannot explain must not offer the escape hatch, got %q", detail)
			}
		})
	}
}
