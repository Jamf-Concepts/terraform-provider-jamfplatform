// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/aigovernance"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/aischemas"
)

// TestRenderProblem pins the wording each finding produces, and that every one names the schema
// version it was checked against — a finding an operator cannot attribute to a schema version is a
// finding they cannot act on.
func TestRenderProblem(t *testing.T) {
	cases := []struct {
		name           string
		problem        aischemas.Problem
		wantSummary    string
		wantMentions   []string
		mentionsSchema bool
	}{
		{
			name:           "wrong type names the setting",
			problem:        aischemas.Problem{Kind: aischemas.WrongType, Path: "/verbose", Detail: "expected a boolean, found a string."},
			wantSummary:    "Setting does not match the tool's schema",
			wantMentions:   []string{"The setting at /verbose", "expected a boolean"},
			mentionsSchema: true,
		},
		{
			name:           "undeclared key points at a newer schema version",
			problem:        aischemas.Problem{Kind: aischemas.UnrecognisedKey, Path: "/nope", Detail: `"nope" is not declared.`},
			wantSummary:    "Setting is not declared for this schema version",
			wantMentions:   []string{"The setting at /nope", "newer schema version"},
			mentionsSchema: true,
		},
		{
			name:         "missing required key",
			problem:      aischemas.Problem{Kind: aischemas.MissingRequiredKey, Path: "", Detail: `"tier" is required.`},
			wantSummary:  "Required setting is missing",
			wantMentions: []string{"The settings", `"tier" is required.`},
		},
		{
			name:           "a finding about the whole object falls back to naming the settings",
			problem:        aischemas.Problem{Kind: aischemas.UndeclaredKey, Path: "", Detail: "not accepted."},
			wantSummary:    "Setting does not match the tool's schema",
			wantMentions:   []string{"The settings:"},
			mentionsSchema: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			summary, detail := renderProblem(c.problem, "2026-08-14")
			if summary != c.wantSummary {
				t.Errorf("summary = %q, want %q", summary, c.wantSummary)
			}
			for _, mention := range c.wantMentions {
				if !strings.Contains(detail, mention) {
					t.Errorf("detail does not mention %q: %s", mention, detail)
				}
			}
			if c.mentionsSchema && !strings.Contains(detail, "2026-08-14") {
				t.Errorf("detail must name the schema version: %s", detail)
			}
		})
	}
}

// TestSortExpressionValidatorAcceptsOnlySupportedProperties pins the set the platform accepts:
// anything else is refused with VALIDATION_FAILED naming the sort parameter, mid-read.
func TestSortExpressionValidatorAcceptsOnlySupportedProperties(t *testing.T) {
	pattern := mustCompile("^(" + joinAlternatives(sortableProperties) + "):(asc|desc)$")

	for _, good := range []string{"name:asc", "name:desc", "createdAt:asc", "updatedAt:desc"} {
		if !pattern.MatchString(good) {
			t.Errorf("%q should be accepted", good)
		}
	}
	for _, bad := range []string{"name", "name:", ":asc", "bogus:asc", "name:sideways", "NAME:asc", "name:asc,extra"} {
		if pattern.MatchString(bad) {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

// stubService describes an AI Governance catalogue and schema surface for a plan-time validation
// test, including the two ways it can be unavailable.
//
// The seam is the HTTP boundary rather than an injected interface, matching
// security_cloud/dns_zone/crud_partial_state_test.go: aischemas.Cache holds a concrete SDK client,
// and an interface introduced only for a test would be a bigger change than the behaviour it pins.
// The stub is local rather than testhelpers.NewMockClient because testhelpers reaches the provider
// package under the acceptance build tag, and the provider registers this package — importing it
// from an in-package test makes that a cycle.
type stubService struct {
	tools      []aigovernance.ToolSummary
	schema     string
	failTools  bool
	failSchema bool
}

// testSchema declares one boolean setting and one nested object, both open to extra keys — the
// Claude Code shape, where an undeclared key is stored and never applied.
const testSchema = `{"type":"object","additionalProperties":true,"properties":` +
	`{"verbose":{"type":"boolean"},"permissions":{"type":"object","additionalProperties":true,` +
	`"properties":{"allow":{"type":"array"}}}}}`

// claudeCode is the catalogue entry the settings tests validate against, carrying one superseded
// schema version so the drift warning has something to fire on.
func claudeCode() aigovernance.ToolSummary {
	return aigovernance.ToolSummary{
		ID:             "com.anthropic.claudecode",
		DisplayName:    "Claude Code",
		SchemaVersion:  "2026-08-14",
		SchemaVersions: []string{"2026-08-14", "2026-05-19"},
	}
}

// resource builds a PolicyResource whose schema cache is backed by the stub.
func (s stubService) resource(t *testing.T) *PolicyResource {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "test-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case strings.Contains(r.URL.Path, "/schemas/"):
			if s.failSchema {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "the schema read failed"})
				return
			}
			_ = json.NewEncoder(w).Encode(aigovernance.ToolSchemaResponse{
				ToolID:        claudeCode().ID,
				SchemaVersion: claudeCode().SchemaVersion,
				Schema:        json.RawMessage(s.schema),
			})
		case strings.HasSuffix(r.URL.Path, "/tools"):
			if s.failTools {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "the catalogue read failed"})
				return
			}
			_ = json.NewEncoder(w).Encode(aigovernance.ToolListResponse{Results: s.tools, TotalCount: len(s.tools)})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := jamfplatform.NewClient(server.URL, "test-id", "test-secret",
		jamfplatform.WithRetryPolicy(0, 0, 0),
		jamfplatform.WithMinRequestInterval(0),
	)
	return &PolicyResource{client: aigovernance.New(client), schemas: aischemas.NewCache(client)}
}

