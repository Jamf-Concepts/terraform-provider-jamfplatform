// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package user_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProUser_ByUsernameThenID looks a user up by username, then
// re-reads the resolved ID through the same data source and asserts the records
// agree. The provider ships no user resource to create a fixture, so the test is
// skipped unless JAMFPLATFORM_ACC_USERNAME names an existing inventory user.
func TestAccDataSource_ProUser_ByUsernameThenID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	username := os.Getenv("JAMFPLATFORM_ACC_USERNAME")
	if username == "" {
		t.Skip("set JAMFPLATFORM_ACC_USERNAME to an existing Jamf Pro inventory username to run this test")
	}

	cfg := fmt.Sprintf(`
		data "jamfplatform_pro_user" "by_name" {
			username = %q
		}
		data "jamfplatform_pro_user" "by_id" {
			id = data.jamfplatform_pro_user.by_name.id
		}
	`, username)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_user.by_name", "id"),
				resource.TestCheckResourceAttr("data.jamfplatform_pro_user.by_name", "username", username),
				// The id-keyed read must resolve the same record.
				resource.TestCheckResourceAttrPair(
					"data.jamfplatform_pro_user.by_id", "username",
					"data.jamfplatform_pro_user.by_name", "username",
				),
				resource.TestCheckResourceAttrPair(
					"data.jamfplatform_pro_user.by_id", "id",
					"data.jamfplatform_pro_user.by_name", "id",
				),
			),
		}},
	})
}
