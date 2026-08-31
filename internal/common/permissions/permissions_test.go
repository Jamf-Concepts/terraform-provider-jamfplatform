// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package permissions

import (
	"slices"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/account"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/aigovernance"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/audit"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/compliancebenchmarks"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/ddmreport"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/deviceactions"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devicegroups"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devices"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
)

func TestSection_BuildingCRUD(t *testing.T) {
	got := Section(pro.Privileges,
		"CreateBuildingV1", "GetBuildingV1", "UpdateBuildingV1", "DeleteBuildingV1")

	for _, want := range []string{
		"**Required Jamf permissions**",
		"| Category | Permission | Actions | API capability |",
		"| Organizational context | Buildings | Create, Read, Update, Delete | `buildings` |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Section() missing %q\n--- got ---\n%s", want, got)
		}
	}
	// One row per capability, not one per action.
	if n := strings.Count(got, "| `buildings` |"); n != 1 {
		t.Errorf("buildings should occupy one row, got %d\n%s", n, got)
	}
	// The pre-GA privilege names are gone for good.
	if strings.Contains(got, "Read Buildings") || strings.Contains(got, "Jamf Pro privilege") {
		t.Errorf("Section() still renders pre-GA privilege names\n%s", got)
	}
}

func TestSection_DeduplicatesAcrossMethods(t *testing.T) {
	// Read + List for the same resource both require buildings:read; the
	// table must list the capability once, with Read ticked once.
	got := Section(pro.Privileges, "GetBuildingV1", "ListBuildingsV1")
	if n := strings.Count(got, "| `buildings` |"); n != 1 {
		t.Fatalf("buildings should appear once, got %d\n%s", n, got)
	}
	if !strings.Contains(got, "| Buildings | Read | `buildings` |") {
		t.Fatalf("want a read-only Buildings row, got:\n%s", got)
	}
}

func TestSection_NoPrivilegeEndpoint(t *testing.T) {
	// Find a registry entry that requires no special permission and assert the
	// "None" wording renders.
	var none string
	for name, mp := range pro.Privileges {
		if len(mp.Scoped) == 0 {
			none = name
			break
		}
	}
	if none == "" {
		t.Skip("no zero-privilege method in registry")
	}
	got := Section(pro.Privileges, none)
	if !strings.Contains(got, "None — any authenticated Jamf Platform API integration") {
		t.Errorf("expected no-privilege wording for %s, got:\n%s", none, got)
	}
}

// TestSection_NoPrivilegeWordingNeedsEveryMethodResolved covers the asymmetry
// between "no method needs a permission" and "no method resolved". The
// affirmative wording is a positive claim about the underlying endpoints, so a
// list mixing a permission-free method with one absent from the registry must
// render nothing rather than an assurance that was never read.
func TestSection_NoPrivilegeWordingNeedsEveryMethodResolved(t *testing.T) {
	reg, names := synth([]string{})
	if got := Section(reg, names...); !strings.Contains(got, "None — any authenticated") {
		t.Fatalf("want the no-privilege wording when every method resolved, got:\n%s", got)
	}
	if got := Section(reg, append(slices.Clone(names), "NoSuchMethodXYZ")...); got != "" {
		t.Fatalf("want an empty section when a method is unresolved, got:\n%s", got)
	}
}

func TestSection_EmptyWhenAllMissing(t *testing.T) {
	if got := Section(pro.Privileges, "NoSuchMethodXYZ"); got != "" {
		t.Errorf("expected empty section for unknown method, got:\n%s", got)
	}
}

func TestMissing_DetectsUnknown(t *testing.T) {
	missing := Missing(pro.Privileges, "GetBuildingV1", "NoSuchMethodXYZ")
	if len(missing) != 1 || missing[0] != "NoSuchMethodXYZ" {
		t.Fatalf("want [NoSuchMethodXYZ], got %v", missing)
	}
}

func TestMerge(t *testing.T) {
	a := Registry{"X": jamfplatform.MethodPrivileges{Method: "X"}}
	b := Registry{"Y": jamfplatform.MethodPrivileges{Method: "Y"}}
	merged := Merge(a, b)
	if _, ok := merged["X"]; !ok {
		t.Error("merged registry missing X")
	}
	if _, ok := merged["Y"]; !ok {
		t.Error("merged registry missing Y")
	}
}

