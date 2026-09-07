// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// GenerateConfigStep returns a test step asserting that a resource can be
// adopted from a tenant through Terraform's configuration generation.
//
// # What this catches that nothing else does
//
// `terraform plan -generate-config-out` writes state back out as configuration,
// so a Read that commits a value the resource's own schema or validators refuse
// produces a file that cannot be planned. Issue #379 collected seven of those
// across six resources, and not one was reachable from an ordinary acceptance
// test: a step that creates a resource and reads it back can only ever produce a
// state the configuration already described. The failures live in states the test
// did not create — a connector authenticating as a platform tenant, an SMTP
// server nobody has set up, a patch title with no packages assigned — and
// generation is the one workflow whose entire purpose is meeting them.
//
// # What the step actually asserts
//
// Three things, in one pass. Terraform generates configuration from the object's
// state; the generated configuration is then planned, which fails on any schema
// or validator diagnostic; and the resulting plan must be a NO-OP import, so an
// attribute that generates legally but disagrees with what the tenant holds
// fails too. That last part is stronger than running `terraform validate` over
// the generated file, which is what the issue proposed: validate would accept a
// generated value the very next plan would want to change.
//
// # How to use it
//
// Append it to a lifecycle test after at least one Config step has applied, so
// there is a real object to adopt. resourceName is the address the earlier step
// created; the framework rewrites it to `<type>.generated` for the import block
// it appends, so the generated resource is a second address adopting the same
// object rather than a conflicting declaration of the first.
//
// The kind is ImportBlockWithResourceIdentity, which is the form
// `terraform plan -generate-config-out` actually emits and the only one
// Terraform's query-driven generation produces, so the step exercises the path a
// practitioner is put on rather than a parallel one. Every construct it is
// applied to therefore needs an IdentitySchema.
//
// That choice is load-bearing: it is what caught all 27 singleton settings
// resources rejecting an identity-based import, each comparing req.ID against
// "singleton" when the identity form leaves req.ID empty
// (helpers.ImportSingletonState now reads whichever field carries it). An
// ImportBlockWithID step would have passed and said nothing.
//
// Pass expectations only where a resource genuinely cannot round-trip — and
// prefer fixing the resource, since a non-empty plan here is the very defect
// this step exists to report.
func GenerateConfigStep(resourceName string) resource.TestStep {
	return resource.TestStep{
		ResourceName:    resourceName,
		ImportState:     true,
		ImportStateKind: resource.ImportBlockWithResourceIdentity,
		GenerateConfig:  true,
	}
}
