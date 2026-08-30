// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_app

import (
	"context"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestNormalisedHostname pins the two normalisations Jamf Security Cloud applies and
// the provider therefore refuses, because `hostnames` is Optional rather than
// Optional+Computed and so cannot be rewritten at plan time.
func TestNormalisedHostname(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr string
	}{
		{"already normalised", "sub.example.com", ""},
		{"wildcard", "*.example.com", ""},
		{"bare wildcard", "*", ""},
		{"digits and hyphens", "web-01.example.com", ""},

		{"uppercase", "Sub.Example.COM", "lower-case"},
		{"single uppercase letter", "sub.example.coM", "lower-case"},
		{"trailing dot", "sub.example.com.", "trailing"},
		{"uppercase and trailing dot", "SUB.EXAMPLE.COM.", "trailing"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			normalisedHostname().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("hostnames"),
				ConfigValue: types.StringValue(tc.value),
			}, resp)

			if tc.wantErr == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected %q to be accepted, got %s", tc.value, resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected %q to be refused", tc.value)
			}
			if !strings.Contains(diagnosticsText(resp.Diagnostics), tc.wantErr) {
				t.Fatalf("expected the refusal of %q to mention %q, got %s", tc.value, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestNormalisedHostnameSuggestsTheFix pins that the lower-case refusal names the
// spelling to write, so the fix does not have to be worked out.
func TestNormalisedHostnameSuggestsTheFix(t *testing.T) {
	resp := &validator.StringResponse{}
	normalisedHostname().ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("hostnames"),
		ConfigValue: types.StringValue("Sub.Example.COM"),
	}, resp)
	if !strings.Contains(diagnosticsText(resp.Diagnostics), "sub.example.com") {
		t.Fatalf("expected the refusal to name the normalised form, got %s", resp.Diagnostics)
	}
}

// TestCoversHostname pins the overlap rule the server enforces: a wildcard covers
// subdomains but not the parent, so a parent listed alongside its own wildcard is
// legal and a subdomain listed alongside it is not.
//
// Coverage is strictly about wildcards, which is why an entry does not cover an
// identical one. Two identical entries would be a duplicate rather than an overlap,
// and the Set type rules that out before any validator sees it.
func TestCoversHostname(t *testing.T) {
	cases := []struct {
		pattern   string
		candidate string
		want      bool
	}{
		{"*", "anything.example.com", true},
		{"*", "*.example.com", true},
		{"*.example.com", "sub.example.com", true},
		{"*.example.com", "deep.sub.example.com", true},
		{"*.example.com", "example.com", false},
		{"*.example.com", "notexample.com", false},
		{"*.example.com", "sub.other.com", false},
		{"example.com", "sub.example.com", false},
		{"sub.example.com", "sub.example.com", false},
	}
	for _, tc := range cases {
		t.Run(tc.pattern+" covers "+tc.candidate, func(t *testing.T) {
			if got := coversHostname(tc.pattern, tc.candidate); got != tc.want {
				t.Errorf("coversHostname(%q, %q) = %v, want %v", tc.pattern, tc.candidate, got, tc.want)
			}
		})
	}
}

