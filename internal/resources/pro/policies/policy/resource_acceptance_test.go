// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /policies endpoint. Classic
// has known concurrency issues when multiple writes hit the same resource
// type — keep these tests serial with any other classic acceptance work.

package policy_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// testAccCheckPolicyDestroy verifies policies created during the test were
// destroyed.
func testAccCheckPolicyDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_policy" {
				continue
			}
			_, err := c.GetPolicyByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro policy %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro policy %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

func policyConfigMinimal(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name    = %q
    enabled = true
  }
}
`, name)
}

func policyConfigEnabledFlip(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name    = %q
    enabled = false
  }
}
`, name)
}

func policyConfigSelfService(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name    = %q
    enabled = true
  }
  self_service = {
    use_for_self_service      = true
    self_service_display_name = %q
    notification_enabled      = true
    notification_type       = "Self Service"
    notification_subject      = "tf-acc"
    notification_message      = "Test policy"
  }
}
`, name, name)
}

func policyConfigAllComputers(name string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name = %q
  }
  scope = {
    all_computers = true
  }
}
`, name)
}

// TestAccPolicyResource_Minimal covers the smallest viable policy: name only.
// Verifies Create/Read/Update/Delete + import.
func TestAccPolicyResource_Minimal(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-min-" + suffix
	renamed := name + "-renamed"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigMinimal(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("enabled"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				Config: policyConfigMinimal(renamed),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("name"),
						knownvalue.StringExact(renamed),
					),
				},
			},
			{
				Config: policyConfigEnabledFlip(renamed),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("general").AtMapKey("enabled"),
						knownvalue.Bool(false),
					),
				},
			},
			{
				ResourceName:                         "jamfplatform_pro_policy.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
			},
		},
	})
}

// TestAccPolicyResource_SelfServiceNotificationSplit confirms the two
// <notification> elements round-trip cleanly through the schema's split
// notification_enabled + notification_type attributes.
func TestAccPolicyResource_SelfServiceNotificationSplit(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-ss-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigSelfService(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("self_service").AtMapKey("notification_enabled"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("self_service").AtMapKey("notification_type"),
						knownvalue.StringExact("Self Service"),
					),
				},
			},
		},
	})
}

// TestAccPolicyResource_AllComputers verifies all_computers=true creates
// without per-computer targets.
func TestAccPolicyResource_AllComputers(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-policy-all-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: policyConfigAllComputers(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_policy.test",
						tfjsonpath.New("scope").AtMapKey("all_computers"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}
