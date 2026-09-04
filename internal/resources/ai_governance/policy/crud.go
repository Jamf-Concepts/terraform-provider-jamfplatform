// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/aigovernance"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// policyIdentityModel is the resource identity: the policy ID, which is the only thing that
// identifies a policy uniquely. Names are not unique.
type policyIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// Create saves the policy and, unless publishing is disabled, publishes it as version 1.
//
// Creating a policy stages a draft and deploys nothing — a policy with no published version cannot
// be referenced by a blueprint at all — so the publish is part of what an apply means here rather
// than a separate operation.
//
// A failed publish is reported only after state has been committed. The framework returns whatever
// resp.State holds when this function ends and never backfills it, so adding the diagnostic before
// the write would make the error check that guards it fire and record nothing for a policy that now
// exists — and because policy names are not unique, the next apply would create a second one.
func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, defaultCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	created, err := r.client.CreatePolicy(ctx, buildCreateRequest(&plan))
	if err != nil {
		if !appendWriteDiagnostics(&resp.Diagnostics, err) {
			resp.Diagnostics.AddError("Unable to create AI policy", err.Error())
		}
		return
	}

	publishErr := r.publishIfNeeded(ctx, created.ID, plan.Publish.ValueBool())

	if !r.hydrate(ctx, &plan, created.ID, &resp.Diagnostics) {
		resp.State.RemoveResource(ctx)
		appendCreatePublishFailure(&resp.Diagnostics, created.ID, publishErr)
		return
	}
	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, policyIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	appendCreatePublishFailure(&resp.Diagnostics, created.ID, publishErr)
}

// Read refreshes the policy. A policy that has been archived is reported as absent: archiving is a
// soft delete the API renders as a 404 from every operation, so there is nothing to distinguish it
// from a policy that never existed.
//
// The returned status is checked as well, rather than relying on that 404 alone. The service's own
// spec declares ARCHIVED as a value GET /policies/{id} may report, and the SDK generates a constant
// for it; wire probing on 2026-08-30 saw only the 404, but a service release that starts honouring
// the spec would otherwise leave a plan reporting no changes for a policy delivered to no device.
func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state policyModel
	if req.State.Raw.IsNull() {
		if !readIdentity(ctx, req.Identity, &state, &resp.Diagnostics) {
			return
		}
	} else {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	timeout, diags := state.Timeouts.Read(ctx, defaultReadTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	detail, err := r.client.GetPolicy(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read AI policy", err.Error())
		return
	}

	if detail.Status == aigovernance.PolicyDetailStatusArchived {
		resp.Diagnostics.AddWarning(
			"AI policy has been archived",
			"The platform reports the policy with ID "+state.ID.ValueString()+" as archived, which is how it records a "+
				"policy deleted outside Terraform. An archived policy is delivered to no device and its published "+
				"versions are no longer served, so it has been removed from state and the next plan will propose "+
				"creating it again.",
		)
		resp.State.RemoveResource(ctx)
		return
	}

	if err := applyPolicyToState(&state, detail); err != nil {
		resp.Diagnostics.AddError("Unable to read AI policy settings", err.Error())
		return
	}
	state.Publish = resolvePublish(state.Publish)
	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, policyIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update saves the draft and, unless publishing is disabled, publishes it.
//
// The platform compares the settings it holds against the ones sent, so an update that changes
// nothing leaves no draft behind and the publish that follows is a no-op rather than a version
// minted for nothing.
//
// A failed publish is reported only after state has been committed, for the reason Create records:
// the diagnostic would otherwise fire the error check that guards the state write, leaving state at
// the values it held before the update and hiding the draft the platform now holds.
func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan policyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state policyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, defaultUpdateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id := state.ID.ValueString()
	if err := r.client.UpdatePolicy(ctx, id, buildUpdateRequest(&plan)); err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			resp.Diagnostics.AddWarning(
				"AI policy no longer exists",
				"The policy was archived outside Terraform, so it could not be updated. It has been removed from "+
					"state and the next plan will propose creating it again.",
			)
			return
		}
		if !appendWriteDiagnostics(&resp.Diagnostics, err) {
			resp.Diagnostics.AddError("Unable to update AI policy", err.Error())
		}
		return
	}

	publishErr := r.publishIfNeeded(ctx, id, plan.Publish.ValueBool())

	if !r.hydrate(ctx, &plan, id, &resp.Diagnostics) {
		resp.State.RemoveResource(ctx)
		appendUpdatePublishFailure(&resp.Diagnostics, publishErr)
		return
	}
	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, policyIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	appendUpdatePublishFailure(&resp.Diagnostics, publishErr)
}

