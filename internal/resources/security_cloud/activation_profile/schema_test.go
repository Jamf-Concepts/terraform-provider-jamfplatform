// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package activation_profile

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// resourceSchema builds the resource schema for inspection.
func resourceSchema(t *testing.T) schema.Schema {
	t.Helper()
	r := &ActivationProfileResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// TestSchema_AttributeSet pins the attribute surface, so an addition or removal is
// a deliberate change rather than a surprise.
func TestSchema_AttributeSet(t *testing.T) {
	s := resourceSchema(t)
	want := []string{"id", "name", "platforms", "capabilities", "device_group_id", "paused", "timeouts"}
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if got, wantLen := len(s.Attributes), len(want); got != wantLen {
		t.Errorf("schema has %d attributes, expected %d: %v", got, wantLen, s.Attributes)
	}
}

// TestSchema_RequiredAndComputed checks the required set, and that the activation
// code is both server-minted and treated as a credential: holding it is enough to
// enrol a device, so `id` must be Sensitive as well as Computed-only.
func TestSchema_RequiredAndComputed(t *testing.T) {
	s := resourceSchema(t)
	for _, name := range []string{"name", "platforms", "capabilities"} {
		if !s.Attributes[name].IsRequired() {
			t.Errorf("%q should be required", name)
		}
	}
	for _, name := range []string{"device_group_id", "paused"} {
		if !s.Attributes[name].IsOptional() {
			t.Errorf("%q should be optional", name)
		}
	}
	if !s.Attributes["id"].IsComputed() || s.Attributes["id"].IsRequired() || s.Attributes["id"].IsOptional() {
		t.Error("id should be computed-only — Jamf Security Cloud mints the activation code")
	}
	if !s.Attributes["id"].IsSensitive() {
		t.Error("id should be sensitive — the activation code on its own is enough to enrol a device")
	}
}

// TestSchema_EveryConfiguredAttributeRequiresReplace is the structural expression
// of this resource's central constraint.
//
// Jamf Security Cloud has no update endpoint for an activation profile and returns
// only the activation code when one is read, so nothing configured can be changed
// in place or refreshed. `paused` is the one exception: the pause and resume
// operations change it without replacing the profile.
//
// The assertion is behavioural rather than nominal: each modifier is invoked with
// a request representing an update to an existing resource — non-null state, plan
// and config, and a prior value differing from the planned one — and must answer
// that replacement is required. A presence check (`len(PlanModifiers) != 0`) would
// stay green if `RequiresReplace()` were swapped for any other modifier, and since
// every Jamf Security Cloud acceptance test skips in CI, this is CI's only gate.
// The modifier counts are exact for the same reason: an added modifier should be a
// deliberate change, not a silent one.
func TestSchema_EveryConfiguredAttributeRequiresReplace(t *testing.T) {
	ctx := context.Background()
	s := resourceSchema(t)
	raw := existingResourceRaw(ctx, s)
	state := tfsdk.State{Schema: s, Raw: raw}
	plan := tfsdk.Plan{Schema: s, Raw: raw}
	config := tfsdk.Config{Schema: s, Raw: raw}

	for _, name := range []string{"name", "device_group_id"} {
		attribute, ok := s.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Errorf("%q is not a StringAttribute", name)
			continue
		}
		if got := len(attribute.PlanModifiers); got != 1 {
			t.Errorf("%q has %d plan modifiers, expected exactly 1 (RequiresReplace)", name, got)
		}
		for i, modifier := range attribute.PlanModifiers {
			resp := &planmodifier.StringResponse{}
			modifier.PlanModifyString(ctx, planmodifier.StringRequest{
				Path:        path.Root(name),
				Config:      config,
				State:       state,
				Plan:        plan,
				ConfigValue: types.StringValue("after"),
				StateValue:  types.StringValue("before"),
				PlanValue:   types.StringValue("after"),
			}, resp)
			if !resp.RequiresReplace {
				t.Errorf("%q plan modifier %d does not require replacement when the value changes; expected RequiresReplace", name, i)
			}
		}
	}

	platforms, ok := s.Attributes["platforms"].(schema.SetAttribute)
	if !ok {
		t.Fatal("platforms is not a SetAttribute")
	}
	if got := len(platforms.PlanModifiers); got != 1 {
		t.Errorf("platforms has %d plan modifiers, expected exactly 1 (RequiresReplace)", got)
	}
	priorSet := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("ios")})
	plannedSet := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("mac")})
	for i, modifier := range platforms.PlanModifiers {
		resp := &planmodifier.SetResponse{}
		modifier.PlanModifySet(ctx, planmodifier.SetRequest{
			Path:        path.Root("platforms"),
			Config:      config,
			State:       state,
			Plan:        plan,
			ConfigValue: plannedSet,
			StateValue:  priorSet,
			PlanValue:   plannedSet,
		}, resp)
		if !resp.RequiresReplace {
			t.Errorf("platforms plan modifier %d does not require replacement when the value changes; expected RequiresReplace", i)
		}
	}

	capabilities, ok := s.Attributes["capabilities"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("capabilities is not a SingleNestedAttribute")
	}
	if got := len(capabilities.PlanModifiers); got != 1 {
		t.Errorf("capabilities has %d plan modifiers, expected exactly 1 (RequiresReplace)", got)
	}
	priorObject := capabilityObject(t, capabilities, false)
	plannedObject := capabilityObject(t, capabilities, true)
	for i, modifier := range capabilities.PlanModifiers {
		resp := &planmodifier.ObjectResponse{}
		modifier.PlanModifyObject(ctx, planmodifier.ObjectRequest{
			Path:        path.Root("capabilities"),
			Config:      config,
			State:       state,
			Plan:        plan,
			ConfigValue: plannedObject,
			StateValue:  priorObject,
			PlanValue:   plannedObject,
		}, resp)
		if !resp.RequiresReplace {
			t.Errorf("capabilities plan modifier %d does not require replacement when the value changes; expected RequiresReplace", i)
		}
	}

	paused, ok := s.Attributes["paused"].(schema.BoolAttribute)
	if !ok {
		t.Fatal("paused is not a BoolAttribute")
	}
	if len(paused.PlanModifiers) != 0 {
		t.Error("paused must not require replacement — pause and resume change it in place")
	}
}

