// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_domain

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// validateDomain runs the domain validator over one value.
func validateDomain(value types.String) validator.StringResponse {
	var resp validator.StringResponse
	DomainName().ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("domain"),
		ConfigValue: value,
	}, &resp)
	return resp
}

func TestDomainName_AcceptsABareLowerCaseDomain(t *testing.T) {
	for _, value := range []string{"example.com", "corp.example", "a.b.c.example.com", "xn--80ak6aa92e.com", "localdomain"} {
		if resp := validateDomain(types.StringValue(value)); resp.Diagnostics.HasError() {
			t.Errorf("%q was refused: %v", value, resp.Diagnostics)
		}
	}
}

// TestDomainName_RefusesMixedCase is the one that matters. Jamf lower-cases the
// value it stores, so a mixed-case configuration would apply and then read back
// changed, which Terraform reports as an inconsistent result. `domain` is
// Required, so a plan modifier cannot canonicalise it — the framework enforces
// plan == config for a non-Computed attribute — leaving strict acceptance.
func TestDomainName_RefusesMixedCase(t *testing.T) {
	resp := validateDomain(types.StringValue("Corp.Example"))

	if !resp.Diagnostics.HasError() {
		t.Fatal("a mixed-case domain must be refused at plan time")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "corp.example") {
		t.Errorf("detail %q must name the spelling to use", detail)
	}
}

// TestDomainName_RefusesAURL covers the paste-a-URL mistake, which Jamf refuses
// with a message naming neither the value nor the part that offends.
func TestDomainName_RefusesAURL(t *testing.T) {
	for _, value := range []string{
		"https://example.com",
		"example.com/path",
		"example.com:443",
		"user@example.com",
		"example.com?x=1",
		"exa mple.com",
	} {
		resp := validateDomain(types.StringValue(value))
		if !resp.Diagnostics.HasError() {
			t.Errorf("%q must be refused: it is not a bare domain name", value)
			continue
		}
		if summary := resp.Diagnostics.Errors()[0].Summary(); !strings.Contains(summary, "bare domain") {
			t.Errorf("%q produced the wrong diagnostic: %q", value, summary)
		}
	}
}

// TestDomainName_DefersOnUnknownAndNull pins STYLE_GUIDE §"Config-time validators
// MUST defer on unknown values". A domain sourced from a variable or another
// resource is unknown at validate time, and erroring on that would make the
// resource unusable from anything but a hard-coded literal.
func TestDomainName_DefersOnUnknownAndNull(t *testing.T) {
	if resp := validateDomain(types.StringUnknown()); resp.Diagnostics.HasError() {
		t.Errorf("an unknown domain must defer, got %v", resp.Diagnostics)
	}
	if resp := validateDomain(types.StringNull()); resp.Diagnostics.HasError() {
		t.Errorf("a null domain must defer to the Required check, got %v", resp.Diagnostics)
	}
}

func TestDomainName_Descriptions(t *testing.T) {
	ctx := context.Background()
	v := DomainName()
	if v.Description(ctx) == "" {
		t.Error("Description must not be empty")
	}
	if v.MarkdownDescription(ctx) != v.Description(ctx) {
		t.Error("MarkdownDescription must match Description for a plain-text rule")
	}
}
