// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// planreportStub is a minimal in-memory Source for the ReportPlan lifecycle
// tests. It is deliberately separate from the stub in impact_test.go so these
// tests do not depend on that file's fixtures.
type planreportStub struct {
	groups  []Group
	totals  Totals
	members map[string][]string
}

func (s *planreportStub) Groups(context.Context) ([]Group, error) { return s.groups, nil }
func (s *planreportStub) Totals(context.Context) (Totals, error)  { return s.totals, nil }

func (s *planreportStub) Members(_ context.Context, platformID string) ([]string, error) {
	ids, ok := s.members[platformID]
	if !ok {
		return nil, errors.New("membership unavailable")
	}
	return ids, nil
}

func (s *planreportStub) ComputerManagementIDs(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *planreportStub) MobileManagementIDs(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *planreportStub) PlaceNames(context.Context) (Places, error) { return Places{}, nil }

// planreportIDs builds a contiguous run of synthetic management ids.
func planreportIDs(from, count int) []string {
	out := make([]string, 0, count)
	for i := range count {
		out = append(out, fmt.Sprintf("pr-%d", from+i))
	}
	return out
}

// planreportSource holds two computer groups with distinct sizes, so a test can
// tell from the reported figure which side of the plan the scope was read from.
func planreportSource() *planreportStub {
	return &planreportStub{
		totals: Totals{ManagedComputers: 100},
		groups: []Group{
			{PlatformID: "uuid-pr-a", JamfProID: "12", Name: "Marketing", DeviceType: DeviceTypeComputer, Smart: true, MembershipCount: 30},
			{PlatformID: "uuid-pr-b", JamfProID: "13", Name: "Lab Macs", DeviceType: DeviceTypeComputer, Smart: false, MembershipCount: 5},
		},
		members: map[string][]string{
			"uuid-pr-a": planreportIDs(1, 30),
			"uuid-pr-b": planreportIDs(500, 5),
		},
	}
}

// planreportModel is a trivial resource model: one attribute driving the scope
// and one payload attribute so a payload-only diff can be expressed.
type planreportModel struct {
	GroupID types.String `tfsdk:"group_id"`
	Payload types.String `tfsdk:"payload"`
}

func planreportSchema() rschema.Schema {
	return rschema.Schema{
		Attributes: map[string]rschema.Attribute{
			"group_id": rschema.StringAttribute{Optional: true},
			"payload":  rschema.StringAttribute{Optional: true},
		},
	}
}

var planreportObjectType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"group_id": tftypes.String,
		"payload":  tftypes.String,
	},
}

// planreportRaw builds a fully-known raw object value for the trivial schema.
func planreportRaw(groupID, payload string) tftypes.Value {
	return tftypes.NewValue(planreportObjectType, map[string]tftypes.Value{
		"group_id": tftypes.NewValue(tftypes.String, groupID),
		"payload":  tftypes.NewValue(tftypes.String, payload),
	})
}

// planreportNullRaw is the null object a plan carries for the missing side of a
// create (no prior state) or a delete (no planned configuration).
func planreportNullRaw() tftypes.Value {
	return tftypes.NewValue(planreportObjectType, nil)
}

// planreportExtract reduces the trivial model to a computer scope naming the one
// group the model carries.
func planreportExtract(_ context.Context, m *planreportModel) Scope {
	s := Scope{DeviceType: DeviceTypeComputer}
	if !m.GroupID.IsNull() && !m.GroupID.IsUnknown() && m.GroupID.ValueString() != "" {
		s.ProGroups = []ProGroupRef{{DeviceType: DeviceTypeComputer, ID: m.GroupID.ValueString()}}
	}
	return s
}

// runReportPlan drives the shared ModifyPlan hook with real framework request
// and response values, exactly as a resource's ModifyPlan would.
func runReportPlan(t *testing.T, cache *Cache, state, plan tftypes.Value) diag.Diagnostics {
	t.Helper()
	sch := planreportSchema()
	req := resource.ModifyPlanRequest{
		State: tfsdk.State{Schema: sch, Raw: state},
		Plan:  tfsdk.Plan{Schema: sch, Raw: plan},
	}
	resp := &resource.ModifyPlanResponse{}
	ReportPlan(context.Background(), req, resp, PlanReport{
		Cache: cache,
		Path:  path.Root("group_id"),
		Label: "test object",
	}, planreportExtract)
	return resp.Diagnostics
}

