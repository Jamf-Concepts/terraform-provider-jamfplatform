// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// testItem stands in for an SDK item type — kept local so the tests do not
// depend on the SDK package.
type testItem struct {
	ID   *int
	Name *string
}

func mkID(id int) testItem      { return testItem{ID: &id} }
func mkName(n string) testItem  { return testItem{Name: &n} }
func extractID(i testItem) *int { return i.ID }
func extractName(i testItem) *string {
	return i.Name
}

func newStringSet(t *testing.T, values []string) types.Set {
	t.Helper()
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	out, diags := types.SetValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("newStringSet: %v", diags)
	}
	return out
}

func sortedSetValues(t *testing.T, s types.Set) []string {
	t.Helper()
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	var out []string
	diags := s.ElementsAs(context.Background(), &out, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs: %v", diags)
	}
	sort.Strings(out)
	return out
}

func TestBuildIDSlice_NullSet(t *testing.T) {
	got, diags := BuildIDSlice(context.Background(), types.SetNull(types.StringType), mkID)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got != nil {
		t.Errorf("expected nil slice, got %+v", got)
	}
}

func TestBuildIDSlice_UnknownSet(t *testing.T) {
	got, diags := BuildIDSlice(context.Background(), types.SetUnknown(types.StringType), mkID)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got != nil {
		t.Errorf("expected nil slice, got %+v", got)
	}
}

func TestBuildIDSlice_EmptySet(t *testing.T) {
	got, diags := BuildIDSlice(context.Background(), newStringSet(t, nil), mkID)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got != nil {
		t.Errorf("expected nil slice for empty set, got %+v", got)
	}
}

func TestBuildIDSlice_Populated(t *testing.T) {
	set := newStringSet(t, []string{"1", "2"})
	got, diags := BuildIDSlice(context.Background(), set, mkID)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(*got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(*got))
	}
	ids := []int{*(*got)[0].ID, *(*got)[1].ID}
	sort.Ints(ids)
	if ids[0] != 1 || ids[1] != 2 {
		t.Errorf("expected IDs [1 2], got %v", ids)
	}
}

func TestBuildIDSlice_ParseErrorsAreCollected(t *testing.T) {
	set := newStringSet(t, []string{"1", "abc", "3", "xyz"})
	got, diags := BuildIDSlice(context.Background(), set, mkID)
	if got != nil {
		t.Errorf("expected nil slice on parse failure, got %+v", got)
	}
	if !diags.HasError() {
		t.Fatal("expected diagnostics with errors")
	}
	if diags.ErrorsCount() < 2 {
		t.Errorf("expected at least 2 parse errors collected in one pass, got %d", diags.ErrorsCount())
	}
}

func TestFlattenIDSlice_NilSlice(t *testing.T) {
	got, diags := FlattenIDSlice[testItem](context.Background(), nil, extractID)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.IsNull() {
		t.Errorf("expected null set, got %+v", got)
	}
}

func TestFlattenIDSlice_EmptySlice(t *testing.T) {
	empty := []testItem{}
	got, diags := FlattenIDSlice(context.Background(), &empty, extractID)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.IsNull() {
		t.Errorf("expected null set for empty slice, got %+v", got)
	}
}

func TestFlattenIDSlice_Populated(t *testing.T) {
	items := []testItem{mkID(7), mkID(42)}
	got, diags := FlattenIDSlice(context.Background(), &items, extractID)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	values := sortedSetValues(t, got)
	if len(values) != 2 || values[0] != "42" || values[1] != "7" {
		t.Errorf("expected sorted [42 7] (string sort), got %v", values)
	}
}

func TestFlattenIDSlice_SkipsNilExtract(t *testing.T) {
	items := []testItem{mkID(1), {ID: nil}, mkID(3)}
	got, diags := FlattenIDSlice(context.Background(), &items, extractID)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	values := sortedSetValues(t, got)
	if len(values) != 2 || values[0] != "1" || values[1] != "3" {
		t.Errorf("expected [1 3], got %v", values)
	}
}

func TestFlattenIDSlice_AllNilCollapsesToNull(t *testing.T) {
	items := []testItem{{ID: nil}, {ID: nil}}
	got, diags := FlattenIDSlice(context.Background(), &items, extractID)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.IsNull() {
		t.Errorf("expected null set when every item extracts to nil, got %+v", got)
	}
}

func TestBuildNameSlice_NullSet(t *testing.T) {
	got, diags := BuildNameSlice(context.Background(), types.SetNull(types.StringType), mkName)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got != nil {
		t.Errorf("expected nil slice, got %+v", got)
	}
}

func TestBuildNameSlice_UnknownSet(t *testing.T) {
	got, diags := BuildNameSlice(context.Background(), types.SetUnknown(types.StringType), mkName)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got != nil {
		t.Errorf("expected nil slice, got %+v", got)
	}
}

func TestBuildNameSlice_EmptySet(t *testing.T) {
	got, diags := BuildNameSlice(context.Background(), newStringSet(t, nil), mkName)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got != nil {
		t.Errorf("expected nil slice for empty set, got %+v", got)
	}
}

func TestBuildNameSlice_Populated(t *testing.T) {
	set := newStringSet(t, []string{"alice", "bob"})
	got, diags := BuildNameSlice(context.Background(), set, mkName)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got == nil || len(*got) != 2 {
		t.Fatalf("expected 2 items, got %+v", got)
	}
	names := []string{*(*got)[0].Name, *(*got)[1].Name}
	sort.Strings(names)
	if names[0] != "alice" || names[1] != "bob" {
		t.Errorf("expected [alice bob], got %v", names)
	}
}

func TestFlattenNameSlice_NilSlice(t *testing.T) {
	got, diags := FlattenNameSlice[testItem](context.Background(), nil, extractName)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.IsNull() {
		t.Errorf("expected null set, got %+v", got)
	}
}

func TestFlattenNameSlice_Populated(t *testing.T) {
	items := []testItem{mkName("alice"), mkName("bob")}
	got, diags := FlattenNameSlice(context.Background(), &items, extractName)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	values := sortedSetValues(t, got)
	if len(values) != 2 || values[0] != "alice" || values[1] != "bob" {
		t.Errorf("expected [alice bob], got %v", values)
	}
}

func TestFlattenNameSlice_SkipsNilAndEmptyString(t *testing.T) {
	empty := ""
	items := []testItem{mkName("alice"), {Name: nil}, {Name: &empty}, mkName("carol")}
	got, diags := FlattenNameSlice(context.Background(), &items, extractName)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	values := sortedSetValues(t, got)
	if len(values) != 2 || values[0] != "alice" || values[1] != "carol" {
		t.Errorf("expected [alice carol], got %v", values)
	}
}
