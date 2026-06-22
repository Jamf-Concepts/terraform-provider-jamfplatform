// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestPatchPolicyResource_Metadata(t *testing.T) {
	r := NewPatchPolicyResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PatchPolicyResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_policy" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_patch_policy", resp.TypeName)
	}
}

func TestPatchPolicyResource_Schema(t *testing.T) {
	r := NewPatchPolicyResource()
	var resp resource.SchemaResponse
	r.(*PatchPolicyResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := []string{
		"id", "software_title_configuration_id", "name", "enabled", "target_version",
		"distribution_method", "allow_downgrade", "patch_unknown", "release_date",
		"incremental_update", "reboot", "minimum_os", "kill_apps", "scope",
		"user_interaction", "timeouts",
	}
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// id is computed-only.
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	// Required general fields.
	for _, req := range []string{"software_title_configuration_id", "name", "target_version"} {
		if !s.Attributes[req].IsRequired() {
			t.Errorf("%q must be required", req)
		}
	}

	// Writable Optional+Computed general fields.
	for _, oc := range []string{"enabled", "distribution_method", "allow_downgrade", "patch_unknown"} {
		a := s.Attributes[oc]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed, got optional=%v computed=%v", oc, a.IsOptional(), a.IsComputed())
		}
	}

	// Server-derived Computed-only general fields.
	for _, c := range []string{"release_date", "incremental_update", "reboot", "minimum_os", "kill_apps"} {
		a := s.Attributes[c]
		if a.IsOptional() || a.IsRequired() || !a.IsComputed() {
			t.Errorf("%q must be computed-only, got optional=%v required=%v computed=%v", c, a.IsOptional(), a.IsRequired(), a.IsComputed())
		}
	}

	// scope and user_interaction are Optional-only blocks (NOT Optional+Computed:
	// the server echoes a full default superset, which trips the framework's
	// Unknown-decode at apply when an Optional+Computed *struct block is used).
	for _, b := range []string{"scope", "user_interaction"} {
		a := s.Attributes[b]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", b)
		}
		if a.IsComputed() {
			t.Errorf("%q must NOT be computed (Optional-only state-gated block)", b)
		}
	}

	// software_title_configuration_id is RequiresReplace.
	stcID, ok := s.Attributes["software_title_configuration_id"].(rschema.StringAttribute)
	if !ok {
		t.Fatalf("software_title_configuration_id is not a StringAttribute")
	}
	if len(stcID.PlanModifiers) == 0 {
		t.Errorf("software_title_configuration_id must carry a RequiresReplace plan modifier")
	}

	// distribution_method validator: OneOf{selfservice, prompt}.
	dm, ok := s.Attributes["distribution_method"].(rschema.StringAttribute)
	if !ok {
		t.Fatalf("distribution_method is not a StringAttribute")
	}
	if len(dm.Validators) == 0 {
		t.Errorf("distribution_method must carry a OneOf validator")
	}

	// kill_apps nested object exposes kill_app_name + kill_app_bundle_id.
	ka, ok := s.Attributes["kill_apps"].(rschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("kill_apps is not a ListNestedAttribute")
	}
	for _, n := range []string{"kill_app_name", "kill_app_bundle_id"} {
		if _, ok := ka.NestedObject.Attributes[n]; !ok {
			t.Errorf("kill_apps missing nested attribute %q", n)
		}
	}

	// scope exposes the limited targets/limitations/exclusions set; no users.
	sc, ok := s.Attributes["scope"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("scope is not a SingleNestedAttribute")
	}
	for _, n := range []string{"targets", "limitations", "exclusions"} {
		if _, ok := sc.Attributes[n]; !ok {
			t.Errorf("scope missing nested attribute %q", n)
		}
	}

	// targets holds the all-flag + per-category target ID sets (admin UI Targets tab).
	tgt, ok := sc.Attributes["targets"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("scope.targets is not a SingleNestedAttribute")
	}
	for _, n := range []string{"all_computers", "computer_ids", "computer_group_ids", "building_ids", "department_ids"} {
		if _, ok := tgt.Attributes[n]; !ok {
			t.Errorf("scope.targets missing nested attribute %q", n)
		}
	}
	for _, forbidden := range []string{"users", "user_ids", "user_group_ids", "directory_service_user_group_names"} {
		if _, ok := tgt.Attributes[forbidden]; ok {
			t.Errorf("scope.targets must NOT model user-based attribute %q", forbidden)
		}
	}

	lim, ok := sc.Attributes["limitations"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("scope.limitations is not a SingleNestedAttribute")
	}
	for _, n := range []string{"network_segment_ids", "ibeacon_ids"} {
		if _, ok := lim.Attributes[n]; !ok {
			t.Errorf("scope.limitations missing nested attribute %q", n)
		}
	}

	excl, ok := sc.Attributes["exclusions"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("scope.exclusions is not a SingleNestedAttribute")
	}
	for _, n := range []string{"computer_ids", "computer_group_ids", "building_ids", "department_ids", "network_segment_ids", "ibeacon_ids"} {
		if _, ok := excl.Attributes[n]; !ok {
			t.Errorf("scope.exclusions missing nested attribute %q", n)
		}
	}

	// user_interaction nested shape.
	ui, ok := s.Attributes["user_interaction"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("user_interaction is not a SingleNestedAttribute")
	}
	for _, n := range []string{"install_button_text", "self_service_description", "self_service_icon_id", "notifications", "deadlines", "grace_period"} {
		if _, ok := ui.Attributes[n]; !ok {
			t.Errorf("user_interaction missing nested attribute %q", n)
		}
	}
	notif, ok := ui.Attributes["notifications"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("user_interaction.notifications is not a SingleNestedAttribute")
	}
	if _, ok := notif.Attributes["reminders"]; !ok {
		t.Errorf("user_interaction.notifications missing reminders")
	}
}

func TestPatchPolicyDataSource_Metadata(t *testing.T) {
	d := NewPatchPolicyDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*PatchPolicyDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_policy" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_patch_policy", resp.TypeName)
	}
}

func TestPatchPolicyDataSource_Schema(t *testing.T) {
	d := NewPatchPolicyDataSource()
	var resp datasource.SchemaResponse
	d.(*PatchPolicyDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	if id := s.Attributes["id"]; !id.IsRequired() {
		t.Errorf("data source id must be required (sole selector)")
	}

	for _, c := range []string{
		"software_title_configuration_id", "name", "enabled", "target_version",
		"distribution_method", "allow_downgrade", "patch_unknown", "release_date",
		"incremental_update", "reboot", "minimum_os", "kill_apps",
	} {
		a, ok := s.Attributes[c]
		if !ok {
			t.Errorf("missing attribute %q", c)
			continue
		}
		if a.IsOptional() || a.IsRequired() || !a.IsComputed() {
			t.Errorf("%q must be computed-only, got optional=%v required=%v computed=%v", c, a.IsOptional(), a.IsRequired(), a.IsComputed())
		}
	}

	if _, ok := s.Attributes["kill_apps"].(dsschema.ListNestedAttribute); !ok {
		t.Errorf("data source kill_apps is not a ListNestedAttribute")
	}
}

func TestPatchPolicyListResource_Metadata(t *testing.T) {
	r := NewPatchPolicyListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PatchPolicyListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_policy" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_patch_policy", resp.TypeName)
	}
}

func TestPatchPolicyListResource_Schema(t *testing.T) {
	r := NewPatchPolicyListResource()
	var resp list.ListResourceSchemaResponse
	r.(*PatchPolicyListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
