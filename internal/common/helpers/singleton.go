// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

// SingletonID is the fixed identifier used by Pro singleton resources — Jamf Pro
// objects that are scoped one-per-tenant and exposed by the API as Update-only
// (no Create or Delete). Every Pro singleton resource stores this constant as its
// Terraform state ID and accepts it as the import identifier.
//
// Rationale: the API has no natural ID for these objects (one record per tenant).
// Terraform's uniqueness scope is per resource type + name, so two different
// singleton resource types both holding ID "singleton" do not collide.
//
// Import: terraform import <type>.<name> singleton
const SingletonID = "singleton"
