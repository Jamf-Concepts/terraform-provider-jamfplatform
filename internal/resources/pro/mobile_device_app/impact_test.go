// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
)

// impactStubSource is a minimal in-memory impact.Source. It deliberately holds
// the same numeric group id in BOTH estates with different sizes: Jamf Pro
// numbers computer groups and mobile device groups independently, so an adapter
// that tagged this resource's scope with the wrong estate would resolve id 66 to
// the computer group and report 40 computers instead of 2 mobile devices.
type impactStubSource struct{}

func (impactStubSource) Groups(context.Context) ([]impact.Group, error) {
	return []impact.Group{
		{PlatformID: "uuid-ipads", JamfProID: "66", Name: "Some iPads", DeviceType: impact.DeviceTypeMobile, Smart: true, MembershipCount: 2},
		{PlatformID: "uuid-macs", JamfProID: "66", Name: "Some Macs", DeviceType: impact.DeviceTypeComputer, Smart: true, MembershipCount: 40},
	}, nil
}

func (impactStubSource) Totals(context.Context) (impact.Totals, error) {
	return impact.Totals{ManagedComputers: 50, ManagedMobileDevices: 8}, nil
}

func (impactStubSource) Members(_ context.Context, platformID string) ([]string, error) {
	if platformID == "uuid-ipads" {
		return []string{"m-1", "m-2"}, nil
	}
	return nil, nil
}

func (impactStubSource) ComputerManagementIDs(context.Context, string) ([]string, error) {
	return nil, nil
}

func (impactStubSource) MobileManagementIDs(context.Context, string) ([]string, error) {
	return nil, nil
}

func (impactStubSource) PlaceNames(context.Context) (impact.Places, error) {
	return impact.Places{}, nil
}

// mobileAppSchema resolves the resource's real schema, so the test exercises the
// same model decode the plan hook performs in production.
func mobileAppSchema(t *testing.T) rschema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	(&MobileAppResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// objectWith builds a raw object value with every attribute null except the
// overrides, deriving the attribute types from the schema so the fixture never
// drifts from the real resource shape.
func objectWith(t *testing.T, typ tftypes.Type, overrides map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	obj, ok := typ.(tftypes.Object)
	if !ok {
		t.Fatalf("expected an object type, got %T", typ)
	}
	vals := make(map[string]tftypes.Value, len(obj.AttributeTypes))
	for name, at := range obj.AttributeTypes {
		if v, found := overrides[name]; found {
			vals[name] = v
			continue
		}
		vals[name] = tftypes.NewValue(at, nil)
	}
	return tftypes.NewValue(obj, vals)
}

// mobileAppPlanRaw builds a raw plan value that is null everywhere except one
// mobile device group target in the scope block.
func mobileAppPlanRaw(t *testing.T, root tftypes.Object, groupID string) tftypes.Value {
	t.Helper()
	scopeType, ok := root.AttributeTypes["scope"].(tftypes.Object)
	if !ok {
		t.Fatalf("scope attribute type: %T", root.AttributeTypes["scope"])
	}
	targetsType, ok := scopeType.AttributeTypes["targets"].(tftypes.Object)
	if !ok {
		t.Fatalf("targets attribute type: %T", scopeType.AttributeTypes["targets"])
	}
	groupsType := targetsType.AttributeTypes["mobile_device_group_ids"]
	groups := tftypes.NewValue(groupsType, []tftypes.Value{
		tftypes.NewValue(tftypes.String, groupID),
	})
	targets := objectWith(t, targetsType, map[string]tftypes.Value{"mobile_device_group_ids": groups})
	scopeVal := objectWith(t, scopeType, map[string]tftypes.Value{"targets": targets})
	return objectWith(t, root, map[string]tftypes.Value{"scope": scopeVal})
}

func TestMobileAppImpactReportsTheMobileEstate(t *testing.T) {
	ctx := context.Background()
	sch := mobileAppSchema(t)
	root, ok := sch.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("schema type: %T", sch.Type().TerraformType(ctx))
	}

	r := &MobileAppResource{impact: impact.NewCache(impactStubSource{})}
	req := resource.ModifyPlanRequest{
		State: tfsdk.State{Schema: sch, Raw: tftypes.NewValue(root, nil)},
		Plan:  tfsdk.Plan{Schema: sch, Raw: mobileAppPlanRaw(t, root, "66")},
	}
	resp := &resource.ModifyPlanResponse{}
	r.reportScopeImpact(ctx, req, resp)

	if len(resp.Diagnostics) != 1 {
		t.Fatalf("expected one impact alert, got %d: %v", len(resp.Diagnostics), resp.Diagnostics)
	}
	if resp.Diagnostics[0].Severity() != diag.SeverityWarning {
		t.Fatalf("impact alerts are advisory, got severity %v", resp.Diagnostics[0].Severity())
	}
	s := resp.Diagnostics[0].Summary()
	// Group id 66 exists in both estates in the fixture. Only the mobile-estate
	// adapter reaches "2 of 8 mobile devices"; a computer-estate conversion would
	// resolve the colliding computer group and report 40 of 50 computers.
	if !strings.Contains(s, "2 of 8 mobile devices") {
		t.Fatalf("the figure must come from the mobile estate: %q", s)
	}
	if strings.Contains(s, "computers") {
		t.Fatalf("a mobile device app must never be measured in computers: %q", s)
	}
}

func TestMobileAppImpactAbsentScopeIsSilent(t *testing.T) {
	// A create with no scope block converts a nil *MobileScopeModelNoIbeacons,
	// which must neither panic nor alert.
	ctx := context.Background()
	sch := mobileAppSchema(t)
	root, ok := sch.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("schema type: %T", sch.Type().TerraformType(ctx))
	}

	r := &MobileAppResource{impact: impact.NewCache(impactStubSource{})}
	req := resource.ModifyPlanRequest{
		State: tfsdk.State{Schema: sch, Raw: tftypes.NewValue(root, nil)},
		Plan:  tfsdk.Plan{Schema: sch, Raw: objectWith(t, root, nil)},
	}
	resp := &resource.ModifyPlanResponse{}
	r.reportScopeImpact(ctx, req, resp)

	if len(resp.Diagnostics) != 0 {
		t.Fatalf("a scope naming nothing must not alert, got %v", resp.Diagnostics)
	}
}
