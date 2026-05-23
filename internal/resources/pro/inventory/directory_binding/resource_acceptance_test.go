// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /directorybindings endpoint.
// Classic has known concurrency issues when multiple writes hit the same
// resource type — keep these tests serial with any other classic
// acceptance work in this package.

package directory_binding_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// testAccCheckDirectoryBindingDestroy verifies directory bindings created
// during the test were destroyed.
func testAccCheckDirectoryBindingDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_directory_binding" {
				continue
			}
			_, err := c.GetDirectoryBindingByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro directory binding %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro directory binding %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// TestAccResource_ProDirectoryBinding_ActiveDirectory exercises the full
// AD path: every documented field in the nested block, plus the flat
// envelope. Step 2 mutates a few fields to confirm PUT partial-merge
// preserves the rest. Step 3 imports and verifies state.
func TestAccResource_ProDirectoryBinding_ActiveDirectory(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	original := "tf-acc-directory-binding-ad-" + suffix
	renamed := "tf-acc-directory-binding-ad-renamed-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryBindingDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "test" {
						name        = %q
						priority    = 1
						type        = "Active Directory"
						domain      = "corp.example.com"
						username    = "joiner-svc"
						password    = "change-me"
						computer_ou = "OU=Macs,DC=corp,DC=example,DC=com"

						active_directory = {
							create_mobile_account      = true
							require_confirmation       = true
							force_local_home_directory = true
							use_unc_path               = true
							network_protocol           = "smb"
							default_shell              = "/bin/bash"
							uid_attribute_mapping      = "uidNumber"
							user_gid_attribute_mapping = "gidNumber"
							gid_attribute_mapping      = "primaryGroupID"
							multiple_domains           = false
							preferred_domain           = "dc01.corp.example.com"
							admin_groups               = "Mac Admins,Domain Admins"
						}
					}
				`, original),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_directory_binding.test", "id"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "name", original),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "type", "Active Directory"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "priority", "1"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "active_directory.create_mobile_account", "true"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "active_directory.network_protocol", "smb"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "active_directory.uid_attribute_mapping", "uidNumber"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "active_directory.admin_groups", "Mac Admins,Domain Admins"),
					// Server hashes the supplied password and returns the hash on read.
					resource.TestCheckResourceAttrSet("jamfplatform_pro_directory_binding.test", "password_sha256"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "test" {
						name        = %q
						priority    = 2
						type        = "Active Directory"
						domain      = "corp.example.com"
						username    = "joiner-svc"
						password    = "change-me"
						computer_ou = "OU=Macs,DC=corp,DC=example,DC=com"

						active_directory = {
							create_mobile_account      = false
							require_confirmation       = true
							force_local_home_directory = true
							use_unc_path               = false
							network_protocol           = "afp"
							default_shell              = "/bin/zsh"
							uid_attribute_mapping      = "uidNumber"
							user_gid_attribute_mapping = "gidNumber"
							gid_attribute_mapping      = "primaryGroupID"
							multiple_domains           = true
							preferred_domain           = "dc02.corp.example.com"
							admin_groups               = "Mac Admins"
						}
					}
				`, renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "name", renamed),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "priority", "2"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "active_directory.create_mobile_account", "false"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "active_directory.network_protocol", "afp"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "active_directory.multiple_domains", "true"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "active_directory.admin_groups", "Mac Admins"),
				),
			},
			{
				ResourceName:      "jamfplatform_pro_directory_binding.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
					// `password` is write-only — the server never echoes it on
					// reads, so an imported resource cannot reconstruct the
					// plaintext. `password_sha256` carries the canonical
					// server-side value across imports.
					"password",
				},
			},
		},
	})
}

// TestAccResource_ProDirectoryBinding_OpenDirectory covers the Apple Open
// Directory path. The UI labels this "Apple Open Directory" but the wire
// `type` value is the bare "Open Directory" — this test pins the mapping.
func TestAccResource_ProDirectoryBinding_OpenDirectory(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-directory-binding-od-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryBindingDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "test" {
						name     = %q
						priority = 3
						type     = "Open Directory"
						domain   = "ldap.staging.example.com"
						username = "cn=joiner,dc=staging,dc=example,dc=com"
						password = "change-me"

						open_directory = {
							encrypt_using_ssl      = true
							perform_secure_bind    = true
							use_for_authentication = true
							use_for_contacts       = false
						}
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "type", "Open Directory"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "open_directory.encrypt_using_ssl", "true"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "open_directory.perform_secure_bind", "true"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "open_directory.use_for_authentication", "true"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "open_directory.use_for_contacts", "false"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_directory_binding.test", "password_sha256"),
				),
			},
			{
				ResourceName:            "jamfplatform_pro_directory_binding.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "password"},
			},
		},
	})
}

