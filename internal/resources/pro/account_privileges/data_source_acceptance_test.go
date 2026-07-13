// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package account_privileges_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const privsAddr = "data.jamfplatform_pro_account_privileges.test"

// TestAccDataSource_ProAccountPrivileges_Read reads the tenant privilege catalog
// end-to-end. This is the regression guard for issue #290: the classic
// Administrator grid echoes some privilege strings twice within a category
// (Create/Read/Update Cloud Distribution Point, Read/Update Computer Check-In),
// which previously produced a "Duplicate Set Element" error so the data source
// never landed in state. A successful apply with populated sets proves the
// dedup holds; the set-element checks assert the once-duplicated CDP privileges
// are each present exactly once.
func TestAccDataSource_ProAccountPrivileges_Read(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_pro_account_privileges" "test" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(privsAddr, "jamf_pro_server_objects.#"),
					resource.TestCheckResourceAttrSet(privsAddr, "jamf_pro_server_settings.#"),
					resource.TestCheckResourceAttrSet(privsAddr, "all.#"),
					resource.TestCheckTypeSetElemAttr(privsAddr, "jamf_pro_server_objects.*", "Create Cloud Distribution Point"),
					resource.TestCheckTypeSetElemAttr(privsAddr, "jamf_pro_server_objects.*", "Read Cloud Distribution Point"),
					resource.TestCheckTypeSetElemAttr(privsAddr, "jamf_pro_server_objects.*", "Update Cloud Distribution Point"),
				),
			},
		},
	})
}