// --- rendering ----------------------------------------------------------------

// synth builds a registry from capability permission slices, so the rendering
// tests below state their input as the wire form rather than hunting the SDK
// for a method that happens to require the shape under test.
func synth(scoped ...[]string) (Registry, []string) {
	reg := make(Registry, len(scoped))
	names := make([]string, 0, len(scoped))
	for i, s := range scoped {
		name := string(rune('A' + i))
		reg[name] = jamfplatform.MethodPrivileges{Method: name, Scoped: s}
		names = append(names, name)
	}
	return reg, names
}

func TestActionList_FollowsPlatformOrder(t *testing.T) {
	// Alphabetical order would render "Create, Delete, Read, Update"; the
	// article's order is CRUD then deploy then execute.
	reg, names := synth([]string{
		"blueprints:execute", "blueprints:delete", "blueprints:deploy",
		"blueprints:create", "blueprints:update", "blueprints:read",
	})
	got := Section(reg, names...)
	if want := "| Blueprints | Create, Read, Update, Delete, Deploy, Execute | `blueprints` |"; !strings.Contains(got, want) {
		t.Fatalf("want %q, got:\n%s", want, got)
	}
}

// TestSection_RowsSortAlphabetically pins the sort to category name then
// permission name. Deployment sorts ahead of Inventory ahead of Organizational
// context, and Policies ahead of Scripts within Deployment — an order derived
// from the catalogue rather than from a hand-maintained list of sections.
func TestSection_RowsSortAlphabetically(t *testing.T) {
	reg, names := synth([]string{"policies:read", "buildings:read", "devices:read", "scripts:read"})
	got := Section(reg, names...)
	var order []int
	for _, row := range []string{"| Policies |", "| Scripts |", "| Devices |", "| Buildings |"} {
		i := strings.Index(got, row)
		if i < 0 {
			t.Fatalf("missing row %q:\n%s", row, got)
		}
		order = append(order, i)
	}
	if !slices.IsSorted(order) {
		t.Fatalf("rows out of alphabetical order (offsets %v):\n%s", order, got)
	}
}

// TestSection_UnresolvedRowsSortLast keeps the contract collect documents: a
// capability the catalogue does not know, and a value whose shape splitScoped
// rejected, both follow every row that resolved to picker names. Neither has a
// section to sort into, and both render as dashes, so sorting them among the
// real rows would put an unreadable row in the middle of a readable table.
func TestSection_UnresolvedRowsSortLast(t *testing.T) {
	reg, names := synth([]string{"zzz-unknown:read", "buildings:read", "devices"})
	got := Section(reg, names...)
	iBuildings := strings.Index(got, "| Organizational context | Buildings |")
	iUnknown := strings.Index(got, "`zzz-unknown`")
	iMalformed := strings.Index(got, "`devices`")
	if iBuildings < 0 || iUnknown < 0 || iMalformed < 0 {
		t.Fatalf("missing a row (buildings %d, unknown %d, malformed %d):\n%s",
			iBuildings, iUnknown, iMalformed, got)
	}
	if iBuildings > iUnknown || iBuildings > iMalformed {
		t.Fatalf("a resolved row sorted after an unresolved one:\n%s", got)
	}
}