// existingResourceRaw builds the raw object of an already-created resource: every
// attribute null, but the object itself known. The framework's RequiresReplace
// modifiers read nothing else from a request's State and Plan — they no-op when
// either raw value is null, which is how they recognise create and destroy — so a
// known object is what turns the request under test into an update.
func existingResourceRaw(ctx context.Context, s schema.Schema) tftypes.Value {
	objectType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		return tftypes.Value{}
	}
	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	return tftypes.NewValue(objectType, values)
}

// capabilityObject builds a capabilities object with every boolean set to enabled
// and every other attribute null, so two calls differing in `enabled` give the
// changed prior-versus-planned pair the modifier is asked about. Attribute types
// come from the schema, so an added capability fails here rather than being
// silently excluded from the comparison.
func capabilityObject(t *testing.T, capabilities schema.SingleNestedAttribute, enabled bool) types.Object {
	t.Helper()
	objectType, ok := capabilities.GetType().(types.ObjectType)
	if !ok {
		t.Fatal("capabilities type is not an ObjectType")
	}
	attributeTypes := objectType.AttributeTypes()
	values := make(map[string]attr.Value, len(attributeTypes))
	for name, attributeType := range attributeTypes {
		if attributeType.Equal(types.BoolType) {
			values[name] = types.BoolValue(enabled)
			continue
		}
		values[name] = nullValue(t, attributeType)
	}
	object, diags := types.ObjectValue(attributeTypes, values)
	if diags.HasError() {
		t.Fatalf("building a capabilities object: %v", diags)
	}
	return object
}

// nullValue returns the null value of an attribute type, so a request can be
// built from the schema's own types instead of a hand-written list.
func nullValue(t *testing.T, attributeType attr.Type) attr.Value {
	t.Helper()
	ctx := context.Background()
	value, err := attributeType.ValueFromTerraform(ctx, tftypes.NewValue(attributeType.TerraformType(ctx), nil))
	if err != nil {
		t.Fatalf("building a null %T: %v", attributeType, err)
	}
	return value
}

// TestSchema_CapabilityAttributes pins the capability surface, including that the
// coupled wire pair is modelled as one attribute.
func TestSchema_CapabilityAttributes(t *testing.T) {
	s := resourceSchema(t)
	capabilities, ok := s.Attributes["capabilities"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("capabilities is not a SingleNestedAttribute")
	}
	want := []string{"content_controls", "network_security", "note"}
	for _, name := range want {
		if _, ok := capabilities.Attributes[name]; !ok {
			t.Errorf("missing capability attribute %q", name)
		}
	}
	if got := len(capabilities.Attributes); got != len(want) {
		t.Errorf("capabilities has %d attributes, expected %d — networkSecurity and vulnerabilityManagement are one attribute here, not two", got, len(want))
	}
	if _, exists := capabilities.Attributes["vulnerability_management"]; exists {
		t.Error("vulnerability_management should not be a separate attribute: the server refuses it disagreeing with network_security, and the console shows one checkbox")
	}
}

// TestSchema_ImportIsNotSupported pins a deliberate omission.
//
// Jamf Security Cloud returns only the activation code, so an imported profile
// would carry null for every RequiresReplace attribute and the next plan would
// replace — destroying the profile it had just adopted.
func TestSchema_ImportIsNotSupported(t *testing.T) {
	var r any = &ActivationProfileResource{}
	if _, ok := r.(resource.ResourceWithImportState); ok {
		t.Error("resource implements ResourceWithImportState; import is deliberately unsupported because a GET returns only the activation code")
	}
}

// wireJargon matches protocol vocabulary that must not reach the Terraform
// Registry, per STYLE_GUIDE §User-facing descriptions are UI-aligned. Product
// framing ("Jamf Security Cloud refuses…") is fine; protocol framing is not.
var wireJargon = regexp.MustCompile(`(?i)\b(api|endpoint|wire|payload|sdk|http|PUT|POST|DELETE|GET|/v1/|json|4\d\d|5\d\d)\b`)

// TestSchema_DescriptionsAreUIAligned keeps wire vocabulary out of every
// user-facing description, including the nested ones.
func TestSchema_DescriptionsAreUIAligned(t *testing.T) {
	s := resourceSchema(t)
	check := func(label, text string) {
		if text == "" {
			t.Errorf("%s has an empty description", label)
			return
		}
		if match := wireJargon.FindString(text); match != "" {
			t.Errorf("%s description contains wire vocabulary %q: %s", label, match, text)
		}
	}
	check("resource", stripPrivilegeSection(s.MarkdownDescription))
	for name, attribute := range s.Attributes {
		if name == "timeouts" {
			continue
		}
		check(name, attribute.GetMarkdownDescription())
	}
	capabilities, ok := s.Attributes["capabilities"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("capabilities is not a SingleNestedAttribute")
	}
	for name, attribute := range capabilities.Attributes {
		check("capabilities."+name, attribute.GetMarkdownDescription())
	}
}

// stripPrivilegeSection removes the generated "Required Jamf permissions" table,
// which legitimately names privileges rather than describing the resource.
func stripPrivilegeSection(text string) string {
	if before, _, ok := strings.Cut(text, "Required Jamf permissions"); ok {
		return before
	}
	return text
}
