// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file create real Jamf Pro administrator accounts via the Pro API
// and write the privilege grid via the classic API. Base-field updates route
// through Pro PUT, which the platform gateway currently rejects (403); that path
// is covered by a separately-skipped test until the permission lands.

package account_test

import (
	"context"
	"fmt"
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

func customAccountConfig(suffix string, objects ...string) string {
	quoted := ""
	for i, p := range objects {
		if i > 0 {
			quoted += ", "
		}
		quoted += fmt.Sprintf("%q", p)
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
`, suffix, quoted)
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
				ResourceName:      "jamfplatform_pro_account.test",
				ImportState:       true,
				ImportStateVerify: false,
			},
		},
	})
}

// TestAccResource_ProAccount_BaseUpdate exercises an in-place base-field update,
// which routes through Pro PUT. This is currently rejected by the platform
// gateway (403 BAD_PERMISSIONS); un-skip when that permission is granted.
func TestAccResource_ProAccount_BaseUpdate(t *testing.T) {
	t.Skip("base-field updates route through Pro PUT, which the platform gateway currently rejects with 403; un-skip when the Pro update permission lands")
}
