// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package printer_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestAccDataSource_ProPrinter_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-printer-ds-id-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPrinterDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_printer" "src" {
						name = %q
						uri  = "ipp://10.1.20.120/"
					}

					data "jamfplatform_pro_printer" "lookup" {
						id = jamfplatform_pro_printer.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_printer.lookup", "name", "jamfplatform_pro_printer.src", "name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_printer.lookup", "uri", "jamfplatform_pro_printer.src", "uri"),
					// use_generic defaults true, so the PPD trio collapses to
					// null in state (server echo notwithstanding).
					resource.TestCheckNoResourceAttr("data.jamfplatform_pro_printer.lookup", "ppd_path"),
				),
			},
		},
	})
}

func TestAccDataSource_ProPrinter_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-printer-ds-name-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPrinterDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_printer" "src" {
						name = %q
					}

					data "jamfplatform_pro_printer" "lookup" {
						name = jamfplatform_pro_printer.src.name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_printer.lookup", "id", "jamfplatform_pro_printer.src", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_printer.lookup", "name", name),
				),
			},
		},
	})
}
