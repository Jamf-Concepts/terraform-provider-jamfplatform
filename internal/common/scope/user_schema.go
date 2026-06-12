// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// UserScopeAttributes returns the attribute map for a user-based classic
// resource's <scope> block (targets + limitations + exclusions). This is the
// THIRD scope shape (distinct from ComputerScopeAttributes / MobileScopeAttributes):
// targets are Jamf Pro users and Jamf Pro user groups (id-keyed) plus the
// all_jss_users flag; limitations and exclusions carry directory-service (LDAP)
// user groups by NAME. The caller wraps the map in its own
// schema.SingleNestedAttribute with a resource-specific description:
//
//	"scope": schema.SingleNestedAttribute{
//	    MarkdownDescription: "...",
//	    Optional:            true,
//	    Attributes:          scope.UserScopeAttributes(),
//	}
//
// all_jss_users (inside the `targets` sub-block) is Optional+Computed with
// booldefault.StaticBool(false) and the value-discriminated AllFlagConflictsWith
// validator (relative paths, so they resolve against its siblings inside
// `targets`). Unlike the computer/mobile flags it does NOT use a state-forwarding
// plan modifier — the static default keeps the value known at plan time, so the
// null→present transition on `targets` cannot surface a "was null, but now …"
// consistency error here.
//
// Consumed by vpp_invitation and vpp_assignment. The build/flatten glue stays
// per-resource because VppInvitation* and VppAssignment* are distinct generated
// SDK structs with no shared interface (see STYLE_GUIDE.md §Scope helper).
func UserScopeAttributes() map[string]schema.Attribute {
	targets := map[string]schema.Attribute{
		"all_jss_users": schema.BoolAttribute{
			MarkdownDescription: "Target all Jamf Pro users. Conflicts with `jss_user_ids` / `jss_user_group_ids`. Defaults to `false`.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
			Validators: []validator.Bool{
				AllFlagConflictsWith(
					path.MatchRelative().AtParent().AtName("jss_user_ids"),
					path.MatchRelative().AtParent().AtName("jss_user_group_ids"),
				),
			},
		},
		"jss_user_ids":       IDSetAttribute("user"),
		"jss_user_group_ids": IDSetAttribute("user group"),
	}

	return map[string]schema.Attribute{
		"targets": schema.SingleNestedAttribute{
			MarkdownDescription: "Scope targets — the Jamf Pro users and user groups the assignment applies to. Mirrors the admin UI's Targets tab.",
			Optional:            true,
			Attributes:          targets,
		},
		"limitations": schema.SingleNestedAttribute{
			MarkdownDescription: "Scope limitations.",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"directory_service_user_group_names": NameSetAttribute("directory service (LDAP) user group"),
			},
		},
		"exclusions": schema.SingleNestedAttribute{
			MarkdownDescription: "Scope exclusions.",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"jss_user_ids":                       IDSetAttribute("user"),
				"jss_user_group_ids":                 IDSetAttribute("user group"),
				"directory_service_user_group_names": NameSetAttribute("directory service (LDAP) user group"),
			},
		},
	}
}
