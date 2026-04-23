// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package device_group_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devicegroups"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckDeviceGroupDestroy verifies that device groups created during the test
// have been destroyed.
func testAccCheckDeviceGroupDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testhelpers.NewAcceptanceClient(t)
		dgClient := devicegroups.New(c)
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_device_group" {
				continue
			}
			deadline := time.Now().Add(60 * time.Second)
			for time.Now().Before(deadline) {
				_, err := dgClient.GetDeviceGroup(ctx, rs.Primary.ID)
				if err != nil {
					if helpers.IsNotFoundError(err) {
						break
					}
					return fmt.Errorf("error checking device group %s: %s", rs.Primary.ID, err)
				}
				time.Sleep(2 * time.Second)
			}
		}
		return nil
	}
}

func TestAccResource_DeviceGroup_StaticComputer(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-static-computer-" + suffix
	nameUpdated := "tf-acc-static-computer-updated-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test_static" {
						name        = %q
						description = "Acceptance test — safe to delete"
						group_type  = "static"
						device_type = "computer"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_device_group.test_static", "id"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "group_type", "static"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "device_type", "computer"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test_static" {
						name        = %q
						description = "Updated description"
						group_type  = "static"
						device_type = "computer"
					}
				`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "name", nameUpdated),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccResource_DeviceGroup_SmartComputer(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-smart-computer-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test_smart" {
						name        = %q
						description = "Acceptance test — safe to delete"
						group_type  = "smart"
						device_type = "computer"
						criteria = [{
							criteria = "Serial Number"
							operator = "like"
							value    = ""
						}]
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_device_group.test_smart", "id"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_smart", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_smart", "group_type", "smart"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_smart", "device_type", "computer"),
					resource.TestCheckResourceAttrSet("jamfplatform_device_group.test_smart", "member_count"),
				),
			},
		},
	})
}

func TestAccResource_DeviceGroup_SmartMobile(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-smart-mobile-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test_mobile" {
						name        = %q
						description = "Acceptance test — safe to delete"
						group_type  = "smart"
						device_type = "mobile"
						criteria = [{
							criteria = "Serial Number"
							operator = "like"
							value    = ""
						}]
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_device_group.test_mobile", "id"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_mobile", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_mobile", "device_type", "mobile"),
				),
			},
		},
	})
}

func TestAccResource_DeviceGroup_ImportState(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-import-test-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test_import" {
						name        = %q
						group_type  = "static"
						device_type = "computer"
					}
				`, name),
			},
			{
				ResourceName:      "jamfplatform_device_group.test_import",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDataSource_DeviceGroup(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ds-device-group-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "source" {
						name        = %q
						group_type  = "static"
						device_type = "computer"
					}

					data "jamfplatform_device_group" "test" {
						id = jamfplatform_device_group.source.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_device_group.test", "name", name),
					resource.TestCheckResourceAttrSet("data.jamfplatform_device_group.test", "group_type"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_device_group.test", "device_type"),
				),
			},
		},
	})
}

func TestAccDataSource_DeviceGroups(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_device_groups" "all" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_device_groups.all", "device_groups.#"),
				),
			},
		},
	})
}

func TestAccResource_DeviceGroup_DescriptionNullVsEmpty(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-desc-nullempty-" + suffix
	rn := "jamfplatform_device_group.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			// Step 1: create with a real description value
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test" {
						name        = %q
						description = "initial value"
						group_type  = "static"
						device_type = "computer"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "description", "initial value"),
				),
			},
			// Step 2: set description to explicit empty string — must be preserved as ""
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test" {
						name        = %q
						description = ""
						group_type  = "static"
						device_type = "computer"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "description", ""),
				),
			},
			// Step 3: set description to explicit null — must become unset in state
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test" {
						name        = %q
						description = null
						group_type  = "static"
						device_type = "computer"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(rn, "description"),
				),
			},
			// Step 4: omit description entirely — equivalent to null, plan must be empty
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test" {
						name        = %q
						group_type  = "static"
						device_type = "computer"
					}
				`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			// Step 5: re-add description to verify it can be restored after being unset
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_device_group" "test" {
						name        = %q
						description = "restored"
						group_type  = "static"
						device_type = "computer"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "description", "restored"),
				),
			},
		},
	})
}
