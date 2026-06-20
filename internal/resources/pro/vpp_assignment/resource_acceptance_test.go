// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests talk to the Jamf ProClassic /vppassignments endpoint (user-based VPP
// content assignment). Keep serial with other classic acceptance work in this
// domain.
//
// Writes are a MERGE; content collections are opt-out (null retains, [] clears,
// populated replaces) and scope is always-emitted (full-replace, empty=clear).
// The update steps mutate the name, grow/shrink a scope group set, and toggle
// all_jss_users to exercise the always-emit clear path.
//
// Apply tests provision their own VPP account via a jamfplatform_pro_volume_-
// purchasing_location fixture, so they are gated on JAMFPLATFORM_VPP_TOKEN (a real
// ABM/ASM .vpptoken — same gate as the location + invitation VPP tests). The
// location's id is the VPP account id the assignment references. Token material
// MUST come from env — never commit it.
//
// Assigning actual content requires the fixture account to OWN the adam_id, which
// varies by token. A content step is gated behind JAMFPLATFORM_ACC_VPP_ADAM_ID
// (an owned iOS-app adam_id); skipped when unset. There is NO ebook acc step (the
// tenant has no book-owning VPP account fixture).
//
// Directory-service-group tests stand up the shared Okta LDAP server fixture via
// the SDK (so the directory exists before the plan-time scope preflight) and use
// JAMFPLATFORM_ACC_LDAP_GROUP_NAME for the real group name.

package vpp_assignment_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const resAddr = "jamfplatform_pro_vpp_assignment.test"

// vppTokenEnvVar holds the base64 `.vpptoken` contents used to stand up a VPP
// location fixture (which owns the VPP account the assignment references).
const vppTokenEnvVar = "JAMFPLATFORM_VPP_TOKEN"

// adamIDEnvVar names an iOS-app adam_id the fixture account owns (content step).
const adamIDEnvVar = "JAMFPLATFORM_ACC_VPP_ADAM_ID"

func vppToken(t *testing.T) string {
	v := os.Getenv(vppTokenEnvVar)
	if v == "" {
		t.Skipf("%s not set; skipping VPP assignment acceptance test (needs a VPP location fixture)", vppTokenEnvVar)
	}
	return v
}

func testAccCheckVPPAssignmentDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_vpp_assignment" {
				continue
			}
			_, err := c.GetVPPAssignmentByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking VPP assignment %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("VPP assignment %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// vppLocationFixture stands up a VPP location from the token; its id is the VPP
// account the assignment distributes content from.
func vppLocationFixture(token, suffix string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_volume_purchasing_location" "vpp" {
  name                                     = "tf-acc-vpp-loc-%[2]s"
  service_token                            = %[1]q
  service_token_wo_version                 = 1
  automatically_populate_purchased_content = true
}
`, token, suffix)
}

// lifecycleConfig builds an assignment plus the location fixture and two static
// user-group fixtures so scope targets reference real IDs. groupCount selects how
// many target groups are in scope (1 or 2) to exercise add/remove; allUsers
// toggles the all_jss_users flag (mutually exclusive with target groups, so it is
// only used with groupCount=0).
func lifecycleConfig(token, suffix, name string, groupCount int, allUsers bool) string {
	scopeBlock := ""
	switch {
	case allUsers:
		scopeBlock = `
  scope = {
    targets = {
      all_jss_users = true
    }
  }`
	case groupCount == 2:
		scopeBlock = `
  scope = {
    targets = {
      jss_user_group_ids = [jamfplatform_pro_user_group.a.id, jamfplatform_pro_user_group.b.id]
    }
  }`
	default:
		scopeBlock = `
  scope = {
    targets = {
      jss_user_group_ids = [jamfplatform_pro_user_group.a.id]
    }
  }`
	}
	return vppLocationFixture(token, suffix) + fmt.Sprintf(`
resource "jamfplatform_pro_user_group" "a" {
  name       = "%[1]s-grp-a"
  group_type = "static"
}

resource "jamfplatform_pro_user_group" "b" {
  name       = "%[1]s-grp-b"
  group_type = "static"
}

resource "jamfplatform_pro_vpp_assignment" "test" {
  name                 = %[1]q
  vpp_admin_account_id = jamfplatform_pro_volume_purchasing_location.vpp.id
%[2]s
}
`, name, scopeBlock)
}

func TestAccResource_ProVPPAssignment(t *testing.T) {
	token := vppToken(t)
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-vppa-" + suffix
	renamed := "tf-acc-vppa-renamed-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVPPAssignmentDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: lifecycleConfig(token, suffix, name, 1, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resAddr, "id"),
					resource.TestCheckResourceAttr(resAddr, "name", name),
					resource.TestCheckResourceAttrPair(resAddr, "vpp_admin_account_id", "jamfplatform_pro_volume_purchasing_location.vpp", "id"),
					resource.TestCheckResourceAttrSet(resAddr, "vpp_admin_account_name"),
					resource.TestCheckResourceAttr(resAddr, "scope.targets.jss_user_group_ids.#", "1"),
				),
			},
			{
				// Merge update: rename + add the second target group (nested-set growth).
				Config: lifecycleConfig(token, suffix, renamed, 2, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resAddr, "name", renamed),
					resource.TestCheckResourceAttr(resAddr, "scope.targets.jss_user_group_ids.#", "2"),
				),
			},
			{
				// Shrink the target set back to one (nested-set removal → always-emit clears).
				Config: lifecycleConfig(token, suffix, renamed, 1, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resAddr, "scope.targets.jss_user_group_ids.#", "1"),
				),
			},
			{
				// Toggle all_jss_users on. The target groups clear via always-emit;
				// resource.Test runs a post-step plan that fails on any residual
				// diff, so the "groups cleared" invariant is enforced implicitly —
				// no explicit count assertion (a cleared id set flattens to a null
				// Set, whose "#" key is unreliable; mirrors vpp_invitation).
				Config: lifecycleConfig(token, suffix, renamed, 0, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resAddr, "scope.targets.all_jss_users", "true"),
				),
			},
			{
				ResourceName:      resAddr,
				ImportState:       true,
				ImportStateVerify: true,
				// Import refreshes only general — scope and the content sets are not
				// reconstructed (state nil), so ignore them on verify.
				ImportStateVerifyIgnore: []string{"timeouts", "scope", "ios_app_adam_ids", "mac_app_adam_ids", "ebook_adam_ids"},
			},
		},
	})
}

// contentConfig assigns a single iOS app by adam_id (owned by the fixture
// account) and then clears it ([]).
func contentConfig(token, suffix, name, adamID string, assign bool) string {
	ios := "[]"
	if assign {
		ios = "[" + adamID + "]"
	}
	return vppLocationFixture(token, suffix) + fmt.Sprintf(`
resource "jamfplatform_pro_vpp_assignment" "test" {
  name                 = %[1]q
  vpp_admin_account_id = jamfplatform_pro_volume_purchasing_location.vpp.id
  ios_app_adam_ids     = %[2]s

  scope = {
    targets = {
      all_jss_users = true
    }
  }
}
`, name, ios)
}

// TestAccResource_ProVPPAssignment_Content exercises the content opt-out path
// (assign then clear an iOS app). Requires an owned adam_id.
func TestAccResource_ProVPPAssignment_Content(t *testing.T) {
	token := vppToken(t)
	adamID := os.Getenv(adamIDEnvVar)
	if adamID == "" {
		t.Skipf("%s not set; skipping VPP assignment content test", adamIDEnvVar)
	}
	if _, err := strconv.ParseInt(adamID, 10, 64); err != nil {
		t.Fatalf("%s must be a numeric adam_id, got %q", adamIDEnvVar, adamID)
	}
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-vppa-content-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVPPAssignmentDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: contentConfig(token, suffix, name, adamID, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resAddr, "ios_app_adam_ids.#", "1"),
					resource.TestCheckTypeSetElemAttr(resAddr, "ios_app_adam_ids.*", adamID),
				),
			},
			{
				// Clear ([]) — opt-out empty emits a clearing <ios_apps/> element.
				Config: contentConfig(token, suffix, name, adamID, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resAddr, "ios_app_adam_ids.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSource_ProVPPAssignment_BySelectors(t *testing.T) {
	token := vppToken(t)
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-vppa-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVPPAssignmentDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: lifecycleConfig(token, suffix, name, 1, false) + `
data "jamfplatform_pro_vpp_assignment" "by_id" {
  id = jamfplatform_pro_vpp_assignment.test.id
}
data "jamfplatform_pro_vpp_assignment" "by_name" {
  name = jamfplatform_pro_vpp_assignment.test.name
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_vpp_assignment.by_id", "name", resAddr, "name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_vpp_assignment.by_name", "id", resAddr, "id"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_vpp_assignment.by_id", "vpp_admin_account_name"),
				),
			},
		},
	})
}

