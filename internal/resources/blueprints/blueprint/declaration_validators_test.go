// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jamf/terraform-provider-jamfplatform/internal/common/appledeclarations"
)

// declarationPath is a representative attribute path, so a failure message shows where a diagnostic
// would land in a real configuration.
func declarationPath() path.Path {
	return path.Root("component_blocks").AtListIndex(0).
		AtName("apple_declarations").AtName("declaration").AtListIndex(0).AtName("payload")
}

// TestValidateDeclarationPayloadAcceptsValid checks the happy path produces no diagnostics, so the
// validator cannot pass its other tests simply by rejecting everything.
func TestValidateDeclarationPayloadAcceptsValid(t *testing.T) {
	diags := validateDeclarationPayload(
		types.StringValue("com.apple.configuration.siri.settings"),
		types.StringValue(`{"Enabled":true,"ForceProfanityFilter":true}`),
		declarationPath(), "",
	)
	if diags.HasError() {
		t.Fatalf("valid declaration produced errors: %v", diags.Errors())
	}
}

// TestValidateDeclarationPayloadSkipsUnresolvedValues pins the plan-time behaviour: a value
// Terraform has yet to compute must be left alone rather than guessed at, or an interpolated
// payload would fail every plan before its inputs exist.
func TestValidateDeclarationPayloadSkipsUnresolvedValues(t *testing.T) {
	cases := map[string]struct{ declarationType, payload types.String }{
		"unknown payload": {
			types.StringValue("com.apple.configuration.siri.settings"),
			types.StringUnknown(),
		},
		"unknown type": {
			types.StringUnknown(),
			types.StringValue(`{"Enabled":true}`),
		},
		"null payload": {
			types.StringValue("com.apple.configuration.siri.settings"),
			types.StringNull(),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			diags := validateDeclarationPayload(tc.declarationType, tc.payload, declarationPath(), "")
			if diags.HasError() {
				t.Errorf("unresolved value produced errors: %v", diags.Errors())
			}
		})
	}
}

// TestValidateDeclarationPayloadRejectsMalformedJSON covers the one failure that is not a schema
// finding: a payload that is not a JSON object at all.
func TestValidateDeclarationPayloadRejectsMalformedJSON(t *testing.T) {
	diags := validateDeclarationPayload(
		types.StringValue("com.apple.configuration.siri.settings"),
		types.StringValue(`not json`),
		declarationPath(), "",
	)
	if !diags.HasError() {
		t.Fatal("malformed payload produced no error")
	}
	if summary := diags.Errors()[0].Summary(); !strings.Contains(summary, "not a JSON object") {
		t.Errorf("summary = %q, want it to name the decoding failure", summary)
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "jsonencode") {
		t.Errorf("detail = %q, want it to name jsonencode as the fix", detail)
	}
}

// TestValidateDeclarationPayloadReportsSchemaFindings checks that each finding reaches the caller as
// an error on the payload's own path, with a summary a reader can act on.
func TestValidateDeclarationPayloadReportsSchemaFindings(t *testing.T) {
	cases := []struct {
		name            string
		declarationType string
		payload         string
		wantSummary     string
	}{
		{
			name:            "unknown key",
			declarationType: "com.apple.configuration.siri.settings",
			payload:         `{"ZzNotAKey":true}`,
			wantSummary:     "Unknown key in declaration payload",
		},
		{
			name:            "miscased key",
			declarationType: "com.apple.configuration.siri.settings",
			payload:         `{"forceprofanityfilter":true}`,
			wantSummary:     "Unknown key in declaration payload",
		},
		{
			name:            "unknown declaration type",
			declarationType: "com.apple.configuration.not.a.thing",
			payload:         `{"Enabled":true}`,
			wantSummary:     "Unknown Apple declaration type",
		},
		{
			name:            "wrong value type",
			declarationType: "com.apple.configuration.siri.settings",
			payload:         `{"Enabled":"yes"}`,
			wantSummary:     "Declaration does not match Apple's schema",
		},
		{
			name:            "out of range",
			declarationType: "com.apple.configuration.passcode.settings",
			payload:         `{"MinimumLength":999}`,
			wantSummary:     "Declaration does not match Apple's schema",
		},
		{
			name:            "unknown status item",
			declarationType: "com.apple.configuration.management.status-subscriptions",
			payload:         `{"StatusItems":[{"Name":"device.identifier.not-a-thing"}]}`,
			wantSummary:     "Unknown status item",
		},
	}

	at := declarationPath()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateDeclarationPayload(
				types.StringValue(tc.declarationType), types.StringValue(tc.payload), at, "",
			)
			if !diags.HasError() {
				t.Fatal("no error reported")
			}
			first := diags.Errors()[0]
			if first.Summary() != tc.wantSummary {
				t.Errorf("summary = %q, want %q", first.Summary(), tc.wantSummary)
			}
			if first.Detail() == "" {
				t.Error("detail is empty; the diagnostic would say nothing")
			}
		})
	}
}

// TestRenderDeclarationProblemNamesTheSnapshotOnlyWhenRelevant pins which findings mention the
// schema snapshot. A name the table has never heard of might be a key Apple published after the
// snapshot, so that diagnostic must say where the schemas came from and how to proceed. A wrong
// value type is wrong against every version of the schema, so pointing at the snapshot there would
// only invite someone to dismiss a real mistake.
func TestRenderDeclarationProblemNamesTheSnapshotOnlyWhenRelevant(t *testing.T) {
	cases := map[appledeclarations.ProblemKind]bool{
		appledeclarations.UnknownKey:             true,
		appledeclarations.UnknownDeclarationType: true,
		appledeclarations.UnknownStatusItem:      true,
		appledeclarations.WrongType:              false,
		appledeclarations.OutOfRange:             false,
		appledeclarations.NotInEnum:              false,
		appledeclarations.MissingRequiredKey:     false,
		appledeclarations.MiscasedKey:            false,
	}

	for kind, wantSnapshot := range cases {
		t.Run(kind.String(), func(t *testing.T) {
			_, detail := renderDeclarationProblem(
				appledeclarations.Problem{Kind: kind, Path: "Enabled", Detail: "something is wrong."}, "",
			)
			mentions := strings.Contains(detail, "raw_component")
			if mentions != wantSnapshot {
				t.Errorf("detail mentions the escape hatch = %v, want %v\ndetail: %s", mentions, wantSnapshot, detail)
			}
		})
	}
}

// TestRenderDeclarationProblemAttributesSetElements checks that a finding against a set-backed
// component, where no element index exists to address, still identifies which declaration it came
// from.
func TestRenderDeclarationProblemAttributesSetElements(t *testing.T) {
	_, detail := renderDeclarationProblem(
		appledeclarations.Problem{Kind: appledeclarations.UnknownKey, Path: "Foo", Detail: "unknown."},
		"com.apple.configuration.siri.settings",
	)
	if !strings.Contains(detail, "com.apple.configuration.siri.settings") {
		t.Errorf("detail does not name the declaration: %s", detail)
	}
}

// TestDeclarationValidatorDescription checks the schema description names the upstream release, so
// generated documentation states which schemas a given provider build validates against.
func TestDeclarationValidatorDescription(t *testing.T) {
	description := appleDeclarationsSchemaValidator().MarkdownDescription(context.Background())
	_, release := appledeclarations.Provenance()
	if release != "" && !strings.Contains(description, release) {
		t.Errorf("description %q does not name the upstream release %q", description, release)
	}
}
