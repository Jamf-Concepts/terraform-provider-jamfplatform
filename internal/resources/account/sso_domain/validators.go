// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_domain

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// domainNameValidator refuses the two configuration mistakes Jamf Account either
// silently corrects or reports without naming the value.
//
// Lower case is the load-bearing half. Jamf lower-cases a domain when it stores
// it, so a mixed-case configuration would be claimed successfully and read back
// changed, which Terraform reports as "Provider produced inconsistent result
// after apply". `domain` is Required, and STYLE_GUIDE §"Plan-modifier rewrites
// are NOT a valid option for Required attributes" rules out canonicalising it in
// a plan modifier — the framework enforces plan == config for a non-Computed
// attribute — so the remaining correct option is strict acceptance with a
// diagnostic that names the spelling to use.
//
// The URL check is convenience rather than necessity: Jamf refuses a value
// carrying a scheme or a path, but its message names neither the value nor the
// part that offends. Everything else about domain syntax is left to Jamf, which
// is deliberately permissive — a reserved top-level domain such as `.example` is
// accepted, and a plan-time syntax check strict enough to be useful would risk
// refusing a name Jamf would have taken.
type domainNameValidator struct{}

// urlishCharacters are the characters that cannot appear in a bare domain name
// and that show up when a URL or an email address has been pasted in place of
// one.
const urlishCharacters = "/:?#@ \t"

// DomainName returns a validator enforcing a lower-case, bare domain name.
func DomainName() validator.String {
	return domainNameValidator{}
}

// Description returns a plain-text description of the validator's behaviour.
func (v domainNameValidator) Description(_ context.Context) string {
	return "must be a bare domain name in lower case"
}

// MarkdownDescription returns a Markdown description of the validator's behaviour.
func (v domainNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString checks the configured domain name.
func (v domainNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()

	if strings.ContainsAny(value, urlishCharacters) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Domain name is not a bare domain",
			"`domain` takes a bare domain name — no scheme, no path, no port, no user part and no "+
				"whitespace. Set it to `example.com` rather than `https://example.com/` or "+
				"`user@example.com`. Configured value: \""+value+"\".",
		)
		return
	}

	if lowered := strings.ToLower(value); lowered != value {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Domain name must be lower case",
			"Jamf lower-cases a domain name when it records the claim, so a mixed-case value would be "+
				"stored differently from the configuration and Terraform would report the result of every "+
				"apply as inconsistent. Set `domain` to \""+lowered+"\".",
		)
	}
}
