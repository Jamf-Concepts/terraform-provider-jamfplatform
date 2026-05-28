//go:build acceptance

// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_prestage_enrollment_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// requireADETokenFixture skips the test when JAMFPLATFORM_ADE_TOKEN is not
// set. The env var holds the ADE / DEP instance ID Jamf Pro uses when
// creating a Computer PreStage; without it the POST is rejected with
// `400 INVALID_ID`.
func requireADETokenFixture(t *testing.T) string {
	t.Helper()
	v := os.Getenv("JAMFPLATFORM_ADE_TOKEN")
	if v == "" {
		t.Skip("JAMFPLATFORM_ADE_TOKEN not set — acc tests require a Jamf Pro Device Enrollment Program instance ID seeded in the tenant under test.")
	}
	return v
}

func TestAccResource_ProComputerPrestageEnrollment_Minimal(t *testing.T) {
	testhelpers.AccPreCheck(t)
	adeID := requireADETokenFixture(t)

	resourceName := "jamfplatform_pro_computer_prestage_enrollment.test"
	displayName := "tf-acc-computer-prestage"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccComputerPrestageMinimalConfig(displayName, adeID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "display_name", displayName),
					resource.TestCheckResourceAttr(resourceName, "device_enrollment_program_instance_id", adeID),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "profile_uuid"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
					"recovery_lock_password",
					"recovery_lock_password_wo_version",
					"account_settings.admin_password",
					"account_settings.admin_password_wo_version",
				},
			},
			{
				Config: testAccComputerPrestageMinimalConfig(displayName+"-renamed", adeID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "display_name", displayName+"-renamed"),
				),
			},
		},
	})
}

func testAccComputerPrestageMinimalConfig(name, adeID string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_computer_prestage_enrollment" "test" {
  display_name                          = %q
  mandatory                             = true
  mdm_removable                         = true
  require_authentication                = false
  device_enrollment_program_instance_id = %q
  keep_existing_location_information    = false
  keep_existing_site_membership         = false
  auto_advance_setup                    = false
  install_profiles_during_setup         = false
  prevent_activation_lock               = false
  enable_device_based_activation_lock   = false

  location_information   = {}
  purchasing_information = {}
  account_settings       = {}
  skip_setup_items       = {}

  scope_serial_numbers = []
}
`, name, adeID)
}
