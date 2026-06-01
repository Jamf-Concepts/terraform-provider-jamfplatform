// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package api_role_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func testAccCheckApiRoleDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_api_role" {
				continue
			}
			_, err := c.GetApiRoleV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro API role %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro API role %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// TestAccResource_ProApiRole exercises the create/update round-trip: rename the
// role, and both add and remove a privilege. Import round-trip verifies state.
func TestAccResource_ProApiRole(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-api-role-" + suffix
	nameUpdated := "tf-acc-pro-api-role-upd-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiRoleDestroy(t),
		Steps: []resource.TestStep{
			{
				// Start with two privileges.
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_api_role" "test" {
						display_name = %q
						privileges   = ["Read API Roles", "Read API Integrations"]
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_api_role.test", "id"),
					resource.TestCheckResourceAttr("jamfplatform_pro_api_role.test", "display_name", name),
					resource.TestCheckResourceAttr("jamfplatform_pro_api_role.test", "privileges.#", "2"),
				),
			},
			{
				// Rename + add a privilege (3 total).
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_api_role" "test" {
						display_name = %q
						privileges   = ["Read API Roles", "Read API Integrations", "Update API Roles"]
					}
				`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_api_role.test", "display_name", nameUpdated),
					resource.TestCheckResourceAttr("jamfplatform_pro_api_role.test", "privileges.#", "3"),
				),
			},
			{
				// Remove privileges down to one.
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_api_role" "test" {
						display_name = %q
						privileges   = ["Read API Roles"]
					}
				`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_api_role.test", "privileges.#", "1"),
				),
			},
			{
				ResourceName:            "jamfplatform_pro_api_role.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

// TestAccResource_ProApiRole_InvalidPrivilege asserts the plan-time privilege
// validator rejects an unknown privilege string with a clear error.
func TestAccResource_ProApiRole_InvalidPrivilege(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-api-role-bad-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_api_role" "test" {
						display_name = %q
						privileges   = ["Not A Real Privilege XYZ"]
					}
				`, name),
				ExpectError: regexp.MustCompile(`not\s+a\s+valid\s+Jamf\s+Pro\s+privilege`),
			},
		},
	})
}

func TestAccDataSource_ProApiRole(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-api-role-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiRoleDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_api_role" "src" {
						display_name = %q
						privileges   = ["Read API Roles"]
					}

					data "jamfplatform_pro_api_role" "lookup" {
						id = jamfplatform_pro_api_role.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_api_role.lookup", "display_name", "jamfplatform_pro_api_role.src", "display_name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_api_role.lookup", "privileges.#", "1"),
				),
			},
		},
	})
}

func TestAccDataSource_ProApiRoles_FilterByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-api-roles-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiRoleDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_api_role" "src" {
						display_name = %q
						privileges   = ["Read API Roles"]
					}

					data "jamfplatform_pro_api_roles" "lookup" {
						filter = [
							{
								selector = "displayName"
								argument = jamfplatform_pro_api_role.src.display_name
							}
						]
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_api_roles.lookup", "api_roles.#", "1"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_api_roles.lookup", "api_roles.0.display_name", name),
				),
			},
		},
	})
}

// TestAccDataSource_ProApiRolePrivileges asserts the privilege-discovery data
// source returns a non-empty privilege set.
func TestAccDataSource_ProApiRolePrivileges(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_pro_api_role_privileges" "all" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_api_role_privileges.all", "privileges.#"),
				),
			},
		},
	})
}