// TestValidateRouting pins all four cross-field rules on a routing block. The server
// answers every one of them with the same opaque sentence, so this is where the
// distinction actually gets made.
func TestValidateRouting(t *testing.T) {
	viaZTNA := routingModeLabels[securitycloud.RoutingTypeCustom]
	direct := routingModeLabels[securitycloud.RoutingTypeDirect]

	cases := []struct {
		name    string
		routing *RoutingModel
		wantErr string
	}{
		{"nil block defers", nil, ""},
		{
			"via ztna, complete",
			&RoutingModel{Mode: types.StringValue(viaZTNA), GatewayID: types.StringValue("a7d2"), RoutingMode: types.StringValue("Standard")},
			"",
		},
		{
			"direct, empty",
			&RoutingModel{Mode: types.StringValue(direct), GatewayID: types.StringNull(), RoutingMode: types.StringNull()},
			"",
		},
		{
			"via ztna without a gateway",
			&RoutingModel{Mode: types.StringValue(viaZTNA), GatewayID: types.StringNull(), RoutingMode: types.StringValue("Standard")},
			"needs an access gateway",
		},
		{
			"via ztna without a routing mode",
			&RoutingModel{Mode: types.StringValue(viaZTNA), GatewayID: types.StringValue("a7d2"), RoutingMode: types.StringNull()},
			"needs a routing mode",
		},
		{
			"direct with a gateway",
			&RoutingModel{Mode: types.StringValue(direct), GatewayID: types.StringValue("a7d2"), RoutingMode: types.StringNull()},
			"does not use an access gateway",
		},
		{
			"direct with a routing mode",
			&RoutingModel{Mode: types.StringValue(direct), GatewayID: types.StringNull(), RoutingMode: types.StringValue("Standard")},
			"has no routing mode",
		},
		{
			"unknown mode defers",
			&RoutingModel{Mode: types.StringUnknown(), GatewayID: types.StringNull(), RoutingMode: types.StringNull()},
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			validateRouting(tc.routing, path.Root("routing"), &diags)

			if tc.wantErr == "" {
				if diags.HasError() {
					t.Fatalf("expected no error, got %s", diags)
				}
				return
			}
			if !diags.HasError() {
				t.Fatal("expected an error")
			}
			if !strings.Contains(diagnosticsText(diags), tc.wantErr) {
				t.Fatalf("expected the error to mention %q, got %s", tc.wantErr, diags)
			}
		})
	}
}

// TestValidateRoutingReportsBothMissingFields pins that a `CUSTOM` block missing both
// members reports both, rather than stopping at the first. The server names neither,
// so a caller fixing one at a time would round-trip twice.
func TestValidateRoutingReportsBothMissingFields(t *testing.T) {
	var diags diag.Diagnostics
	validateRouting(&RoutingModel{
		Mode:        types.StringValue(routingModeLabels[securitycloud.RoutingTypeCustom]),
		GatewayID:   types.StringNull(),
		RoutingMode: types.StringNull(),
	}, path.Root("routing"), &diags)

	if got := diags.ErrorsCount(); got != 2 {
		t.Fatalf("expected 2 errors, got %d: %s", got, diags)
	}
}

// diagnosticsText flattens diagnostics into one string for substring assertions.
func diagnosticsText(diags diag.Diagnostics) string {
	var b strings.Builder
	for _, d := range diags.Errors() {
		b.WriteString(d.Summary())
		b.WriteString(" ")
		b.WriteString(d.Detail())
		b.WriteString("\n")
	}
	return b.String()
}

// TestValidateAppForm pins the mutual exclusivity of the two forms. The
// combination case is the one that matters most, because the server accepts it and
// then discards the name — so this validator is the only thing standing between an
// operator and a silently ignored configuration.
func TestValidateAppForm(t *testing.T) {
	cases := []struct {
		name       string
		appName    types.String
		predefined types.String
		wantErr    string
	}{
		{"custom", types.StringValue("Internal CRM"), types.StringNull(), ""},
		{"predefined", types.StringNull(), types.StringValue("2aaa401c"), ""},
		{"both", types.StringValue("My Slack"), types.StringValue("2aaa401c"), "cannot be renamed"},
		{"neither", types.StringNull(), types.StringNull(), "needs a name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			validateAppForm(ZtnaAppResourceModel{Name: tc.appName, PredefinedAppID: tc.predefined}, &diags)

			if tc.wantErr == "" {
				if diags.HasError() {
					t.Fatalf("expected no error, got %s", diags)
				}
				return
			}
			if !strings.Contains(diagnosticsText(diags), tc.wantErr) {
				t.Fatalf("expected the error to mention %q, got %s", tc.wantErr, diags)
			}
		})
	}
}

// TestValidateHostnameOverlap pins the plan-time overlap check against the pairing
// the server refuses, and against the pairing it allows — a parent domain listed
// alongside its own wildcard, which is the whole reason coversHostname excludes the
// parent.
func TestValidateHostnameOverlap(t *testing.T) {
	cases := []struct {
		name      string
		hostnames []string
		wantErr   bool
	}{
		{"disjoint", []string{"a.example.com", "b.example.com"}, false},
		{"parent alongside its wildcard", []string{"example.com", "*.example.com"}, false},
		{"two unrelated wildcards", []string{"*.example.com", "*.other.com"}, false},
		{"absent", nil, false},

		{"subdomain under a wildcard", []string{"*.example.com", "sub.example.com"}, true},
		{"deep subdomain under a wildcard", []string{"*.example.com", "deep.sub.example.com"}, true},
		{"full tunnel alongside anything", []string{"*", "example.com"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := ZtnaAppResourceModel{Hostnames: stringSetFor(t, tc.hostnames)}
			var diags diag.Diagnostics
			validateHostnameOverlap(context.Background(), config, &diags)

			if got := diags.HasError(); got != tc.wantErr {
				t.Fatalf("HasError() = %v, want %v (%s)", got, tc.wantErr, diags)
			}
		})
	}
}

