// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_macos_settings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fullPlan returns a plan with every attribute declared, distinct per field.
func fullPlan() SelfServiceMacosSettingsResourceModel {
	return SelfServiceMacosSettingsResourceModel{
		InstallAutomatically:             types.BoolValue(true),
		InstallLocation:                  types.StringValue("/Applications"),
		LoginMethod:                      types.StringValue("Required"),
		AuthenticationType:               types.StringValue("Saml"),
		KeychainCredentialStorageEnabled: types.BoolValue(false),
		Fido2Enabled:                     types.BoolValue(true),
		NotificationsEnabled:             types.BoolValue(true),
		AlertUserApprovedMdm:             types.BoolValue(false),
		DefaultLandingPage:               types.StringValue("BROWSE"),
		DefaultHomeCategoryID:            types.Int64Value(7),
		BookmarksDisplayName:             types.StringValue("Websites"),
	}
}

// TestBuildInput_DeclaredPlanWins verifies every declared plan value reaches the payload
// and the payload always carries all three setting groups (the wire 500s without them).
func TestBuildInput_DeclaredPlanWins(t *testing.T) {
	body := buildSelfServiceMacosSettingsInput(fullPlan(), nil)

	if body.InstallSettings == nil || body.LoginSettings == nil || body.ConfigurationSettings == nil {
		t.Fatalf("all three setting groups must always be present, got %+v", body)
	}
	if body.InstallSettings.InstallAutomatically == nil || !*body.InstallSettings.InstallAutomatically {
		t.Errorf("installAutomatically = %v, want true", body.InstallSettings.InstallAutomatically)
	}
	if body.InstallSettings.InstallLocation != "/Applications" {
		t.Errorf("installLocation = %q, want /Applications", body.InstallSettings.InstallLocation)
	}
	if body.LoginSettings.UserLoginLevel != "Required" {
		t.Errorf("userLoginLevel = %q, want Required", body.LoginSettings.UserLoginLevel)
	}
	if body.LoginSettings.AuthType != "Saml" {
		t.Errorf("authType = %q, want Saml", body.LoginSettings.AuthType)
	}
	if body.LoginSettings.AllowRememberMe == nil || *body.LoginSettings.AllowRememberMe {
		t.Errorf("allowRememberMe = %v, want false", body.LoginSettings.AllowRememberMe)
	}
	if body.LoginSettings.UseFido2 == nil || !*body.LoginSettings.UseFido2 {
		t.Errorf("useFido2 = %v, want true", body.LoginSettings.UseFido2)
	}
	if body.ConfigurationSettings.NotificationsEnabled == nil || !*body.ConfigurationSettings.NotificationsEnabled {
		t.Errorf("notificationsEnabled = %v, want true", body.ConfigurationSettings.NotificationsEnabled)
	}
	if body.ConfigurationSettings.AlertUserApprovedMDM == nil || *body.ConfigurationSettings.AlertUserApprovedMDM {
		t.Errorf("alertUserApprovedMdm = %v, want false", body.ConfigurationSettings.AlertUserApprovedMDM)
	}
	if body.ConfigurationSettings.DefaultLandingPage == nil || *body.ConfigurationSettings.DefaultLandingPage != "BROWSE" {
		t.Errorf("defaultLandingPage = %v, want BROWSE", body.ConfigurationSettings.DefaultLandingPage)
	}
	if body.ConfigurationSettings.DefaultHomeCategoryID == nil || *body.ConfigurationSettings.DefaultHomeCategoryID != 7 {
		t.Errorf("defaultHomeCategoryId = %v, want 7", body.ConfigurationSettings.DefaultHomeCategoryID)
	}
	if body.ConfigurationSettings.BookmarksName != "Websites" {
		t.Errorf("bookmarksName = %q, want Websites", body.ConfigurationSettings.BookmarksName)
	}
}

