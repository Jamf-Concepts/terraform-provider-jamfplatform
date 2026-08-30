// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"strings"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/aischemas"
)

// TestRenderProblem pins the wording each finding produces, and that every one names the schema
// version it was checked against — a finding an operator cannot attribute to a schema version is a
// finding they cannot act on.
func TestRenderProblem(t *testing.T) {
	cases := []struct {
		name           string
		problem        aischemas.Problem
		wantSummary    string
		wantMentions   []string
		mentionsSchema bool
	}{
		{
			name:           "wrong type names the setting",
			problem:        aischemas.Problem{Kind: aischemas.WrongType, Path: "/verbose", Detail: "expected a boolean, found a string."},
			wantSummary:    "Setting does not match the tool's schema",
			wantMentions:   []string{"The setting at /verbose", "expected a boolean"},
			mentionsSchema: true,
		},
		{
			name:           "undeclared key points at a newer schema version",
			problem:        aischemas.Problem{Kind: aischemas.UnrecognisedKey, Path: "/nope", Detail: `"nope" is not declared.`},
			wantSummary:    "Setting is not declared for this schema version",
			wantMentions:   []string{"newer schema version"},
			mentionsSchema: true,
		},
		{
			name:         "missing required key",
			problem:      aischemas.Problem{Kind: aischemas.MissingRequiredKey, Path: "", Detail: `"tier" is required.`},
			wantSummary:  "Required setting is missing",
			wantMentions: []string{"The settings", `"tier" is required.`},
		},
		{
			name:           "a finding about the whole object falls back to naming the settings",
			problem:        aischemas.Problem{Kind: aischemas.UndeclaredKey, Path: "", Detail: "not accepted."},
			wantSummary:    "Setting does not match the tool's schema",
			wantMentions:   []string{"The settings:"},
			mentionsSchema: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			summary, detail := renderProblem(c.problem, "2026-08-14")
			if summary != c.wantSummary {
				t.Errorf("summary = %q, want %q", summary, c.wantSummary)
			}
			for _, mention := range c.wantMentions {
				if !strings.Contains(detail, mention) {
					t.Errorf("detail does not mention %q: %s", mention, detail)
				}
			}
			if c.mentionsSchema && !strings.Contains(detail, "2026-08-14") {
				t.Errorf("detail must name the schema version: %s", detail)
			}
		})
	}
}

// TestSortExpressionValidatorAcceptsOnlySupportedProperties pins the set the platform accepts:
// anything else is refused with VALIDATION_FAILED naming the sort parameter, mid-read.
func TestSortExpressionValidatorAcceptsOnlySupportedProperties(t *testing.T) {
	pattern := mustCompile("^(" + joinAlternatives(sortableProperties) + "):(asc|desc)$")

	for _, good := range []string{"name:asc", "name:desc", "createdAt:asc", "updatedAt:desc"} {
		if !pattern.MatchString(good) {
			t.Errorf("%q should be accepted", good)
		}
	}
	for _, bad := range []string{"name", "name:", ":asc", "bogus:asc", "name:sideways", "NAME:asc", "name:asc,extra"} {
		if pattern.MatchString(bad) {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
