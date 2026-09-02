// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   account.CreateConnection
//   account.GetConnection    (read, create read-back, update read-back)
//   account.ListConnections  (the second half of every read; data sources; list resource)
//   account.UpdateConnection
//   account.DeleteConnection
//
// ListConnections is on the resource's own path, not only the data sources',
// because no single call returns a whole connection: the read of one carries the
// per-provider settings and no products, and the collection read carries the
// products and the consent ticket and no settings. It is also what disambiguates
// a connection Jamf's collection lists but cannot read on its own identifier
// from one that has genuinely gone.
//
// Status: current. Last reviewed 2026-09-02.

package sso_connection

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/account"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create makes the connection and reads back the half the create response does
// not carry.
//
// Every attempt is currently refused with an internal failure from Jamf, for
// every payload, in every region — see the package doc. The path is written for
// the fix rather than around the fault, and the diagnostic says whose fault it
// is.
func (r *ConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, secretDiags := configuredClientSecret(ctx, req.Config)
	resp.Diagnostics.Append(secretDiags...)
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

	request, buildDiags := buildConnectionRequest(createCtx, plan, secret)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateConnection(createCtx, request)
	if err != nil {
		if !appendWriteDiagnostics(&resp.Diagnostics, "create", err) {
			resp.Diagnostics.AddError("Error creating Jamf Account SSO connection", err.Error())
		}
		return
	}

	summary, summaryDiags := r.readSummary(createCtx, created.ID)
	resp.Diagnostics.Append(summaryDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(assignConnectionResourceModel(&plan, created, summary, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, connectionIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created Jamf Account SSO connection", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest representation of the
// connection.
//
// The identity branch serves the identity-only refresh Terraform performs when
// it holds an identity and no state. Both branches are treated as an adoption
// when `name` is absent, which is what an import leaves: STYLE_GUIDE's usual
// warning that state is not null after an import is why the test is a Required
// attribute rather than the raw state. An adoption takes the stored name and
// fills every settings block; an ordinary refresh does neither, for the reasons
// state_builders.go gives.
//
// A connection built with Microsoft admin consent is refused here, which covers
// both the read after an import and the case of one that grew a consent after
// Terraform adopted it. helpers.go says why it cannot be a managed resource.
func (r *ConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConnectionResourceModel

	if req.State.Raw.IsNull() {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this Jamf Account SSO connection without existing state or identity data, so the provider cannot determine which connection to read.",
			)
			return
		}
		var identity connectionIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !helpers.IsConfiguredValue(identity.ID) || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing connection identifier",
				"The resource identity did not include an 'id' attribute, so the provider cannot refresh the Jamf Account SSO connection.",
			)
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(connectionTimeoutAttributeTypes)
	} else {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !helpers.IsConfiguredValue(state.ID) || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing connection identifier", "Cannot read a Jamf Account SSO connection without an identifier.")
		return
	}
	id := state.ID.ValueString()
	adopt := state.Name.IsNull() || state.Name.ValueString() == ""

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultReadTimeout, state.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	found, err := r.client.GetConnection(readCtx, id)
	if err != nil {
		if !helpers.IsNotFoundError(err) {
			resp.Diagnostics.AddError("Error reading Jamf Account SSO connection", err.Error())
			return
		}
		r.reportMissingConnection(readCtx, ctx, resp, id)
		return
	}

	if appendConsentFlowDiagnostics(&resp.Diagnostics, found) {
		return
	}

	summary, summaryDiags := r.readSummary(readCtx, id)
	resp.Diagnostics.Append(summaryDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(assignConnectionResourceModel(&state, found, summary, adopt)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, connectionIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update replaces the connection's settings.
//
// The whole settings body is sent, because Jamf documents this as a replacement
// rather than a patch — spec-derived, not wire-verified, since the write path is
// refused for every request. Consequently the plan is what is sent, and the
// carry-forward plan modifiers on the optional-and-reported attributes are what
// keep a value the operator left out from being cleared.
//
// A connection built with Microsoft admin consent is refused before any request
// is issued, keyed on the value already in state. That is deterministic and needs
// no guess about how Jamf would answer such a write — a guess that would be
// especially poor here, since every write is currently refused identically
// whatever the reason.
func (r *ConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state ConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, secretDiags := configuredClientSecret(ctx, req.Config)
	resp.Diagnostics.Append(secretDiags...)
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

	if state.ConsentFlow.ValueBool() {
		appendConsentFlowUpdateDiagnostics(&resp.Diagnostics, state.DisplayName.ValueString())
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing Jamf Account SSO connection identifier",
			"Terraform has no identifier recorded for this connection, and changing one needs it. Run "+
				"`terraform apply -refresh-only` to populate it, then apply again.",
		)
		return
	}
	plan.ID = state.ID

	request, buildDiags := buildConnectionRequest(updateCtx, plan, secret)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateConnection(updateCtx, id, request)
	if err != nil {
		if !appendWriteDiagnostics(&resp.Diagnostics, "change", err) {
			resp.Diagnostics.AddError("Error updating Jamf Account SSO connection", err.Error())
		}
		return
	}

	summary, summaryDiags := r.readSummary(updateCtx, id)
	resp.Diagnostics.Append(summaryDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(assignConnectionResourceModel(&plan, updated, summary, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, connectionIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "updated Jamf Account SSO connection", map[string]any{"id": id})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the connection.
//
// Removing a connection leaves its domains claimed and verified, which is the
// asymmetry worth knowing: withdrawing a domain shrinks every connection naming
// it, while removing a connection touches no domain. So nothing here has to be
// ordered against the domain resources.
//
// A connection built with Microsoft admin consent is deliberately *not* refused
// here, unlike in Read and Update. Removal works on one, and refusing it would
// leave an operator who adopted one under an earlier provider version unable to
// get rid of it by any means but editing state by hand.
func (r *ConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConnectionResourceModel
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

	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing Jamf Account SSO connection identifier",
			"Terraform has no identifier recorded for this connection, and removing one needs it. Run "+
				"`terraform apply -refresh-only` to populate it, then destroy again.",
		)
		return
	}

	if err := r.client.DeleteConnection(deleteCtx, id); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Account SSO connection already removed", map[string]any{"id": id})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Account SSO connection", fmt.Sprintf("API error: %v", err))
	}
}

// readSummary returns the collection entry for one connection.
//
// Absence is not an error: a connection readable on its own identifier but
// missing from the collection is the mirror image of the disagreement inside
// Jamf that this package guards against, and the only cost is that the two
// attributes the collection supplies come back empty. The read that matters has
// already succeeded, so failing the refresh over the lesser half would be worse
// than reporting it partial.
func (r *ConnectionResource) readSummary(ctx context.Context, id string) (*account.ConnectionSummary, diag.Diagnostics) {
	var diags diag.Diagnostics
	summaries, err := r.client.ListConnections(ctx)
	if err != nil {
		diags.AddError(
			"Unable to list Jamf Account SSO connections",
			"The connection itself was read, but the organization's connection list — which is the only place "+
				"the enabled products and the consent ticket appear — could not be. Underlying error: "+
				err.Error(),
		)
		return nil, diags
	}
	return findSummary(summaries, id), diags
}

// reportMissingConnection decides what a not-found means.
//
// Two things look identical at the single-connection read and mean opposite
// things. A connection genuinely removed outside Terraform has to be dropped from
// state so the next plan makes it again. A connection Jamf's own collection still
// lists has to be kept, because it exists and planning a fresh create would risk
// a duplicate — that disagreement inside Jamf was observed on a real connection,
// and it is why this branch consults the collection instead of trusting the
// not-found.
func (r *ConnectionResource) reportMissingConnection(readCtx, ctx context.Context, resp *resource.ReadResponse, id string) {
	summaries, err := r.client.ListConnections(readCtx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to confirm whether the Jamf Account SSO connection still exists",
			"Reading the connection on its identifier reported it missing, and the organization's connection "+
				"list could not be read to confirm that. Terraform has left it in state rather than assuming it "+
				"is gone, because Jamf is known to list a connection it cannot read on its own identifier. "+
				"Underlying error: "+err.Error(),
		)
		return
	}

	if summary := findSummary(summaries, id); summary != nil {
		appendGhostConnectionDiagnostics(&resp.Diagnostics, id, summary.Name)
		return
	}

	tflog.Info(ctx, "Jamf Account SSO connection not found, removing from state", map[string]any{"id": id})
	resp.State.RemoveResource(ctx)
}

// configuredClientSecret reads the write-only client secret out of the
// configuration.
//
// It comes from the configuration rather than the plan because a write-only value
// is stripped from a plan, so the plan would always report it absent — and absent
// is meaningful here: it is how an operator says "keep the stored secret". Only
// the one attribute is read, so an unknown value elsewhere in a nested block
// cannot take the read down with it.
func configuredClientSecret(ctx context.Context, config tfsdk.Config) (types.String, diag.Diagnostics) {
	var secret types.String
	diags := config.GetAttribute(ctx, path.Root("client_secret"), &secret)
	return secret, diags
}