// TestValidateHostnameOverlapNamesBothEntries pins that the diagnostic names the
// covered entry and the wildcard covering it. The server's own refusal names
// neither, so a set of twenty host names would otherwise be a manual search.
func TestValidateHostnameOverlapNamesBothEntries(t *testing.T) {
	config := ZtnaAppResourceModel{Hostnames: stringSetFor(t, []string{"*.example.com", "sub.example.com"})}
	var diags diag.Diagnostics
	validateHostnameOverlap(context.Background(), config, &diags)

	text := diagnosticsText(diags)
	for _, want := range []string{"sub.example.com", "*.example.com"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected the error to name %q, got %s", want, text)
		}
	}
}

// TestValidateDeviceGroupAssignment pins the three assignment rules, including the
// one the server does not enforce: a group list sent alongside "all device groups"
// is accepted and ignored, so it has to be refused here or it would disappear from
// state on the first refresh.
func TestValidateDeviceGroupAssignment(t *testing.T) {
	const groupA = "aaaaaaaa-0000-0000-0000-000000000000"
	const groupB = "bbbbbbbb-0000-0000-0000-000000000000"
	const groupC = "cccccccc-0000-0000-0000-000000000000"

	cases := []struct {
		name      string
		allGroups bool
		assigned  []string
		overrides [][]string
		wantErr   string
	}{
		{"all groups, no list", true, nil, nil, ""},
		{"selected groups", false, []string{groupA, groupB}, nil, ""},
		{"override on an assigned group", false, []string{groupA, groupB}, [][]string{{groupA}}, ""},
		{"overrides on distinct assigned groups", false, []string{groupA, groupB}, [][]string{{groupA}, {groupB}}, ""},
		{"all groups with an override on any group", true, nil, [][]string{{groupC}}, ""},
		{"selected groups, empty list", false, nil, nil, ""},

		{"all groups with a list", true, []string{groupA}, nil, "conflict with all device groups"},
		{"override on an unassigned group", false, []string{groupA}, [][]string{{groupC}}, "unassigned device group"},
		{"group in two overrides", false, []string{groupA}, [][]string{{groupA}, {groupA}}, "more than one routing override"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := ZtnaAppResourceModel{
				AllDeviceGroups:  types.BoolValue(tc.allGroups),
				DeviceGroupIDs:   stringSetFor(t, tc.assigned),
				RoutingOverrides: overrideListFor(t, tc.overrides),
			}
			var diags diag.Diagnostics
			validateDeviceGroupAssignment(context.Background(), config, &diags)

			if tc.wantErr == "" {
				if diags.HasError() {
					t.Fatalf("expected no error, got %s", diags)
				}
				return
			}
			if !strings.Contains(diagnosticsText(diags), tc.wantErr) {
				t.Fatalf("expected the error to mention %q, got %s", tc.wantErr, diags)
			}
		})
	}
}

// TestValidateAllRoutingWalksOverrides pins that an override's routing is held to the
// same rules as the application's own, and that the diagnostic points at the
// offending index. The server does index its own message here, but only for the
// routing shape — not for the group rules.
func TestValidateAllRoutingWalksOverrides(t *testing.T) {
	direct := routingModeLabels[securitycloud.RoutingTypeDirect]
	viaZTNA := routingModeLabels[securitycloud.RoutingTypeCustom]

	config := ZtnaAppResourceModel{
		Routing: &RoutingModel{
			Mode:        types.StringValue(direct),
			GatewayID:   types.StringNull(),
			RoutingMode: types.StringNull(),
		},
		RoutingOverrides: overrideListWithRouting(t, []*RoutingModel{
			{Mode: types.StringValue(direct), GatewayID: types.StringNull(), RoutingMode: types.StringNull()},
			{Mode: types.StringValue(viaZTNA), GatewayID: types.StringNull(), RoutingMode: types.StringValue("Standard")},
		}),
	}

	var diags diag.Diagnostics
	validateAllRouting(context.Background(), config, &diags)

	if got := diags.ErrorsCount(); got != 1 {
		t.Fatalf("expected exactly the second override to fail, got %d errors: %s", got, diags)
	}
	withPath, ok := diags.Errors()[0].(diag.DiagnosticWithPath)
	if !ok {
		t.Fatal("expected the error to be attached to an attribute path")
	}
	if got := withPath.Path().String(); got != "routing_overrides[1].routing.gateway_id" {
		t.Errorf("error is attached to %s, want routing_overrides[1].routing.gateway_id", got)
	}
}

