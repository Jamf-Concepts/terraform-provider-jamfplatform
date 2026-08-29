// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   securitycloud.GetDnsSearchDomainV1
//   securitycloud.SetDnsSearchDomainV1
//   securitycloud.ClearDnsSearchDomainV1
//
// Status: current. Last reviewed 2026-08-29.

package dns_search_domain

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// providerNotConfiguredError is the diagnostic every handler raises when the client
// is missing, kept in one place so the three wordings cannot drift apart.
func providerNotConfiguredError() (string, string) {
	return "Provider not configured",
		"The provider client was not configured. Please ensure the provider block is set up correctly."
}

// Create sets the tenant's search domain, refusing to overwrite an existing one.
//
// The preflight read is the whole point. PUT is an unconditional upsert that reports
// no conflict, so without reading first a plan that looks like a create would
// silently replace a search domain an administrator set by hand, and two Terraform
// configurations both managing this resource would overwrite each other in silence.
// Refusing and pointing at import is the only signal Terraform can give.
func (r *SearchDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(providerNotConfiguredError())
		return
	}

	var plan SearchDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	existing, err := r.client.GetDnsSearchDomainV1(createCtx)
	switch {
	case err == nil && existing != nil && existing.Suffix != "":
		resp.Diagnostics.AddError(
			"Search domain already configured",
			"This tenant already has the search domain \""+existing.Suffix+"\" set, and Jamf Security Cloud "+
				"allows only one. Creating it here would replace that value without warning. Import the existing "+
				"search domain instead:\n\n"+
				"    terraform import jamfplatform_security_cloud_dns_search_domain.<name> "+helpers.SingletonID,
		)
		return
	case err != nil && !helpers.IsNotFoundError(err):
		if !appendWriteDiagnostics(&resp.Diagnostics, err) {
			resp.Diagnostics.AddError("Error checking for an existing Jamf Security Cloud search domain", err.Error())
		}
		return
	}

	if err := r.client.SetDnsSearchDomainV1(createCtx, &securitycloud.SearchDomain{Suffix: plan.DomainName.ValueString()}); err != nil {
		if !appendWriteDiagnostics(&resp.Diagnostics, err) {
			resp.Diagnostics.AddError("Error setting Jamf Security Cloud search domain", err.Error())
		}
		return
	}

	got, err := r.client.GetDnsSearchDomainV1(createCtx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the Jamf Security Cloud search domain just written", err.Error())
		return
	}
	assignSearchDomainResourceModel(&plan, got)
	plan.ID = types.StringValue(helpers.SingletonID)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, searchDomainIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "set Jamf Security Cloud search domain")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state with the tenant's stored search domain.
//
// An unset search domain answers 404 with SEARCH_DOMAIN_NOT_SET rather than an
// empty 200, so absence is unambiguous and the resource is dropped from state — the
// ordinary contract for a deleted object, and the divergence from
// STYLE_GUIDE §Singleton resources described in the package doc comment.
func (r *SearchDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(providerNotConfiguredError())
		return
	}

	var state SearchDomainResourceModel
	if req.State.Raw.IsNull() {
		state.ID = types.StringValue(helpers.SingletonID)
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(searchDomainTimeoutAttributeTypes)
	} else {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultReadTimeout, state.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	got, err := r.client.GetDnsSearchDomainV1(readCtx)
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Security Cloud search domain not set, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Security Cloud search domain", err.Error())
		return
	}

	assignSearchDomainResourceModel(&state, got)
	state.ID = types.StringValue(helpers.SingletonID)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, searchDomainIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update replaces the tenant's search domain.
//
// No preflight here, unlike Create: state already says this configuration owns the
// value, so overwriting it is the intent rather than an accident.
func (r *SearchDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(providerNotConfiguredError())
		return
	}

	var plan SearchDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	if err := r.client.SetDnsSearchDomainV1(updateCtx, &securitycloud.SearchDomain{Suffix: plan.DomainName.ValueString()}); err != nil {
		if !appendWriteDiagnostics(&resp.Diagnostics, err) {
			resp.Diagnostics.AddError("Error updating Jamf Security Cloud search domain", err.Error())
		}
		return
	}

	got, err := r.client.GetDnsSearchDomainV1(updateCtx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the Jamf Security Cloud search domain just written", err.Error())
		return
	}
	assignSearchDomainResourceModel(&plan, got)
	plan.ID = types.StringValue(helpers.SingletonID)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, searchDomainIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "updated Jamf Security Cloud search domain")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete clears the tenant's search domain.
//
// A real clear, not the no-op STYLE_GUIDE §Singleton resources describes: the
// endpoint honours DELETE and answers 204 whether or not anything was set, so an
// already-cleared search domain needs no special case. Destroying this resource
// removes the search domain for the whole tenant.
func (r *SearchDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(providerNotConfiguredError())
		return
	}

	var state SearchDomainResourceModel
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

	if err := r.client.ClearDnsSearchDomainV1(deleteCtx); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Security Cloud search domain already cleared")
			return
		}
		resp.Diagnostics.AddError("Error clearing Jamf Security Cloud search domain", fmt.Sprintf("API error: %v", err))
		return
	}

	tflog.Trace(ctx, "cleared Jamf Security Cloud search domain")
}
