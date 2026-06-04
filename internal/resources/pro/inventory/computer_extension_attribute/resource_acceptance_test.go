// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf Pro /v1/computer-extension-attributes
// endpoint. Load-bearing behaviours the acc run verifies (see the build spike):
//   - input-type discriminator validators (FIELD_REQUIRED / INVALID_CONTENT mirror)
//   - full-replace PUT + GET-after-write
//   - SCRIPT→POPUP transition auto-clears the orphaned script (step 4)
//   - empty-echo normalization (no perma-diff on a bare TEXT EA)
//   - script trailing-newline tolerance (import ignores `script`; server appends \n)
//   - manage_existing_data is write-only (import ignores it; never returned)

package computer_extension_attribute_test

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

const ceaResource = "jamfplatform_pro_computer_extension_attribute.test"

func testAccCheckCEADestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_computer_extension_attribute" {
				continue
			}
			_, err := c.GetComputerExtensionAttributeV1(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking computer EA %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("computer EA %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// Step 1: bare TEXT EA — exercises empty-echo normalization (no companion fields).
func ceaText(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_computer_extension_attribute" "test" {
			name              = %q
			description       = "probe text"
			data_type         = "STRING"
			input_type        = "TEXT"
			inventory_display = "GENERAL"
		}
	`, name)
}

// Step 2: → SCRIPT (disabled, manage_existing_data) — mutates most attributes.
func ceaScript(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_computer_extension_attribute" "test" {
			name                 = %q
			description          = "probe script"
			data_type            = "STRING"
			input_type           = "SCRIPT"
			inventory_display    = "HARDWARE"
			enabled              = false
			script               = "#!/bin/bash\necho \"<result>ok</result>\""
			manage_existing_data = "RETAIN"
		}
	`, name)
}

// Step 4: SCRIPT→POPUP transition — server auto-clears the orphaned script.
func ceaPopup(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_computer_extension_attribute" "test" {
			name               = %q
			data_type          = "STRING"
			input_type         = "POPUP"
			inventory_display  = "GENERAL"
			popup_menu_choices = ["Red", "Green", "Blue"]
		}
	`, name)
}

// Step 6: POPUP→DSAM transition with allow_multiple_values.
//
// A DIRECTORY_SERVICE_ATTRIBUTE_MAPPING extension attribute requires LDAP to be
// configured on the tenant (else Create 400s with "[INVALID_CONTENT] inputType:
// Input type can not be 'DIRECTORY_SERVICE_ATTRIBUTE_MAPPING' if LDAP is not
// configured"). It also reads user/location data from the directory service
// (inventory_display = USER_AND_LOCATION). So the config stands up two ordered
// fixtures the EA depends_on: a dummy LDAP server (no reachable host needed —
// the ldap_server resource does not verify connectivity), then the computer
// inventory collection setting that enables directory-service user/location
// collection. The inventory-settings Delete is state-only (singleton); the LDAP
// server is removed on teardown.
func ceaDSAM(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_ldap_server" "ea_fixture" {
			connection_settings = {
				display_name        = "tf-acc-cea-dsam-ldap"
				directory_service   = "Open Directory"
				hostname            = "ldap.acc-anon.example.com"
				port                = 389
				use_ssl             = false
				authentication_type = "none"
			}
		}

		resource "jamfplatform_pro_computer_inventory_collection_settings" "ea_fixture" {
			depends_on                                       = [jamfplatform_pro_ldap_server.ea_fixture]
			collect_user_and_location_from_directory_service = true
		}

		resource "jamfplatform_pro_computer_extension_attribute" "test" {
			depends_on                  = [jamfplatform_pro_computer_inventory_collection_settings.ea_fixture]
			name                        = %q
			data_type                   = "STRING"
			input_type                  = "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"
			inventory_display           = "USER_AND_LOCATION"
			directory_service_attribute = "mail"
			allow_multiple_values       = true
		}
	`, name)
}

