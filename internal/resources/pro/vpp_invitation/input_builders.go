// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_invitation

import (
	"context"
	"net/url"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// buildVPPInvitationInput projects the plan into an SDK *proclassic.VppInvitation
// for Create / Update.
//
// General scalars are emitted as the plan holds them; email-mode fields are
// omitted when null (the server stores them only for "Send emails"). Scope, when
// declared, is emitted as a FULL skeleton — every collection wrapper is present
// (empty when its set is null/empty) so the server full-replaces it. A nil Scope
// omits <scope> entirely and leaves the server's scope untouched.
func buildVPPInvitationInput(ctx context.Context, plan VPPInvitationResourceModel) (*proclassic.VppInvitation, diag.Diagnostics) {
	var diags diag.Diagnostics

	general := &proclassic.VppInvitationGeneral{
		Name:                     helpers.OptionalStringPointer(plan.Name),
		DistributionMethod:       helpers.OptionalStringPointer(plan.DistributionMethod),
		AutoRegisterManagedUsers: helpers.OptionalBoolPointer(plan.AutoRegisterManagedUsers),
		SenderName:               helpers.OptionalStringPointer(plan.SenderName),
		SenderEmailAddress:       helpers.OptionalStringPointer(plan.SenderEmailAddress),
		Subject:                  helpers.OptionalStringPointer(plan.Subject),
		Message:                  encodedMessagePointer(plan.Message),
		RequireLogin:             helpers.OptionalBoolPointer(plan.RequireLogin),
	}
	if acctID := helpers.StringIDPtr(plan.VPPAccountID); acctID != nil {
		general.VppAccount = &proclassic.VppInvitationGeneralVppAccount{ID: acctID}
	}

	out := &proclassic.VppInvitation{General: general}

	if plan.Scope != nil {
		s, d := buildScope(ctx, plan.Scope)
		diags.Append(d...)
		out.Scope = s
	}

	return out, diags
}

// buildScope emits the full <scope> skeleton (always-emit). Every collection
// wrapper is present so the server full-replaces it; a nil inner slice marshals
// as an empty element, which clears that collection.
func buildScope(ctx context.Context, m *scope.UserScopeModel) (*proclassic.VppInvitationScope, diag.Diagnostics) {
	var diags diag.Diagnostics
	t := m.TargetsOrZero()
	s := &proclassic.VppInvitationScope{
		AllJssUsers: helpers.OptionalBoolPointer(t.AllJssUsers),
	}

	jssUsers, d := scope.BuildIDSlice(ctx, t.JssUserIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	s.JssUsers = &proclassic.VppInvitationScopeJssUsers{User: jssUsers}

	jssUserGroups, d := scope.BuildIDSlice(ctx, t.JssUserGroupIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	s.JssUserGroups = &proclassic.VppInvitationScopeJssUserGroups{UserGroup: jssUserGroups}

	// limitations.user_groups: directory-service groups, NAME-keyed (populate Name).
	var limNames *[]proclassic.IDName
	if m.Limitations != nil {
		limNames, d = scope.BuildNameSlice(ctx, m.Limitations.DirectoryServiceUserGroupNames, func(name string) proclassic.IDName {
			n := name
			return proclassic.IDName{Name: &n}
		})
		diags.Append(d...)
	}
	s.Limitations = &proclassic.VppInvitationScopeLimitations{
		UserGroups: &proclassic.VppInvitationScopeLimitationsUserGroups{UserGroup: limNames},
	}

	// exclusions: id-keyed jss_users / jss_user_groups + name-keyed user_groups.
	excl := &proclassic.VppInvitationScopeExclusions{}
	var exclJssUsers, exclJssUserGroups *[]proclassic.IDName
	var exclDSGroups *[]proclassic.VppInvitationScopeExclusionsUserGroupsUserGroupItem
	if m.Exclusions != nil {
		exclJssUsers, d = scope.BuildIDSlice(ctx, m.Exclusions.JssUserIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
		diags.Append(d...)
		exclJssUserGroups, d = scope.BuildIDSlice(ctx, m.Exclusions.JssUserGroupIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
		diags.Append(d...)
		exclDSGroups, d = scope.BuildNameSlice(ctx, m.Exclusions.DirectoryServiceUserGroupNames, func(name string) proclassic.VppInvitationScopeExclusionsUserGroupsUserGroupItem {
			n := name
			return proclassic.VppInvitationScopeExclusionsUserGroupsUserGroupItem{Name: &n}
		})
		diags.Append(d...)
	}
	excl.JssUsers = &proclassic.VppInvitationScopeExclusionsJssUsers{User: exclJssUsers}
	excl.JssUserGroups = &proclassic.VppInvitationScopeExclusionsJssUserGroups{UserGroup: exclJssUserGroups}
	excl.UserGroups = &proclassic.VppInvitationScopeExclusionsUserGroups{UserGroup: exclDSGroups}
	s.Exclusions = excl

	return s, diags
}

// encodedMessagePointer form-URL-encodes the invitation email message so the
// server's form-decode of the <message> field round-trips it verbatim. The
// classic endpoint form-decodes this field (wire-probed: bare `%` not followed by
// two hex digits → HTTP 500; `+` → space; `%XX` → byte). url.QueryEscape makes
// the message survive that decode exactly — including the `%@` registration-URL
// placeholder and embedded newlines — so state (decoded on GET) matches config.
// Returns nil for null/unknown so the field is omitted.
func encodedMessagePointer(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := url.QueryEscape(v.ValueString())
	return &s
}
