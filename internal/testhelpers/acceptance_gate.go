// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers/accrequire"
)

// The credential gate itself lives in internal/testhelpers/accrequire, a leaf
// package importing nothing but the standard library. It has to: this package
// imports internal/provider to build the test provider factories, and provider
// reaches internal/common/impact through providerdata, so every package below
// testhelpers in the import graph — impact included, and it has acceptance tests
// of its own — would be unable to reach the gate if it lived here.
//
// These three forwarders exist so that ordinary call sites, which are the large
// majority, keep saying testhelpers.AccEnv and testhelpers.SkipOrFailUnset
// rather than carrying a second import for one call. A package that cannot
// import testhelpers imports accrequire directly.

// AccEnv reads an acceptance variable, falling back to its pre-rename name. See
// accrequire.AccEnv.
func AccEnv(name string) string { return accrequire.AccEnv(name) }

// SkipOrFailUnset ends the calling test for an unconfigured credential set,
// entitlement declaration or scope variable: fatally when JAMFPLATFORM_ACC_REQUIRE
// names the lane's set, otherwise as a skip. See accrequire.SkipOrFailUnset.
func SkipOrFailUnset(t *testing.T, precheck, reason string) {
	t.Helper()
	accrequire.SkipOrFailUnset(t, precheck, reason)
}

// CredentialRejectedMessage builds the report to hand Jamf Support when the
// estate refuses credentials that were supplied. See
// accrequire.CredentialRejectedMessage.
func CredentialRejectedMessage(baseURL string, err error) string {
	return accrequire.CredentialRejectedMessage(baseURL, err)
}
