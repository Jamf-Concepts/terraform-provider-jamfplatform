// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIDSetAttribute_DefaultShape(t *testing.T) {
	attr := IDSetAttribute("computer")
	if !attr.Optional {
		t.Error("expected Optional=true")
	}
	if attr.Required {
		t.Error("expected Required=false")
	}
	// Optional-only: the null/[] distinction carries per-category ownership
	// (omit = unmanaged/preserved, [] = declared clear), so the attribute
	// must not be Computed and needs no plan modifiers.
	if attr.Computed {
		t.Error("expected Computed=false")
	}
	if len(attr.PlanModifiers) != 0 {
		t.Errorf("expected no plan modifiers, got %d", len(attr.PlanModifiers))
	}
	if attr.ElementType != types.StringType {
		t.Errorf("expected ElementType=types.StringType, got %T", attr.ElementType)
	}
	if attr.MarkdownDescription == "" {
		t.Error("expected non-empty MarkdownDescription")
	}
}

func TestNameSetAttribute_DefaultShape(t *testing.T) {
	attr := NameSetAttribute("directory service or local user")
	if !attr.Optional {
		t.Error("expected Optional=true")
	}
	if attr.Required {
		t.Error("expected Required=false")
	}
	if attr.Computed {
		t.Error("expected Computed=false")
	}
	if len(attr.PlanModifiers) != 0 {
		t.Errorf("expected no plan modifiers, got %d", len(attr.PlanModifiers))
	}
	if attr.ElementType != types.StringType {
		t.Errorf("expected ElementType=types.StringType, got %T", attr.ElementType)
	}
	if attr.MarkdownDescription == "" {
		t.Error("expected non-empty MarkdownDescription")
	}
}

func TestIDSetAttribute_LabelInterpolated(t *testing.T) {
	attr := IDSetAttribute("computer")
	if !strings.Contains(attr.MarkdownDescription, "computer") {
		t.Errorf("expected MarkdownDescription to contain %q, got %q", "computer", attr.MarkdownDescription)
	}
}

func TestNameSetAttribute_LabelInterpolated(t *testing.T) {
	attr := NameSetAttribute("limit-to-user group")
	if !strings.Contains(attr.MarkdownDescription, "limit-to-user group") {
		t.Errorf("expected MarkdownDescription to contain %q, got %q", "limit-to-user group", attr.MarkdownDescription)
	}
}
