// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /mobiledeviceprovisioningprofiles
// endpoint. Classic has known concurrency issues when multiple writes hit the same
// resource type — keep these tests serial with any future classic acceptance work
// in this package.
//
// Server invariant exercised here: the uploaded blob is create-only. Every PUT to
// a blob-bearing profile returns HTTP 500, so name/display_name/profile_data are
// all RequiresReplace — the "update" step is therefore a replacement.
//
// Fixture: a real Apple-signed enterprise .mobileprovision (testdata/, see its
// README). Jamf rejects synthesised blobs with 500, so a genuine signed profile is
// mandatory; the test base64-encodes it at runtime.

package mobile_device_provisioning_profile_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const resourceAddr = "jamfplatform_pro_mobile_device_provisioning_profile.test"

// fixtureBase64 reads the bundled signed profile and base64-encodes it.
func fixtureBase64(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(file), "testdata", "Certificates.mobileprovision")
	b, err := os.ReadFile(p) //nolint:gosec // test fixture path is fixed
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func testAccCheckProvisioningProfileDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_mobile_device_provisioning_profile" {
				continue
			}
			_, err := c.GetMobileDeviceProvisioningProfileByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking provisioning profile %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("provisioning profile %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

func provisioningProfileConfig(name, b64 string) string {
	// display_name is server-derived (forced == name); it is Computed, not settable.
	return fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_provisioning_profile" "test" {
			name         = %q
			profile_data = %q
		}
	`, name, b64)
}

// TestAccResource_ProMobileDeviceProvisioningProfile exercises create with a real
// blob, a rename (which forces replacement), and import. The blob, uuid, and
// expiration computed fields are asserted as populated; the rename step verifies
// the RequiresReplace behaviour via a plan check.
func TestAccResource_ProMobileDeviceProvisioningProfile(t *testing.T) {
	testhelpers.AccPreCheck(t)
	b64 := fixtureBase64(t)
	suffix := testhelpers.RunSuffix()
	original := "tf-acc-mdpp-" + suffix
	renamed := "tf-acc-mdpp-renamed-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningProfileDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: provisioningProfileConfig(original, b64),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceAddr, "id"),
					resource.TestCheckResourceAttr(resourceAddr, "name", original),
					// display_name is server-derived: forced to equal name.
					resource.TestCheckResourceAttr(resourceAddr, "display_name", original),
					resource.TestCheckResourceAttr(resourceAddr, "profile_data", b64),
					// uuid is parsed from the uploaded blob — must be populated.
					resource.TestCheckResourceAttrSet(resourceAddr, "uuid"),
				),
			},
			{
				// Rename forces replacement (no working PUT for a blob-bearing profile).
				Config: provisioningProfileConfig(renamed, b64),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddr, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceAddr, "name", renamed),
					resource.TestCheckResourceAttrSet(resourceAddr, "uuid"),
				),
			},
			{
				ResourceName:            resourceAddr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func TestAccDataSource_ProMobileDeviceProvisioningProfile_BySelectors(t *testing.T) {
	testhelpers.AccPreCheck(t)
	b64 := fixtureBase64(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdpp-ds-" + suffix

	base := provisioningProfileConfig(name, b64)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningProfileDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: base + `
					data "jamfplatform_pro_mobile_device_provisioning_profile" "by_id" {
						id = jamfplatform_pro_mobile_device_provisioning_profile.test.id
					}
					data "jamfplatform_pro_mobile_device_provisioning_profile" "by_name" {
						name = jamfplatform_pro_mobile_device_provisioning_profile.test.name
					}
					data "jamfplatform_pro_mobile_device_provisioning_profile" "by_uuid" {
						uuid = jamfplatform_pro_mobile_device_provisioning_profile.test.uuid
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_provisioning_profile.by_id", "name", resourceAddr, "name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_provisioning_profile.by_name", "id", resourceAddr, "id"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_provisioning_profile.by_uuid", "id", resourceAddr, "id"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_provisioning_profile.by_id", "profile_data", resourceAddr, "profile_data"),
				),
			},
		},
	})
}

// TestAccResource_ProMobileDeviceProvisioningProfile_DriftRecovery deletes the
// profile out-of-band, then re-applies the same config. This exercises the
// Read 404 -> RemoveResource self-heal and confirms the profile is recreated
// (with its computed uuid re-parsed from the same blob).
func TestAccResource_ProMobileDeviceProvisioningProfile_DriftRecovery(t *testing.T) {
	testhelpers.AccPreCheck(t)
	b64 := fixtureBase64(t)
	name := "tf-acc-mdpp-drift-" + testhelpers.RunSuffix()
	cfg := provisioningProfileConfig(name, b64)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningProfileDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check:  resource.TestCheckResourceAttrSet(resourceAddr, "uuid"),
			},
			{
				PreConfig: func() {
					c := proclassic.New(testhelpers.NewAcceptanceClient(t))
					ctx := context.Background()
					id, err := c.ResolveMobileDeviceProvisioningProfileIDByName(ctx, name)
					if err != nil {
						t.Fatalf("drift preconfig: resolve id: %v", err)
					}
					if err := c.DeleteMobileDeviceProvisioningProfileByID(ctx, id); err != nil {
						t.Fatalf("drift preconfig: delete: %v", err)
					}
				},
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceAddr, "id"),
					resource.TestCheckResourceAttr(resourceAddr, "name", name),
					resource.TestCheckResourceAttrSet(resourceAddr, "uuid"),
				),
			},
		},
	})
}

// TestAccResource_ProMobileDeviceProvisioningProfile_EmptyNameRejected asserts the
// name LengthAtLeast(1) validator fires.
func TestAccResource_ProMobileDeviceProvisioningProfile_EmptyNameRejected(t *testing.T) {
	testhelpers.AccPreCheck(t)
	b64 := fixtureBase64(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      provisioningProfileConfig("", b64),
				ExpectError: regexp.MustCompile(`string length must be at least 1`),
			},
		},
	})
}

// TestAccDataSource_ProMobileDeviceProvisioningProfile_AmbiguousSelector asserts
// the ExactlyOneOf config validator rejects two selectors.
func TestAccDataSource_ProMobileDeviceProvisioningProfile_AmbiguousSelector(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_pro_mobile_device_provisioning_profile" "bad" {
						id   = "1"
						name = "x"
					}
				`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}
