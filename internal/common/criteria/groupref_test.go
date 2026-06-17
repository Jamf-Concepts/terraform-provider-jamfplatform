// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package criteria

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fakeGroupResolver models the proclassic group endpoints with an in-memory
// id<->name table. calls counts every lookup so a test can assert the
// backward-compatibility fast path performs NO lookup. err forces every lookup to
// fail (transient / missing-privilege simulation).
type fakeGroupResolver struct {
	byID  map[string]string // id -> canonical name
	calls int
	err   error
}

func (f *fakeGroupResolver) NameByID(_ context.Context, _ ObjectType, id string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	if n, ok := f.byID[id]; ok {
		return n, nil
	}
	return "", errors.New("not found")
}

func (f *fakeGroupResolver) IDByName(_ context.Context, _ ObjectType, name string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	for id, n := range f.byID {
		if strings.EqualFold(n, name) { // /name/<n> classic lookup is case-insensitive
			return id, nil
		}
	}
	return "", errors.New("not found")
}

func newResolver() *fakeGroupResolver {
	return &fakeGroupResolver{byID: map[string]string{"3": "Excluded Users", "25": "All 10.16+ Devices"}}
}

func crit(name, op, value string) CriterionModel {
	return CriterionModel{
		Name:       types.StringValue(name),
		SearchType: types.StringValue(op),
		Value:      types.StringValue(value),
		Priority:   types.Int64Value(0),
	}
}

func TestIsJamfGroupCriterion(t *testing.T) {
	tests := []struct {
		ot       ObjectType
		name, op string
		want     bool
		desc     string
	}{
		{ObjectTypeUser, "User Group", "member of", true, "user group member of"},
		{ObjectTypeUser, "User Group", "not member of", true, "user group not member of"},
		{ObjectTypeUser, "User Group", "MEMBER OF", true, "operator case-insensitive"},
		{ObjectTypeComputer, "Computer Group", "member of", true, "computer group"},
		{ObjectTypeMobile, "Mobile Device Group", "member of", true, "mobile group"},
		{ObjectTypeUser, "User Group", "is", false, "non-membership operator"},
		{ObjectTypeUser, "Full Name", "member of", false, "non-group criterion name"},
		{ObjectTypeComputer, "User Group", "member of", false, "wrong name for class"},
		{ObjectTypeUser, "user group", "member of", false, "criterion name is case-sensitive"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := IsJamfGroupCriterion(tt.ot, tt.name, tt.op); got != tt.want {
				t.Fatalf("IsJamfGroupCriterion(%v,%q,%q) = %v, want %v", tt.ot, tt.name, tt.op, got, tt.want)
			}
		})
	}
}

// TestReadGroupValue_BackwardCompatNoLookup is the load-bearing backward-compat
// assertion: when the wire value already equals the configured value (a pre-11.29
// server returning the name, or the steady state), the value is kept verbatim and
// the resolver is NEVER called. This is what lets the same code path serve old and
// new Jamf Pro with no version branch and no group-read privilege requirement.
func TestReadGroupValue_BackwardCompatNoLookup(t *testing.T) {
	r := newResolver()
	got := ReadGroupValue(context.Background(), r, ObjectTypeUser, "Excluded Users", "Excluded Users")
	if got != "Excluded Users" {
		t.Fatalf("got %q, want the name unchanged", got)
	}
	if r.calls != 0 {
		t.Fatalf("resolver called %d times on the wire==prior fast path; want 0", r.calls)
	}
}

func TestReadGroupValue(t *testing.T) {
	tests := []struct {
		desc        string
		wire, prior string
		err         error
		want        string
	}{
		{"11.29 id maps back to configured name", "3", "Excluded Users", nil, "Excluded Users"},
		{"case-insensitive wire==prior keeps authored casing", "excluded users", "Excluded Users", nil, "Excluded Users"},
		{"id names a different group -> drift (wire)", "25", "Excluded Users", nil, "25"},
		{"unresolvable id -> drift (wire)", "999", "Excluded Users", nil, "999"},
		{"resolver error -> drift (wire), never panic", "3", "Excluded Users", errors.New("403"), "3"},
		{"import (no prior): id reverse-resolves to name", "3", "", nil, "Excluded Users"},
		{"import (no prior) old-Jamf name passes through", "Excluded Users", "", nil, "Excluded Users"},
		{"import (no prior) unresolvable -> wire", "999", "", nil, "999"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			r := newResolver()
			r.err = tt.err
			if got := ReadGroupValue(context.Background(), r, ObjectTypeUser, tt.wire, tt.prior); got != tt.want {
				t.Fatalf("ReadGroupValue(%q,%q) = %q, want %q", tt.wire, tt.prior, got, tt.want)
			}
		})
	}
}