// diagnosticPath returns the attribute path a diagnostic names, or the empty string for one that
// names none.
func diagnosticPath(diagnostic diag.Diagnostic) string {
	if withPath, ok := diagnostic.(diag.DiagnosticWithPath); ok {
		return withPath.Path().String()
	}
	return ""
}

// assertCounts fails unless the diagnostics hold exactly the given number of errors and warnings,
// reporting what they actually said when they do not.
func assertCounts(t *testing.T, diags diag.Diagnostics, wantErrors, wantWarnings int) {
	t.Helper()
	if got := len(diags.Errors()); got != wantErrors {
		t.Errorf("got %d errors, want %d: %v", got, wantErrors, diags.Errors())
	}
	if got := len(diags.Warnings()); got != wantWarnings {
		t.Errorf("got %d warnings, want %d: %v", got, wantWarnings, diags.Warnings())
	}
}

// TestCheckSettingsUndeclaredKeyIsAWarning pins the classification the whole feature turns on: a key
// the schema does not declare is stored by the platform and never applied by the tool, which is
// advisory — the key may simply postdate the schema version — so it must not block a plan. Inverting
// the Advisory branch fails here.
func TestCheckSettingsUndeclaredKeyIsAWarning(t *testing.T) {
	r := stubService{tools: []aigovernance.ToolSummary{claudeCode()}, schema: testSchema}.resource(t)
	plan := policyModel{SettingsJSON: newJSONObjectValue(`{"permissions":{"allow":[],"allowlist":true}}`)}

	var diags diag.Diagnostics
	r.checkSettings(context.Background(), &diags, &plan, claudeCode().ID, claudeCode().SchemaVersion)

	assertCounts(t, diags, 0, 1)
	if len(diags.Warnings()) == 0 {
		return
	}
	warning := diags.Warnings()[0]
	if got := diagnosticPath(warning); got != "settings_json" {
		t.Errorf("warning path = %q, want settings_json", got)
	}
	if !strings.Contains(warning.Detail(), "The setting at /permissions/allowlist") {
		t.Errorf("the warning must locate the key by JSON pointer: %s", warning.Detail())
	}
}

// TestCheckSettingsWrongTypeIsAnError pins the other half of the split: a value of the wrong type is
// a write Jamf refuses, so reporting it as advisory would let the apply fail instead of the plan.
func TestCheckSettingsWrongTypeIsAnError(t *testing.T) {
	r := stubService{tools: []aigovernance.ToolSummary{claudeCode()}, schema: testSchema}.resource(t)
	plan := policyModel{SettingsJSON: newJSONObjectValue(`{"verbose":"yes"}`)}

	var diags diag.Diagnostics
	r.checkSettings(context.Background(), &diags, &plan, claudeCode().ID, claudeCode().SchemaVersion)

	assertCounts(t, diags, 1, 0)
	if len(diags.Errors()) == 0 {
		return
	}
	if got := diagnosticPath(diags.Errors()[0]); got != "settings_json" {
		t.Errorf("error path = %q, want settings_json", got)
	}
}

