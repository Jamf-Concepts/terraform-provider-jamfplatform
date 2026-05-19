// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package script_test

import (
	"context"
	"fmt"
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

// testAccCheckScriptDestroy verifies scripts created during the test were destroyed.
func testAccCheckScriptDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_script" {
				continue
			}
			_, err := c.GetScriptV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro script %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro script %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

func TestAccResource_ProScript_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-script-" + suffix
	nameUpdated := "tf-acc-pro-script-updated-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScriptDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_script" "test" {
						name            = %q
						priority        = "AFTER"
						info            = "tf-acc info"
						notes           = "tf-acc notes"
						os_requirements = "13.0.x,14.0.x"
						parameter_4     = "param4-label"
						script_contents = "#!/bin/sh\necho hello"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_script.test", "id"),
					resource.TestCheckResourceAttr("jamfplatform_pro_script.test", "name", name),
					resource.TestCheckResourceAttr("jamfplatform_pro_script.test", "priority", "AFTER"),
					resource.TestCheckResourceAttr("jamfplatform_pro_script.test", "parameter_4", "param4-label"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_script" "test" {
						name            = %q
						priority        = "BEFORE"
						script_contents = "#!/bin/sh\necho updated"
					}
				`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jamfplatform_pro_script.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("jamfplatform_pro_script.test", "priority", "BEFORE"),
				),
			},
			{
				ResourceName:      "jamfplatform_pro_script.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
				},
			},
		},
	})
}

func TestAccDataSource_ProScript_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-script-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScriptDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_script" "src" {
						name            = %q
						priority        = "AFTER"
						script_contents = "echo ds-test"
					}

					data "jamfplatform_pro_script" "lookup" {
						id = jamfplatform_pro_script.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_script.lookup", "name", "jamfplatform_pro_script.src", "name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_script.lookup", "priority", "jamfplatform_pro_script.src", "priority"),
				),
			},
		},
	})
}

// TestAccListResource_ProScript_Basic exercises the jamfplatform_pro_script list
// resource via the `terraform query` workflow.
func TestAccListResource_ProScript_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-script-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScriptDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_script" "src" {
						name            = %q
						priority        = "AFTER"
						script_contents = "echo list-test"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_script.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_script" "test" {
						provider         = jamfplatform
						include_resource = true

						config {
							filter = [
								{
									selector = "name"
									argument = %q
								}
							]
						}
					}
				`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("jamfplatform_pro_script.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_script.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
							{Path: tfjsonpath.New("priority"), KnownValue: knownvalue.StringExact("AFTER")},
						},
					),
				},
			},
		},
	})
}

func TestAccDataSource_ProScripts_FilterByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-scripts-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScriptDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_script" "src" {
						name            = %q
						priority        = "AFTER"
						script_contents = "echo plural-test"
					}

					data "jamfplatform_pro_scripts" "lookup" {
						filter = [
							{
								selector = "name"
								argument = jamfplatform_pro_script.src.name
							}
						]
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.jamfplatform_pro_scripts.lookup", "scripts.#", "1"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_scripts.lookup", "scripts.0.name", name),
				),
			},
		},
	})
}
