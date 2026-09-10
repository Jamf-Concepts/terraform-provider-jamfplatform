// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
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

// isVersionConflict reports whether an error is the platform refusing an update because the policy
// has moved on since the version the request named. Only update can produce it, so it is checked
// there rather than in appendWriteDiagnostics, which create shares.
func isVersionConflict(err error) bool {
	return hasCode(err, codePolicyVersionConflict)
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

// privateKeyVersion holds the policy's optimistic-lock counter between one apply and the next, so
// that an update can make its PATCH conditional on nobody else having written the policy since the
// provider last read it.
//
// It lives in private state rather than in the schema for two reasons. The counter is machinery an
// operator can neither set nor usefully read — it increments on every PATCH, including one the
// platform diffs to nothing — so surfacing it would put churn in every plan for a value nothing can
// consume. And adding an attribute to a shipped schema needs a state upgrader, which private state
// does not: a policy recorded before this key existed reads back empty and updates unconditionally,
// exactly as it did before.
const privateKeyVersion = "policy_version"

// privateStateReader is the read half of the framework's private-state surface. The concrete type is
// internal to the framework, so a narrow interface is the only way to write a helper against it —
// and it makes the round trip testable without a live resource.
type privateStateReader interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

// privateStateWriter is the write half, satisfied uniformly by the create, read and update response
// private-state surfaces.
//
// Every call site assigns the response field to a variable of this type only when the field is
// non-nil, and must keep doing so. The framework's concrete type is unnameable here, so a nil
// pointer assigned straight into the interface would arrive as a non-nil interface holding a nil
// pointer — past the guard below and into SetKey, which reports an uninitialized-ProviderData error
// on a nil receiver. The framework populates the field on every real operation; a response built
// without one is a unit test, and recording nothing is the right outcome there.
type privateStateWriter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// readLockToken returns the If-Match precondition recorded by the last read of this policy, or the
// empty string when there is none — a policy the platform created before it gained the counter, one
// imported in this run, or state written before the provider began recording it. The empty string is
// what UpdatePolicy takes to mean "unconditional", so each of those cases degrades to the behaviour
// the resource had before this key existed.
func readLockToken(ctx context.Context, r privateStateReader) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if r == nil {
		return "", diags
	}
	raw, d := r.GetKey(ctx, privateKeyVersion)
	diags.Append(d...)
	if diags.HasError() || len(raw) == 0 {
		return "", diags
	}
	var token string
	if err := json.Unmarshal(raw, &token); err != nil {
		diags.AddError(
			"Unable to read the AI policy's recorded version",
			"Terraform's private state for this policy holds a value the provider cannot decode: "+err.Error()+
				". Remove the resource from state and import it again to clear it.",
		)
		return "", diags
	}
	return token, diags
}

// writeLockToken records the counter a just-completed read reported, ready for the next update. A
// policy the platform reports without one — every policy that predates the counter — clears the key
// instead, so a value that has gone stale can never be sent as a precondition.
//
// The value is stored already rendered as the header the update will send. Private state must hold
// valid JSON and the header is a decimal string, so storing the rendered form keeps one spelling of
// the value rather than a number here and a formatting step at the call site.
func writeLockToken(ctx context.Context, w privateStateWriter, version *int64) diag.Diagnostics {
	var diags diag.Diagnostics
	if w == nil {
		return diags
	}
	if version == nil {
		diags.Append(w.SetKey(ctx, privateKeyVersion, nil)...)
		return diags
	}
	encoded, err := json.Marshal(strconv.FormatInt(*version, 10))
	if err != nil {
		diags.AddError(
			"Unable to record the AI policy's version",
			fmt.Sprintf("JSON-encoding the version counter %d for private state: %s", *version, err.Error()),
		)
		return diags
	}
	diags.Append(w.SetKey(ctx, privateKeyVersion, encoded)...)
	return diags
}

// appendVersionConflict reports an update the platform refused because the policy changed since
// Terraform last read it.
//
// The remedy is a plain re-run: the refresh at the start of the next plan picks up whatever was
// written, so the change appears in the plan the operator approves instead of being replaced
// unseen. That matters more here than it would elsewhere because settings_json is sent whole — the
// platform holds no merge for it, so an update built on a stale read overwrites every setting
// another editor had made.
func appendVersionConflict(diags *diag.Diagnostics, id string) {
	diags.AddError(
		"AI policy changed outside Terraform",
		"Policy "+id+" was modified by something else after Terraform last read it, so the update was refused "+
			"rather than applied over the top, and nothing has been changed. Run Terraform again: the refresh "+
			"picks up the current policy and the next plan reports what this configuration would alter, "+
			"including any setting the other change introduced. Terraform checks this because settings_json is "+
			"written whole — Jamf holds no merge for it, so an update built on a stale read replaces every "+
			"setting another editor had made.",
	)
}
