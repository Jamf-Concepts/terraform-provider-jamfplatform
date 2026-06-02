// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf Pro /v1/mobile-device-extension-attributes
// endpoint. Mobile EAs cannot run scripts; the discriminator covers POPUP and
// DIRECTORY_SERVICE_ATTRIBUTE_MAPPING only.

package mobile_device_extension_attribute_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const mdeaResource = "jamfplatform_pro_mobile_device_extension_attribute.test"

func testAccCheckMDEADestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_mobile_device_extension_attribute" {
				continue
			}
			_, err := c.GetMobileDeviceExtensionAttributeV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking mobile EA %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("mobile EA %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

func mdeaText(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_extension_attribute" "test" {
			name              = %q
			description       = "probe"
			data_type         = "STRING"
			input_type        = "TEXT"
			inventory_display = "GENERAL"
		}
	`, name)
}

func mdeaPopup(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_extension_attribute" "test" {
			name               = %q
			data_type          = "STRING"
			input_type         = "POPUP"
			inventory_display  = "EXTENSION_ATTRIBUTES"
			popup_menu_choices = ["One", "Two"]
		}
	`, name)
}

func mdeaDSAM(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_extension_attribute" "test" {
			name                        = %q
			data_type                   = "STRING"
			input_type                  = "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"
			inventory_display           = "USER_AND_LOCATION"
			directory_service_attribute = "mail"
		}
	`, name)
}

func TestAccResource_ProMobileDeviceExtensionAttribute_Lifecycle(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mdea-" + suffix
	renamed := "tf-acc-pro-mdea-renamed-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMDEADestroy(t),
		Steps: []resource.TestStep{
			{
				Config: mdeaText(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(mdeaResource, "id"),
					resource.TestCheckResourceAttr(mdeaResource, "input_type", "TEXT"),
					resource.TestCheckNoResourceAttr(mdeaResource, "popup_menu_choices.0"),
				),
			},
			{
				Config: mdeaPopup(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mdeaResource, "name", renamed),
					resource.TestCheckResourceAttr(mdeaResource, "input_type", "POPUP"),
					resource.TestCheckResourceAttr(mdeaResource, "popup_menu_choices.#", "2"),
				),
			},
			{
				ResourceName:            mdeaResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
			{
				Config: mdeaDSAM(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mdeaResource, "input_type", "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"),
					resource.TestCheckResourceAttr(mdeaResource, "directory_service_attribute", "mail"),
					resource.TestCheckNoResourceAttr(mdeaResource, "popup_menu_choices.0"),
				),
			},
		},
	})
}

func TestAccResource_ProMobileDeviceExtensionAttribute_ValidatorErrors(t *testing.T) {
	testhelpers.AccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// popup_menu_choices forbidden on TEXT.
				Config: `
					resource "jamfplatform_pro_mobile_device_extension_attribute" "test" {
						name               = "tf-acc-mdea-bad"
						data_type          = "STRING"
						input_type         = "TEXT"
						inventory_display  = "GENERAL"
						popup_menu_choices = ["a"]
					}
				`,
				ExpectError: regexp.MustCompile(`POPUP`),
			},
		},
	})
}

func TestAccDataSource_ProMobileDeviceExtensionAttribute_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mdea-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMDEADestroy(t),
		Steps: []resource.TestStep{
			{
				Config: mdeaText(name) + `
					data "jamfplatform_pro_mobile_device_extension_attribute" "by_name" {
						name = jamfplatform_pro_mobile_device_extension_attribute.test.name
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_extension_attribute.by_name", "id", mdeaResource, "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mobile_device_extension_attribute.by_name", "input_type", "TEXT"),
				),
			},
		},
	})
}

func TestAccListResource_ProMobileDeviceExtensionAttribute_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mdea-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMDEADestroy(t),
		Steps: []resource.TestStep{
			{
				Config: mdeaText(name),
				Check:  resource.TestCheckResourceAttrSet(mdeaResource, "id"),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_mobile_device_extension_attribute" "test" {
						provider         = jamfplatform
						include_resource = true

						config {
							filter = {
								name_substring = %q
							}
						}
					}
				`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("jamfplatform_pro_mobile_device_extension_attribute.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_mobile_device_extension_attribute.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
							{Path: tfjsonpath.New("input_type"), KnownValue: knownvalue.StringExact("TEXT")},
						},
					),
				},
			},
		},
	})
}