// TestCheckSettingsAcceptsAValidBody pins that a body matching the schema produces nothing at all,
// so neither half of the split fires on a correct configuration.
func TestCheckSettingsAcceptsAValidBody(t *testing.T) {
	r := stubService{tools: []aigovernance.ToolSummary{claudeCode()}, schema: testSchema}.resource(t)
	plan := policyModel{SettingsJSON: newJSONObjectValue(`{"verbose":true,"permissions":{"allow":["Read"]}}`)}

	var diags diag.Diagnostics
	r.checkSettings(context.Background(), &diags, &plan, claudeCode().ID, claudeCode().SchemaVersion)

	assertCounts(t, diags, 0, 0)
}

// TestCheckCatalogueWarnsOnASupersededSchemaVersion pins the drift warning. The platform keeps
// serving the older schema, so this cannot be an error — but schema_drift as a computed attribute
// only reports it after the apply.
func TestCheckCatalogueWarnsOnASupersededSchemaVersion(t *testing.T) {
	r := stubService{tools: []aigovernance.ToolSummary{claudeCode()}, schema: testSchema}.resource(t)

	var diags diag.Diagnostics
	if !r.checkCatalogue(context.Background(), &diags, claudeCode().ID, "2026-05-19") {
		t.Error("a superseded schema version must still be checked against")
	}

	assertCounts(t, diags, 0, 1)
	if len(diags.Warnings()) == 0 {
		return
	}
	if got := diagnosticPath(diags.Warnings()[0]); got != "schema_version" {
		t.Errorf("warning path = %q, want schema_version", got)
	}
}

// TestCheckCatalogueRejectsAnUnknownTool pins the one plan-time hard error the catalogue produces,
// and that it names the attribute the operator wrote rather than the wire field.
func TestCheckCatalogueRejectsAnUnknownTool(t *testing.T) {
	r := stubService{tools: []aigovernance.ToolSummary{claudeCode()}, schema: testSchema}.resource(t)

	var diags diag.Diagnostics
	if r.checkCatalogue(context.Background(), &diags, "com.example.nope", "2026-08-14") {
		t.Error("an unknown tool must stop the settings check")
	}

	assertCounts(t, diags, 1, 0)
	if len(diags.Errors()) == 0 {
		return
	}
	failure := diags.Errors()[0]
	if got := diagnosticPath(failure); got != "tool_id" {
		t.Errorf("error path = %q, want tool_id", got)
	}
	if !strings.Contains(failure.Detail(), claudeCode().ID) {
		t.Errorf("the error must list the identifiers the catalogue does offer: %s", failure.Detail())
	}
}

// TestCheckCatalogueRejectsAnUnknownSchemaVersion pins the second hard error, which the platform
// would otherwise report mid-apply as SCHEMA_VERSION_UNKNOWN.
func TestCheckCatalogueRejectsAnUnknownSchemaVersion(t *testing.T) {
	r := stubService{tools: []aigovernance.ToolSummary{claudeCode()}, schema: testSchema}.resource(t)

	var diags diag.Diagnostics
	if r.checkCatalogue(context.Background(), &diags, claudeCode().ID, "1999-01-01") {
		t.Error("an unknown schema version must stop the settings check")
	}

	assertCounts(t, diags, 1, 0)
	if len(diags.Errors()) == 0 {
		return
	}
	if got := diagnosticPath(diags.Errors()[0]); got != "schema_version" {
		t.Errorf("error path = %q, want schema_version", got)
	}
}

