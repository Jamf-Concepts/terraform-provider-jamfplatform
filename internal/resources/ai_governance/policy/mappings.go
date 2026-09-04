// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

// Machine-readable error codes the AI Governance policies endpoints return. Wire-probed against the
// EU sandbox on 2026-08-30 (raw bodies in local-testing/ai-governance/).
//
// The SDK generates four enums for this namespace — policy status, blueprint deployment state and
// deployment run state — and none of them is an error vocabulary: `aigovernance.ApiErrorItem.Code`
// is a plain string with no generated constants at all, so every code below is a literal by
// necessity rather than by choice. enum_literals_test.go records that per value, so an SDK release
// that starts generating any of them fails the build instead of leaving a silent duplicate.
const (
	// codeToolIDUnknown accompanies the 422 for a tool identifier that is not in the catalogue.
	// Reported against the toolId field.
	codeToolIDUnknown = "TOOL_ID_UNKNOWN"

	// codeSchemaVersionUnknown accompanies the 422 for a schema version the tool does not offer.
	// Reported against the schemaVersion field.
	codeSchemaVersionUnknown = "SCHEMA_VERSION_UNKNOWN"

	// codeSchemaValidationFailed accompanies the 422 for settings that do not match the tool's
	// schema. `field` is a JSON pointer into the settings when the offending value can be located
	// (`/verbose`) and empty when the failure is about the settings object itself — a JSON array
	// where an object was expected, or an undeclared key on a schema that refuses extras. Both
	// shapes were probed; do not assume a field is always given.
	codeSchemaValidationFailed = "SCHEMA_VALIDATION_FAILED"

	// codeValidationFailed accompanies the 400 for a field the request shape itself rejects: a
	// missing schemaVersion ("must not be blank"), a null settings object ("must not be null"), or
	// an unsupported sort property.
	codeValidationFailed = "VALIDATION_FAILED"

	// codePolicyNotFound accompanies the 404 from read, update, publish and delete. It is also what
	// a malformed identifier returns, so there is no separate invalid-id shape to handle, and it is
	// what an already-archived policy returns — archiving is a soft delete the API renders as total
	// absence.
	codePolicyNotFound = "POLICY_NOT_FOUND"

	// codeNoDraftToPublish accompanies the 409 from publish when nothing is staged. The platform
	// diffs settings itself, so an update that changes nothing leaves no draft and the publish that
	// follows lands here. Treated as success rather than translated — see publishIfNeeded in crud.go.
	codeNoDraftToPublish = "NO_DRAFT_TO_PUBLISH"

	// codeRequestContextNotProvided is the gateway's own 400, returned before the service is reached
	// when the integration sends no scope header. Not translated into a diagnostic: providerdata's
	// scope gate refuses organization scope at Configure, so an operator reaching this code has a
	// client that disagrees with its own scope and the raw error is the more useful signal.
	codeRequestContextNotProvided = "REQUEST_CONTEXT_NOT_PROVIDED"
)
