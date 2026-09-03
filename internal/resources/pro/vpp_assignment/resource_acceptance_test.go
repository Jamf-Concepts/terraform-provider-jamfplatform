// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests talk to the Jamf ProClassic /vppassignments endpoint (user-based VPP
// content assignment). Keep serial with other classic acceptance work in this
// domain.
//
// Writes are a MERGE; content collections are opt-out (null retains, [] clears,
// populated replaces) and scope follows per-category granular ownership: a
// declared category (including explicit `[]`, which clears) is owned by
// Terraform, an omitted category is preserved via read-merge-write on update.
// The update steps mutate the name, grow/shrink a scope group set, and toggle
// all_jss_users to exercise the all-flag precedence (the flag wipes its
// conflicting target categories).
//
// Apply tests provision their own VPP account via a jamfplatform_pro_volume_-
// purchasing_location fixture, so they are gated on JAMFPLATFORM_ACC_PRO_VPP_TOKEN (a real
// ABM/ASM .vpptoken — same gate as the location + invitation VPP tests). The
// location's id is the VPP account id the assignment references. Token material
// MUST come from env — never commit it.
//
// Assigning actual content requires the fixture account to OWN the adam_id. The
// adam_id is read live from the location fixture's Computed `content` catalog
// (one row per owned adam_id) — Create polls until Apple's first sync completes,
// so `content` is populated by the time the assignment plans. We select the
// first iOS app (content_type "IOS_APP"), so the test follows whatever the
// token currently owns rather than a hand-maintained env var. There
// is NO ebook acc step (the tenant has no book-owning VPP account fixture).
//
// Directory-service-group tests stand up the shared Okta LDAP server fixture via
// the SDK (so the directory exists before the plan-time scope preflight) and use
// JAMFPLATFORM_ACC_PRO_LDAP_GROUP_NAME for the real group name.

package vpp_assignment_test

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

const resAddr = "jamfplatform_pro_vpp_assignment.test"

// vppTokenEnvVar holds the base64 `.vpptoken` contents used to stand up a VPP
// location fixture (which owns the VPP account the assignment references).
const vppTokenEnvVar = "JAMFPLATFORM_ACC_PRO_VPP_TOKEN"

func vppToken(t *testing.T) string {
	v := testhelpers.AccEnv(vppTokenEnvVar)
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
				// Shrink the target set back to one (nested-set removal — the declared
				// category is owned, so its members full-replace).
				Config: lifecycleConfig(token, suffix, renamed, 1, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resAddr, "scope.targets.jss_user_group_ids.#", "1"),
				),
			},
			{
				// Toggle all_jss_users on. The flag wipes the target group categories
				// (all-flag precedence — the merge mirrors it); the now-undeclared
				// jss_user_group_ids stays null in state (unmanaged), and resource.Test
				// runs a post-step plan that fails on any residual diff, so the
				// "groups cleared" invariant is enforced implicitly — no explicit
				// count assertion (a null Set has no reliable "#" key; mirrors
				// vpp_invitation).
				Config: lifecycleConfig(token, suffix, renamed, 0, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resAddr, "scope.targets.all_jss_users", "true"),
				),
			},
			{
				ResourceName:      resAddr,
				ImportState:       true,
				ImportStateVerify: true,
				// Import hydrates every scope category and content set; apply keeps
				// declared-only, so verify against this subset config must ignore them.
				ImportStateVerifyIgnore: []string{"timeouts", "scope", "ios_app_adam_ids", "mac_app_adam_ids", "ebook_adam_ids"},
			},
		},
	})
}

// contentConfig assigns a single iOS app then clears it ([]). The adam_id is
// selected live from the location fixture's Computed `content` catalog (first
// content_type=="IOS_APP" row — the Jamf VolumePurchasingContent contentType
// enum is IOS_APP/MAC_APP/BOOK/UNKNOWN), so it tracks whatever the fixture
// token currently owns instead of a hand-set env var. A resource
// ARGUMENT may be unknown at plan time (unlike count/for_each), so the location
// applies + syncs first and the adam_id resolves before the assignment applies.
func contentConfig(token, suffix, name string, assign bool) string {
	ios := "[]"
	if assign {
		ios = `[
    [for c in jamfplatform_pro_volume_purchasing_location.vpp.content :
      c.adam_id if c.content_type == "IOS_APP"
    ][0],
  ]`
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
// (assign then clear an iOS app). The adam_id is read live from the location
// fixture's owned content, so the only gate is JAMFPLATFORM_ACC_PRO_VPP_TOKEN (a token
// whose account owns at least one iOS app).
func TestAccResource_ProVPPAssignment_Content(t *testing.T) {
	token := vppToken(t)
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-vppa-content-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVPPAssignmentDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: contentConfig(token, suffix, name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Exactly one app assigned; its value is server-owned (derived
					// from the location catalog), so we assert the count, not a
					// hard-coded id.
					resource.TestCheckResourceAttr(resAddr, "ios_app_adam_ids.#", "1"),
				),
			},
			{
				// Clear ([]) — opt-out empty emits a clearing <ios_apps/> element.
				Config: contentConfig(token, suffix, name, false),
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
			{
				// Detach the DS group from both limitations and exclusions BEFORE
				// the framework destroys the assignment, via an empty set `[]` (the
				// natural "remove all" gesture). Destroying while a DS group is still
				// scoped can leave an orphaned scope->LDAP association that blocks the
				// LDAP server's deletion (a server-side data-integrity bug). `[]` is
				// the declared-clear gesture (plans as an empty set, emits an explicit
				// empty element — omission would preserve the group); the post-step
				// empty-plan check enforces the clear round-tripped.
				Config: vppLocationFixture(token, suffix) + fmt.Sprintf(`
resource "jamfplatform_pro_vpp_assignment" "test" {
  name                 = %[1]q
  vpp_admin_account_id = jamfplatform_pro_volume_purchasing_location.vpp.id

  scope = {
    limitations = {
      directory_service_user_group_names = []
    }
    exclusions = {
      directory_service_user_group_names = []
    }
  }
}
`, name),
				Check: resource.TestCheckResourceAttrSet(resAddr, "id"),
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

// NOTE: the directory-service-group preflight is intentionally advisory now (a
// no-match warns, never errors at plan) so a same-apply directory+scope bootstrap
// isn't blocked — see helpers.RetryOnDirectoryGroupMatchConflict and the
// criteria/scope unit tests. A wrong group name therefore surfaces at apply (after a
// bounded retry), not plan, so there is no plan-time "not found" acc test here.

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