func TestGroupValuesEquivalent(t *testing.T) {
	tests := []struct {
		desc string
		a, b string
		err  error
		want bool
	}{
		{"identical", "Excluded Users", "Excluded Users", nil, true},
		{"case-only", "excluded users", "Excluded Users", nil, true},
		{"name vs its id", "Excluded Users", "3", nil, true},
		{"id vs name", "3", "Excluded Users", nil, true},
		{"different groups", "Excluded Users", "All 10.16+ Devices", nil, false},
		{"resolve failure -> not equivalent", "Excluded Users", "3", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			r := newResolver()
			r.err = tt.err
			if got := GroupValuesEquivalent(context.Background(), r, ObjectTypeUser, tt.a, tt.b); got != tt.want {
				t.Fatalf("GroupValuesEquivalent(%q,%q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestResolveAuthoredGroupRefMap(t *testing.T) {
	r := newResolver()
	models := []CriterionModel{
		crit("User Group", "member of", "Excluded Users"), // name -> keyed by id "3"
		crit("User Group", "member of", "3"),              // authored as id -> keyed by "3"
		crit("Full Name", "is", "Excluded Users"),         // not a group criterion -> skipped
	}
	got := ResolveAuthoredGroupRefMap(context.Background(), r, ObjectTypeUser, models)
	if v, ok := got["3"]; !ok || v == "" {
		t.Fatalf("expected id 3 keyed in authored map, got %#v", got)
	}
	// nil resolver -> empty map, no panic.
	if m := ResolveAuthoredGroupRefMap(context.Background(), nil, ObjectTypeUser, models); len(m) != 0 {
		t.Fatalf("nil resolver should yield empty map, got %#v", m)
	}
}

func TestRestoreAuthoredGroupRefCriteria(t *testing.T) {
	authored := map[string]string{"3": "Excluded Users"}
	// 11.29 flatten returns the id; restore puts the authored name back.
	flattened := []CriterionModel{crit("User Group", "member of", "3")}
	out := RestoreAuthoredGroupRefCriteria(flattened, authored, ObjectTypeUser)
	if out[0].Value.ValueString() != "Excluded Users" {
		t.Fatalf("got %q, want restored name", out[0].Value.ValueString())
	}
	// pre-11.29 flatten returns the name; not in the id-keyed map -> left as-is.
	oldJamf := []CriterionModel{crit("User Group", "member of", "Excluded Users")}
	out = RestoreAuthoredGroupRefCriteria(oldJamf, authored, ObjectTypeUser)
	if out[0].Value.ValueString() != "Excluded Users" {
		t.Fatalf("old-Jamf name should pass through, got %q", out[0].Value.ValueString())
	}
	// empty authored map / nil slice preserved.
	if RestoreAuthoredGroupRefCriteria(flattened, nil, ObjectTypeUser)[0].Value.ValueString() != "3" {
		t.Fatal("empty map must leave the value as flattened")
	}
	if RestoreAuthoredGroupRefCriteria(nil, authored, ObjectTypeUser) != nil {
		t.Fatal("nil slice must be preserved")
	}
}

func TestReadbackGroupRefCriteria(t *testing.T) {
	r := newResolver()
	wire := []CriterionModel{
		crit("User Group", "member of", "3"), // -> Excluded Users
		crit("Full Name", "is", "tf-acc"),    // untouched
	}
	prior := []CriterionModel{
		crit("User Group", "member of", "Excluded Users"),
		crit("Full Name", "is", "tf-acc"),
	}
	out := ReadbackGroupRefCriteria(context.Background(), r, ObjectTypeUser, wire, prior)
	if out[0].Value.ValueString() != "Excluded Users" {
		t.Fatalf("group criterion not mapped: %q", out[0].Value.ValueString())
	}
	if out[1].Value.ValueString() != "tf-acc" {
		t.Fatalf("non-group criterion must be untouched: %q", out[1].Value.ValueString())
	}
	// nil resolver / nil slice preserved.
	if ReadbackGroupRefCriteria(context.Background(), nil, ObjectTypeUser, wire, prior)[0].Value.ValueString() != "3" {
		t.Fatal("nil resolver must leave wire unchanged")
	}
	if ReadbackGroupRefCriteria(context.Background(), r, ObjectTypeUser, nil, prior) != nil {
		t.Fatal("nil slice must be preserved")
	}
}

func TestSuppressEquivalentGroupRefValues(t *testing.T) {
	r := newResolver()
	// Planned pastes the id; prior holds the name for the same group -> collapse to prior.
	planned := []CriterionModel{crit("User Group", "member of", "3")}
	planned[0].Priority = types.Int64Unknown() // Optional+Computed churns to Unknown on a criteria change
	prior := []CriterionModel{crit("User Group", "member of", "Excluded Users")}
	out := SuppressEquivalentGroupRefValues(context.Background(), r, ObjectTypeUser, planned, prior)
	if out[0].Value.ValueString() != "Excluded Users" {
		t.Fatalf("equivalent swap not collapsed: %q", out[0].Value.ValueString())
	}
	if out[0].Priority.IsUnknown() || out[0].Priority.ValueInt64() != 0 {
		t.Fatalf("Unknown sibling not reconciled from prior: %v", out[0].Priority)
	}
	// Different group -> not suppressed.
	planned2 := []CriterionModel{crit("User Group", "member of", "25")}
	prior2 := []CriterionModel{crit("User Group", "member of", "Excluded Users")}
	out = SuppressEquivalentGroupRefValues(context.Background(), r, ObjectTypeUser, planned2, prior2)
	if out[0].Value.ValueString() != "25" {
		t.Fatalf("non-equivalent value must be left intact: %q", out[0].Value.ValueString())
	}
	// nil resolver / nil slice preserved.
	if SuppressEquivalentGroupRefValues(context.Background(), nil, ObjectTypeUser, planned, prior)[0].Value.ValueString() != "3" {
		t.Fatal("nil resolver must leave the plan unchanged")
	}
	if SuppressEquivalentGroupRefValues(context.Background(), r, ObjectTypeUser, nil, prior) != nil {
		t.Fatal("nil slice must be preserved")
	}
}
