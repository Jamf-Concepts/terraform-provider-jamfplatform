// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package class

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestReconcileStringSet_EmptyPreservesShape(t *testing.T) {
	ctx := context.Background()

	// null current + empty API → null
	got, d := reconcileStringSet(ctx, nil, types.SetNull(types.StringType), false)
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	if !got.IsNull() {
		t.Errorf("null current + empty API should stay null")
	}

	// empty-set current + empty API → empty set (not null)
	got, d = reconcileStringSet(ctx, nil, strSet(t), false)
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	if got.IsNull() {
		t.Errorf("empty-set current + empty API should stay an empty set")
	}
	if len(setElems(t, got)) != 0 {
		t.Errorf("expected empty set")
	}
}

func TestReconcileStringSet_CaseInsensitivePreservesConfigCasing(t *testing.T) {
	ctx := context.Background()
	current := strSet(t, "Kyle.Hoare@JAMF.com", "Other.User@X.com")
	// Server canonicalises to lowercase, and adds a brand-new member.
	api := []string{"kyle.hoare@jamf.com", "new.member@y.com"}

	got, d := reconcileStringSet(ctx, api, current, true)
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	elems := map[string]bool{}
	for _, e := range setElems(t, got) {
		elems[e] = true
	}
	if !elems["Kyle.Hoare@JAMF.com"] {
		t.Errorf("matched member must keep config casing; got %v", setElems(t, got))
	}
	if !elems["new.member@y.com"] {
		t.Errorf("new member must use server value; got %v", setElems(t, got))
	}
}

func TestReconcileStringSet_CaseSensitiveExactValues(t *testing.T) {
	ctx := context.Background()
	got, d := reconcileStringSet(ctx, []string{"66", "876"}, strSet(t, "66"), false)
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	if len(setElems(t, got)) != 2 {
		t.Errorf("expected both ids, got %v", setElems(t, got))
	}
}

func TestIntSliceFromSet_Errors(t *testing.T) {
	ctx := context.Background()
	if _, d := intSliceFromSet(ctx, strSet(t, "abc"), "student_group_ids"); !d.HasError() {
		t.Errorf("expected error for non-integer id")
	}
	ids, d := intSliceFromSet(ctx, strSet(t, "3", "66"), "student_group_ids")
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 ids, got %v", ids)
	}
	// null set → empty (non-nil) slice for always-emit.
	ids, d = intSliceFromSet(ctx, types.SetNull(types.StringType), "x")
	if d.HasError() || ids == nil || len(ids) != 0 {
		t.Errorf("null set must yield empty non-nil slice")
	}
}
