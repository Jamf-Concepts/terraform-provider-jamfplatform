// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_domain

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildDomainRequest_CarriesTheDomain(t *testing.T) {
	got := buildDomainRequest(DomainResourceModel{Domain: types.StringValue("corp.example")})

	if got == nil {
		t.Fatal("buildDomainRequest returned nil")
	}
	if got.Domain != "corp.example" {
		t.Errorf("Domain = %q, want %q", got.Domain, "corp.example")
	}
}

// TestBuildDomainRequest_SendsNothingElse pins the payload shape. Jamf reads only
// `domain` from a claim request and assigns every other field itself, so a
// request that grew a second key would be sending something Jamf ignores — or,
// worse, something it validates.
func TestBuildDomainRequest_SendsNothingElse(t *testing.T) {
	encoded, err := json.Marshal(buildDomainRequest(DomainResourceModel{
		Domain:          types.StringValue("corp.example"),
		ID:              types.StringValue("26917"),
		VerificationKey: types.StringValue("should-not-be-sent"),
	}))
	if err != nil {
		t.Fatalf("encoding the request: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatalf("decoding the request: %v", err)
	}
	if len(body) != 1 {
		t.Errorf("request carries %d fields, want only the domain: %s", len(body), encoded)
	}
	if body["domain"] != "corp.example" {
		t.Errorf("request domain = %v, want %q", body["domain"], "corp.example")
	}
}
