// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package activation_profile

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// resourceSchema builds the resource schema for inspection.
func resourceSchema(t *testing.T) schema.Schema {
	t.Helper()
	r := &ActivationProfileResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// TestSchema_AttributeSet pins the attribute surface, so an addition or removal is
// a deliberate change rather than a surprise.
func TestSchema_AttributeSet(t *testing.T) {
	s := resourceSchema(t)
	want := []string{"id", "name", "platforms", "capabilities", "device_group_id", "paused", "timeouts"}
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if got, wantLen := len(s.Attributes), len(want); got != wantLen {
		t.Errorf("schema has %d attributes, expected %d: %v", got, wantLen, s.Attributes)
	}
}

// TestSchema_RequiredAndComputed checks the required set and that the activation
// code is server-minted.
func TestSchema_RequiredAndComputed(t *testing.T) {
	s := resourceSchema(t)
	for _, name := range []string{"name", "platforms", "capabilities"} {
		if !s.Attributes[name].IsRequired() {
			t.Errorf("%q should be required", name)
		}
	}
	for _, name := range []string{"device_group_id", "paused"} {
		if !s.Attributes[name].IsOptional() {
			t.Errorf("%q should be optional", name)
		}
	}
	if !s.Attributes["id"].IsComputed() || s.Attributes["id"].IsRequired() || s.Attributes["id"].IsOptional() {
		t.Error("id should be computed-only — Jamf Security Cloud mints the activation code")
	}
}

// TestSchema_EveryConfiguredAttributeRequiresReplace is the structural expression
// of this resource's central constraint.
//
// Jamf Security Cloud has no update endpoint for an activation profile and returns
// only the activation code when one is read, so nothing configured can be changed
// in place or refreshed. `paused` is the one exception: the pause and resume
// operations change it without replacing the profile.
func TestSchema_EveryConfiguredAttributeRequiresReplace(t *testing.T) {
	s := resourceSchema(t)

	name, ok := s.Attributes["name"].(schema.StringAttribute)
	if !ok || len(name.PlanModifiers) == 0 {
		t.Error("name has no plan modifiers; expected RequiresReplace")
	}
	group, ok := s.Attributes["device_group_id"].(schema.StringAttribute)
	if !ok || len(group.PlanModifiers) == 0 {
		t.Error("device_group_id has no plan modifiers; expected RequiresReplace")
	}
	platforms, ok := s.Attributes["platforms"].(schema.SetAttribute)
	if !ok || len(platforms.PlanModifiers) == 0 {
		t.Error("platforms has no plan modifiers; expected RequiresReplace")
	}
	capabilities, ok := s.Attributes["capabilities"].(schema.SingleNestedAttribute)
	if !ok || len(capabilities.PlanModifiers) == 0 {
		t.Error("capabilities has no plan modifiers; expected RequiresReplace")
	}

	paused, ok := s.Attributes["paused"].(schema.BoolAttribute)
	if !ok {
		t.Fatal("paused is not a BoolAttribute")
	}
	if len(paused.PlanModifiers) != 0 {
		t.Error("paused must not require replacement — pause and resume change it in place")
	}
}

// TestSchema_CapabilityAttributes pins the capability surface, including that the
// coupled wire pair is modelled as one attribute.
func TestSchema_CapabilityAttributes(t *testing.T) {
	s := resourceSchema(t)
	capabilities, ok := s.Attributes["capabilities"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("capabilities is not a SingleNestedAttribute")
	}
	want := []string{"content_controls", "network_security", "note"}
	for _, name := range want {
		if _, ok := capabilities.Attributes[name]; !ok {
			t.Errorf("missing capability attribute %q", name)
		}
	}
	if got := len(capabilities.Attributes); got != len(want) {
		t.Errorf("capabilities has %d attributes, expected %d — networkSecurity and vulnerabilityManagement are one attribute here, not two", got, len(want))
	}
	if _, exists := capabilities.Attributes["vulnerability_management"]; exists {
		t.Error("vulnerability_management should not be a separate attribute: the server refuses it disagreeing with network_security, and the console shows one checkbox")
	}
}

// TestSchema_ImportIsNotSupported pins a deliberate omission.
//
// Jamf Security Cloud returns only the activation code, so an imported profile
// would carry null for every RequiresReplace attribute and the next plan would
// replace — destroying the profile it had just adopted.
func TestSchema_ImportIsNotSupported(t *testing.T) {
	var r any = &ActivationProfileResource{}
	if _, ok := r.(resource.ResourceWithImportState); ok {
		t.Error("resource implements ResourceWithImportState; import is deliberately unsupported because a GET returns only the activation code")
	}
}

// wireJargon matches protocol vocabulary that must not reach the Terraform
// Registry, per STYLE_GUIDE §User-facing descriptions are UI-aligned. Product
// framing ("Jamf Security Cloud refuses…") is fine; protocol framing is not.
var wireJargon = regexp.MustCompile(`(?i)\b(api|endpoint|wire|payload|sdk|http|PUT|POST|DELETE|GET|/v1/|json|4\d\d|5\d\d)\b`)

// TestSchema_DescriptionsAreUIAligned keeps wire vocabulary out of every
// user-facing description, including the nested ones.
func TestSchema_DescriptionsAreUIAligned(t *testing.T) {
	s := resourceSchema(t)
	check := func(label, text string) {
		if text == "" {
			t.Errorf("%s has an empty description", label)
			return
		}
		if match := wireJargon.FindString(text); match != "" {
			t.Errorf("%s description contains wire vocabulary %q: %s", label, match, text)
		}
	}
	check("resource", stripPrivilegeSection(s.MarkdownDescription))
	for name, attribute := range s.Attributes {
		if name == "timeouts" {
			continue
		}
		check(name, attribute.GetMarkdownDescription())
	}
	capabilities, ok := s.Attributes["capabilities"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("capabilities is not a SingleNestedAttribute")
	}
	for name, attribute := range capabilities.Attributes {
		check("capabilities."+name, attribute.GetMarkdownDescription())
	}
}

// stripPrivilegeSection removes the generated "Required Jamf permissions" table,
// which legitimately names privileges rather than describing the resource.
func stripPrivilegeSection(text string) string {
	if before, _, ok := strings.Cut(text, "Required Jamf permissions"); ok {
		return before
	}
	return text
}
