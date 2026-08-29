// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

// Machine-readable error codes the Jamf Security Cloud device group endpoints
// return. Wire-probed against the EU sandbox on 2026-08-29 (raw bodies in
// local-testing/securitycloud-groups/).
//
// Unlike the custom DNS surface, these codes are not in the SDK's generated
// ApiErrorItemCode enum — that enum is the DNS namespace's error schema and
// carries none of the group codes — so they are restated here as string
// literals, per STYLE_GUIDE §Security Cloud shapes that recur ("where the SDK
// documents a set but generates no helper, restate it and say why").
// mappings_test.go pins each literal against the body captured during the probe.
const (
	// codeGroupAlreadyExists is returned by both create and update when the name
	// collides with an existing group. Uniqueness is per customer and, unlike the
	// reserved-name check below, case-SENSITIVE: "Example" and "example" coexist.
	codeGroupAlreadyExists = "GROUP_ALREADY_EXISTS"

	// codeReservedGroupName is returned when the name resolves to a reserved
	// name. The check runs after the server trims surrounding whitespace and
	// compares case-insensitively, so "default group" and "Default Group " are
	// both refused.
	codeReservedGroupName = "RESERVED_GROUP_NAME"

	// codeGroupNotFound accompanies the 404 from read and update. Delete never
	// returns it — see the Delete doc comment in crud.go.
	codeGroupNotFound = "GROUP_NOT_FOUND"

	// codeInvalidField accompanies the 400 for a blank name. This surface fills
	// in `field` and a human description, which the shared Security Cloud enum
	// failure does not.
	codeInvalidField = "INVALID_FIELD"

	// codeNotEntitled means the credentials authenticated but the tenant does not
	// hold the surface. Not observed on this endpoint — the probe tenant is
	// entitled and cannot be un-entitled — but translated for consistency with
	// every other Security Cloud construct.
	codeNotEntitled = "NOT_ENTITLED"

	// codeBadPermissions is deliberately NOT translated into a diagnostic. The
	// gateway returns it both for a genuine privilege gap and for a route it does
	// not serve at all: a control probe on a deliberately bogus path returned the
	// identical body. Any wording we chose would be wrong half the time, so the
	// raw error is surfaced instead.
	codeBadPermissions = "BAD_PERMISSIONS"
)

// defaultGroupName is the name of the implicit group every tenant carries.
//
// It is not a stored group: the list endpoint returns it with no `id` key at all,
// so it cannot be read, updated or deleted by id, and creating or renaming to it
// is refused with RESERVED_GROUP_NAME. The comparison the server performs is
// case-insensitive and runs after trimming, which is what reservedGroupName in
// validators.go reproduces.
const defaultGroupName = "Default Group"