// stringSetFor builds a set of strings, or a null set for an absent collection.
func stringSetFor(t *testing.T, values []string) types.Set {
	t.Helper()
	if values == nil {
		return types.SetNull(types.StringType)
	}
	set, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("building set: %s", diags)
	}
	return set
}

// overrideListFor builds a routing_overrides list whose entries differ only in the
// groups they name, with a valid direct-routing block throughout.
func overrideListFor(t *testing.T, groups [][]string) types.List {
	t.Helper()
	if groups == nil {
		return types.ListNull(routingOverrideObjectType)
	}
	routings := make([]*RoutingModel, 0, len(groups))
	for range groups {
		routings = append(routings, &RoutingModel{
			Mode:        types.StringValue(routingModeLabels[securitycloud.RoutingTypeDirect]),
			GatewayID:   types.StringNull(),
			RoutingMode: types.StringNull(),
		})
	}
	return buildOverrideList(t, groups, routings)
}

// overrideListWithRouting builds a routing_overrides list whose entries differ in
// their routing block, each naming a distinct group.
func overrideListWithRouting(t *testing.T, routings []*RoutingModel) types.List {
	t.Helper()
	groups := make([][]string, 0, len(routings))
	for i := range routings {
		groups = append(groups, []string{"group-" + string(rune('a'+i))})
	}
	return buildOverrideList(t, groups, routings)
}

// buildOverrideList assembles the framework list value from parallel slices.
func buildOverrideList(t *testing.T, groups [][]string, routings []*RoutingModel) types.List {
	t.Helper()
	models := make([]RoutingOverrideModel, 0, len(groups))
	for i := range groups {
		set, diags := types.SetValueFrom(context.Background(), types.StringType, groups[i])
		if diags.HasError() {
			t.Fatalf("building override group set: %s", diags)
		}
		models = append(models, RoutingOverrideModel{DeviceGroupIDs: set, Routing: routings[i]})
	}
	list, diags := types.ListValueFrom(context.Background(), routingOverrideObjectType, models)
	if diags.HasError() {
		t.Fatalf("building override list: %s", diags)
	}
	return list
}

// TestKnownStrings pins the unknown-value handling every config validator here rests
// on. ElementsAs, which this replaced, refuses a collection holding an unknown
// element — and an unknown element is the commonest case there is, because a group ID
// usually references a device group the same plan is about to create. That made every
// check silently dead in exactly the configuration they exist for.
func TestKnownStrings(t *testing.T) {
	known, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"a", "b"})
	if diags.HasError() {
		t.Fatalf("building set: %s", diags)
	}
	partial, diags := types.SetValue(types.StringType, []attr.Value{
		types.StringValue("a"),
		types.StringUnknown(),
	})
	if diags.HasError() {
		t.Fatalf("building set: %s", diags)
	}

	cases := []struct {
		name         string
		set          types.Set
		wantValues   int
		wantComplete bool
	}{
		{"null", types.SetNull(types.StringType), 0, true},
		{"unknown", types.SetUnknown(types.StringType), 0, false},
		{"fully known", known, 2, true},
		{"one unknown element", partial, 1, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values, complete := knownStrings(tc.set)
			if len(values) != tc.wantValues {
				t.Errorf("got %d known values, want %d: %v", len(values), tc.wantValues, values)
			}
			if complete != tc.wantComplete {
				t.Errorf("complete = %v, want %v", complete, tc.wantComplete)
			}
		})
	}
}

