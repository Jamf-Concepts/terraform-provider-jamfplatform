// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /mobiledeviceenrollmentprofiles
// endpoint. Classic has known concurrency issues when multiple writes hit the same
// resource type — keep these tests serial with any future classic acceptance work
// in this package.
//
// Writes are a MERGE (omit=retain, empty=clear); the update step mutates scalars,
// clears a field, and moves the site to exercise that path. Attachments are
// read-only (the upload endpoint rejects bearer auth for this resource) so no
// attachment write is exercised. location/purchasing are not populated on import
// (they refresh only when authored) → ImportStateVerifyIgnore covers them.

package mobile_device_enrollment_profile_test

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

const resAddr = "jamfplatform_pro_mobile_device_enrollment_profile.test"

func testAccCheckEnrollmentProfileDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_mobile_device_enrollment_profile" {
				continue
			}
			_, err := c.GetMobileDeviceEnrollmentProfileByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking enrollment profile %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("enrollment profile %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// config builds a profile that references a managed site (tenant-agnostic).
// An empty room omits the line entirely (clear-by-omission — the real Terraform
// path); a merge write then clears the field on the server and state reconciles
// to null. (Setting room = "" explicitly would be a known plan value the server
// drops to null, which is an inconsistency — users omit to clear.)
func config(name, desc, room string) string {
	roomLine := ""
	if room != "" {
		roomLine = fmt.Sprintf("    room      = %q\n", room)
	}
	return fmt.Sprintf(`
resource "jamfplatform_pro_site" "s" {
  name = "%s-site"
}

resource "jamfplatform_pro_mobile_device_enrollment_profile" "test" {
  name        = %q
  description = %q
  site_id     = jamfplatform_pro_site.s.id

  location = {
    username  = "alice"
    real_name = "Alice A"
%s  }

  purchasing = {
    is_purchased = true
    vendor       = "Acme"
  }
}
`, name, name, desc, roomLine)
}

func TestAccResource_ProMobileDeviceEnrollmentProfile(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdep-" + suffix
	renamed := "tf-acc-mdep-renamed-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnrollmentProfileDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config(name, "first", "R1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resAddr, "id"),
					resource.TestCheckResourceAttr(resAddr, "name", name),
					resource.TestCheckResourceAttr(resAddr, "description", "first"),
					resource.TestCheckResourceAttrSet(resAddr, "invitation"),
					resource.TestCheckResourceAttrSet(resAddr, "uuid"),
					resource.TestCheckResourceAttrSet(resAddr, "site_id"),
					resource.TestCheckResourceAttrSet(resAddr, "site_name"),
					resource.TestCheckResourceAttr(resAddr, "location.username", "alice"),
					resource.TestCheckResourceAttr(resAddr, "location.room", "R1"),
					resource.TestCheckResourceAttr(resAddr, "purchasing.vendor", "Acme"),
					resource.TestCheckResourceAttr(resAddr, "purchasing.is_purchased", "true"),
				),
			},
			{
				// Merge update: rename, change description, clear room (empty=clear), keep invitation/uuid stable.
				Config: config(renamed, "second", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resAddr, "name", renamed),
					resource.TestCheckResourceAttr(resAddr, "description", "second"),
					resource.TestCheckNoResourceAttr(resAddr, "location.room"),
					resource.TestCheckResourceAttr(resAddr, "location.username", "alice"),
				),
			},
			{
				ResourceName:            resAddr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "location", "purchasing"},
			},
		},
	})
}

func TestAccDataSource_ProMobileDeviceEnrollmentProfile_BySelectors(t *testing.T) {
	testhelpers.AccPreCheck(t)
	name := "tf-acc-mdep-ds-" + testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnrollmentProfileDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config(name, "ds", "R1") + `
data "jamfplatform_pro_mobile_device_enrollment_profile" "by_id" {
  id = jamfplatform_pro_mobile_device_enrollment_profile.test.id
}
data "jamfplatform_pro_mobile_device_enrollment_profile" "by_name" {
  name = jamfplatform_pro_mobile_device_enrollment_profile.test.name
}
data "jamfplatform_pro_mobile_device_enrollment_profile" "by_invitation" {
  invitation = jamfplatform_pro_mobile_device_enrollment_profile.test.invitation
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_enrollment_profile.by_id", "name", resAddr, "name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_enrollment_profile.by_name", "id", resAddr, "id"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_enrollment_profile.by_invitation", "id", resAddr, "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mobile_device_enrollment_profile.by_id", "location.username", "alice"),
				),
			},
		},
	})
}

func TestAccResource_ProMobileDeviceEnrollmentProfile_DriftRecovery(t *testing.T) {
	testhelpers.AccPreCheck(t)
	name := "tf-acc-mdep-drift-" + testhelpers.RunSuffix()
	cfg := config(name, "drift", "R1")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnrollmentProfileDestroy(t),
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.TestCheckResourceAttrSet(resAddr, "invitation")},
			{
				PreConfig: func() {
					c := proclassic.New(testhelpers.NewAcceptanceClient(t))
					ctx := context.Background()
					id, err := c.ResolveMobileDeviceEnrollmentProfileIDByName(ctx, name)
					if err != nil {
						t.Fatalf("drift preconfig resolve: %v", err)
					}
					if err := c.DeleteMobileDeviceEnrollmentProfileByID(ctx, id); err != nil {
						t.Fatalf("drift preconfig delete: %v", err)
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

func TestAccResource_ProMobileDeviceEnrollmentProfile_EmptyNameRejected(t *testing.T) {
	testhelpers.AccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "jamfplatform_pro_mobile_device_enrollment_profile" "test" {
  name = ""
}
`,
				ExpectError: regexp.MustCompile(`string length must be at least 1`),
			},
		},
	})
}

func TestAccDataSource_ProMobileDeviceEnrollmentProfile_AmbiguousSelector(t *testing.T) {
	testhelpers.AccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "jamfplatform_pro_mobile_device_enrollment_profile" "bad" {
  id   = "1"
  name = "x"
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}
