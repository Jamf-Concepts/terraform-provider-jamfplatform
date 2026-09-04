// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

// namespaceMigrationSummary and namespaceMigrationDetail carry the advance
// notice that this provider will publish under `jamf/jamfplatform` instead of
// `jamf-concepts/jamfplatform`. The registry namespace is part of the provider
// address Terraform records in state, so consumers need
// `terraform state replace-provider` on every workspace and state file rather
// than a version bump alone, and they need to hear about it before the release
// that moves it.
//
// The notice lives in its own file so that retiring it after the move is
// deleting a file and one call in Configure, and the same reason keeps the
// wording out of the Configure body. The matching banner on the registry index
// page sits in the provider Schema description, and README.md carries it for
// anyone reading the repository directly.
//
// Configure raises the warning before it validates anything, so a configuration
// that is missing credentials still carries the notice. A configuration that uses
// only the provider-defined functions never reaches Configure and so never sees
// it; the registry banner covers that case.
//
// Terraform renders diagnostic detail as plain text, so this string carries no
// markdown.
const (
	namespaceMigrationSummary = "This provider is moving to the jamf namespace"

	namespaceMigrationDetail = "A future release will carry the address jamf/jamfplatform in place of " +
		"jamf-concepts/jamfplatform.\n\n" +
		"Terraform records that namespace in state, so the change breaks existing configurations. " +
		"Change nothing today; releases under jamf-concepts/jamfplatform keep working. When the release " +
		"lands, for each workspace and state file holding jamfplatform resources:\n\n" +
		"  1. Run: terraform state replace-provider jamf-concepts/jamfplatform jamf/jamfplatform\n" +
		"  2. Point source at jamf/jamfplatform in the required_providers block.\n" +
		"  3. Run: terraform init -upgrade\n\n" +
		"Step 1 asks for confirmation and writes a state backup; add -auto-approve to run it unattended. " +
		"OpenTofu works the same way, with tofu in place of terraform.\n\n" +
		"Reference: https://developer.hashicorp.com/terraform/cli/commands/state/replace-provider"
)
