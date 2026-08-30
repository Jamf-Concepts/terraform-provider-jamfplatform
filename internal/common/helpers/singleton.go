// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

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
