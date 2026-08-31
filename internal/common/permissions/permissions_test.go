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

func TestSection_RowsFollowPickerOrder(t *testing.T) {
	// Devices (Inventory) precedes Buildings (Organizational context) precedes
	// Policies (Deployment) in Jamf Account, though not alphabetically.
	reg, names := synth([]string{"policies:read", "buildings:read", "devices:read"})
	got := Section(reg, names...)
	iDevices := strings.Index(got, "| Devices |")
	iBuildings := strings.Index(got, "| Buildings |")
	iPolicies := strings.Index(got, "| Policies |")
	if iDevices < 0 || iBuildings < 0 || iPolicies < 0 {
		t.Fatalf("missing a row:\n%s", got)
	}
	if iDevices > iBuildings || iBuildings > iPolicies {
		t.Fatalf("rows out of picker order (devices %d, buildings %d, policies %d):\n%s",
			iDevices, iBuildings, iPolicies, got)
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

// TestCatalogueCategoriesAreOrdered keeps categoryOrder and the categories the
// catalogue actually uses in agreement. A category absent from categoryOrder
// gets slices.Index == -1 and would sort ahead of everything, silently.
func TestCatalogueCategoriesAreOrdered(t *testing.T) {
	for capability, e := range catalogue {
		if !slices.Contains(categoryOrder, e.category) {
			t.Errorf("capability %q: category %q missing from categoryOrder", capability, e.category)
		}
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
