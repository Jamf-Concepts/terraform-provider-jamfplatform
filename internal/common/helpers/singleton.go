// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SingletonID is the fixed identifier used by singleton resources — objects scoped
// one-per-tenant with no identifier of their own. Every singleton resource stores
// this constant as its Terraform state ID and accepts it as the import identifier.
//
// Two shapes share it. Jamf Pro singletons are exposed by the API as Update-only,
// with no Create and no Delete (STYLE_GUIDE §Singleton resources). The Jamf Security
// Cloud custom DNS singletons are one-per-tenant in the same way but have a Delete
// the API honours and an absence that can be read back
// (STYLE_GUIDE §Tenant singletons with a real clear). The identifier is the same; the
// Delete and Read semantics are not.
//
// Rationale: the API has no natural ID for these objects (one record per tenant).
// Terraform's uniqueness scope is per resource type + name, so two different
// singleton resource types both holding ID "singleton" do not collide.
//
// Import: terraform import <type>.<name> singleton
const SingletonID = "singleton"

// ImportSingletonState implements ImportState for a singleton resource, accepting
// the identifier from either import form and refusing anything but SingletonID.
//
// typeName is the Terraform type being imported, named in the diagnostic so an
// operator sees which resource refused them.
//
// # Why this exists rather than a bare req.ID check
//
// A practitioner can supply the identifier two ways, and only one of them fills
// req.ID. `terraform import`, and an `import` block written with `id =`, set
// req.ID. An `import` block written with `identity = { id = … }` — which is the
// form `terraform plan -generate-config-out` emits, and the only form Terraform's
// query-driven generation produces — leaves req.ID EMPTY and puts the value in
// req.Identity instead.
//
// Every singleton resource used to compare req.ID against SingletonID directly, so
// all 29 of them rejected the identity form with "must be imported with id
// \"singleton\". Got \"\"" while advertising an IdentitySchema that promised it
// worked. Reading the identifier from whichever field carries it, once, is what
// keeps the promise; the framework's own ImportStatePassthroughWithIdentity then
// stores it and mirrors the identity through.
//
// # Why the identity is written even on the flat-ID path
//
// ImportStatePassthroughWithIdentity writes the identity only when the identifier
// arrived *as* an identity: on the req.ID path it sets the state attribute and
// returns, leaving resp.Identity as the fully-null value the framework
// pre-populated. Nothing later fills it in, so `terraform import <addr> singleton`
// produces an imported resource with no identity — and the framework's own
// ReadResource then refuses any read that returns without one:
//
//	Missing Resource Identity After Read
//	The Terraform Provider unexpectedly returned no resource identity data after
//	having no errors in the resource read. This is always an issue in the
//	Terraform Provider and should be reported to the provider developers.
//
// (fwserver/server_readresource.go: resp.NewIdentity is seeded from
// req.CurrentIdentity, and a fully-null one is an error whenever the resource
// declares an IdentitySchema.)
//
// A singleton that exists never reaches that, because its Read sets the identity
// on the way out. One the tenant has NOT configured does: Read finds nothing,
// drops the resource from state and returns with no error and no identity. So
// `terraform import` against an unconfigured singleton told the practitioner to
// report a provider bug, while the identity form — carrying an identity from the
// import block — reported Terraform's own "Cannot import non-existent remote
// object". Setting it here is what makes the two forms agree, in the one place
// both of them pass through.
func ImportSingletonState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, typeName string) {
	supplied := req.ID
	if supplied == "" && req.Identity != nil {
		var fromIdentity types.String
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &fromIdentity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		supplied = fromIdentity.ValueString()
	}

	if supplied != SingletonID {
		resp.Diagnostics.AddError(
			"Invalid singleton import identifier",
			fmt.Sprintf(
				"%s is a singleton resource and must be imported with id %q. Got %q.",
				typeName, SingletonID, supplied,
			),
		)
		return
	}

	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
	if resp.Diagnostics.HasError() || resp.Identity == nil {
		return
	}
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), SingletonID)...)
}
