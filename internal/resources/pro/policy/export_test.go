// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package policy

// EncodeNoExecuteTimeForTest exposes encodeNoExecuteTime to the external
// acceptance test package.
//
// The bridge exists because internal/testhelpers imports internal/provider,
// which imports this package, so an acceptance test written in package `policy`
// is an import cycle. The one test that needs it holds the encoder against the
// live endpoint, which is worth a two-line bridge — see
// TestAccPolicyResource_NoExecuteWindowEncoding. Delete this file along with
// the encoding when Jamf PI-1661 is fixed.
func EncodeNoExecuteTimeForTest(value string) string { return encodeNoExecuteTime(value) }
