// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /accounts/groupid endpoint and
// the Pro v1 /account-groups endpoint. Classic has known concurrency issues when
// multiple writes hit the same resource type — keep these serial.

package account_group_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func testAccCheckAccountGroupDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_account_group" {
				continue
			}
			_, err := c.GetAccountGroupByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking account group %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("account group %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// customGroupConfig builds a Custom-privilege-set group with the given
// jamf_pro_server_objects privileges.
func customGroupConfig(displayName string, objects ...string) string {
	quoted := ""
	for i, p := range objects {
		if i > 0 {
			quoted += ", "
		}
		quoted += fmt.Sprintf("%q", p)
	}
	return fmt.Sprintf(`
resource "jamfplatform_pro_account_group" "test" {
  display_name  = %q
  access_level  = "Full Access"
  privilege_set = "Custom"
  privileges = {
    jamf_pro_server_objects = [%s]
  }
}
`, displayName, quoted)
}

// TestAccResource_ProAccountGroup exercises the full lifecycle plus the privilege
// intersect-on-read behaviour: a privilege set is grown then shrunk, and state
// must reflect exactly the declared set each time (proving server-added
// dependency privileges are reconciled out and removals are honoured).
func TestAccResource_ProAccountGroup(t *testing.T) {
	name := fmt.Sprintf("tf-acc-account-group-%d", os.Getpid())
	renamed := name + "-renamed"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccountGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				// Create with two privileges.
				Config: customGroupConfig(name, "Read Computers", "Update Computers"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_account_group.test", "id"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account_group.test", "display_name", name),
					resource.TestCheckResourceAttr("jamfplatform_pro_account_group.test", "privilege_set", "Custom"),
					resource.TestCheckResourceAttr("jamfplatform_pro_account_group.test", "privileges.jamf_pro_server_objects.#", "2"),
					resource.TestCheckTypeSetElemAttr("jamfplatform_pro_account_group.test", "privileges.jamf_pro_server_objects.*", "Read Computers"),
					resource.TestCheckTypeSetElemAttr("jamfplatform_pro_account_group.test", "privileges.jamf_pro_server_objects.*", "Update Computers"),
				),
			},
			{
				// Shrink privileges (remove Update Computers) and rename. State
				// must show exactly the declared single privilege — removals work
				// and server-added dependencies do not leak in.
				Config: customGroupConfig(renamed, "Read Computers"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_account_group.test", "display_name", renamed),
					resource.TestCheckResourceAttr("jamfplatform_pro_account_group.test", "privileges.jamf_pro_server_objects.#", "1"),
					resource.TestCheckTypeSetElemAttr("jamfplatform_pro_account_group.test", "privileges.jamf_pro_server_objects.*", "Read Computers"),
				),
			},
			{
				// Import smoke. ImportStateVerify is intentionally omitted: import
				// faithfully materialises the full server privilege grid (a
				// superset of the declared subset), so a full-fidelity compare
				// against the prior managed state would not match by design.
				ResourceName:      "jamfplatform_pro_account_group.test",
				ImportState:       true,
				ImportStateVerify: false,
			},
		},
	})
}

// TestAccResource_ProAccountGroup_NonCustom verifies a preset privilege set
// (Auditor) with no privileges block, and the Pro v1 data source read.
func TestAccResource_ProAccountGroup_NonCustom(t *testing.T) {
	name := fmt.Sprintf("tf-acc-account-group-auditor-%d", os.Getpid())
	config := fmt.Sprintf(`
resource "jamfplatform_pro_account_group" "auditor" {
  display_name  = %q
  access_level  = "Full Access"
  privilege_set = "Auditor"
}

data "jamfplatform_pro_account_group" "by_name" {
  display_name = jamfplatform_pro_account_group.auditor.display_name
}
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccountGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_account_group.auditor", "privilege_set", "Auditor"),
					// DS is classic-sourced (same spellings as the resource).
					resource.TestCheckResourceAttr("data.jamfplatform_pro_account_group.by_name", "privilege_set", "Auditor"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_account_group.by_name", "access_level", "Full Access"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_account_group.by_name", "id", "jamfplatform_pro_account_group.auditor", "id"),
				),
			},
		},
	})
}

// TestAccResource_ProAccountGroup_Members exercises membership add and remove.
// Gated on JAMFPLATFORM_ACC_ACCOUNT_MEMBER_ID (an existing account ID) since it
// requires a real member to reference.
func TestAccResource_ProAccountGroup_Members(t *testing.T) {
	memberID := os.Getenv("JAMFPLATFORM_ACC_ACCOUNT_MEMBER_ID")
	if memberID == "" {
		t.Skip("set JAMFPLATFORM_ACC_ACCOUNT_MEMBER_ID to an existing Jamf Pro account ID to run the membership test")
	}
	name := fmt.Sprintf("tf-acc-account-group-members-%d", os.Getpid())

	withMember := fmt.Sprintf(`
resource "jamfplatform_pro_account_group" "members" {
  display_name  = %q
  access_level  = "Full Access"
  privilege_set = "Auditor"
  members       = [%s]
}
`, name, memberID)
	withoutMember := fmt.Sprintf(`
resource "jamfplatform_pro_account_group" "members" {
  display_name  = %q
  access_level  = "Full Access"
  privilege_set = "Auditor"
  members       = []
}
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccountGroupDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: withMember,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_account_group.members", "members.#", "1"),
					resource.TestCheckTypeSetElemAttr("jamfplatform_pro_account_group.members", "members.*", memberID),
				),
			},
			{
				Config: withoutMember,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_account_group.members", "members.#", "0"),
				),
			},
		},
	})
}
