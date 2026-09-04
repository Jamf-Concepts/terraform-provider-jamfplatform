// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   pro.CreateAccountV1               (POST /accounts; sets accountType incl. FEDERATED)
//   pro.GetAccountV1                  (base-field read)
//   pro.UpdateAccountV1               (PUT base fields)
//   pro.DeleteAccountV1
//   pro.ResolveAccountV1IDByName      (data source username lookup)
//   proclassic.GetAccountByUserID     (Custom privilege grid read)
//   proclassic.UpdateAccountByUserID  (Custom privilege grid write — merge)
//   proclassic.ListAccounts           (privilege-catalog discovery, ModifyPlan)
//
// Status: current. Last reviewed 2026-06-24.

package account

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create creates a Jamf Pro admin account via the Pro API, then (for a Custom +
// Full Access account) writes the privilege grid via the classic API. If the
// classic step fails, the just-created account is deleted so no half-built
// account is left behind. Privileges are trusted from the plan (not re-read).
func (r *AccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, cfg AccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultCreateTimeout, plan.Timeouts.Create)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	created, err := r.proClient.CreateAccountV1(createCtx, buildProUserAccount(plan, helpers.OptionalStringPointer(cfg.Password)))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro account", err.Error())
		return
	}
	if created == nil || created.ID == nil {
		resp.Diagnostics.AddError("Create response missing account ID", "Jamf Pro returned success with no account ID; cannot persist state.")
		return
	}
	plan.ID = helpers.StringPointerValueOrNull(created.ID)
	id := plan.ID.ValueString()

	if custPrivApplicable(plan.PrivilegeSet, plan.AccessLevel) {
		classicInput, d := buildClassicPrivileges(createCtx, plan)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if classicInput != nil {
			if err := r.classicClient.UpdateAccountByUserID(createCtx, id, classicInput); err != nil {
				// Roll back the account so we do not leave a privilege-less husk.
				if delErr := r.proClient.DeleteAccountV1(createCtx, id); delErr != nil {
					tflog.Error(ctx, "failed to roll back account after privilege write failure", map[string]any{"id": id, "delete_error": delErr.Error()})
				}
				resp.Diagnostics.AddError("Error writing Jamf Pro account privileges", err.Error())
				return
			}
		}
	}

	got, err := r.proClient.GetAccountV1(createCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro account", err.Error())
		return
	}
	assignProBaseFields(&plan, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, accountIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "created Jamf Pro account", map[string]any{"id": id})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes state: base fields from Pro, the Custom privilege grid from
// classic (intersect-on-read) when the account is Custom + Full Access.
func (r *AccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccountResourceModel
	stateAbsent := req.State.Raw.IsNull()

	if stateAbsent {
		if req.Identity == nil {
			resp.Diagnostics.AddError("Missing resource identity", "Terraform requested a refresh without existing state or identity data.")
			return
		}
		var identity accountIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError("Missing account ID", "The resource identity did not include an 'id' attribute.")
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(accountTimeoutAttributeTypes)
	} else {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	priorUsername := types.StringNull()
	if !stateAbsent {
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("username"), &priorUsername)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	hydrating := importHydration(stateAbsent, priorUsername)

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultReadTimeout, state.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro account without ID.")
		return
	}
	id := state.ID.ValueString()

	got, err := r.proClient.GetAccountV1(readCtx, id)
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro account not found, removing from state", map[string]any{"id": id})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, accountIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro account", err.Error())
		return
	}
	assignProBaseFields(&state, got)

	if custPrivApplicable(state.PrivilegeSet, state.AccessLevel) {
		classicGot, err := r.classicClient.GetAccountByUserID(readCtx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error reading Jamf Pro account privileges", err.Error())
			return
		}
		resp.Diagnostics.Append(assignClassicPrivileges(readCtx, &state, classicGot, hydrating)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		state.Privileges = nil
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, accountIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies base-field changes via Pro PUT and Custom privilege changes via
// the classic API. Each side is called only when its inputs actually changed, so
// a privilege-only edit skips the Pro PUT and a base-only edit skips the classic
// write.
func (r *AccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state, cfg AccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultUpdateTimeout, plan.Timeouts.Update)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	id := plan.ID.ValueString()
	passwordRotated := !plan.PasswordWOVersion.Equal(state.PasswordWOVersion)

	if baseFieldsChanged(plan, state) || passwordRotated {
		var password *string
		if passwordRotated {
			password = helpers.OptionalStringPointer(cfg.Password)
		}
		if _, err := r.proClient.UpdateAccountV1(updateCtx, id, buildProUserAccount(plan, password)); err != nil {
			resp.Diagnostics.AddError("Error updating Jamf Pro account", err.Error())
			return
		}
	}

	if custPrivApplicable(plan.PrivilegeSet, plan.AccessLevel) {
		classicInput, d := buildClassicPrivileges(updateCtx, plan)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if classicInput != nil {
			if err := r.classicClient.UpdateAccountByUserID(updateCtx, id, classicInput); err != nil {
				resp.Diagnostics.AddError("Error updating Jamf Pro account privileges", err.Error())
				return
			}
		}
	}

	got, err := r.proClient.GetAccountV1(updateCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro account", err.Error())
		return
	}
	assignProBaseFields(&plan, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, accountIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Pro account via the Pro API.
func (r *AccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultDeleteTimeout, state.Timeouts.Delete)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro account without ID.")
		return
	}

	if err := r.proClient.DeleteAccountV1(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro account already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro account", fmt.Sprintf("API error: %v", err))
	}
}

// baseFieldsChanged reports whether any Pro-owned base field differs between
// plan and state (account_type is RequiresReplace, so it is not compared here).
func baseFieldsChanged(plan, state AccountResourceModel) bool {
	return !plan.Username.Equal(state.Username) ||
		!plan.FullName.Equal(state.FullName) ||
		!plan.EmailAddress.Equal(state.EmailAddress) ||
		!plan.AccessLevel.Equal(state.AccessLevel) ||
		!plan.PrivilegeSet.Equal(state.PrivilegeSet) ||
		!plan.AccessStatus.Equal(state.AccessStatus) ||
		!plan.LdapServerID.Equal(state.LdapServerID) ||
		!plan.SiteID.Equal(state.SiteID) ||
		!plan.ForcePasswordChange.Equal(state.ForcePasswordChange)
}
