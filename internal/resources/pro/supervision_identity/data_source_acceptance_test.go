// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package supervision_identity_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProSupervisionIdentity_ByID looks up a generated identity by id.
func TestAccDataSource_ProSupervisionIdentity_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-supervision-ds-id-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSupervisionIdentityDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "jamfplatform_pro_supervision_identity" "src" {
  display_name = %q
  password     = "AccSupervisionPassw0rd!"
}

data "jamfplatform_pro_supervision_identity" "lookup" {
  id = jamfplatform_pro_supervision_identity.src.id
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_supervision_identity.lookup", "display_name", "jamfplatform_pro_supervision_identity.src", "display_name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_supervision_identity.lookup", "common_name", "jamfplatform_pro_supervision_identity.src", "common_name"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_supervision_identity.lookup", "expiration_date", "jamfplatform_pro_supervision_identity.src", "expiration_date"),
				),
			},
		},
	})
}

// TestAccDataSource_ProSupervisionIdentity_ByDisplayName looks up a generated
// identity by its (unique-in-test) display name.
func TestAccDataSource_ProSupervisionIdentity_ByDisplayName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-supervision-ds-name-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSupervisionIdentityDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "jamfplatform_pro_supervision_identity" "src" {
  display_name = %q
  password     = "AccSupervisionPassw0rd!"
}

data "jamfplatform_pro_supervision_identity" "lookup" {
  display_name = jamfplatform_pro_supervision_identity.src.display_name
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_supervision_identity.lookup", "id", "jamfplatform_pro_supervision_identity.src", "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_supervision_identity.lookup", "common_name", "Jamf Identity - "+name),
				),
			},
		},
	})
}
