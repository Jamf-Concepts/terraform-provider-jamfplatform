// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file create real Jamf Pro administrator accounts via the Pro API
// and write the privilege grid via the classic API. Base-field updates route
// through Pro PUT (now accepted via the platform gateway) and are exercised by
// TestAccResource_ProAccount_BaseUpdate.

package account_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func testAccCheckAccountDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_account" {
				continue
			}
			_, err := c.GetAccountV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking account %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("account %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// importedPrivilegeCheck asserts that an imported account carries a populated
// Jamf-Pro-server-objects privilege category including the named privilege. It
// matches on the element values rather than on an index, because a Set lands in
// the shimmed import state under keys the provider does not choose, and it
// asserts membership rather than an exact count, because Jamf Pro adds
// dependency privileges of its own to the grid an import materialises.
func importedPrivilegeCheck(want string) resource.ImportStateCheckFunc {
	const category = "privileges.jamf_pro_server_objects."
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("expected one imported instance, got %d", len(states))
		}
		attrs := states[0].Attributes
		if count := attrs[category+"#"]; count == "" || count == "0" {
			return fmt.Errorf("imported account has no jamf_pro_server_objects privileges; the grid the classic endpoint returns must reach state on import")
		}
		for key, value := range attrs {
			if strings.HasPrefix(key, category) && !strings.HasSuffix(key, "#") && value == want {
				return nil
			}
		}
		return fmt.Errorf("imported jamf_pro_server_objects does not contain %q", want)
	}
}

func customAccountConfig(suffix string, objects ...string) string {
	var quoted strings.Builder
	for i, p := range objects {
		if i > 0 {
			quoted.WriteString(", ")
		}
		fmt.Fprintf(&quoted, "%q", p)
	}
	return fmt.Sprintf(`
resource "jamfplatform_pro_account" "test" {
  username      = "tf-acc-acct-%[1]s"
  full_name     = "TF Acc Account"
  email_address = "tf-acc-acct-%[1]s@example.invalid"
  access_level  = "Full Access"
  privilege_set = "Custom"

  password            = "Pr0bePassw0rd-%[1]s"
  password_wo_version = 1

  privileges = {
    jamf_pro_server_objects = [%[2]s]
  }
}
`, suffix, quoted.String())
}

// TestAccResource_ProAccount covers the create + privilege-only update + import
// lifecycle, all of which work through the gateway today (Pro create, classic
// privilege write, Pro delete). The privilege set is grown then shrunk to prove
// intersect-on-read (server-added dependency privileges do not leak; removals
// are honoured).
func TestAccResource_ProAccount(t *testing.T) {
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccountDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: customAccountConfig(suffix, "Read Computers", "Update Computers"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_account.test", "id"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "username", "tf-acc-acct-"+suffix),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "access_level", "Full Access"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "privilege_set", "Custom"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "account_type", "DEFAULT"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "privileges.jamf_pro_server_objects.#", "2"),
				),
			},
			{
				// Privilege-only update (classic write; no base change ⇒ no Pro PUT).
				Config: customAccountConfig(suffix, "Read Computers"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "privileges.jamf_pro_server_objects.#", "1"),
					resource.TestCheckTypeSetElemAttr("jamfplatform_pro_account.test", "privileges.jamf_pro_server_objects.*", "Read Computers"),
				),
			},
			{
				// Import smoke. ImportStateVerify omitted: password is WriteOnly
				// (never returned) and import materialises the full server
				// privilege grid, so a full-fidelity compare would not match.
				// ImportStateCheck carries the assertion instead. The grid is
				// what an earlier revision dropped on this path, and a compare
				// that cannot run is no cover for it (issue #372).
				ResourceName:      "jamfplatform_pro_account.test",
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateCheck:  importedPrivilegeCheck("Read Computers"),
			},
		},
	})
}

// baseAccountConfig renders a non-Custom (Auditor) account so the base-update
// path is exercised in isolation: no Custom privilege grid means no classic
// write, so only the Pro PUT base-field path runs.
func baseAccountConfig(suffix, fullName, accessStatus string, passwordWOVersion int) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_account" "test" {
  username      = "tf-acc-base-%[1]s"
  full_name     = %[2]q
  email_address = "tf-acc-base-%[1]s@example.invalid"
  access_level  = "Full Access"
  privilege_set = "Auditor"
  access_status = %[3]q

  password            = "Pr0bePassw0rd-%[1]s-v%[4]d"
  password_wo_version = %[4]d
}
`, suffix, fullName, accessStatus, passwordWOVersion)
}

// TestAccResource_ProAccount_BaseUpdate exercises in-place base-field updates,
// which route through Pro PUT. Step 2 changes plain base fields (full name,
// access status); step 3 rotates the WriteOnly password (bumped wo_version →
// password re-sent on the same PUT). Both confirm the gateway now accepts the
// Pro update that previously returned 403 BAD_PERMISSIONS.
func TestAccResource_ProAccount_BaseUpdate(t *testing.T) {
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccountDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: baseAccountConfig(suffix, "TF Acc Base", "Enabled", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_account.test", "id"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "full_name", "TF Acc Base"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "access_status", "Enabled"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "privilege_set", "Auditor"),
				),
			},
			{
				// In-place base-field update (no password change ⇒ Pro PUT without
				// re-sending the password).
				Config: baseAccountConfig(suffix, "TF Acc Base Renamed", "Disabled", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "full_name", "TF Acc Base Renamed"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "access_status", "Disabled"),
				),
			},
			{
				// Password rotation: bump password_wo_version ⇒ password re-sent on
				// the Pro PUT.
				Config: baseAccountConfig(suffix, "TF Acc Base Renamed", "Disabled", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "password_wo_version", "2"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account.test", "full_name", "TF Acc Base Renamed"),
				),
			},
		},
	})
}