func TestAccResource_ProVPPAssignment_DriftRecovery(t *testing.T) {
	token := vppToken(t)
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-vppa-drift-" + suffix
	cfg := lifecycleConfig(token, suffix, name, 1, false)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVPPAssignmentDestroy(t),
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.TestCheckResourceAttrSet(resAddr, "id")},
			{
				PreConfig: func() {
					c := proclassic.New(testhelpers.NewAcceptanceClient(t))
					ctx := context.Background()
					listed, err := c.ListVPPAssignments(ctx)
					if err != nil {
						t.Fatalf("drift preconfig list: %v", err)
					}
					for _, item := range listed.VppAssignments {
						if item.Name != nil && *item.Name == name && item.ID != nil {
							if err := c.DeleteVPPAssignmentByID(ctx, helpers.StringValueFromIntPtr(item.ID).ValueString()); err != nil {
								t.Fatalf("drift preconfig delete: %v", err)
							}
						}
					}
				},
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resAddr, "id"),
					resource.TestCheckResourceAttr(resAddr, "name", name),
				),
			},
		},
	})
}

// TestAccResource_ProVPPAssignment_DSGroupScope exercises a real directory-service
// group in both limitations and exclusions. Requires both a VPP token (for the
// account fixture) and a real LDAP group name.
func TestAccResource_ProVPPAssignment_DSGroupScope(t *testing.T) {
	token := vppToken(t)
	ldapEnv := testhelpers.RequireOktaLdapEnv(t)
	group := testhelpers.RequireLdapGroupName(t)
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-vppa-dsscope-" + suffix
	// Plan-time scope preflight resolves the group, so the directory must exist
	// before plan — pre-create it via the SDK (not an in-config fixture).
	testhelpers.EnsureLdapServerFixture(t, name, ldapEnv)
	// Wait until the fresh fixture's bind is up so the plan-time scope preflight
	// resolves the group instead of failing "not found".
	testhelpers.WaitForLdapGroupResolvable(t, group)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVPPAssignmentDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: vppLocationFixture(token, suffix) + fmt.Sprintf(`
resource "jamfplatform_pro_vpp_assignment" "test" {
  name                 = %[1]q
  vpp_admin_account_id = jamfplatform_pro_volume_purchasing_location.vpp.id

  scope = {
    limitations = {
      directory_service_user_group_names = [%[2]q]
    }
    exclusions = {
      directory_service_user_group_names = [%[2]q]
    }
  }
}
`, name, group),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resAddr, "scope.limitations.directory_service_user_group_names.#", "1"),
					resource.TestCheckResourceAttr(resAddr, "scope.exclusions.directory_service_user_group_names.#", "1"),
				),
			},
		},
	})
}

// ---- ExpectError: plan-time validators (no account / token needed) -------------

// litAccount is a placeholder vpp_admin_account_id for validation-only tests that
// fail at plan time before any API call — they never apply, so the id need not exist.
const litAccount = "1"

func TestAccResource_ProVPPAssignment_AllJSSUsersConflict(t *testing.T) {
	testhelpers.AccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "jamfplatform_pro_user_group" "a" {
  name       = "tf-acc-vppa-conflict-grp"
  group_type = "static"
}

resource "jamfplatform_pro_vpp_assignment" "test" {
  name                 = "tf-acc-vppa-conflict"
  vpp_admin_account_id = %q

  scope = {
    targets = {
      all_jss_users      = true
      jss_user_group_ids = [jamfplatform_pro_user_group.a.id]
    }
  }
}
`, litAccount),
				ExpectError: regexp.MustCompile(`all-flag`),
			},
		},
	})
}

// TestAccResource_ProVPPAssignment_DSGroupNotFound exercises the directory-service
// preflight (a plan-time check; no apply, so no account fixture needed). Pre-creates
// the LDAP directory so the preflight errors (group genuinely not found) rather than
// warning (directory unreachable).
func TestAccResource_ProVPPAssignment_DSGroupNotFound(t *testing.T) {
	testhelpers.AccPreCheck(t)
	ldapEnv := testhelpers.RequireOktaLdapEnv(t)
	testhelpers.EnsureLdapServerFixture(t, "tf-acc-vppa-dsbad", ldapEnv)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "jamfplatform_pro_vpp_assignment" "test" {
  name                 = "tf-acc-vppa-dsbad"
  vpp_admin_account_id = %q

  scope = {
    exclusions = {
      directory_service_user_group_names = ["tf-acc-no-such-ldap-group-zzz"]
    }
  }
}
`, litAccount),
				ExpectError: regexp.MustCompile(`Directory-service group not found`),
			},
		},
	})
}

func TestAccDataSource_ProVPPAssignment_AmbiguousSelector(t *testing.T) {
	testhelpers.AccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "jamfplatform_pro_vpp_assignment" "bad" {
  id   = "1"
  name = "x"
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}
