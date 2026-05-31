// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /restrictedsoftware endpoint.
// Classic has known concurrency issues when multiple writes hit the same
// resource type — keep these tests serial with any future classic acceptance
// work in this package.
//
// No pre-existing tenant target objects are required: scope happy-paths use
// all_computers = true, exclusion users are free-text local usernames, and
// general.site is left unset (defaults to "None"). This sidesteps the
// HTTP 409 "bad scope reference" invariant (a target/exclusion ID that does not
// exist is rejected) without provisioning fixtures.

package restricted_software_test

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

const restrictedSoftwareResourceAddr = "jamfplatform_pro_restricted_software.test"

// testAccCheckRestrictedSoftwareDestroy verifies records created during the test
// were destroyed.
func testAccCheckRestrictedSoftwareDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_restricted_software" {
				continue
			}
			_, err := c.GetRestrictedSoftwareByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro restricted software %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro restricted software %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// generalOnlyConfig is the import-stable shape: only the required general
// fields. The importer populates general post-Read but leaves the optional
// scope block null, so ImportStateVerify must run against a general-only config.
func generalOnlyConfig(name, processName string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_restricted_software" "test" {
			general = {
				name         = %q
				process_name = %q
			}
		}
	`, name, processName)
}

// generalFullWithSiteConfig sets every mutable general attribute explicitly,
// including site_id sourced from a jamfplatform_pro_site fixture (a real site
// is required — an unknown site ID returns HTTP 409 "Problem with site ID").
func generalFullWithSiteConfig(name, processName, message string, restrictExact, sendEmail, kill, del bool) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_site" "test" {
			name = "%s-site"
		}

		resource "jamfplatform_pro_restricted_software" "test" {
			general = {
				name                                 = %q
				process_name                         = %q
				restrict_exact_process_name          = %t
				send_email_notification_on_violation = %t
				kill_process                         = %t
				delete_application                   = %t
				display_message                      = %q
				site_id                              = jamfplatform_pro_site.test.id
			}
		}
	`, name, name, processName, restrictExact, sendEmail, kill, del, message)
}

// scopeTargetedConfig sets a targeted scope (a department target sourced from a
// jamfplatform_pro_department fixture) plus a free-text user-exclusion set.
// A targeted apply needs a real object — this is the applied targeted-scope
// happy-path, distinct from the all_computers variant and the PlanOnly
// conflict test.
func scopeTargetedConfig(name, processName, excludedUsers string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_department" "test" {
			name = "%s-dept"
		}

		resource "jamfplatform_pro_restricted_software" "test" {
			general = {
				name         = %q
				process_name = %q
			}
			scope = {
				department_ids = [jamfplatform_pro_department.test.id]
				exclusions = {
					directory_service_or_local_user_names = [%s]
				}
			}
		}
	`, name, name, processName, excludedUsers)
}

// TestAccResource_ProRestrictedSoftware_Basic exercises create, computed-default
// resolution, import, and an in-place general update. The mutated general bools
// and process_name verify the GET-after-Update path (classic PUT returns 201
// with an empty body).
func TestAccResource_ProRestrictedSoftware_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-restsw-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRestrictedSoftwareDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: generalOnlyConfig(name, "Chess.app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(restrictedSoftwareResourceAddr, "id"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.name", name),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.process_name", "Chess.app"),
					// Server-applied defaults: match_exact_process_name=true, the rest false.
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.restrict_exact_process_name", "true"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.send_email_notification_on_violation", "false"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.kill_process", "false"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.delete_application", "false"),
					// Server-derived site echo resolves to the "None" site.
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.site_id", "-1"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.site_name", "NONE"),
				),
			},
			{
				ResourceName:            restrictedSoftwareResourceAddr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
			{
				// In-place update: flip every mutable general bool, change the
				// process name, set a display message, and assign a site (every
				// non-RequiresReplace general attribute is mutated here).
				Config: generalFullWithSiteConfig(name, "Chess2.app", "This application is restricted.", false, true, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.process_name", "Chess2.app"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.restrict_exact_process_name", "false"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.send_email_notification_on_violation", "true"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.kill_process", "true"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.delete_application", "true"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.display_message", "This application is restricted."),
					resource.TestCheckResourceAttrPair(restrictedSoftwareResourceAddr, "general.site_id", "jamfplatform_pro_site.test", "id"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "general.site_name", name+"-site"),
				),
			},
		},
	})
}

// scopeAllComputersConfig keeps the department fixture declared (so it is not
// destroyed mid-test) but switches the record scope to all_computers, dropping
// the department target entirely. Used to test present→absent target clearing.
func scopeAllComputersConfig(name, processName, excludedUsers string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_department" "test" {
			name = "%s-dept"
		}

		resource "jamfplatform_pro_restricted_software" "test" {
			general = {
				name         = %q
				process_name = %q
			}
			scope = {
				all_computers = true
				exclusions = {
					directory_service_or_local_user_names = [%s]
				}
			}
		}
	`, name, name, processName, excludedUsers)
}

// TestAccResource_ProRestrictedSoftware_ScopeUpdate exercises an applied
// targeted scope (a department target), a nested-set add+remove on the
// free-text user exclusions, and a target-removal transition to all_computers.
func TestAccResource_ProRestrictedSoftware_ScopeUpdate(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-restsw-scope-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRestrictedSoftwareDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: scopeTargetedConfig(name, "Solitaire.app", `"alice", "bob"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "scope.department_ids.#", "1"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "scope.exclusions.directory_service_or_local_user_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(restrictedSoftwareResourceAddr, "scope.exclusions.directory_service_or_local_user_names.*", "alice"),
					resource.TestCheckTypeSetElemAttr(restrictedSoftwareResourceAddr, "scope.exclusions.directory_service_or_local_user_names.*", "bob"),
				),
			},
			{
				// Nested-set churn: remove "alice", add "carol" (target unchanged).
				Config: scopeTargetedConfig(name, "Solitaire.app", `"bob", "carol"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "scope.department_ids.#", "1"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "scope.exclusions.directory_service_or_local_user_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(restrictedSoftwareResourceAddr, "scope.exclusions.directory_service_or_local_user_names.*", "bob"),
					resource.TestCheckTypeSetElemAttr(restrictedSoftwareResourceAddr, "scope.exclusions.directory_service_or_local_user_names.*", "carol"),
				),
			},
			{
				// Present→absent target clearing: drop department_ids, switch to
				// all_computers. Verifies an omitted target category within a sent
				// scope is cleared server-side (not retained → would trip the
				// post-apply consistency check on department_ids).
				Config: scopeAllComputersConfig(name, "Solitaire.app", `"bob", "carol"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "scope.all_computers", "true"),
					resource.TestCheckResourceAttr(restrictedSoftwareResourceAddr, "scope.department_ids.#", "0"),
				),
			},
		},
	})
}

// TestAccResource_ProRestrictedSoftware_AllComputersConflict asserts the shared
// scope validator rejects all_computers = true alongside a computer target.
func TestAccResource_ProRestrictedSoftware_AllComputersConflict(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-restsw-allc-" + suffix
	config := fmt.Sprintf(`
		resource "jamfplatform_pro_restricted_software" "test" {
			general = {
				name         = %q
				process_name = "Chess.app"
			}
			scope = {
				all_computers      = true
				computer_group_ids = ["1"]
			}
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Conflicts with all-flag"),
			},
		},
	})
}