// TestSplitScoped covers the shapes the renderer must refuse. The
// three-part beta slug is the one that used to slip through: strings.Cut takes
// the first colon, so "create:pro:buildings" split into the capability "create"
// and rendered a row naming a permission that does not exist.
func TestSplitScoped(t *testing.T) {
	for _, tc := range []struct {
		scoped     string
		capability string
		action     string
		ok         bool
	}{
		{"buildings:read", "buildings", "read", true},
		{":read", "", "", false},
		{"buildings:", "", "", false},
		{"create:pro:buildings", "", "", false},
		{"weird-shape", "", "", false},
		{"", "", "", false},
		{":", "", "", false},
	} {
		capability, action, ok := splitScoped(tc.scoped)
		if capability != tc.capability || action != tc.action || ok != tc.ok {
			t.Errorf("splitScoped(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.scoped, capability, action, ok, tc.capability, tc.action, tc.ok)
		}
	}
}

// TestSection_MalformedValueOnCataloguedCapabilityIsVisiblyBroken guards the
// worst rendering of an unparseable value: "buildings" alone resolves to the
// Buildings row with an empty Actions cell, which in Jamf Account's model is a
// permission granting nothing — and, because buildings IS catalogued, without
// the footnote that would explain it.
func TestSection_MalformedValueOnCataloguedCapabilityIsVisiblyBroken(t *testing.T) {
	reg, names := synth([]string{"buildings"})
	got := Section(reg, names...)
	if strings.Contains(got, "| Organizational context | Buildings |") {
		t.Errorf("malformed value rendered as a grantable Buildings row:\n%s", got)
	}
	if want := "| — | — | " + malformedAction + " | `buildings` |"; !strings.Contains(got, want) {
		t.Errorf("want %q, got:\n%s", want, got)
	}
	if !strings.Contains(got, "no Jamf Account name recorded for") {
		t.Errorf("malformed value should carry the explanatory footnote, got:\n%s", got)
	}
}

// TestSection_MalformedValueDoesNotMergeIntoWellFormedRow guards the quieter
// failure: an unparseable value sharing a capability with a well-formed one used
// to be absorbed into its row and leave no trace at all.
func TestSection_MalformedValueDoesNotMergeIntoWellFormedRow(t *testing.T) {
	reg, names := synth([]string{"buildings", "buildings:read"})
	got := Section(reg, names...)
	if want := "| Organizational context | Buildings | Read | `buildings` |"; !strings.Contains(got, want) {
		t.Errorf("want the well-formed row %q, got:\n%s", want, got)
	}
	if want := "| — | — | " + malformedAction + " | `buildings` |"; !strings.Contains(got, want) {
		t.Errorf("malformed value absorbed into the well-formed row, want %q, got:\n%s", want, got)
	}
	if n := strings.Count(got, "| `buildings` |"); n != 2 {
		t.Errorf("want two buildings rows, one real and one malformed, got %d:\n%s", n, got)
	}
}

// TestActionList_UnlabelledOrderedActionRendersAsSlugOnce pins actionOrder to
// actionLabels. An entry added to the order but not the labels used to append
// an empty string and then repeat itself through the extras path, rendering a
// cell like "Read, , annotate" whose blank the operator cannot act on.
func TestActionList_UnlabelledOrderedActionRendersAsSlugOnce(t *testing.T) {
	original := actionOrder
	actionOrder = append(slices.Clone(original), "annotate")
	t.Cleanup(func() { actionOrder = original })

	if got, want := actionList(map[string]bool{"read": true, "annotate": true}), "Read, annotate"; got != want {
		t.Fatalf("actionList() = %q, want %q", got, want)
	}
}

func TestSection_UncataloguedCapabilityIsMarkedNotGuessed(t *testing.T) {
	reg, names := synth([]string{"buildings:read", "brand-new-thing:read"})
	got := Section(reg, names...)
	if !strings.Contains(got, "| — | — | Read | `brand-new-thing` |") {
		t.Errorf("uncatalogued capability should render dashes, got:\n%s", got)
	}
	if !strings.Contains(got, "no Jamf Account name recorded for") {
		t.Errorf("uncatalogued capability should carry the explanatory note, got:\n%s", got)
	}
	// A known capability alongside it keeps its name, and the note is not
	// emitted for tables without an unknown.
	clean, cleanNames := synth([]string{"buildings:read"})
	if strings.Contains(Section(clean, cleanNames...), "no Jamf Account name recorded for") {
		t.Error("note emitted for a fully catalogued table")
	}
}

func TestSection_UnsplittableScopedSurvivesVerbatim(t *testing.T) {
	// Defensive: the SDK documents the two-part GA form as the only one it
	// emits, so a value without an action is a shape we have never seen. It
	// must still appear rather than be silently dropped.
	reg, names := synth([]string{"weird-shape"})
	got := Section(reg, names...)
	if !strings.Contains(got, "`weird-shape`") {
		t.Fatalf("unsplittable capability dropped:\n%s", got)
	}
}

// --- catalogue drift ----------------------------------------------------------

// sdkRegistries is every privilege registry the SDK publishes. Listed
// explicitly because Go has no way to enumerate a module's packages at
// runtime; a new SDK sub-package is added here by hand, which is the same
// motion as wiring its resources up.
func sdkRegistries() map[string]Registry {
	return map[string]Registry{
		"account":              account.Privileges,
		"aigovernance":         aigovernance.Privileges,
		"audit":                audit.Privileges,
		"blueprints":           blueprints.Privileges,
		"compliancebenchmarks": compliancebenchmarks.Privileges,
		"ddmreport":            ddmreport.Privileges,
		"deviceactions":        deviceactions.Privileges,
		"devicegroups":         devicegroups.Privileges,
		"devices":              devices.Privileges,
		"pro":                  pro.Privileges,
		"proclassic":           proclassic.Privileges,
		"securitycloud":        securitycloud.Privileges,
	}
}

// TestCatalogueCoversEverySDKCapability is the drift guard on catalogue.go.
// Jamf's permissions map is documentation, not a machine-readable artefact, so
// the transcription cannot be regenerated — this test is what tells us the
// article has moved on. A failure is not a bug in the renderer: it means a
// capability now reachable through the SDK has no Jamf Account permission name
// recorded, and the fix is to add the row from the current article.
func TestCatalogueCoversEverySDKCapability(t *testing.T) {
	type where struct{ pkg, method string }
	missing := map[string]where{}
	for pkgName, reg := range sdkRegistries() {
		for method, mp := range reg {
			for _, scoped := range mp.Scoped {
				capability, action, ok := splitScoped(scoped)
				if !ok {
					t.Errorf("%s.%s: scoped permission %q is not {capability}:{action}", pkgName, method, scoped)
					continue
				}
				if _, known := actionLabels[action]; !known {
					t.Errorf("%s.%s: scoped permission %q uses unknown action %q", pkgName, method, scoped, action)
				}
				if _, known := catalogue[capability]; !known {
					if _, dup := missing[capability]; !dup {
						missing[capability] = where{pkgName, method}
					}
				}
			}
		}
	}
	for capability, w := range missing {
		t.Errorf("capability %q (required by %s.%s) has no catalogue entry — add it from Jamf's permissions map",
			capability, w.pkg, w.method)
	}
}

// TestCataloguePermissionNamesAreUniquePerCategory catches a copy-paste slip in
// the transcription: two capabilities pointing at the same picker row.
func TestCataloguePermissionNamesAreUniquePerCategory(t *testing.T) {
	seen := map[entry]string{}
	for capability, e := range catalogue {
		if prev, dup := seen[e]; dup {
			t.Errorf("capabilities %q and %q both map to %s / %s", prev, capability, e.category, e.name)
			continue
		}
		seen[e] = capability
	}
}

// --- Renders ------------------------------------------------------------------

func TestRenders(t *testing.T) {
	section := Section(pro.Privileges,
		"CreateBuildingV1", "GetBuildingV1", "UpdateBuildingV1", "DeleteBuildingV1")

	for _, want := range []string{
		"buildings:create", "buildings:read", "buildings:update", "buildings:delete",
	} {
		if !Renders(section, want) {
			t.Errorf("Renders(%q) = false, want true\n%s", want, section)
		}
	}
	for _, notWant := range []string{
		"buildings:deploy",   // right row, box not ticked
		"buildings:execute",  // right row, box not ticked
		"departments:read",   // wrong row entirely
		"building:read",      // capability is not a prefix match
		"buildings",          // not a scoped permission
		"buildings:nonsense", // action outside the six
	} {
		if Renders(section, notWant) {
			t.Errorf("Renders(%q) = true, want false\n%s", notWant, section)
		}
	}
}

func TestRenders_IgnoresNonTableText(t *testing.T) {
	if Renders("**Required Jamf permissions**\n\nNone — any authenticated integration", "buildings:read") {
		t.Error("Renders matched the no-permission wording")
	}
	if Renders("", "buildings:read") {
		t.Error("Renders matched an empty section")
	}
}
