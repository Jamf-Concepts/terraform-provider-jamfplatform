// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /licensedsoftware endpoint.
// Classic has known concurrency issues when multiple writes hit the same
// resource type — keep these tests serial with any future classic acceptance
// work in this package.
//
// No pre-existing tenant target objects are required: the record header leaves
// site unset (defaults to "None"), and software definitions / licences are
// free-text. No fixture strings contain a literal '&' (the ProClassic content
// 409 bug — see project_proclassic_payload_amp_content_bug).

package licensed_software_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
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

const licensedSoftwareResourceAddr = "jamfplatform_pro_licensed_software.test"

// testAccCheckLicensedSoftwareDestroy verifies records created during the test
// were destroyed.
func testAccCheckLicensedSoftwareDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_licensed_software" {
				continue
			}
			_, err := c.GetLicensedSoftwareByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro licensed software %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro licensed software %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// definitionsOnlyConfig: one software definition, licenses OMITTED (opt-out —
// unmanaged; nothing exists to retain on a fresh record). The header bools are
// left unset so the server defaults resolve (every general bool false, platform
// "Any", site "None").
func definitionsOnlyConfig(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_licensed_software" "test" {
			name = %q
			software_definitions = [
				{ name = "Acme Editor", version = "1.0", compare_type = "is" },
			]
		}
	`, name)
}

// fullConfig: every mutable header field set, two software definitions (add),
// and two licences (add) — the first carrying a full purchasing block, the
// second a bare licence. Used for the import round-trip and the grow step.
func fullConfig(name, poDate, expires string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_licensed_software" "test" {
			name                                    = %q
			publisher                               = "Acme Corp"
			platform                                = "Mac"
			notes                                   = "managed by terraform"
			send_email_on_violation                 = true
			remove_titles_from_inventory_reports    = true
			exclude_titles_purchased_from_app_store = true

			software_definitions = [
				{ name = "Acme Editor", version = "2.0", compare_type = "is" },
				{ name = "Acme Viewer", compare_type = "like" },
			]

			licenses = [
				{
					serial_number_1   = "SER-0001"
					organization_name = "Acme Corp"
					registered_to     = "IT Department"
					license_type      = "Standard"
					license_count     = 25
					notes             = "primary licence"
					purchasing = {
						license_term       = "perpetual"
						po_number          = "PO-12345"
						po_date            = %q
						vendor             = "Acme Reseller"
						license_expires    = %q
						purchase_price     = "1999.00"
						life_expectancy    = 3
						purchasing_account = "Finance"
						purchasing_contact = "Jane Doe"
					}
				},
				{
					serial_number_1 = "SER-0002"
					license_type    = "Concurrent"
					license_count   = 0
					purchasing = {
						license_term = "annual"
						vendor       = "Acme Reseller"
					}
				},
			]
		}
	`, name, poDate, expires)
}

