// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package accountprivileges provides the shared privilege-grid model used by
// the jamfplatform_pro_account and jamfplatform_pro_account_group resources.
//
// Jamf Pro administrator accounts and account groups carry a privilege grid
// split into seven wire categories (jss_objects, jss_settings, jss_actions,
// casper_admin, casper_remote, casper_imaging, recon), each a list of
// free-text privilege strings. Two server behaviours make this grid awkward to
// manage declaratively (both wire-probed 2026-06-12):
//
//   - The server SILENTLY EXPANDS a submitted set, adding dependency
//     privileges the caller did not request — sometimes in a different
//     category (e.g. submitting jss_objects=["Update Buildings"] yields a
//     stored jss_settings=["Read Activation Code"]). Re-submitting the same
//     subset reproduces the same superset, stably.
//   - The server SILENTLY DROPS an unrecognised privilege string, returning
//     201/200 with the bad value simply absent — no error.
//
// To keep a correct configuration free of perpetual diffs we use
// INTERSECT-ON-READ: refresh-Read stores, per category, only the intersection
// of the user-declared set and the server set (see IntersectIntoState). Server-
// added extras therefore never enter Terraform state, and a genuine removal is
// preserved (it is not a superset/subset suppression — see the rejected
// alternative in the resource spike). Writes trust the planned value for the
// privilege grid rather than echoing the server response, so a silently-dropped
// privilege degrades to a soft diff instead of a hard "inconsistent result
// after apply" abort. The plan-time Validator (discovering the tenant's catalog
// from an Administrator account/group) is what keeps invalid privileges from
// reaching apply in the first place.
package accountprivileges

// Category describes one of the seven privilege buckets: its Jamf classic wire
// element name, the UI-aligned Terraform attribute name, and a short
// description for the schema.
type Category struct {
	WireKey  string // classic XML element, e.g. "jss_objects"
	AttrName string // Terraform attribute, e.g. "jamf_pro_server_objects"
	Desc     string
}

// Categories is the canonical ordered list of the seven privilege buckets.
// Order is fixed so generated schema, state, and tests are deterministic.
// Attribute names mirror the Jamf Pro admin UI Privileges tab
// ("Jamf Pro Server Objects/Settings/Actions") and product names (Casper Admin,
// Casper Remote, Recon); casper_imaging is retained for round-trip fidelity
// though Jamf Imaging is a legacy product.
var Categories = []Category{
	{WireKey: "jss_objects", AttrName: "jamf_pro_server_objects", Desc: "Create/Read/Update/Delete privileges for Jamf Pro server objects (the UI \"Jamf Pro Server Objects\" tab)."},
	{WireKey: "jss_settings", AttrName: "jamf_pro_server_settings", Desc: "Read/Update privileges for Jamf Pro server settings (the UI \"Jamf Pro Server Settings\" tab)."},
	{WireKey: "jss_actions", AttrName: "jamf_pro_server_actions", Desc: "Action privileges (the UI \"Jamf Pro Server Actions\" tab)."},
	{WireKey: "casper_admin", AttrName: "casper_admin", Desc: "Jamf Admin (Casper Admin) application privileges."},
	{WireKey: "casper_remote", AttrName: "casper_remote", Desc: "Jamf Remote (Casper Remote) application privileges."},
	{WireKey: "casper_imaging", AttrName: "casper_imaging", Desc: "Jamf Imaging (Casper Imaging) application privileges (legacy)."},
	{WireKey: "recon", AttrName: "recon", Desc: "Recon application privileges."},
}
