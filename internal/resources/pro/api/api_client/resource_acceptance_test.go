// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package api_client_test

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

func testAccCheckApiClientDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_api_client" {
				continue
			}
			_, err := c.GetApiIntegrationV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro API client %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro API client %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// roleFixture returns HCL for an API role the client can reference by display name.
func roleFixture(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_api_role" "role" {
			display_name = %q
			privileges   = ["Read Computers"]
		}
	`, name)
}

// TestAccResource_ProApiClient drives the full lifecycle: create with a generated
// secret, rotate the secret, update other attributes without rotating (secret
// must be stable), add/remove an api_role scope, and import. It asserts the
// secret is produced, rotates on trigger change, and that the credential
// generation path works end-to-end against a same-apply api_role fixture.
func TestAccResource_ProApiClient(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	roleName := "tf-acc-pro-api-client-role-" + suffix
	roleName2 := "tf-acc-pro-api-client-role2-" + suffix
	name := "tf-acc-pro-api-client-" + suffix
	nameUpdated := "tf-acc-pro-api-client-upd-" + suffix

	var secretCreate, secretRotated string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiClientDestroy(t),
		Steps: []resource.TestStep{
			{
				// Create, enabled, with first secret generation.
				Config: roleFixture(roleName) + fmt.Sprintf(`
					resource "jamfplatform_pro_api_client" "test" {
						display_name                  = %q
						api_roles                     = [jamfplatform_pro_api_role.role.display_name]
						enabled                       = true
						access_token_lifetime_seconds = 300
						credential_rotation            = "1"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_api_client.test", "id"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_api_client.test", "client_id"),
					resource.TestCheckResourceAttr("jamfplatform_pro_api_client.test", "app_type", "CLIENT_CREDENTIALS"),
					resource.TestCheckResourceAttr("jamfplatform_pro_api_client.test", "enabled", "true"),
					resource.TestCheckResourceAttr("jamfplatform_pro_api_client.test", "api_roles.#", "1"),
					resource.TestCheckResourceAttrWith("jamfplatform_pro_api_client.test", "client_secret", func(v string) error {
						if v == "" {
							return fmt.Errorf("client_secret should be set after generation")
						}
						secretCreate = v
						return nil
					}),
				),
			},
			{
				// Rotate: trigger change mints a fresh secret, same client_id.
				Config: roleFixture(roleName) + fmt.Sprintf(`
					resource "jamfplatform_pro_api_client" "test" {
						display_name                  = %q
						api_roles                     = [jamfplatform_pro_api_role.role.display_name]
						enabled                       = true
						access_token_lifetime_seconds = 300
						credential_rotation            = "2"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("jamfplatform_pro_api_client.test", "client_secret", func(v string) error {
						if v == "" || v == secretCreate {
							return fmt.Errorf("client_secret should rotate to a new value on trigger change")
						}
						secretRotated = v
						return nil
					}),
				),
			},
			{
				// Update non-credential attrs (rename, lifetime, add a role) WITHOUT
				// changing the trigger: the secret must remain stable.
				Config: roleFixture(roleName) + fmt.Sprintf(`
					resource "jamfplatform_pro_api_role" "role2" {
						display_name = %q
						privileges   = ["Read Mobile Devices"]
					}

					resource "jamfplatform_pro_api_client" "test" {
						display_name                  = %q
						api_roles                     = [jamfplatform_pro_api_role.role.display_name, jamfplatform_pro_api_role.role2.display_name]
						enabled                       = true
						access_token_lifetime_seconds = 600
						credential_rotation            = "2"
					}
				`, roleName2, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_api_client.test", "display_name", nameUpdated),
					resource.TestCheckResourceAttr("jamfplatform_pro_api_client.test", "access_token_lifetime_seconds", "600"),
					resource.TestCheckResourceAttr("jamfplatform_pro_api_client.test", "api_roles.#", "2"),
					resource.TestCheckResourceAttrWith("jamfplatform_pro_api_client.test", "client_secret", func(v string) error {
						if v != secretRotated {
							return fmt.Errorf("client_secret must be stable when the trigger is unchanged")
						}
						return nil
					}),
				),
			},
			{
				// Remove the second role scope from the client. role2 stays DEFINED
				// (just unreferenced) so Terraform updates the client to drop the
				// scope rather than destroying a role that is still in use — deleting
				// a role still assigned to a client returns 406 HAS_DEPENDENCIES.
				Config: roleFixture(roleName) + fmt.Sprintf(`
					resource "jamfplatform_pro_api_role" "role2" {
						display_name = %q
						privileges   = ["Read Mobile Devices"]
					}

					resource "jamfplatform_pro_api_client" "test" {
						display_name                  = %q
						api_roles                     = [jamfplatform_pro_api_role.role.display_name]
						enabled                       = true
						access_token_lifetime_seconds = 600
						credential_rotation            = "2"
					}
				`, roleName2, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_api_client.test", "api_roles.#", "1"),
				),
			},
			{
				ResourceName:      "jamfplatform_pro_api_client.test",
				ImportState:       true,
				ImportStateVerify: true,
				// client_secret is never returned by the API; credential_rotation is a
				// client-side trigger not stored on the server; timeouts are local.
				ImportStateVerifyIgnore: []string{"client_secret", "credential_rotation", "timeouts"},
			},
		},
	})
}

// TestAccResource_ProApiClient_RotationRequiresEnabled asserts the cross-field
// validator: credential_rotation cannot be set while the client is disabled.
func TestAccResource_ProApiClient_RotationRequiresEnabled(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	roleName := "tf-acc-pro-api-client-vrole-" + suffix
	name := "tf-acc-pro-api-client-v-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiClientDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: roleFixture(roleName) + fmt.Sprintf(`
					resource "jamfplatform_pro_api_client" "test" {
						display_name        = %q
						api_roles           = [jamfplatform_pro_api_role.role.display_name]
						enabled             = false
						credential_rotation = "1"
					}
				`, name),
				ExpectError: regexp.MustCompile(`credential_rotation\s+requires\s+enabled`),
			},
		},
	})
}

// TestAccResource_ProApiClient_NoCredentials asserts a client created without a
// credential_rotation trigger has no secret and app_type NONE.
func TestAccResource_ProApiClient_NoCredentials(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	roleName := "tf-acc-pro-api-client-ncrole-" + suffix
	name := "tf-acc-pro-api-client-nc-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiClientDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: roleFixture(roleName) + fmt.Sprintf(`
					resource "jamfplatform_pro_api_client" "test" {
						display_name = %q
						api_roles    = [jamfplatform_pro_api_role.role.display_name]
						enabled      = true
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_api_client.test", "app_type", "NONE"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_api_client.test", "client_id"),
					resource.TestCheckNoResourceAttr("jamfplatform_pro_api_client.test", "client_secret"),
				),
			},
		},
	})
}

