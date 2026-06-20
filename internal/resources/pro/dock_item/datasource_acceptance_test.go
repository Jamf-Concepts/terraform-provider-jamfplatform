// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package dock_item_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestAccDataSource_ProDockItem_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-dock-item-ds-id-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDockItemDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_dock_item" "src" {
						name = %q
						type = "App"
						path = "/Applications/Calculator.app"
					}

					data "jamfplatform_pro_dock_item" "lookup" {
						id = jamfplatform_pro_dock_item.src.id
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_dock_item.lookup", "name", "jamfplatform_pro_dock_item.src", "name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_dock_item.lookup", "type", "jamfplatform_pro_dock_item.src", "type"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_dock_item.lookup", "path", "jamfplatform_pro_dock_item.src", "path"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_pro_dock_item.lookup", "contents"),
				),
			},
		},
	})
}

func TestAccDataSource_ProDockItem_ByName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-dock-item-ds-name-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDockItemDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_dock_item" "src" {
						name = %q
						type = "Folder"
						path = "~/Downloads"
					}

					data "jamfplatform_pro_dock_item" "lookup" {
						name = jamfplatform_pro_dock_item.src.name
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_dock_item.lookup", "id", "jamfplatform_pro_dock_item.src", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_dock_item.lookup", "type", "Folder"),
				),
			},
		},
	})
}
