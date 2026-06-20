// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package automated_device_enrollment_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccListResource_ProAutomatedDeviceEnrollment_Basic exercises the
// jamfplatform_pro_automated_device_enrollment list resource via the
// `terraform query` workflow. The Pro `/v1/device-enrollments` list endpoint
// returns the full instance shape per row, so include_resource = true does
// not trigger an N+1 follow-up GET. Gated on JAMFPLATFORM_ADE_TOKEN to keep
// CI deterministic — provisioning the source resource needs a real
// Apple-issued server token.
func TestAccListResource_ProAutomatedDeviceEnrollment_Basic(t *testing.T) {
	token := os.Getenv(adeTokenEnvVar)
	if token == "" {
		t.Skipf("%s not set; skipping ADE list resource acceptance test", adeTokenEnvVar)
	}

	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-ade-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAutomatedDeviceEnrollmentDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_automated_device_enrollment" "src" {
						name                    = %q
						server_token            = %q
						server_token_wo_version = 1
					}
				`, name, token),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_automated_device_enrollment.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_automated_device_enrollment" "test" {
						provider         = jamfplatform
						include_resource = true

						config {
							filter = {
								name_substring = %q
							}
						}
					}
				`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("jamfplatform_pro_automated_device_enrollment.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_automated_device_enrollment.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
						},
					),
				},
			},
		},
	})
}