func TestAccResource_ProComputerExtensionAttribute_Lifecycle(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-cea-" + suffix
	renamed := "tf-acc-pro-cea-renamed-" + suffix

	// script and manage_existing_data are excluded from ImportStateVerify:
	//   - script: Jamf appends a trailing newline; reconcileScript keeps the
	//     config value during apply, but import has no prior value so it takes
	//     the server's "…\n" verbatim — a benign mismatch.
	//   - manage_existing_data: write-only; never returned by the API.
	importIgnore := []string{"timeouts", "script", "manage_existing_data"}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCEADestroy(t),
		Steps: []resource.TestStep{
			{
				Config: ceaText(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ceaResource, "id"),
					resource.TestCheckResourceAttr(ceaResource, "name", name),
					resource.TestCheckResourceAttr(ceaResource, "input_type", "TEXT"),
					resource.TestCheckResourceAttr(ceaResource, "enabled", "true"),
					// Empty-echo normalization: companion fields stay null.
					resource.TestCheckNoResourceAttr(ceaResource, "script"),
					resource.TestCheckNoResourceAttr(ceaResource, "popup_menu_choices.0"),
					resource.TestCheckNoResourceAttr(ceaResource, "directory_service_attribute"),
				),
			},
			{
				Config: ceaScript(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ceaResource, "name", renamed),
					resource.TestCheckResourceAttr(ceaResource, "input_type", "SCRIPT"),
					resource.TestCheckResourceAttr(ceaResource, "inventory_display", "HARDWARE"),
					resource.TestCheckResourceAttr(ceaResource, "enabled", "false"),
					resource.TestCheckResourceAttrSet(ceaResource, "script"),
				),
			},
			{
				ResourceName:            ceaResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnore,
			},
			{
				Config: ceaPopup(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ceaResource, "input_type", "POPUP"),
					resource.TestCheckResourceAttr(ceaResource, "enabled", "true"),
					resource.TestCheckResourceAttr(ceaResource, "popup_menu_choices.#", "3"),
					// popup_menu_choices is a Set (server sorts) — match by element.
					resource.TestCheckTypeSetElemAttr(ceaResource, "popup_menu_choices.*", "Red"),
					resource.TestCheckTypeSetElemAttr(ceaResource, "popup_menu_choices.*", "Green"),
					resource.TestCheckTypeSetElemAttr(ceaResource, "popup_menu_choices.*", "Blue"),
					// Orphaned script auto-cleared by the server on transition.
					resource.TestCheckNoResourceAttr(ceaResource, "script"),
				),
			},
			{
				Config: ceaDSAM(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ceaResource, "input_type", "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"),
					resource.TestCheckResourceAttr(ceaResource, "directory_service_attribute", "mail"),
					resource.TestCheckResourceAttr(ceaResource, "allow_multiple_values", "true"),
					resource.TestCheckNoResourceAttr(ceaResource, "popup_menu_choices.0"),
				),
			},
			{
				ResourceName:            ceaResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnore,
			},
		},
	})
}

// Validator matrix: each ExpectError matches a single no-space token to dodge
// Terraform's ~80-col diagnostic wrap.
func TestAccResource_ProComputerExtensionAttribute_ValidatorErrors(t *testing.T) {
	testhelpers.AccPreCheck(t)

	cases := []struct {
		name   string
		config string
		expect *regexp.Regexp
	}{
		{
			name:   "bad data_type",
			config: ceaBad(`data_type = "BOOLEAN"`, `input_type = "TEXT"`, `inventory_display = "GENERAL"`),
			expect: regexp.MustCompile(`must be one of`),
		},
		{
			name:   "script required when SCRIPT",
			config: ceaBad(`data_type = "STRING"`, `input_type = "SCRIPT"`, `inventory_display = "GENERAL"`),
			expect: regexp.MustCompile(`required`),
		},
		{
			name:   "dsa required when DSAM",
			config: ceaBad(`data_type = "STRING"`, `input_type = "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"`, `inventory_display = "USER_AND_LOCATION"`),
			expect: regexp.MustCompile(`required`),
		},
		{
			name:   "script forbidden on TEXT",
			config: ceaBad(`data_type = "STRING"`, `input_type = "TEXT"`, `inventory_display = "GENERAL"`, `script = "echo hi"`),
			expect: regexp.MustCompile(`SCRIPT`),
		},
		{
			name:   "popup forbidden on TEXT",
			config: ceaBad(`data_type = "STRING"`, `input_type = "TEXT"`, `inventory_display = "GENERAL"`, `popup_menu_choices = ["a"]`),
			expect: regexp.MustCompile(`POPUP`),
		},
		{
			name:   "enabled false forbidden on TEXT",
			config: ceaBad(`data_type = "STRING"`, `input_type = "TEXT"`, `inventory_display = "GENERAL"`, `enabled = false`),
			expect: regexp.MustCompile(`SCRIPT`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: tc.config, ExpectError: tc.expect},
				},
			})
		})
	}
}

// ceaBad assembles a resource block from arbitrary attribute lines, for the
// negative validator cases.
func ceaBad(lines ...string) string {
	body := ""
	for _, l := range lines {
		body += "\t\t\t" + l + "\n"
	}
	return fmt.Sprintf(`
		resource "jamfplatform_pro_computer_extension_attribute" "test" {
			name = "tf-acc-cea-bad"
%s		}
	`, body)
}

func TestAccDataSource_ProComputerExtensionAttribute_ByIDAndName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-cea-ds-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCEADestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_computer_extension_attribute" "test" {
						name               = %q
						data_type          = "STRING"
						input_type         = "POPUP"
						inventory_display  = "GENERAL"
						popup_menu_choices = ["a", "b"]
					}

					data "jamfplatform_pro_computer_extension_attribute" "by_id" {
						id = jamfplatform_pro_computer_extension_attribute.test.id
					}

					data "jamfplatform_pro_computer_extension_attribute" "by_name" {
						name = jamfplatform_pro_computer_extension_attribute.test.name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_computer_extension_attribute.by_id", "name", ceaResource, "name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_computer_extension_attribute.by_id", "input_type", "POPUP"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_computer_extension_attribute.by_name", "id", ceaResource, "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_computer_extension_attribute.by_name", "popup_menu_choices.#", "2"),
				),
			},
		},
	})
}

// List endpoint returns full objects; include_resource hydrates every attribute.
func TestAccListResource_ProComputerExtensionAttribute_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-cea-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCEADestroy(t),
		Steps: []resource.TestStep{
			{
				Config: ceaText(name),
				Check:  resource.TestCheckResourceAttrSet(ceaResource, "id"),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_computer_extension_attribute" "test" {
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
					querycheck.ExpectLength("jamfplatform_pro_computer_extension_attribute.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_computer_extension_attribute.test",
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