// TestBuildInput_NullPlanAdoptsCurrent verifies the create-adopt path: a plan declaring
// nothing re-emits every value from the live merge base, so the full-replace PUT does not
// reset undeclared fields.
func TestBuildInput_NullPlanAdoptsCurrent(t *testing.T) {
	body := buildSelfServiceMacosSettingsInput(SelfServiceMacosSettingsResourceModel{}, fullResponse())

	if body.InstallSettings.InstallAutomatically == nil || !*body.InstallSettings.InstallAutomatically {
		t.Errorf("installAutomatically = %v, want adopted true", body.InstallSettings.InstallAutomatically)
	}
	if body.InstallSettings.InstallLocation != "/Applications" {
		t.Errorf("installLocation = %q, want adopted /Applications", body.InstallSettings.InstallLocation)
	}
	if body.LoginSettings.UserLoginLevel != "Anonymous" {
		t.Errorf("userLoginLevel = %q, want adopted Anonymous", body.LoginSettings.UserLoginLevel)
	}
	if body.LoginSettings.AuthType != "Basic" {
		t.Errorf("authType = %q, want adopted Basic", body.LoginSettings.AuthType)
	}
	if body.LoginSettings.AllowRememberMe == nil || !*body.LoginSettings.AllowRememberMe {
		t.Errorf("allowRememberMe = %v, want adopted true", body.LoginSettings.AllowRememberMe)
	}
	if body.ConfigurationSettings.DefaultLandingPage == nil || *body.ConfigurationSettings.DefaultLandingPage != "BROWSE" {
		t.Errorf("defaultLandingPage = %v, want adopted BROWSE", body.ConfigurationSettings.DefaultLandingPage)
	}
	if body.ConfigurationSettings.DefaultHomeCategoryID == nil || *body.ConfigurationSettings.DefaultHomeCategoryID != 42 {
		t.Errorf("defaultHomeCategoryId = %v, want adopted 42", body.ConfigurationSettings.DefaultHomeCategoryID)
	}
	if body.ConfigurationSettings.BookmarksName != "Bookmarks" {
		t.Errorf("bookmarksName = %q, want adopted Bookmarks", body.ConfigurationSettings.BookmarksName)
	}
}

// TestBuildInput_MixedDeclaredAndAdopted verifies a partially-declared plan overlays the
// merge base per-field rather than per-group.
func TestBuildInput_MixedDeclaredAndAdopted(t *testing.T) {
	plan := SelfServiceMacosSettingsResourceModel{
		LoginMethod:          types.StringValue("Required"),
		NotificationsEnabled: types.BoolValue(false),
	}
	body := buildSelfServiceMacosSettingsInput(plan, fullResponse())

	if body.LoginSettings.UserLoginLevel != "Required" {
		t.Errorf("userLoginLevel = %q, want declared Required", body.LoginSettings.UserLoginLevel)
	}
	if body.LoginSettings.AuthType != "Basic" {
		t.Errorf("authType = %q, want adopted Basic (sibling in same group)", body.LoginSettings.AuthType)
	}
	if body.ConfigurationSettings.NotificationsEnabled == nil || *body.ConfigurationSettings.NotificationsEnabled {
		t.Errorf("notificationsEnabled = %v, want declared false", body.ConfigurationSettings.NotificationsEnabled)
	}
	if body.ConfigurationSettings.BookmarksName != "Bookmarks" {
		t.Errorf("bookmarksName = %q, want adopted Bookmarks", body.ConfigurationSettings.BookmarksName)
	}
}

// TestBuildInput_UnknownTreatedAsNull verifies Unknown plan values fall back to the merge
// base the same way nulls do (never serialized as zero values).
func TestBuildInput_UnknownTreatedAsNull(t *testing.T) {
	plan := SelfServiceMacosSettingsResourceModel{
		InstallAutomatically:  types.BoolUnknown(),
		BookmarksDisplayName:  types.StringUnknown(),
		DefaultHomeCategoryID: types.Int64Unknown(),
	}
	body := buildSelfServiceMacosSettingsInput(plan, fullResponse())

	if body.InstallSettings.InstallAutomatically == nil || !*body.InstallSettings.InstallAutomatically {
		t.Errorf("installAutomatically = %v, want adopted true (unknown plan)", body.InstallSettings.InstallAutomatically)
	}
	if body.ConfigurationSettings.BookmarksName != "Bookmarks" {
		t.Errorf("bookmarksName = %q, want adopted Bookmarks (unknown plan)", body.ConfigurationSettings.BookmarksName)
	}
	if body.ConfigurationSettings.DefaultHomeCategoryID == nil || *body.ConfigurationSettings.DefaultHomeCategoryID != 42 {
		t.Errorf("defaultHomeCategoryId = %v, want adopted 42 (unknown plan)", body.ConfigurationSettings.DefaultHomeCategoryID)
	}
}

// TestBuildInput_NilCurrentNullPlan verifies the unreachable-in-practice fallback (no merge
// base, nothing declared) omits pointer fields rather than zero-filling them.
func TestBuildInput_NilCurrentNullPlan(t *testing.T) {
	body := buildSelfServiceMacosSettingsInput(SelfServiceMacosSettingsResourceModel{}, nil)

	if body.InstallSettings.InstallAutomatically != nil {
		t.Errorf("installAutomatically = %v, want nil (omit, server default)", body.InstallSettings.InstallAutomatically)
	}
	if body.ConfigurationSettings.DefaultHomeCategoryID != nil {
		t.Errorf("defaultHomeCategoryId = %v, want nil (omit, server default)", body.ConfigurationSettings.DefaultHomeCategoryID)
	}
	if body.ConfigurationSettings.DefaultLandingPage != nil {
		t.Errorf("defaultLandingPage = %v, want nil (omit, server default)", body.ConfigurationSettings.DefaultLandingPage)
	}
}