// TestUnreadableCatalogueWarnsOncePerPlan pins the difference between a plan where validation passed
// and a plan where it never ran. The plan must still succeed — the platform validates the write —
// but silence would leave the two indistinguishable, while the resource documentation tells the
// operator the settings are checked during plan.
func TestUnreadableCatalogueWarnsOncePerPlan(t *testing.T) {
	r := stubService{schema: testSchema, failTools: true}.resource(t)

	var diags diag.Diagnostics
	if !r.checkCatalogue(context.Background(), &diags, claudeCode().ID, claudeCode().SchemaVersion) {
		t.Error("a catalogue that cannot be read must not stop the plan")
	}

	assertCounts(t, diags, 0, 1)
	if len(diags.Warnings()) == 0 {
		return
	}
	if got := diags.Warnings()[0].Summary(); got != "Settings validation unavailable" {
		t.Errorf("warning summary = %q", got)
	}

	if !r.checkCatalogue(context.Background(), &diags, claudeCode().ID, claudeCode().SchemaVersion) {
		t.Error("a catalogue that cannot be read must not stop the plan")
	}
	assertCounts(t, diags, 0, 1)
}

// TestUnreadableSchemaWarnsOnce pins the same notice on the other fetch. A role holding
// ai-policies:write without ai-policies:read reaches exactly this path.
func TestUnreadableSchemaWarnsOnce(t *testing.T) {
	r := stubService{tools: []aigovernance.ToolSummary{claudeCode()}, schema: testSchema, failSchema: true}.resource(t)
	plan := policyModel{SettingsJSON: newJSONObjectValue(`{"verbose":"yes"}`)}

	var diags diag.Diagnostics
	r.checkSettings(context.Background(), &diags, &plan, claudeCode().ID, claudeCode().SchemaVersion)
	assertCounts(t, diags, 0, 1)

	r.checkSettings(context.Background(), &diags, &plan, claudeCode().ID, claudeCode().SchemaVersion)
	assertCounts(t, diags, 0, 1)
}

// policyRawPlan builds a plan with the three attributes plan-time validation reads set, and
// everything else null, the way Terraform sends a create for a configuration that sets only those.
func policyRawPlan(ctx context.Context, policySchema resourceschema.Schema, toolID, schemaVersion, settings string) tftypes.Value {
	object := policySchema.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	values["tool_id"] = tftypes.NewValue(tftypes.String, toolID)
	values["schema_version"] = tftypes.NewValue(tftypes.String, schemaVersion)
	values["settings_json"] = tftypes.NewValue(tftypes.String, settings)

	return tftypes.NewValue(object, values)
}

// TestModifyPlanClassifiesFindings pins the wiring from a plan to the diagnostics an operator sees:
// an advisory finding must reach them as a warning the plan survives, and everything else as an
// error that stops it. The acceptance suite cannot cover the warning half — terraform-plugin-testing
// has no warning assertion — so inverting the Advisory branch is caught only here.
func TestModifyPlanClassifiesFindings(t *testing.T) {
	cases := []struct {
		name          string
		toolID        string
		schemaVersion string
		settings      string
		wantErrors    int
		wantWarnings  int
	}{
		{
			name:          "undeclared key warns",
			toolID:        claudeCode().ID,
			schemaVersion: claudeCode().SchemaVersion,
			settings:      `{"permissions":{"allow":[],"allowlist":true}}`,
			wantWarnings:  1,
		},
		{
			name:          "wrong type errors",
			toolID:        claudeCode().ID,
			schemaVersion: claudeCode().SchemaVersion,
			settings:      `{"verbose":"yes"}`,
			wantErrors:    1,
		},
		{
			name:          "an unknown tool stops before the settings are read",
			toolID:        "com.example.nope",
			schemaVersion: claudeCode().SchemaVersion,
			settings:      `{"verbose":"yes"}`,
			wantErrors:    1,
		},
		{
			name:          "a valid body reports nothing",
			toolID:        claudeCode().ID,
			schemaVersion: claudeCode().SchemaVersion,
			settings:      `{"verbose":true}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			r := stubService{tools: []aigovernance.ToolSummary{claudeCode()}, schema: testSchema}.resource(t)

			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

			var resp resource.ModifyPlanResponse
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{
				Plan: tfsdk.Plan{
					Schema: schemaResp.Schema,
					Raw:    policyRawPlan(ctx, schemaResp.Schema, c.toolID, c.schemaVersion, c.settings),
				},
			}, &resp)

			assertCounts(t, resp.Diagnostics, c.wantErrors, c.wantWarnings)
		})
	}
}