func TestReportPlanIdenticalPlanIsSilent(t *testing.T) {
	// Terraform calls ModifyPlan for every resource in the configuration, so an
	// object whose plan matches its state exactly must produce nothing.
	c := NewCache(planreportSource())
	same := planreportRaw("12", "a")
	if diags := runReportPlan(t, c, same, same); len(diags) != 0 {
		t.Fatalf("an unchanged resource must not alert, got %d: %v", len(diags), diags)
	}
}

func TestReportPlanPayloadOnlyChangeAlerts(t *testing.T) {
	// The hook's Changed flag is derived from the raw values, not from the scope:
	// a payload edit reaches every device in scope even though the audience is
	// identical. Hardcoding Changed to false silences exactly this case.
	c := NewCache(planreportSource())
	diags := runReportPlan(t, c, planreportRaw("12", "a"), planreportRaw("12", "b"))
	if len(diags) != 1 {
		t.Fatalf("a payload-only change must alert exactly once, got %d: %v", len(diags), diags)
	}
	if diags[0].Severity() != diag.SeverityWarning {
		t.Fatalf("impact alerts are advisory, got severity %v", diags[0].Severity())
	}
	if s := diags[0].Summary(); !strings.Contains(s, "30 of 100 computers") {
		t.Fatalf("the alert must carry the audience figure: %q", s)
	}
}

func TestReportPlanScopeChangeAlertsOnce(t *testing.T) {
	c := NewCache(planreportSource())
	diags := runReportPlan(t, c, planreportRaw("13", "a"), planreportRaw("12", "a"))
	if len(diags) != 1 {
		t.Fatalf("a scope change must alert exactly once, got %d: %v", len(diags), diags)
	}
}

func TestReportPlanCreateUsesThePlannedScope(t *testing.T) {
	// A null prior state is a create: the planned configuration is the only side
	// that exists, and the figure must come from it — group 12 holds 30, so a
	// figure of 5 would mean the hook read the wrong side.
	c := NewCache(planreportSource())
	diags := runReportPlan(t, c, planreportNullRaw(), planreportRaw("12", "a"))
	if len(diags) != 1 {
		t.Fatalf("a create must alert exactly once, got %d: %v", len(diags), diags)
	}
	s := diags[0].Summary()
	if !strings.Contains(s, "will be scoped to") {
		t.Fatalf("a create must use the create wording: %q", s)
	}
	if !strings.Contains(s, "30 of 100 computers") {
		t.Fatalf("the figure must come from the planned scope: %q", s)
	}
}

func TestReportPlanDeleteUsesThePriorScope(t *testing.T) {
	// A null planned value is a delete: the prior state is the only side that
	// exists. Group 13 holds 5, so a figure of 30 would mean the hook swapped the
	// create and delete detection.
	c := NewCache(planreportSource())
	diags := runReportPlan(t, c, planreportRaw("13", "a"), planreportNullRaw())
	if len(diags) != 1 {
		t.Fatalf("a delete must alert exactly once, got %d: %v", len(diags), diags)
	}
	s := diags[0].Summary()
	if !strings.Contains(s, "removing") {
		t.Fatalf("a delete must use the delete wording: %q", s)
	}
	if !strings.Contains(s, "5 of 100 computers") {
		t.Fatalf("the figure must come from the prior scope: %q", s)
	}
}

func TestReportPlanBothSidesNullIsSilent(t *testing.T) {
	c := NewCache(planreportSource())
	if diags := runReportPlan(t, c, planreportNullRaw(), planreportNullRaw()); len(diags) != 0 {
		t.Fatalf("a resource with neither state nor plan must not alert, got %d: %v", len(diags), diags)
	}
}

func TestReportPlanNilCacheIsSilent(t *testing.T) {
	// A nil cache is the disabled state: resources wire the hook unconditionally
	// and rely on this, so a diff on any lifecycle path must produce nothing.
	if diags := runReportPlan(t, nil, planreportRaw("13", "a"), planreportRaw("12", "b")); len(diags) != 0 {
		t.Fatalf("a nil cache must disable reporting, got %d: %v", len(diags), diags)
	}
	if diags := runReportPlan(t, nil, planreportNullRaw(), planreportRaw("12", "a")); len(diags) != 0 {
		t.Fatalf("a nil cache must disable reporting on creates too, got %d: %v", len(diags), diags)
	}
}