// TestValidateDeviceGroupAssignmentDefersOnUnknownAssignment pins the one rule that
// needs the whole collection. A group whose assignment is not yet resolved must not be
// reported as unassigned — that would fail the plan for a perfectly valid
// configuration referencing a device group the same apply creates.
func TestValidateDeviceGroupAssignmentDefersOnUnknownAssignment(t *testing.T) {
	assigned, diags := types.SetValue(types.StringType, []attr.Value{types.StringUnknown()})
	if diags.HasError() {
		t.Fatalf("building set: %s", diags)
	}

	config := ZtnaAppResourceModel{
		AllDeviceGroups:  types.BoolValue(false),
		DeviceGroupIDs:   assigned,
		RoutingOverrides: overrideListFor(t, [][]string{{"aaaaaaaa-0000-0000-0000-000000000000"}}),
	}

	var got diag.Diagnostics
	validateDeviceGroupAssignment(context.Background(), config, &got)
	if got.HasError() {
		t.Fatalf("expected the subset rule to defer while the assignment is unresolved, got %s", got)
	}
}

// TestValidateDeviceGroupAssignmentStillCatchesDuplicatesWhenIncomplete pins the
// other half of that decision: two known colliding values are a real error whatever
// else is unresolved, so the duplicate rule must not defer with the subset rule.
func TestValidateDeviceGroupAssignmentStillCatchesDuplicatesWhenIncomplete(t *testing.T) {
	assigned, diags := types.SetValue(types.StringType, []attr.Value{types.StringUnknown()})
	if diags.HasError() {
		t.Fatalf("building set: %s", diags)
	}

	const group = "aaaaaaaa-0000-0000-0000-000000000000"
	config := ZtnaAppResourceModel{
		AllDeviceGroups:  types.BoolValue(false),
		DeviceGroupIDs:   assigned,
		RoutingOverrides: overrideListFor(t, [][]string{{group}, {group}}),
	}

	var got diag.Diagnostics
	validateDeviceGroupAssignment(context.Background(), config, &got)
	if !strings.Contains(diagnosticsText(got), "more than one routing override") {
		t.Fatalf("expected the duplicate rule to still fire, got %s", got)
	}
}

// TestValidateDeviceGroupAssignmentAllowsSelectedGroups is the regression test for a
// bug this package shipped with briefly: a boolvalidator.ConflictsWith on
// all_device_groups fired whenever both attributes were configured, and
// all_device_groups is Required — so it is always configured, and device_group_ids
// could never be set at all. The rule is conditional on the bool's value, which only a
// config validator can see.
func TestValidateDeviceGroupAssignmentAllowsSelectedGroups(t *testing.T) {
	config := ZtnaAppResourceModel{
		AllDeviceGroups:  types.BoolValue(false),
		DeviceGroupIDs:   stringSetFor(t, []string{"aaaaaaaa-0000-0000-0000-000000000000"}),
		RoutingOverrides: types.ListNull(routingOverrideObjectType),
	}

	var got diag.Diagnostics
	validateDeviceGroupAssignment(context.Background(), config, &got)
	if got.HasError() {
		t.Fatalf("selected device groups must be accepted alongside all_device_groups = false, got %s", got)
	}
}

// TestSchemaDoesNotRefuseSelectedGroups is the schema-level half of the same
// regression. The config validator above cannot see a schema validator, so a
// ConflictsWith reintroduced on either attribute would pass every test here unless
// something asserts its absence.
func TestSchemaDoesNotRefuseSelectedGroups(t *testing.T) {
	s := resourceSchema(t)

	boolAttr, ok := s.Attributes["all_device_groups"].(rschema.BoolAttribute)
	if !ok {
		t.Fatalf("all_device_groups must be a BoolAttribute, got %T", s.Attributes["all_device_groups"])
	}
	if len(boolAttr.Validators) != 0 {
		t.Errorf("all_device_groups must carry no schema validators: a ConflictsWith here fires whenever "+
			"both attributes are configured, and this attribute is Required, so device_group_ids could "+
			"never be set. Got %d", len(boolAttr.Validators))
	}

	setAttr, ok := s.Attributes["device_group_ids"].(rschema.SetAttribute)
	if !ok {
		t.Fatalf("device_group_ids must be a SetAttribute, got %T", s.Attributes["device_group_ids"])
	}
	for _, v := range setAttr.Validators {
		if strings.Contains(v.Description(context.Background()), "all_device_groups") {
			t.Errorf("device_group_ids must not conflict-check against all_device_groups in the schema: %s",
				v.Description(context.Background()))
		}
	}
}
