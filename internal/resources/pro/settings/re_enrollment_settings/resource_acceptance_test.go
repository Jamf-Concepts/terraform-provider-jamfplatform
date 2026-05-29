// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package re_enrollment_settings_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// The Re-enrollment settings object is a tenant-wide singleton that always
// exists and cannot be deleted (Delete is state-only by design). Every test
// therefore uses an INVERTED CheckDestroy: after `terraform destroy` the record
// must still be readable on the tenant. Tests leave the record in a known
// explicit state (all six fields set) so there is no ordering dependence
// between tests — the singleton always exists.

const reEnrollmentResourceAddr = "jamfplatform_pro_re_enrollment_settings.test"

// checkReEnrollmentStillExists verifies Delete did not remove the Re-enrollment
// settings record. The Delete handler is documented as state-only, so a GET must
// still succeed after destroy.
func checkReEnrollmentStillExists(t *testing.T) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		got, err := c.GetReenrollmentSettingsV1(context.Background())
		if err != nil {
			return fmt.Errorf("expected Re-enrollment settings record to persist after destroy: %w", err)
		}
		if got == nil {
			return fmt.Errorf("expected non-nil Re-enrollment settings post-destroy")
		}
		return nil
	}
}

// reEnrollmentConfig renders a full six-field config. Every attribute is
// Required so all six must always be present.
func reEnrollmentConfig(clearPolicyLogs, clearLocationInfo, clearLocationHistory, clearExtAttrs, clearSUPlans bool, clearManagementHistory string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_re_enrollment_settings" "test" {
			clear_policy_logs                  = %t
			clear_location_information         = %t
			clear_location_information_history = %t
			clear_extension_attributes         = %t
			clear_software_update_plans        = %t
			clear_management_history           = %q
		}
	`, clearPolicyLogs, clearLocationInfo, clearLocationHistory, clearExtAttrs, clearSUPlans, clearManagementHistory)
}

// TestAccResource_ProReEnrollmentSettings_Update drives the full Update
// round-trip. Step 1 sets a mix of the five bools plus the enum; step 2 flips
// all five bools and changes the enum. Every attribute is asserted on both
// steps. This is the highest-value test — it proves the full-replace PUT and
// state round-trip across every non-RequiresReplace attribute.
func TestAccResource_ProReEnrollmentSettings_Update(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkReEnrollmentStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: reEnrollmentConfig(true, false, true, false, true, "DELETE_EVERYTHING_EXCEPT_ACKNOWLEDGED"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "id", "singleton"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_policy_logs", "true"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_location_information", "false"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_location_information_history", "true"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_extension_attributes", "false"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_software_update_plans", "true"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_management_history", "DELETE_EVERYTHING_EXCEPT_ACKNOWLEDGED"),
				),
			},
			{
				// Flip all five bools and change the enum.
				Config: reEnrollmentConfig(false, true, false, true, false, "DELETE_NOTHING"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_policy_logs", "false"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_location_information", "true"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_location_information_history", "false"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_extension_attributes", "true"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_software_update_plans", "false"),
					resource.TestCheckResourceAttr(reEnrollmentResourceAddr, "clear_management_history", "DELETE_NOTHING"),
				),
			},
		},
	})
}

// TestAccResource_ProReEnrollmentSettings_Import exercises the import
// round-trip. All six attributes are non-sub-block scalars, so ImportStateVerify
// is safe here (the singleton importer's post-import Read populates every base
// attribute). The second step asserts the non-singleton import guard.
func TestAccResource_ProReEnrollmentSettings_Import(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkReEnrollmentStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: reEnrollmentConfig(true, true, true, true, true, "DELETE_EVERYTHING"),
			},
			{
				ResourceName:      reEnrollmentResourceAddr,
				ImportState:       true,
				ImportStateId:     "singleton",
				ImportStateVerify: true,
			},
			{
				ResourceName:  reEnrollmentResourceAddr,
				ImportState:   true,
				ImportStateId: "not-the-singleton",
				ExpectError:   regexp.MustCompile(`Invalid singleton import identifier`),
			},
		},
	})
}

// TestAccResource_ProReEnrollmentSettings_InvalidEnum verifies the OneOf
// validator on clear_management_history rejects an unknown value. The regex
// matches a single contiguous enum literal embedded in the validator's allowed
// list — no spaces, so it survives Terraform's ~80-col error wrapping.
func TestAccResource_ProReEnrollmentSettings_InvalidEnum(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      reEnrollmentConfig(true, true, true, true, true, "BOGUS"),
				ExpectError: regexp.MustCompile(`DELETE_EVERYTHING_EXCEPT_ACKNOWLEDGED`),
			},
		},
	})
}

// TestAccDataSource_ProReEnrollmentSettings_Basic applies the resource then
// reads it back through the data source and checks a couple of attributes.
func TestAccDataSource_ProReEnrollmentSettings_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkReEnrollmentStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: reEnrollmentConfig(true, false, false, false, false, "DELETE_ERRORS") + `
					data "jamfplatform_pro_re_enrollment_settings" "ds" {
						depends_on = [jamfplatform_pro_re_enrollment_settings.test]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_re_enrollment_settings.ds", "id", "singleton"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_re_enrollment_settings.ds", "clear_policy_logs", "true"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_re_enrollment_settings.ds", "clear_management_history", "DELETE_ERRORS"),
				),
			},
		},
	})
}
