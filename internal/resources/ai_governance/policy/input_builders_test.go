// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// planFixture is a fully-populated plan model.
func planFixture() *policyModel {
	return &policyModel{
		Name:          types.StringValue("Claude Code — Engineering"),
		Description:   types.StringValue("Managed settings for engineering"),
		ToolID:        types.StringValue("com.anthropic.claudecode"),
		SchemaVersion: types.StringValue("2026-08-14"),
		SettingsJSON:  newJSONObjectValue(`{"model":"sonnet"}`),
		Publish:       types.BoolValue(true),
	}
}

func TestBuildCreateRequest(t *testing.T) {
	got := buildCreateRequest(planFixture())

	if got.Name != "Claude Code — Engineering" {
		t.Errorf("name = %q", got.Name)
	}
	if got.Description == nil || *got.Description != "Managed settings for engineering" {
		t.Errorf("description = %v", got.Description)
	}
	if got.ToolID != "com.anthropic.claudecode" {
		t.Errorf("toolId = %q", got.ToolID)
	}
	if got.SchemaVersion != "2026-08-14" {
		t.Errorf("schemaVersion = %q", got.SchemaVersion)
	}
	if string(got.Settings) != `{"model":"sonnet"}` {
		t.Errorf("settings = %q", got.Settings)
	}
}

// TestBuildUpdateRequestAlwaysSendsName pins that a name is written on every update. The wire treats
// an omitted name as "leave unchanged", which would silently preserve a name the operator changed
// in a way Terraform could not then reconcile.
func TestBuildUpdateRequestAlwaysSendsName(t *testing.T) {
	got := buildUpdateRequest(planFixture())
	if got.Name == nil || *got.Name != "Claude Code — Engineering" {
		t.Fatalf("name = %v, want it always sent", got.Name)
	}
	if got.SchemaVersion == "" {
		t.Error("schemaVersion is mandatory on every update")
	}
	if len(got.Settings) == 0 {
		t.Error("settings are mandatory on every update")
	}
}

// TestUnsetDescriptionIsOmittedOnCreateAndBlankedOnUpdate pins the two halves of the description
// contract. A create has nothing to preserve, so an unset description is omitted. An update is a
// PATCH on which both an absent key and a JSON null mean "leave unchanged", so an unset description
// has to be sent as the explicit blank the platform clears on — dropping it preserves the old value,
// which Terraform then rejects as an inconsistent result and cannot plan its way out of.
func TestUnsetDescriptionIsOmittedOnCreateAndBlankedOnUpdate(t *testing.T) {
	plan := planFixture()
	plan.Description = types.StringNull()

	if got := buildCreateRequest(plan); got.Description != nil {
		t.Errorf("create description = %q, want it omitted", *got.Description)
	}

	got := buildUpdateRequest(plan)
	if got.Description == nil {
		t.Fatal("update description was omitted, which preserves the stored value instead of clearing it")
	}
	if *got.Description != "" {
		t.Errorf("update description = %q, want an explicit blank", *got.Description)
	}
}

// TestConfiguredDescriptionIsSentVerbatim pins that the blank only stands in for an unset value.
func TestConfiguredDescriptionIsSentVerbatim(t *testing.T) {
	got := buildUpdateRequest(planFixture())
	if got.Description == nil || *got.Description != "Managed settings for engineering" {
		t.Errorf("update description = %v, want the configured value", got.Description)
	}
}

// TestSettingsPayloadFallsBackToAnEmptyObject pins that an unset settings value never becomes a null
// body: the platform refuses a null settings object outright.
func TestSettingsPayloadFallsBackToAnEmptyObject(t *testing.T) {
	plan := planFixture()
	plan.SettingsJSON = newJSONObjectNull()
	if got := string(settingsPayload(plan)); got != "{}" {
		t.Errorf("settings = %q, want {}", got)
	}
}
