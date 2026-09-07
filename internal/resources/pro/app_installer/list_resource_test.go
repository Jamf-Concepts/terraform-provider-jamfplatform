// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestHydrateAppTitleName_Resolves asserts the hydration path writes the
// catalog's canonical title name, not the id it started from. The stored name is
// what the forward resolver matches on byte equality at plan time, so anything
// other than the canonical spelling would generate configuration that fails
// validation on the very next plan.
func TestHydrateAppTitleName_Resolves(t *testing.T) {
	state := AppInstallerResourceModel{AppTitleID: types.StringValue("027")}
	if !hydrateAppTitleName(context.Background(), &fakeTitleResolver{}, &state) {
		t.Fatalf("expected an id present in the catalog to resolve")
	}
	if got := state.AppTitleName.ValueString(); got != "Adobe Lightroom Classic" {
		t.Errorf("expected the canonical title name, got %q", got)
	}
}

// TestHydrateAppTitleName_DropsItem covers every way naming can fail. Each must
// report false and leave app_title_name null, because the caller drops the
// deployment rather than emitting a required attribute Terraform would refuse.
func TestHydrateAppTitleName_DropsItem(t *testing.T) {
	cases := map[string]struct {
		catalog titleCatalog
		titleID string
	}{
		"title withdrawn from the catalog": {&fakeTitleResolver{}, "no-such-id"},
		"empty catalog":                    {&fakeTitleResolver{notFound: true}, "027"},
		"catalog read failure":             {&fakeTitleResolver{otherErr: errors.New("connection refused")}, "027"},
		"catalog not configured":           {nil, "027"},
		"deployment reports no title":      {&fakeTitleResolver{}, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			state := AppInstallerResourceModel{AppTitleID: types.StringValue(tc.titleID)}
			if hydrateAppTitleName(context.Background(), tc.catalog, &state) {
				t.Fatalf("expected the deployment to be dropped")
			}
			if !state.AppTitleName.IsNull() {
				t.Errorf("a dropped deployment must leave app_title_name null, got %q", state.AppTitleName.ValueString())
			}
		})
	}
}

// TestAppTitleNameSkipWarning_Empty asserts the renderer is silent when nothing
// was dropped, which is what lets the caller append it unconditionally.
func TestAppTitleNameSkipWarning_Empty(t *testing.T) {
	if diags := appTitleNameSkipWarning(nil); len(diags) != 0 {
		t.Errorf("expected no diagnostics for an empty skip list, got %v", diags)
	}
}

// TestAppTitleNameSkipWarning_NamesEverySkip asserts one warning carries every
// dropped deployment, and that it is a warning rather than an error — dropping
// an item must not abandon config generation for the rest of the tenant.
func TestAppTitleNameSkipWarning_NamesEverySkip(t *testing.T) {
	diags := appTitleNameSkipWarning([]string{"1Password 8 (id 4)", "Google Chrome (id 9)"})
	if diags.HasError() {
		t.Fatalf("skipping an item must not be an error: %v", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("expected exactly 1 consolidated warning, got %d", diags.WarningsCount())
	}
	detail := diags.Warnings()[0].Detail()
	for _, want := range []string{"1Password 8 (id 4)", "Google Chrome (id 9)"} {
		if !strings.Contains(detail, want) {
			t.Errorf("warning detail does not name %q:\n%s", want, detail)
		}
	}
}

// TestAppTitleNameSkipWarning_CapsTheList asserts a wholesale catalog failure —
// which drops every deployment in the tenant — still reports one readable
// warning, spelling out a bounded sample and counting the rest.
func TestAppTitleNameSkipWarning_CapsTheList(t *testing.T) {
	skipped := make([]string, 0, maxListedTitleSkips+3)
	for i := range maxListedTitleSkips + 3 {
		skipped = append(skipped, "Deployment "+string(rune('a'+i)))
	}
	diags := appTitleNameSkipWarning(skipped)
	if diags.WarningsCount() != 1 {
		t.Fatalf("expected exactly 1 consolidated warning, got %d", diags.WarningsCount())
	}
	w := diags.Warnings()[0]
	if !strings.Contains(w.Summary(), "13") {
		t.Errorf("summary must report the full count, got %q", w.Summary())
	}
	if got := strings.Count(w.Detail(), "  - "); got != maxListedTitleSkips {
		t.Errorf("expected %d listed entries, got %d", maxListedTitleSkips, got)
	}
	if !strings.Contains(w.Detail(), "and 3 more") {
		t.Errorf("detail must count the unlisted remainder:\n%s", w.Detail())
	}
}