func TestAccDataSource_ProApiClient(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	roleName := "tf-acc-pro-api-client-dsrole-" + suffix
	name := "tf-acc-pro-api-client-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiClientDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: roleFixture(roleName) + fmt.Sprintf(`
					resource "jamfplatform_pro_api_client" "src" {
						display_name = %q
						api_roles    = [jamfplatform_pro_api_role.role.display_name]
						enabled      = true
					}

					data "jamfplatform_pro_api_client" "lookup" {
						id = jamfplatform_pro_api_client.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_api_client.lookup", "display_name", "jamfplatform_pro_api_client.src", "display_name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_api_client.lookup", "client_id", "jamfplatform_pro_api_client.src", "client_id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_api_client.lookup", "api_roles.#", "1"),
				),
			},
		},
	})
}

func TestAccDataSource_ProApiClients_FilterByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	roleName := "tf-acc-pro-api-clients-role-" + suffix
	name := "tf-acc-pro-api-clients-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiClientDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: roleFixture(roleName) + fmt.Sprintf(`
					resource "jamfplatform_pro_api_client" "src" {
						display_name = %q
						api_roles    = [jamfplatform_pro_api_role.role.display_name]
						enabled      = true
					}

					data "jamfplatform_pro_api_clients" "lookup" {
						filter = [
							{
								selector = "displayName"
								argument = jamfplatform_pro_api_client.src.display_name
							}
						]
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_api_clients.lookup", "api_clients.#", "1"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_api_clients.lookup", "api_clients.0.display_name", name),
				),
			},
		},
	})
}
