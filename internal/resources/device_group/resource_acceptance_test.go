// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package device_group_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckDeviceGroupDestroy verifies that device groups created during the test
// have been destroyed.
func testAccCheckDeviceGroupDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testhelpers.NewAcceptanceClient(t)
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_device_group" {
				continue
			}
			_, err := c.GetDeviceGroupByIDV1(ctx, rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("device group %s still exists after destroy", rs.Primary.ID)
			}
			if !helpers.IsNotFoundError(err) {
				return fmt.Errorf("error checking device group %s: %s", rs.Primary.ID, err)
			}
		}
		return nil
	}
}

func TestAccResource_DeviceGroup_StaticComputer(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_device_group" "test_static" {
						name        = "tf-acc-static-computer"
						description = "Acceptance test — safe to delete"
						group_type  = "static"
						device_type = "computer"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_device_group.test_static", "id"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "name", "tf-acc-static-computer"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "group_type", "static"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "device_type", "computer"),
				),
			},
			{
				Config: `
					resource "jamfplatform_device_group" "test_static" {
						name        = "tf-acc-static-computer-updated"
						description = "Updated description"
						group_type  = "static"
						device_type = "computer"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "name", "tf-acc-static-computer-updated"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_static", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccResource_DeviceGroup_SmartComputer(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_device_group" "test_smart" {
						name        = "tf-acc-smart-computer"
						description = "Acceptance test — safe to delete"
						group_type  = "smart"
						device_type = "computer"
						criteria = [{
							criteria = "Serial Number"
							operator = "like"
							value    = ""
						}]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_device_group.test_smart", "id"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_smart", "name", "tf-acc-smart-computer"),
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

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_device_group" "test_mobile" {
						name        = "tf-acc-smart-mobile"
						description = "Acceptance test — safe to delete"
						group_type  = "smart"
						device_type = "mobile"
						criteria = [{
							criteria = "Serial Number"
							operator = "like"
							value    = ""
						}]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_device_group.test_mobile", "id"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_mobile", "name", "tf-acc-smart-mobile"),
					resource.TestCheckResourceAttr("jamfplatform_device_group.test_mobile", "device_type", "mobile"),
				),
			},
		},
	})
}

func TestAccResource_DeviceGroup_ImportState(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_device_group" "test_import" {
						name        = "tf-acc-import-test"
						group_type  = "static"
						device_type = "computer"
					}
				`,
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

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeviceGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_device_group" "source" {
						name        = "tf-acc-ds-device-group"
						group_type  = "static"
						device_type = "computer"
					}

					data "jamfplatform_device_group" "test" {
						id = jamfplatform_device_group.source.id
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_device_group.test", "name", "tf-acc-ds-device-group"),
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
				Config: fmt.Sprintf(`
					data "jamfplatform_device_groups" "all" {}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_device_groups.all", "device_groups.#"),
				),
			},
		},
	})
}