// TestAccResource_ProDirectoryBinding_PowerBroker covers the empty-block
// path. The user supplies no nested block — the input builder synthesises
// the empty SDK struct from `type` alone, the server stores the binding
// with `<powerbroker_identity_services/>` on the wire, and the state
// builder ignores that wire element on the GET path.
func TestAccResource_ProDirectoryBinding_PowerBroker(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-directory-binding-pb-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryBindingDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "test" {
						name        = %q
						priority    = 4
						type        = "PowerBroker Identity Services"
						domain      = "lab.example.com"
						username    = "joiner@lab.example.com"
						password    = "change-me"
						computer_ou = "OU=Macs,DC=lab,DC=example,DC=com"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "type", "PowerBroker Identity Services"),
					// PowerBroker has no nested block on the TF side. Verify the
					// schema does not surface any.
					resource.TestCheckNoResourceAttr("jamfplatform_pro_directory_binding.test", "active_directory"),
					resource.TestCheckNoResourceAttr("jamfplatform_pro_directory_binding.test", "open_directory"),
					resource.TestCheckNoResourceAttr("jamfplatform_pro_directory_binding.test", "admitmac"),
					resource.TestCheckNoResourceAttr("jamfplatform_pro_directory_binding.test", "centrify"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_directory_binding.test", "password_sha256"),
				),
			},
			{
				ResourceName:            "jamfplatform_pro_directory_binding.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "password"},
			},
		},
	})
}

// TestAccResource_ProDirectoryBinding_ADmitMac exercises the ADmitMac
// path with all 16 nested fields populated, including the *int
// `cached_credentials` and the string `home_location` (distinct from
// AD's bool `force_local_home_directory` even though both round-trip
// through a wire element named `local_home`).
func TestAccResource_ProDirectoryBinding_ADmitMac(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-directory-binding-admitmac-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryBindingDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "test" {
						name        = %q
						priority    = 5
						type        = "ADmitMac"
						domain      = "corp.example.com"
						username    = "joiner-svc"
						password    = "change-me"
						computer_ou = "OU=Macs,DC=corp,DC=example,DC=com"

						admitmac = {
							require_confirmation       = false
							home_location              = "Local"
							network_protocol           = "smb"
							default_shell              = "/bin/bash"
							mount_network_home         = false
							place_home_folders         = "/Users"
							uid_attribute_mapping      = "uidNumber"
							user_gid_attribute_mapping = "gidNumber"
							gid_attribute_mapping      = "primaryGroupID"
							admin_group                = "Mac Admins"
							cached_credentials         = 10
							add_user_to_local          = true
							users_ou                   = "OU=Users,DC=corp,DC=example,DC=com"
							groups_ou                  = "OU=Groups,DC=corp,DC=example,DC=com"
							printers_ou                = "OU=Printers,DC=corp,DC=example,DC=com"
							shared_folders_ou          = "OU=Shares,DC=corp,DC=example,DC=com"
						}
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "type", "ADmitMac"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "admitmac.home_location", "Local"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "admitmac.cached_credentials", "10"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "admitmac.network_protocol", "smb"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "admitmac.add_user_to_local", "true"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "admitmac.shared_folders_ou", "OU=Shares,DC=corp,DC=example,DC=com"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_directory_binding.test", "password_sha256"),
				),
			},
			{
				ResourceName:            "jamfplatform_pro_directory_binding.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "password"},
			},
		},
	})
}

// TestAccResource_ProDirectoryBinding_Centrify exercises the Centrify
// path and confirms `update_pam` round-trips through the wire element
// `<update_PAM>` (uppercase preserved on the wire; lowercase in TF).
func TestAccResource_ProDirectoryBinding_Centrify(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-directory-binding-centrify-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryBindingDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "test" {
						name     = %q
						priority = 6
						type     = "Centrify"
						domain   = "corp.example.com"
						username = "joiner-svc"
						password = "change-me"

						centrify = {
							workstation_mode        = false
							overwrite_existing      = true
							update_pam              = true
							zone                    = "macs"
							preferred_domain_server = "dc01.corp.example.com"
						}
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "type", "Centrify"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "centrify.zone", "macs"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "centrify.update_pam", "true"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "centrify.workstation_mode", "false"),
					resource.TestCheckResourceAttr("jamfplatform_pro_directory_binding.test", "centrify.overwrite_existing", "true"),
					resource.TestCheckResourceAttrSet("jamfplatform_pro_directory_binding.test", "password_sha256"),
				),
			},
			{
				ResourceName:            "jamfplatform_pro_directory_binding.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "password"},
			},
		},
	})
}

// TestAccResource_ProDirectoryBinding_TypeBlockMismatch exercises the
// plan-time cross-field validator. type=Active Directory + an admitmac
// block must surface a typed plan-time error and never reach apply.
func TestAccResource_ProDirectoryBinding_TypeBlockMismatch(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-directory-binding-mismatch-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "test" {
						name = %q
						type = "Active Directory"

						admitmac = {
							home_location = "Local"
						}
					}
				`, name),
				ExpectError: regexp.MustCompile(`admitmac forbidden when type`),
			},
		},
	})
}

// TestAccResource_ProDirectoryBinding_TypeBlockMismatch_PowerBroker
// confirms PowerBroker forbids every nested block — its identity is
// conveyed by `type` alone.
func TestAccResource_ProDirectoryBinding_TypeBlockMismatch_PowerBroker(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-directory-binding-pb-mismatch-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_directory_binding" "test" {
						name = %q
						type = "PowerBroker Identity Services"

						active_directory = {
							use_unc_path = true
						}
					}
				`, name),
				ExpectError: regexp.MustCompile(`active_directory forbidden when type`),
			},
		},
	})
}
