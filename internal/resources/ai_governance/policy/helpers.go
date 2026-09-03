// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"regexp"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// isNotFound reports whether an error is the platform saying the policy is gone. A malformed
// identifier and an already-archived policy both answer with this code, so read, update, publish and
// delete all treat it the same way.
func isNotFound(err error) bool {
	return hasCode(err, codePolicyNotFound)
}

// hasCode reports whether an error carries the given machine-readable code.
func hasCode(err error, code string) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil {
		return false
	}
	for _, detail := range apiErr.Details() {
		if detail.Code == code {
			return true
		}
	}
	return false
}

// appendWriteDiagnostics turns a create or update failure into the most specific diagnostic the
// error body supports, and reports whether it recognised one.
//
// Only codes whose remedy is not obvious from the platform's own wording are translated, and each
// one is pointed at the attribute the operator actually wrote — the platform names wire fields
// (`toolId`, `settings`), which a practitioner reading a Terraform diagnostic has never typed.
//
// SCHEMA_VALIDATION_FAILED earns the most work because it is the code plan-time validation is meant
// to pre-empt: reaching it at apply means either the settings changed shape after the plan, or the
// schema declares something the provider's own checker skips. Its `field` is a JSON pointer into the
// settings when the platform can locate the value and empty when the problem is the object itself,
// so the pointer is quoted into the detail rather than used to build an attribute path.
//
// REQUEST_CONTEXT_NOT_PROVIDED is deliberately absent — see mappings.go for why.
func appendWriteDiagnostics(diags *diag.Diagnostics, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil {
		return false
	}

	matched := false
	for _, detail := range apiErr.Details() {
		switch detail.Code {
		case codeToolIDUnknown:
			diags.AddAttributeError(
				path.Root("tool_id"),
				"Unknown AI tool",
				"The platform offers no AI tool with this identifier. Read the available identifiers from the "+
					"jamfplatform_ai_governance_tools data source — they are reverse-domain names such as "+
					"com.anthropic.claudecode, and the match is exact. Reported by Jamf: "+detail.Description,
			)
		case codeSchemaVersionUnknown:
			diags.AddAttributeError(
				path.Root("schema_version"),
				"Unknown settings schema version",
				"This tool does not offer the requested settings schema version. Read the versions it does offer from "+
					"the jamfplatform_ai_governance_tool data source's schema_versions attribute. Reported by Jamf: "+
					detail.Description,
			)
		case codeSchemaValidationFailed:
			diags.AddAttributeError(
				path.Root("settings_json"),
				"Settings do not match the tool's schema",
				schemaFailureDetail(detail.Field, detail.Description),
			)
		case codeValidationFailed:
			diags.AddError(
				"Jamf rejected the policy",
				"Jamf rejected this policy on the field "+quoteOrUnnamed(detail.Field)+". Reported by Jamf: "+
					detail.Description,
			)
		default:
			continue
		}
		matched = true
	}
	return matched
}

// schemaFailureDetail composes the detail for a settings validation failure, locating it by the
// platform's own JSON pointer when there is one.
func schemaFailureDetail(field, description string) string {
	location := "The settings were rejected"
	if field != "" {
		location = "The setting at " + field + " was rejected"
	}
	return location + " when checked against this tool's schema for the requested schema_version. " +
		"The provider checks the same schema during plan, so reaching this at apply means either the settings " +
		"changed after the plan was made, or the rule involved is one the provider's own check does not cover. " +
		"Reported by Jamf: " + description
}

// quoteOrUnnamed renders a wire field name for a diagnostic, saying so when the platform named none.
func quoteOrUnnamed(field string) string {
	if strings.TrimSpace(field) == "" {
		return "it did not name"
	}
	return "\"" + field + "\""
}

// joinAlternatives renders a set of names as a regular-expression alternation.
func joinAlternatives(names []string) string {
	return strings.Join(names, "|")
}

// joinCommas renders a set of names for a diagnostic.
func joinCommas(names []string) string {
	return strings.Join(names, ", ")
}

// mustCompile compiles a pattern the package itself builds from a fixed set of names. A failure is a
// programming error rather than anything an operator can cause, and the schema is built before any
// diagnostic sink exists to report it to.
func mustCompile(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}
