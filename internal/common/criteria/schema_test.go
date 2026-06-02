// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package criteria

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestCriterionAttributes_FieldShape(t *testing.T) {
	attrs := CriterionAttributes(Operators)

	for _, name := range []string{
		"priority", "name", "search_type", "value",
		"and_or", "has_opening_parenthesis", "has_closing_parenthesis",
	} {
		if _, ok := attrs[name]; !ok {
			t.Errorf("CriterionAttributes missing %q", name)
		}
	}
	if len(attrs) != 7 {
		t.Errorf("expected 7 criterion attributes, got %d", len(attrs))
	}

	// name / search_type / value are Required; priority / and_or / parens are
	// Optional+Computed (server is authoritative, defaults fill omitted fields).
	if got := attrs["name"].(schema.StringAttribute); !got.Required {
		t.Errorf("name must be Required")
	}
	if got := attrs["search_type"].(schema.StringAttribute); !got.Required {
		t.Errorf("search_type must be Required")
	}
	if got := attrs["value"].(schema.StringAttribute); !got.Required {
		t.Errorf("value must be Required")
	}
	priority := attrs["priority"].(schema.Int64Attribute)
	if !priority.Optional || !priority.Computed {
		t.Errorf("priority must be Optional+Computed")
	}
}

func TestCriterionAttributes_SearchTypeUsesGivenOperatorVocabulary(t *testing.T) {
	subset := Without("in less than x days", "in more than x days")
	attrs := CriterionAttributes(subset)

	searchType := attrs["search_type"].(schema.StringAttribute)
	if len(searchType.Validators) == 0 {
		t.Fatalf("search_type must carry an OneOf validator")
	}
	// The wired description reflects the passed (subset) vocabulary, not the
	// full set — proves the operators argument is threaded through.
	desc := searchType.MarkdownDescription
	if got, want := desc, Description(subset); got != want {
		t.Errorf("search_type description not derived from given operators\n got: %q\nwant: %q", got, want)
	}
}
