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
// all 27 of them rejected the identity form with "must be imported with id
// \"singleton\". Got \"\"" while advertising an IdentitySchema that promised it
// worked. Reading the identifier from whichever field carries it, once, is what
// keeps the promise; the framework's own ImportStatePassthroughWithIdentity then
// stores it and mirrors the identity through.
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
}
