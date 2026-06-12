// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package accountprivileges

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mustSet(t *testing.T, vals ...string) types.Set {
	t.Helper()
	s, diags := newStringSet(vals)
	if diags.HasError() {
		t.Fatalf("newStringSet(%v): %v", vals, diags)
	}
	return s
}

func setStrings(t *testing.T, s types.Set) []string {
	t.Helper()
	if s.IsNull() {
		return nil
	}
	var out []string
	if d := s.ElementsAs(context.Background(), &out, false); d.HasError() {
		t.Fatalf("ElementsAs: %v", d)
	}
	sort.Strings(out)
	return out
}

func TestGroupPrivilegesRoundTrip(t *testing.T) {
	in := map[string][]string{
		"jss_objects":  {"Read Computers", "Update Computers"},
		"jss_settings": {"Read Activation Code"},
		"jss_actions":  {},
	}
	got := FromGroupPrivileges(ToGroupPrivileges(in))
	for k, want := range in {
		sort.Strings(want)
		gv := got[k]
		sort.Strings(gv)
		if !reflect.DeepEqual(want, gv) && !(len(want) == 0 && len(gv) == 0) {
			t.Errorf("category %s: want %v, got %v", k, want, gv)
		}
	}
}

func TestAccountPrivilegesRoundTrip(t *testing.T) {
	in := map[string][]string{
		"jss_objects": {"Read Scripts"},
		"recon":       {"Use Recon"},
	}
	got := FromAccountPrivileges(ToAccountPrivileges(in))
	if !reflect.DeepEqual(got["jss_objects"], []string{"Read Scripts"}) {
		t.Errorf("jss_objects: %v", got["jss_objects"])
	}
	if !reflect.DeepEqual(got["recon"], []string{"Use Recon"}) {
		t.Errorf("recon: %v", got["recon"])
	}
	if _, ok := got["jss_settings"]; ok {
		t.Errorf("absent category jss_settings should not appear: %v", got)
	}
}

func TestToAccountPrivilegesEmptyMapIsNil(t *testing.T) {
	if ToAccountPrivileges(nil) != nil {
		t.Error("nil map should yield nil privileges")
	}
	if ToGroupPrivileges(map[string][]string{}) != nil {
		t.Error("empty map should yield nil privileges")
	}
}

func TestIntersectIntoState_DropsServerAdded(t *testing.T) {
	// Declared {Update Buildings}; server expanded to add Read Activation Code
	// in a different category. Declared category keeps only what it declared;
	// the undeclared (null) jss_settings category stays null.
	prior := &Model{JamfProServerObjects: mustSet(t, "Update Buildings")}
	server := map[string][]string{
		"jss_objects":  {"Update Buildings"},
		"jss_settings": {"Read Activation Code"},
	}
	out, diags := IntersectIntoState(context.Background(), prior, server)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if got := setStrings(t, out.JamfProServerObjects); !reflect.DeepEqual(got, []string{"Update Buildings"}) {
		t.Errorf("jss_objects: %v", got)
	}
	if !out.JamfProServerSettings.IsNull() {
		t.Errorf("undeclared jss_settings should stay null, got %v", out.JamfProServerSettings)
	}
}

func TestIntersectIntoState_PreservesRemovalIntent(t *testing.T) {
	// Prior declared {A,B}; user is removing B (so config will be {A}). On the
	// read after a write of {A}, the server still has {A} (+maybe deps we don't
	// declare). Intersect with declared {A} keeps {A}; B is gone — removal works.
	prior := &Model{JamfProServerObjects: mustSet(t, "A")}
	server := map[string][]string{"jss_objects": {"A", "ServerDep"}}
	out, _ := IntersectIntoState(context.Background(), prior, server)
	if got := setStrings(t, out.JamfProServerObjects); !reflect.DeepEqual(got, []string{"A"}) {
		t.Errorf("expected [A] (ServerDep dropped), got %v", got)
	}
}

func TestIntersectIntoState_DriftShrinksWhenServerLosesDeclared(t *testing.T) {
	// Declared {A,B}; server now only has {A} (B removed out-of-band). State
	// becomes {A}, so the next plan re-adds B (drift corrected).
	prior := &Model{JamfProServerObjects: mustSet(t, "A", "B")}
	server := map[string][]string{"jss_objects": {"A"}}
	out, _ := IntersectIntoState(context.Background(), prior, server)
	if got := setStrings(t, out.JamfProServerObjects); !reflect.DeepEqual(got, []string{"A"}) {
		t.Errorf("expected [A], got %v", got)
	}
}

func TestIntersectIntoState_ImportMaterialisesFullGrid(t *testing.T) {
	server := map[string][]string{"jss_objects": {"A", "B"}}
	out, _ := IntersectIntoState(context.Background(), nil, server)
	if got := setStrings(t, out.JamfProServerObjects); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("import should materialise full grid, got %v", got)
	}
	if !out.JamfProServerSettings.IsNull() {
		t.Errorf("absent category should be null on import, got %v", out.JamfProServerSettings)
	}
}

