// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package client_test

import (
	"os"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestMain runs all tests in the client_test package and cleans up shared
// fixtures (e.g. the smart group fixture) after they complete.
func TestMain(m *testing.M) {
	code := m.Run()
	testhelpers.CleanupSmartGroupFixture()
	os.Exit(code)
}