// Delete archives the policy.
//
// Archiving succeeds even when a deployed blueprint references one of the policy's published
// versions: the platform accepts the delete and leaves that blueprint pointing at a version it will
// no longer serve, and wire probing on 2026-08-30 established that the blueprint is then unwritable
// in full — a merge PATCH carrying only a new description is refused with POLICY_ARCHIVED against
// the component's policyId, and the steps cannot be emptied either. Nothing can be done about that
// from this end, so the warning on a successful archive is the whole remedy: Terraform cannot see
// the blueprints, and there is no reverse lookup that works (the policy deployment endpoint reports
// no blueprints even for one that is deployed). A failed archive returns before the warning, so a
// policy that still exists is never described as archived.
func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, defaultDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := r.client.ArchivePolicy(ctx, state.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Unable to delete AI policy", err.Error())
		return
	}

	resp.Diagnostics.AddWarning(
		"Archived AI policy may still be referenced by a blueprint",
		"Policy "+state.ID.ValueString()+" has been archived. Jamf does not block this even when a deployed "+
			"blueprint pins one of its published versions, and no API reports which blueprints do. A blueprint "+
			"still naming this policy cannot be written at all until the reference is replaced — even a change "+
			"to its description is refused with POLICY_ARCHIVED.",
	)
}

// appendCreatePublishFailure reports a publish that failed after the policy was created.
//
// The retry the wording promises is real, and is the whole reason planPublishOutcome exists: state
// records the draft this failure left behind, and the next plan turns that into a change to
// has_draft and published_version, so Terraform calls Update and the publish is attempted again.
// What the operator must not do is create the policy a second time, because policy names are not
// unique and both would then exist — so the wording says so before it mentions the retry.
func appendCreatePublishFailure(diags *diag.Diagnostics, id string, err error) {
	if err == nil {
		return
	}
	diags.AddError(
		"AI policy created but not published",
		"The policy was created with ID "+id+" but publishing it failed, so it holds an unpublished draft and "+
			"cannot be deployed by a blueprint yet. Terraform has recorded the policy — do not create it again, "+
			"because policy names are not unique and a second create would leave two. The next apply retries the "+
			"publish, and it can also be published in the Jamf Account admin UI. Reported by Jamf: "+err.Error(),
	)
}

// appendUpdatePublishFailure reports a publish that failed after the draft was saved. The draft is
// recorded in state, so the next apply retries the publish for the reason appendCreatePublishFailure
// describes.
func appendUpdatePublishFailure(diags *diag.Diagnostics, err error) {
	if err == nil {
		return
	}
	diags.AddError(
		"AI policy updated but not published",
		"The policy's draft was saved but publishing it failed, so blueprints continue to deploy the previously "+
			"published version. Terraform has recorded the draft, and the next apply retries the publish — it can "+
			"also be published in the Jamf Account admin UI. Reported by Jamf: "+err.Error(),
	)
}

// publishIfNeeded publishes the policy's draft when the operator asked for it.
//
// A 409 NO_DRAFT_TO_PUBLISH is success, not failure. The platform raises a draft only when the
// settings actually differ from the ones it holds, so this is the ordinary outcome of an apply that
// changed only the name — or of enabling `publish` on a policy that was already published.
func (r *PolicyResource) publishIfNeeded(ctx context.Context, id string, publish bool) error {
	if !publish {
		return nil
	}
	if _, err := r.client.PublishPolicy(ctx, id); err != nil {
		if hasCode(err, codeNoDraftToPublish) {
			return nil
		}
		return err
	}
	return nil
}

// hydrate re-reads the policy and copies it onto the model, reporting whether the model is usable.
// A policy that has vanished between the write and the read leaves nothing to record.
func (r *PolicyResource) hydrate(ctx context.Context, model *policyModel, id string, diags *diag.Diagnostics) bool {
	detail, err := r.client.GetPolicy(ctx, id)
	if err != nil {
		if isNotFound(err) {
			diags.AddError(
				"AI policy disappeared immediately after being written",
				"Jamf reported the policy with ID "+id+" as absent when reading it back. It may have been archived "+
					"outside Terraform between the two calls.",
			)
			return false
		}
		diags.AddError("Unable to read AI policy back after writing it", err.Error())
		return false
	}
	if err := applyPolicyToState(model, detail); err != nil {
		diags.AddError("Unable to read AI policy settings", err.Error())
		return false
	}
	model.Publish = resolvePublish(model.Publish)
	return true
}

// readIdentity fills the model's ID from the resource identity, for a refresh Terraform issues with
// no prior state — an import addressed by identity rather than by the passthrough import ID.
func readIdentity(ctx context.Context, identity *tfsdk.ResourceIdentity, state *policyModel, diags *diag.Diagnostics) bool {
	if identity == nil {
		diags.AddError(
			"Missing resource identity",
			"Terraform requested a refresh for this AI policy with neither prior state nor identity data, so the "+
				"provider cannot tell which policy to read.",
		)
		return false
	}
	var model policyIdentityModel
	diags.Append(identity.Get(ctx, &model)...)
	if diags.HasError() {
		return false
	}
	if !helpers.IsConfiguredValue(model.ID) || model.ID.ValueString() == "" {
		diags.AddError(
			"Missing AI policy ID",
			"The resource identity did not carry an \"id\", so the provider cannot refresh the policy.",
		)
		return false
	}
	state.ID = model.ID
	return true
}