func TestModelToMap_OmitsNullCategories(t *testing.T) {
	m := &Model{JamfProServerObjects: mustSet(t, "Read Computers")}
	got, diags := m.ToMap(context.Background())
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if _, ok := got["jss_objects"]; !ok {
		t.Errorf("declared category missing: %v", got)
	}
	if _, ok := got["jss_settings"]; ok {
		t.Errorf("null category should be omitted: %v", got)
	}
}

// fakeDiscoverer serves a fixed account/group topology for Discover tests.
type fakeDiscoverer struct {
	groups map[int]*proclassic.Group
	users  map[int]*proclassic.Account
}

func (f fakeDiscoverer) ListAccounts(_ context.Context) (*proclassic.Accounts, error) {
	var gs []proclassic.AccountsGroupsGroupItem
	for id := range f.groups {
		id := id
		gs = append(gs, proclassic.AccountsGroupsGroupItem{ID: &id})
	}
	var us []proclassic.AccountsUsersUserItem
	for id := range f.users {
		id := id
		us = append(us, proclassic.AccountsUsersUserItem{ID: &id})
	}
	return &proclassic.Accounts{
		Groups: &proclassic.AccountsGroups{Group: &gs},
		Users:  &proclassic.AccountsUsers{User: &us},
	}, nil
}

func (f fakeDiscoverer) GetAccountGroupByID(_ context.Context, id string) (*proclassic.Group, error) {
	var n int
	fmt.Sscanf(id, "%d", &n)
	if g, ok := f.groups[n]; ok {
		return g, nil
	}
	return nil, fmt.Errorf("not found")
}

func (f fakeDiscoverer) GetAccountByUserID(_ context.Context, id string) (*proclassic.Account, error) {
	var n int
	fmt.Sscanf(id, "%d", &n)
	if u, ok := f.users[n]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("not found")
}

func adminGroup(id int) *proclassic.Group {
	set := administratorPrivilegeSet
	return &proclassic.Group{
		ID:           &id,
		PrivilegeSet: &set,
		Privileges: &proclassic.GroupPrivileges{
			JssObjects: &proclassic.GroupPrivilegesJssObjects{Privilege: &[]string{"Read Computers", "Update Computers"}},
		},
	}
}

func TestDiscover_PrefersAdminGroup(t *testing.T) {
	custom := "Custom"
	f := fakeDiscoverer{
		groups: map[int]*proclassic.Group{
			1: {ID: intPtr(1), PrivilegeSet: &custom},
			8: adminGroup(8),
		},
	}
	cat, err := Discover(context.Background(), f)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !cat.Contains("Read Computers") || !cat.Contains("Update Computers") {
		t.Errorf("catalog missing expected privileges: %v", cat.All())
	}
	if cat.Contains("Nonexistent Privilege") {
		t.Error("catalog should not contain bogus privilege")
	}
}

func TestDiscover_NoAdminErrors(t *testing.T) {
	custom := "Custom"
	f := fakeDiscoverer{groups: map[int]*proclassic.Group{1: {ID: intPtr(1), PrivilegeSet: &custom}}}
	if _, err := Discover(context.Background(), f); err == nil {
		t.Error("expected error when no Administrator exists")
	}
}

func TestValidate_RejectsUnknownWithSuggestion(t *testing.T) {
	cat := catalogFromMap(map[string][]string{"jss_objects": {"Read Computers", "Update Computers"}})
	m := &Model{JamfProServerObjects: mustSet(t, "Read Computer")} // typo: missing trailing s
	diags := Validate(context.Background(), cat, m, path.Root("privileges"))
	if !diags.HasError() {
		t.Fatal("expected an error for unknown privilege")
	}
	if detail := diags[0].Detail(); !contains(detail, "Did you mean") || !contains(detail, "Read Computers") {
		t.Errorf("expected fuzzy suggestion, got: %s", detail)
	}
}

func TestValidate_AcceptsKnown(t *testing.T) {
	cat := catalogFromMap(map[string][]string{"jss_objects": {"Read Computers"}})
	m := &Model{JamfProServerObjects: mustSet(t, "Read Computers")}
	if diags := Validate(context.Background(), cat, m, path.Root("privileges")); diags.HasError() {
		t.Errorf("unexpected error: %v", diags)
	}
}

func TestLevenshtein(t *testing.T) {
	if d := levenshtein("Read Computers", "read computers"); d != 0 {
		t.Errorf("case-insensitive distance should be 0, got %d", d)
	}
	if d := levenshtein("abc", "abd"); d != 1 {
		t.Errorf("want 1, got %d", d)
	}
}

func intPtr(i int) *int { return &i }

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
