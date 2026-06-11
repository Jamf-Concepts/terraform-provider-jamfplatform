// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package users_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProUsers_ReadAll reads the full inventory user list with no
// filter and asserts the read succeeds and stamps the synthetic id. This passes
// on an empty tenant (zero users) — it exercises the endpoint and state plumbing
// without depending on any fixture.
func TestAccDataSource_ProUsers_ReadAll(t *testing.T) {
	testhelpers.AccPreCheck(t)
	cfg := `
		data "jamfplatform_pro_users" "all" {}
	`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.jamfplatform_pro_users.all", "id", "users"),
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_users.all", "users.#"),
			),
		}},
	})
}