// shrunkConfig: one software definition (remove), one licence (remove), header
// fields changed back. Drives the present→absent removal path on both lists.
func shrunkConfig(name string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_licensed_software" "test" {
			name      = %q
			publisher = "Acme Corp Updated"
			platform  = "Any"

			software_definitions = [
				{ name = "Acme Viewer", compare_type = "like" },
			]

			licenses = [
				{
					serial_number_1 = "SER-0002"
					license_type    = "Concurrent"
					license_count   = 10
				},
			]
		}
	`, name)
}

// TestAccResource_ProLicensedSoftware exercises the full lifecycle: create with
// definitions only, import round-trip, a grow update (header mutation + add a
// second definition AND two licences with computed epoch/utc re-resolution),
// then a shrink update (remove a definition AND a licence). Covers the §1.8
// update round-trip, nested-list add+remove on both positional lists, and
// computed-sibling re-resolution.
func TestAccResource_ProLicensedSoftware(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-licsw-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLicensedSoftwareDestroy(t),
		Steps: []resource.TestStep{
			{
				// Create: definitions only, no licences.
				Config: definitionsOnlyConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(licensedSoftwareResourceAddr, "id"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "name", name),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "platform", "Any"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "send_email_on_violation", "false"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "site_id", "-1"),
					// No site: site_name is null (absent) on the "-1" sentinel — the
					// server echo of "NONE" is flaky, so DerivedRefName nulls it.
					resource.TestCheckNoResourceAttr(licensedSoftwareResourceAddr, "site_name"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.#", "1"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.0.name", "Acme Editor"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.0.compare_type", "is"),
					// licenses omitted -> null list (no .# entry).
					resource.TestCheckNoResourceAttr(licensedSoftwareResourceAddr, "licenses.#"),
				),
			},
			{
				// Grow: full header + 2 definitions + 2 licences.
				Config: fullConfig(name, "2026-03-15", "2027-03-15"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "publisher", "Acme Corp"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "platform", "Mac"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "send_email_on_violation", "true"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.#", "2"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.1.name", "Acme Viewer"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.#", "2"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.serial_number_1", "SER-0001"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.license_count", "25"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.purchasing.license_term", "perpetual"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.purchasing.life_expectancy", "3"),
					// Computed epoch/utc re-resolution from the date strings.
					resource.TestCheckResourceAttrSet(licensedSoftwareResourceAddr, "licenses.0.purchasing.po_date_epoch"),
					resource.TestCheckResourceAttrSet(licensedSoftwareResourceAddr, "licenses.0.purchasing.po_date_utc"),
					resource.TestCheckResourceAttrSet(licensedSoftwareResourceAddr, "licenses.0.purchasing.license_expires_epoch"),
					// Second licence: count 0 preserved, annual term round-trips.
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.1.license_count", "0"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.1.purchasing.license_term", "annual"),
				),
			},
			{
				ResourceName:      licensedSoftwareResourceAddr,
				ImportState:       true,
				ImportStateVerify: true,
				// software_definitions and licenses are opt-out lists: the importer
				// has no prior model, so it leaves them unmanaged (null) and they
				// are re-declared to take ownership. computers is a read-only echo.
				ImportStateVerifyIgnore: []string{"timeouts", "computers", "software_definitions", "licenses"},
			},
			{
				// Shrink: 1 definition (remove), 1 licence (remove), header change.
				Config: shrunkConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "publisher", "Acme Corp Updated"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "platform", "Any"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.#", "1"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.0.name", "Acme Viewer"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.#", "1"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.serial_number_1", "SER-0002"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.license_count", "10"),
				),
			},
		},
	})
}

// posListsConfig renders a record whose two positional lists carry the supplied
// ordered definition names and licence serials. Definitions all use the default
// compare_type EXCEPT the one named "Bravo" (which omits it) so the schema
// default ("like") is exercised end-to-end. licence_count is derived from the
// 1-based position so a value landing at the wrong index is detectable.
func posListsConfig(name string, defNames, licSerials []string) string {
	var defs strings.Builder
	for _, d := range defNames {
		if d == "Bravo" {
			fmt.Fprintf(&defs, "\t\t\t\t{ name = %q, version = \"2.0\" },\n", d) // compare_type omitted -> default "like"
		} else {
			fmt.Fprintf(&defs, "\t\t\t\t{ name = %q, version = \"1.0\", compare_type = \"is\" },\n", d)
		}
	}
	var lics strings.Builder
	for i, s := range licSerials {
		lics.WriteString(fmt.Sprintf("\t\t\t\t{ serial_number_1 = %q, license_type = \"Standard\", license_count = %d },\n", s, i+1))
	}
	return fmt.Sprintf(`
		resource "jamfplatform_pro_licensed_software" "test" {
			name = %q
			software_definitions = [
%s			]
			licenses = [
%s			]
		}
	`, name, defs.String(), lics.String())
}

// TestAccResource_ProLicensedSoftware_PositionalLists targets the defining risk
// of the two id-less lists: ordering and per-index correlation. It exercises
// 3-element lists, the compare_type default, a full reorder (asserting the
// server preserves send-order — the load-bearing wire assumption), a mid-list
// element removal with an in-place mutation of a survivor, and a non-empty →
// empty clear. Terraform's implicit post-apply plan check makes each step a
// drift/idempotency guard.
func TestAccResource_ProLicensedSoftware_PositionalLists(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-licsw-pos-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLicensedSoftwareDestroy(t),
		Steps: []resource.TestStep{
			{
				// 3 + 3, with "Bravo" relying on the compare_type default.
				Config: posListsConfig(name, []string{"Alpha", "Bravo", "Charlie"}, []string{"SER-1", "SER-2", "SER-3"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.#", "3"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.0.name", "Alpha"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.1.name", "Bravo"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.2.name", "Charlie"),
					// Default compare_type resolved for the entry that omitted it.
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.1.compare_type", "like"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.#", "3"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.serial_number_1", "SER-1"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.license_count", "1"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.2.serial_number_1", "SER-3"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.2.license_count", "3"),
				),
			},
			{
				// Full reorder: the server must return the new send-order, or the
				// implicit post-apply plan is non-empty and this step fails.
				Config: posListsConfig(name, []string{"Charlie", "Bravo", "Alpha"}, []string{"SER-3", "SER-2", "SER-1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.0.name", "Charlie"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.2.name", "Alpha"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.serial_number_1", "SER-3"),
					// license_count tracks position (SER-3 now first => count 1).
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.license_count", "1"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.2.serial_number_1", "SER-1"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.2.license_count", "3"),
				),
			},
			{
				// Mid-list removal: drop "Bravo" / "SER-2" from the middle. The
				// survivors must re-correlate cleanly at their new indices.
				Config: posListsConfig(name, []string{"Charlie", "Alpha"}, []string{"SER-3", "SER-1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.#", "2"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.0.name", "Charlie"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.1.name", "Alpha"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.#", "2"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.0.serial_number_1", "SER-3"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.1.serial_number_1", "SER-1"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.1.license_count", "2"),
				),
			},
			{
				// Non-empty → cleared via explicit []: under the opt-out contract,
				// `[]` is the clear signal (omitting would instead retain). The
				// provider emits an empty wire element and state reconciles to a
				// known empty list (.# == 0).
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_licensed_software" "test" {
						name                 = %q
						software_definitions = []
						licenses             = []
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "software_definitions.#", "0"),
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.#", "0"),
				),
			},
		},
	})
}

// TestAccResource_ProLicensedSoftware_InvalidPlatform asserts the platform OneOf
// validator rejects an out-of-vocabulary value.
func TestAccResource_ProLicensedSoftware_InvalidPlatform(t *testing.T) {
	testhelpers.AccPreCheck(t)
	config := `
		resource "jamfplatform_pro_licensed_software" "test" {
			name     = "tf-acc-invalid-platform"
			platform = "BadPlatform"
		}
	`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`BadPlatform`),
			},
		},
	})
}

// TestAccResource_ProLicensedSoftware_InvalidCompareType asserts the
// software_definitions.compare_type OneOf validator rejects an unknown operator.
func TestAccResource_ProLicensedSoftware_InvalidCompareType(t *testing.T) {
	testhelpers.AccPreCheck(t)
	config := `
		resource "jamfplatform_pro_licensed_software" "test" {
			name = "tf-acc-invalid-compare"
			software_definitions = [
				{ name = "X", compare_type = "BadCompareOp" },
			]
		}
	`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`BadCompareOp`),
			},
		},
	})
}

// TestAccResource_ProLicensedSoftware_InvalidLicenseTerm asserts the
// purchasing.license_term OneOf validator rejects an unknown term.
func TestAccResource_ProLicensedSoftware_InvalidLicenseTerm(t *testing.T) {
	testhelpers.AccPreCheck(t)
	config := `
		resource "jamfplatform_pro_licensed_software" "test" {
			name = "tf-acc-invalid-term"
			licenses = [
				{
					serial_number_1 = "S1"
					purchasing = { license_term = "BadTermValue" }
				},
			]
		}
	`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`BadTermValue`),
			},
		},
	})
}

// TestAccResource_ProLicensedSoftware_InvalidLifeExpectancy asserts the
// life_expectancy Between(1,5) validator rejects an out-of-range value.
func TestAccResource_ProLicensedSoftware_InvalidLifeExpectancy(t *testing.T) {
	testhelpers.AccPreCheck(t)
	config := `
		resource "jamfplatform_pro_licensed_software" "test" {
			name = "tf-acc-invalid-life"
			licenses = [
				{
					serial_number_1 = "S1"
					purchasing = {
						license_term    = "perpetual"
						life_expectancy = 9
					}
				},
			]
		}
	`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`life_expectancy`),
			},
		},
	})
}

// TestAccDataSource_ProLicensedSoftware_ByID looks up a created record by ID and
// verifies the flat projection resolves the general header.
func TestAccDataSource_ProLicensedSoftware_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-licsw-ds-id-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLicensedSoftwareDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_licensed_software" "src" {
						name      = %q
						publisher = "Acme Corp"
						platform  = "Mac"
					}

					data "jamfplatform_pro_licensed_software" "lookup" {
						id = jamfplatform_pro_licensed_software.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_licensed_software.lookup", "name", "jamfplatform_pro_licensed_software.src", "name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_licensed_software.lookup", "publisher", "Acme Corp"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_licensed_software.lookup", "platform", "Mac"),
				),
			},
		},
	})
}

// TestAccDataSource_ProLicensedSoftware_ByName looks up a created record by exact
// name and verifies the ID resolves.
func TestAccDataSource_ProLicensedSoftware_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-licsw-ds-name-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLicensedSoftwareDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_licensed_software" "src" {
						name = %q
					}

					data "jamfplatform_pro_licensed_software" "lookup" {
						name = jamfplatform_pro_licensed_software.src.name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_licensed_software.lookup", "id", "jamfplatform_pro_licensed_software.src", "id"),
				),
			},
		},
	})
}

// TestAccListResource_ProLicensedSoftware exercises the list resource via the
// `terraform query` workflow, filtered to the record created in the first step.
func TestAccListResource_ProLicensedSoftware(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-licsw-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLicensedSoftwareDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_licensed_software" "src" {
						name = %q
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_licensed_software.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_licensed_software" "test" {
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
					querycheck.ExpectLength("jamfplatform_pro_licensed_software.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_licensed_software.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact(name)},
						},
					),
				},
			},
		},
	})
}

// checkServerLicenseCount reads the record straight from Jamf Pro (bypassing
// Terraform state) and asserts the stored licence count — the only way to prove
// that omitting the attribute RETAINED the server's licences while state shows
// them unmanaged.
func checkServerLicenseCount(t *testing.T, want int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[licensedSoftwareResourceAddr]
		if !ok {
			return fmt.Errorf("%s not found in state", licensedSoftwareResourceAddr)
		}
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		got, err := c.GetLicensedSoftwareByID(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("server GET %s: %w", rs.Primary.ID, err)
		}
		n := 0
		if got.Licenses != nil && got.Licenses.License != nil {
			n = len(*got.Licenses.License)
		}
		if n != want {
			return fmt.Errorf("server licence count = %d, want %d", n, want)
		}
		return nil
	}
}

// TestAccResource_ProLicensedSoftware_OptOut proves the opt-out contract on the
// licenses list: omitting the attribute leaves the server's licences intact
// (managed nowhere in state), while setting it to [] clears them. The live
// server read is the assertion that omit != clear.
func TestAccResource_ProLicensedSoftware_OptOut(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-licsw-optout-" + suffix

	withLicence := fmt.Sprintf(`
		resource "jamfplatform_pro_licensed_software" "test" {
			name = %q
			licenses = [
				{ serial_number_1 = "OPTOUT-1", license_type = "Standard", license_count = 1 },
			]
		}
	`, name)
	omitLicences := fmt.Sprintf(`
		resource "jamfplatform_pro_licensed_software" "test" {
			name = %q
		}
	`, name)
	clearLicences := fmt.Sprintf(`
		resource "jamfplatform_pro_licensed_software" "test" {
			name     = %q
			licenses = []
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLicensedSoftwareDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: withLicence,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.#", "1"),
					checkServerLicenseCount(t, 1),
				),
			},
			{
				// Omit → retain: state no longer manages licences, but the server
				// keeps the licence it had (the merge PUT omits the wrapper).
				Config: omitLicences,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(licensedSoftwareResourceAddr, "licenses.#"),
					checkServerLicenseCount(t, 1),
				),
			},
			{
				// [] → clear: now the server actually drops the licence.
				Config: clearLicences,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(licensedSoftwareResourceAddr, "licenses.#", "0"),
					checkServerLicenseCount(t, 0),
				),
			},
		},
	})
}
