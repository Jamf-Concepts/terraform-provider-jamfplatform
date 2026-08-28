// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package uemconnectactions

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
)

func synchronizeSchema(t *testing.T) actionschema.Schema {
	t.Helper()
	a := NewSynchronizeAction()
	var resp action.SchemaResponse
	a.(*SynchronizeAction).Schema(context.Background(), action.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestSynchronizeAction_Metadata(t *testing.T) {
	a := NewSynchronizeAction()
	var resp action.MetadataResponse
	a.(*SynchronizeAction).Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	want := "jamfplatform_security_cloud_uem_connect_synchronize"
	if resp.TypeName != want {
		t.Errorf("type name = %q, want %q", resp.TypeName, want)
	}
}

// TestSynchronizeAction_IDIsOptional pins the shape the whole ergonomics rest on: a
// tenant holds at most one integration, so requiring its opaque identifier would be
// friction with nothing to disambiguate. The attribute stays available because
// naming the resource's ID is how a configuration orders the action after it.
func TestSynchronizeAction_IDIsOptional(t *testing.T) {
	s := synchronizeSchema(t)

	attr, ok := s.Attributes["uem_connect_id"]
	if !ok {
		t.Fatal("missing attribute uem_connect_id")
	}
	if attr.IsRequired() {
		t.Error("uem_connect_id must be optional — a tenant has only one integration to act on")
	}
	if !attr.IsOptional() {
		t.Error("uem_connect_id must be optional")
	}
}

// TestSynchronizeAction_NoOtherAttributes pins that the action takes nothing else.
// An action is fire-once with no state, so every attribute added here is a
// parameter a caller has to get right for something they cannot observe the result
// of.
func TestSynchronizeAction_NoOtherAttributes(t *testing.T) {
	s := synchronizeSchema(t)

	if len(s.Attributes) != 1 {
		names := make([]string, 0, len(s.Attributes))
		for name := range s.Attributes {
			names = append(names, name)
		}
		t.Errorf("expected only uem_connect_id, got %v", names)
	}
}

// TestSynchronizeAction_DescriptionSaysItDoesNotReport pins the honesty that matters
// most for a fire-once action over an asynchronous operation: the request is
// accepted, not completed, so the description must not imply the sync has finished
// or that its outcome is available here.
func TestSynchronizeAction_DescriptionSaysItDoesNotReport(t *testing.T) {
	s := synchronizeSchema(t)
	desc := strings.TrimSuffix(s.MarkdownDescription, synchronizePrivileges)

	for _, fragment := range []string{"background", "does not wait", "latest_sync"} {
		if !strings.Contains(desc, fragment) {
			t.Errorf("description does not mention %q:\n%s", fragment, desc)
		}
	}
}

// TestSynchronizeAction_DescriptionsAreUIAligned pins STYLE_GUIDE §User-facing
// descriptions are UI-aligned, not wire-aligned.
func TestSynchronizeAction_DescriptionsAreUIAligned(t *testing.T) {
	s := synchronizeSchema(t)

	descriptions := []string{strings.TrimSuffix(s.MarkdownDescription, synchronizePrivileges)}
	for _, attr := range s.Attributes {
		descriptions = append(descriptions, attr.GetMarkdownDescription())
	}

	for _, desc := range descriptions {
		lower := strings.ToLower(desc)
		for _, banned := range []string{"endpoint", "/v1/", " sdk", "202", "http"} {
			if strings.Contains(lower, banned) {
				t.Errorf("description contains wire vocabulary %q:\n%s", banned, desc)
			}
		}
	}
}
