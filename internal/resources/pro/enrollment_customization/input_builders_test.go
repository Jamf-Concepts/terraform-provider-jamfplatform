// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enrollment_customization

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildParentInput_DefaultsAndIconURLPrecedence(t *testing.T) {
	branding := &brandingSettingsModel{
		BodyTextColor:   types.StringValue("111111"),
		ButtonColor:     types.StringValue("222222"),
		ButtonTextColor: types.StringValue("333333"),
		BackgroundColor: types.StringValue("444444"),
		IconURL:         types.StringValue("https://user/typed.png"),
	}

	// Case 1: no upload override, user-supplied URL passes through.
	plan := EnrollmentCustomizationResourceModel{
		DisplayName:      types.StringValue("Welcome"),
		Description:      types.StringValue("desc"),
		SiteID:           types.StringNull(),
		BrandingSettings: branding,
	}
	got := buildParentInput(plan, "")
	if got.DisplayName != "Welcome" {
		t.Fatalf("DisplayName = %q", got.DisplayName)
	}
	if got.Description != "desc" {
		t.Fatalf("Description = %q", got.Description)
	}
	if got.SiteID != siteIDNoneSentinel {
		t.Fatalf("SiteID default = %q, want %q", got.SiteID, siteIDNoneSentinel)
	}
	if got.EnrollmentCustomizationBrandingSettings.IconURL != "https://user/typed.png" {
		t.Fatalf("IconURL = %q, want user-supplied", got.EnrollmentCustomizationBrandingSettings.IconURL)
	}
	if got.EnrollmentCustomizationBrandingSettings.TextColor != "111111" {
		t.Fatalf("TextColor mapping wrong, got %q", got.EnrollmentCustomizationBrandingSettings.TextColor)
	}

	// Case 2: upload override wins over user-supplied URL.
	got = buildParentInput(plan, "https://uploaded/icon.png")
	if got.EnrollmentCustomizationBrandingSettings.IconURL != "https://uploaded/icon.png" {
		t.Fatalf("upload override IconURL = %q", got.EnrollmentCustomizationBrandingSettings.IconURL)
	}

	// Case 3: neither override nor user URL — empty key still serialised.
	branding2 := *branding
	branding2.IconURL = types.StringNull()
	plan.BrandingSettings = &branding2
	got = buildParentInput(plan, "")
	if got.EnrollmentCustomizationBrandingSettings.IconURL != "" {
		t.Fatalf("expected empty IconURL, got %q", got.EnrollmentCustomizationBrandingSettings.IconURL)
	}
}

func TestBuildTextPanelInput_SubtextOptional(t *testing.T) {
	withSubtext := textPaneModel{
		DisplayName:        types.StringValue("p1"),
		Rank:               types.Int64Value(0),
		Title:              types.StringValue("t"),
		Body:               types.StringValue("b"),
		Subtext:            types.StringValue("hello"),
		PreviousButtonText: types.StringValue("prev"),
		NextButtonText:     types.StringValue("next"),
	}
	got := buildTextPanelInput(withSubtext)
	if got.Subtext == nil || *got.Subtext != "hello" {
		t.Fatalf("Subtext = %v, want pointer to %q", got.Subtext, "hello")
	}
	if got.BackButtonText != "prev" || got.ContinueButtonText != "next" {
		t.Fatalf("button text mapping wrong: back=%q continue=%q", got.BackButtonText, got.ContinueButtonText)
	}

	noSubtext := withSubtext
	noSubtext.Subtext = types.StringNull()
	got = buildTextPanelInput(noSubtext)
	if got.Subtext != nil {
		t.Fatalf("nil subtext expected, got pointer to %q", *got.Subtext)
	}
}

func TestBuildLdapPanelInput_GroupAccessRoundTrip(t *testing.T) {
	p := ldapPaneModel{
		DisplayName:        types.StringValue("ldap"),
		Rank:               types.Int64Value(0),
		Title:              types.StringValue("t"),
		UsernameText:       types.StringValue("u"),
		PasswordText:       types.StringValue("p"),
		PreviousButtonText: types.StringValue("prev"),
		LoginButtonText:    types.StringValue("login"),
		DirectoryServiceGroups: []directoryServiceGroupModel{
			{GroupName: types.StringValue("g1"), DirectoryServiceServerID: types.Int64Value(7)},
		},
	}
	got := buildLdapPanelInput(p)
	if got.UsernameLabel != "u" || got.PasswordLabel != "p" {
		t.Fatalf("label mapping wrong: u=%q p=%q", got.UsernameLabel, got.PasswordLabel)
	}
	if got.ContinueButtonText != "login" {
		t.Fatalf("login_button_text should map to continueButtonText; got %q", got.ContinueButtonText)
	}
	if got.LdapGroupAccess == nil || len(*got.LdapGroupAccess) != 1 {
		t.Fatalf("LdapGroupAccess wrong: %+v", got.LdapGroupAccess)
	}
	entry := (*got.LdapGroupAccess)[0]
	if entry.GroupName == nil || *entry.GroupName != "g1" {
		t.Fatalf("group name = %v", entry.GroupName)
	}
	if entry.LdapServerID == nil || *entry.LdapServerID != 7 {
		t.Fatalf("server id = %v", entry.LdapServerID)
	}

	// Empty groups list → nil pointer (wire omits the key).
	p.DirectoryServiceGroups = nil
	got = buildLdapPanelInput(p)
	if got.LdapGroupAccess != nil {
		t.Fatalf("expected nil LdapGroupAccess for empty group list")
	}
}

func TestBuildSsoPanelInput_EnrollmentAccessEnumToBool(t *testing.T) {
	base := ssoPaneModel{
		DisplayName:               types.StringValue("sso"),
		Rank:                      types.Int64Value(0),
		PassUserInfoToJamfConnect: types.BoolValue(true),
		AccountNameAttribute:      types.StringValue("uid"),
		AccountFullNameAttribute:  types.StringValue("name"),
	}

	// specific_group → true + group name preserved.
	p := base
	p.EnrollmentAccess = types.StringValue(enrollmentAccessSpecificGroup)
	p.AccessGroupName = types.StringValue("Admins")
	got := buildSsoPanelInput(p)
	if !got.IsGroupEnrollmentAccessEnabled {
		t.Fatalf("specific_group must map to true")
	}
	if got.GroupEnrollmentAccessName != "Admins" {
		t.Fatalf("group name dropped: %q", got.GroupEnrollmentAccessName)
	}

	// any_idp_user → false + group name omitted regardless of input.
	p2 := base
	p2.EnrollmentAccess = types.StringValue(enrollmentAccessAnyIdpUser)
	p2.AccessGroupName = types.StringValue("Ignored")
	got = buildSsoPanelInput(p2)
	if got.IsGroupEnrollmentAccessEnabled {
		t.Fatalf("any_idp_user must map to false")
	}
	if got.GroupEnrollmentAccessName != "" {
		t.Fatalf("group name should be cleared in any_idp_user mode, got %q", got.GroupEnrollmentAccessName)
	}

	if !got.IsUseJamfConnect {
		t.Fatalf("PassUserInfoToJamfConnect should map directly")
	}
	if got.ShortNameAttribute != "uid" || got.LongNameAttribute != "name" {
		t.Fatalf("attribute mapping wrong: short=%q long=%q", got.ShortNameAttribute, got.LongNameAttribute)
	}
}
