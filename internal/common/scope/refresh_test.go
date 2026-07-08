// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRefreshManagedSet(t *testing.T) {
	wire := newStringSet(t, []string{"5"})

	if got := RefreshManagedSet(types.SetNull(types.StringType), wire, false); !got.IsNull() {
		t.Errorf("unmanaged (null current) must stay null, got %v", got)
	}
	if got := RefreshManagedSet(newStringSet(t, nil), wire, false); got.IsNull() || len(got.Elements()) != 1 {
		t.Errorf("managed [] current must adopt wire, got %v", got)
	}
	if got := RefreshManagedSet(newStringSet(t, []string{"9"}), wire, false); len(sortedSetValues(t, got)) != 1 || sortedSetValues(t, got)[0] != "5" {
		t.Errorf("managed current must refresh to wire, got %v", got)
	}
	if got := RefreshManagedSet(types.SetNull(types.StringType), wire, true); got.IsNull() {
		t.Errorf("includeUnmanaged must hydrate, got %v", got)
	}
}

func TestRefreshManagedBool(t *testing.T) {
	wireTrue := true

	if got := RefreshManagedBool(types.BoolNull(), &wireTrue, false); !got.IsNull() {
		t.Errorf("unmanaged flag must stay null, got %v", got)
	}
	if got := RefreshManagedBool(types.BoolValue(false), &wireTrue, false); !got.ValueBool() {
		t.Errorf("managed flag must refresh to wire, got %v", got)
	}
	if got := RefreshManagedBool(types.BoolValue(false), nil, false); got.IsNull() || got.ValueBool() {
		t.Errorf("nil wire must keep current for managed flag, got %v", got)
	}
	if got := RefreshManagedBool(types.BoolNull(), &wireTrue, true); !got.ValueBool() {
		t.Errorf("includeUnmanaged must hydrate flag, got %v", got)
	}
}
