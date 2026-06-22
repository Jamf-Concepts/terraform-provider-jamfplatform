// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package filters

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type fakeItem struct {
	Name string
}

func fakeName(i fakeItem) string { return i.Name }

func items(names ...string) []fakeItem {
	out := make([]fakeItem, len(names))
	for i, n := range names {
		out[i] = fakeItem{Name: n}
	}
	return out
}

func names(items []fakeItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Name
	}
	return out
}

func TestApplyClassicFilter_PassThrough(t *testing.T) {
	in := items("Primary", "HQ", "Branch")
	tests := []struct {
		name   string
		filter ClassicFilterModel
	}{
		{"null", ClassicFilterModel{NameSubstring: types.StringNull()}},
		{"unknown", ClassicFilterModel{NameSubstring: types.StringUnknown()}},
		{"empty", ClassicFilterModel{NameSubstring: types.StringValue("")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ApplyClassicFilter(in, tc.filter, fakeName)
			if !reflect.DeepEqual(got, in) {
				t.Errorf("expected pass-through for %q, got %v", tc.name, names(got))
			}
		})
	}
}

func TestApplyClassicFilter_SubstringMatch(t *testing.T) {
	in := items("Primary HQ", "Primary Branch", "Remote", "primary basement")
	got := ApplyClassicFilter(in, ClassicFilterModel{NameSubstring: types.StringValue("primary")}, fakeName)
	want := []string{"Primary HQ", "Primary Branch", "primary basement"}
	if !reflect.DeepEqual(names(got), want) {
		t.Errorf("got %v, want %v", names(got), want)
	}
}

func TestApplyClassicFilter_CaseInsensitive(t *testing.T) {
	in := items("PRIMARY", "primary", "Primary", "secondary")
	got := ApplyClassicFilter(in, ClassicFilterModel{NameSubstring: types.StringValue("PRIM")}, fakeName)
	want := []string{"PRIMARY", "primary", "Primary"}
	if !reflect.DeepEqual(names(got), want) {
		t.Errorf("got %v, want %v", names(got), want)
	}
}

func TestApplyClassicFilter_NoMatches(t *testing.T) {
	in := items("alpha", "beta")
	got := ApplyClassicFilter(in, ClassicFilterModel{NameSubstring: types.StringValue("zulu")}, fakeName)
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", names(got))
	}
}

func TestApplyClassicFilter_DoesNotMutateInput(t *testing.T) {
	in := items("alpha", "beta", "ALPHA")
	original := make([]fakeItem, len(in))
	copy(original, in)

	_ = ApplyClassicFilter(in, ClassicFilterModel{NameSubstring: types.StringValue("a")}, fakeName)

	if !reflect.DeepEqual(in, original) {
		t.Errorf("input slice was mutated: got %v, want %v", names(in), names(original))
	}
}

func TestApplyClassicFilter_EmptyInput(t *testing.T) {
	got := ApplyClassicFilter[fakeItem](nil, ClassicFilterModel{NameSubstring: types.StringValue("anything")}, fakeName)
	if got == nil {
		t.Fatalf("expected non-nil result when filter is active (even with nil input)")
	}
	if len(got) != 0 {
		t.Errorf("expected empty result for nil input, got %v", names(got))
	}
}

func TestClassicFilterAttribute_Shape(t *testing.T) {
	a := ClassicFilterAttribute()
	if !a.Optional {
		t.Errorf("expected outer Optional true")
	}
	inner, ok := a.Attributes["name_substring"]
	if !ok {
		t.Fatalf("expected name_substring attribute, got keys %v", keys(a.Attributes))
	}
	str, ok := inner.(schema.StringAttribute)
	if !ok {
		t.Fatalf("name_substring expected to be schema.StringAttribute, got %T", inner)
	}
	if !str.Optional {
		t.Errorf("name_substring should be Optional")
	}
}

func TestClassicListFilterAttribute_Shape(t *testing.T) {
	a := ClassicListFilterAttribute()
	if !a.Optional {
		t.Errorf("expected outer Optional true")
	}
	inner, ok := a.Attributes["name_substring"]
	if !ok {
		t.Fatalf("expected name_substring attribute")
	}
	str, ok := inner.(listschema.StringAttribute)
	if !ok {
		t.Fatalf("name_substring expected to be listschema.StringAttribute, got %T", inner)
	}
	if !str.Optional {
		t.Errorf("name_substring should be Optional")
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
