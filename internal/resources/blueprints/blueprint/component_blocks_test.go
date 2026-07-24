// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strptr(s string) *string { return &s }

// --- Schema ---

func TestBlueprintResource_SchemaComponentBlocks(t *testing.T) {
	r := NewBlueprintResource()
	var resp resource.SchemaResponse
	r.(*BlueprintResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["component_blocks"]
	if !ok {
		t.Fatal("missing component_blocks attribute")
	}
	if !attr.IsOptional() {
		t.Error("component_blocks should be optional")
	}

	list, ok := attr.(resourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("component_blocks should be a ListNestedAttribute, got %T", attr)
	}

	nested := list.NestedObject.Attributes
	for _, name := range []string{"name", "activation_conditions", "raw_component", "passcode_policy", "software_update_settings", "legacy_payloads"} {
		if _, ok := nested[name]; !ok {
			t.Errorf("component_blocks block missing attribute %q", name)
		}
	}

	if nested["passcode_policy"].GetDeprecationMessage() != "" {
		t.Error("component attributes inside component_blocks must not be deprecated")
	}
}

func TestBlueprintResource_FlatComponentAttrsDeprecated(t *testing.T) {
	r := NewBlueprintResource()
	var resp resource.SchemaResponse
	r.(*BlueprintResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	deprecated := []string{
		"activation_conditions",
		"legacy_payloads",
		"raw_component",
		"audio_accessory_settings",
		"custom_declarations",
		"disk_management_settings",
		"math_settings",
		"passcode_policy",
		"safari_bookmarks",
		"safari_extensions",
		"safari_settings",
		"service_background_tasks",
		"service_configuration_files",
		"software_update",
		"software_update_settings",
	}
	for _, name := range deprecated {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing flat attribute %q", name)
			continue
		}
		if attr.GetDeprecationMessage() == "" {
			t.Errorf("flat attribute %q should carry a DeprecationMessage", name)
		}
	}

	// Blueprint-level attributes must NOT be deprecated.
	for _, name := range []string{"name", "description", "deployed", "device_groups"} {
		if resp.Schema.Attributes[name].GetDeprecationMessage() != "" {
			t.Errorf("blueprint-level attribute %q must not be deprecated", name)
		}
	}
}

func TestBlueprintDataSource_SchemaComponentBlocks(t *testing.T) {
	d := NewBlueprintDataSource()
	var resp datasource.SchemaResponse
	d.(*BlueprintDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["component_blocks"]
	if !ok {
		t.Fatal("data source missing component_blocks attribute")
	}
	if !attr.IsComputed() {
		t.Error("data source component_blocks should be computed")
	}
}

// --- buildSteps (input) ---

func TestBuildSteps_BlockMode(t *testing.T) {
	r := &BlueprintResource{}
	data := &BlueprintResourceModel{
		Name: types.StringValue("BP"),
		ComponentBlocks: []ComponentBlockModel{
			{
				Name:                 types.StringValue("Block A"),
				ActivationConditions: types.StringValue("ANY @property(jamf.device.groups)"),
				Components: []ComponentModel{
					{Identifier: types.StringValue("com.jamf.ddm.passcode-settings"), Configuration: types.MapNull(types.StringType)},
				},
			},
			{
				Name: types.StringValue("Block B"),
				Components: []ComponentModel{
					{Identifier: types.StringValue("com.jamf.ddm.passcode-settings"), Configuration: types.MapNull(types.StringType)},
				},
			},
		},
	}

	steps, diags := r.buildSteps(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}

	if steps[0].Name == nil || *steps[0].Name != "Block A" {
		t.Errorf("expected step 0 name 'Block A', got %v", steps[0].Name)
	}
	if steps[0].ActivationPredicate == nil || *steps[0].ActivationPredicate != "ANY @property(jamf.device.groups)" {
		t.Errorf("expected step 0 activation predicate, got %v", steps[0].ActivationPredicate)
	}
	if steps[1].Name == nil || *steps[1].Name != "Block B" {
		t.Errorf("expected step 1 name 'Block B', got %v", steps[1].Name)
	}
	if steps[1].ActivationPredicate != nil {
		t.Errorf("expected step 1 activation predicate nil, got %v", *steps[1].ActivationPredicate)
	}

	// Same identifier appears in both blocks — the wire allows it and buildSteps must not merge.
	if len(steps[0].Components) != 1 || steps[0].Components[0].Identifier != "com.jamf.ddm.passcode-settings" {
		t.Errorf("expected step 0 to carry the passcode component, got %+v", steps[0].Components)
	}
	if len(steps[1].Components) != 1 || steps[1].Components[0].Identifier != "com.jamf.ddm.passcode-settings" {
		t.Errorf("expected step 1 to carry the passcode component, got %+v", steps[1].Components)
	}
}

func TestBuildSteps_FlatMode(t *testing.T) {
	r := &BlueprintResource{}
	data := &BlueprintResourceModel{
		Name:                 types.StringValue("BP"),
		ActivationConditions: types.StringValue("ANY x"),
		Components: []ComponentModel{
			{Identifier: types.StringValue("com.jamf.ddm.disk-management"), Configuration: types.MapNull(types.StringType)},
		},
	}

	steps, diags := r.buildSteps(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 flat step, got %d", len(steps))
	}
	if steps[0].Name == nil || *steps[0].Name != flatStepName {
		t.Errorf("expected flat step name %q, got %v", flatStepName, steps[0].Name)
	}
	if steps[0].ActivationPredicate == nil || *steps[0].ActivationPredicate != "ANY x" {
		t.Errorf("expected flat step activation predicate 'ANY x', got %v", steps[0].ActivationPredicate)
	}
}

// --- Dual-mode read (state) ---

func TestUpdateModelFromAPIResponse_BlockMode(t *testing.T) {
	ctx := context.Background()
	model := &BlueprintResourceModel{} // empty prior → block mode
	blueprint := &blueprints.BlueprintDetail{
		DeploymentState: &blueprints.DeploymentState{State: "DEPLOYED"},
		Steps: []blueprints.BlueprintStep{
			{
				Name:       strptr("Step 1"),
				Components: []blueprints.Component{{Identifier: "com.jamf.custom.thing", Configuration: json.RawMessage(`{"a":"1"}`)}},
			},
			{
				Name:                strptr("Step 2"),
				ActivationPredicate: strptr("ANY @status(x) == 'y'"),
				Components:          []blueprints.Component{{Identifier: "com.jamf.custom.thing", Configuration: json.RawMessage(`{"a":"2"}`)}},
			},
		},
	}

	diags := updateModelFromAPIResponse(ctx, model, blueprint)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(model.ComponentBlocks) != 2 {
		t.Fatalf("expected 2 component blocks, got %d", len(model.ComponentBlocks))
	}
	if model.ComponentBlocks[0].Name.ValueString() != "Step 1" {
		t.Errorf("expected block 0 name 'Step 1', got %q", model.ComponentBlocks[0].Name.ValueString())
	}
	if !model.ComponentBlocks[0].ActivationConditions.IsNull() {
		t.Errorf("expected block 0 activation null, got %q", model.ComponentBlocks[0].ActivationConditions.ValueString())
	}
	if model.ComponentBlocks[1].ActivationConditions.ValueString() != "ANY @status(x) == 'y'" {
		t.Errorf("expected block 1 activation predicate, got %q", model.ComponentBlocks[1].ActivationConditions.ValueString())
	}
	// Same non-typed identifier lands in each block's raw_component independently.
	for i, block := range model.ComponentBlocks {
		if len(block.Components) != 1 || block.Components[0].Identifier.ValueString() != "com.jamf.custom.thing" {
			t.Errorf("expected block %d raw component 'com.jamf.custom.thing', got %+v", i, block.Components)
		}
	}

	// Flat attributes must be cleared in block mode.
	if model.Components != nil {
		t.Error("expected flat raw_component nil in block mode")
	}
	if !model.ActivationConditions.IsNull() {
		t.Error("expected flat activation_conditions null in block mode")
	}
}

func TestUpdateModelFromAPIResponse_FlatModeMultiStepWarns(t *testing.T) {
	ctx := context.Background()
	model := &BlueprintResourceModel{
		Components: []ComponentModel{
			{Identifier: types.StringValue("com.jamf.ddm.disk-management"), Configuration: types.MapNull(types.StringType)},
		},
	}
	blueprint := &blueprints.BlueprintDetail{
		DeploymentState: &blueprints.DeploymentState{State: "DEPLOYED"},
		Steps: []blueprints.BlueprintStep{
			{Components: []blueprints.Component{{Identifier: "com.jamf.ddm.disk-management", Configuration: json.RawMessage(`{"externalStorage":"deny"}`)}}},
			{Components: []blueprints.Component{{Identifier: "com.jamf.ddm.math-settings", Configuration: json.RawMessage(`{}`)}}},
		},
	}

	diags := updateModelFromAPIResponse(ctx, model, blueprint)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
	if diags.WarningsCount() == 0 {
		t.Fatal("expected a migration warning for a multi-step blueprint in flat mode")
	}
	found := false
	for _, d := range diags.Warnings() {
		if d.Summary() == "Blueprint has multiple component blocks" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Blueprint has multiple component blocks' warning, got %v", diags)
	}
	if model.ComponentBlocks != nil {
		t.Error("flat mode must not populate component_blocks")
	}
}

func TestUpdateModelFromAPIResponse_EmptyPriorUsesBlockMode(t *testing.T) {
	ctx := context.Background()
	model := &BlueprintResourceModel{}
	blueprint := &blueprints.BlueprintDetail{
		DeploymentState: &blueprints.DeploymentState{State: "NOT_DEPLOYED"},
		Steps: []blueprints.BlueprintStep{
			{Name: strptr("Only Block"), Components: []blueprints.Component{{Identifier: "com.jamf.custom.x", Configuration: json.RawMessage(`{"k":"v"}`)}}},
		},
	}

	updateModelFromAPIResponse(ctx, model, blueprint)

	if len(model.ComponentBlocks) != 1 {
		t.Fatalf("expected empty prior to read as one block, got %d blocks", len(model.ComponentBlocks))
	}
	if model.Components != nil {
		t.Error("expected flat raw_component nil when reading in block mode")
	}
}
