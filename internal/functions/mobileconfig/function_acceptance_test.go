// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package mobileconfig_test

import (
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// TestAccFunction_Mobileconfig drives the mobileconfig function through a real
// Terraform run and asserts the rendered .mobileconfig. Provider-defined
// functions are offline (no API client, no provider config), so this test does
// NOT call testhelpers.AccPreCheck — there are no tenant credentials to gate on.
// It gates only on the Terraform version, since provider functions are GA from
// Terraform 1.8. This is the reference pattern for future provider functions.
func TestAccFunction_Mobileconfig(t *testing.T) {
	// Offline precheck: sets TF_ACC without gating on tenant credentials, so
	// this test runs under the raw `go test -tags=acceptance` command too rather
	// than skipping silently.
	testhelpers.AccPreCheckOffline(t)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					output "profile" {
						value = provider::jamfplatform::mobileconfig({
							identifier = "com.example.dock"
							payloads = [
								{
									PayloadType = "com.apple.dock"
									tilesize    = 48
								},
							]
						})
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("profile",
						knownvalue.StringRegexp(regexp.MustCompile(`<key>PayloadType</key>`))),
					statecheck.ExpectKnownOutputValue("profile",
						knownvalue.StringRegexp(regexp.MustCompile(`com\.apple\.dock`))),
					// Whole number renders as <integer>, proving the decode →
					// normalize → plist path end to end through Terraform.
					statecheck.ExpectKnownOutputValue("profile",
						knownvalue.StringRegexp(regexp.MustCompile(`<integer>48</integer>`))),
				},
			},
		},
	})
}
