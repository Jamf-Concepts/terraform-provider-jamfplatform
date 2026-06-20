// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package webhook_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccListResource_ProWebhook_Basic exercises the jamfplatform_pro_webhook
// list resource via the `terraform query` workflow. The classic /webhooks list
// endpoint returns id+name only, so the list resource is identity-only; the
// client-side name_substring filter narrows to the record created in step 1.
func TestAccListResource_ProWebhook_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	name := "tf-acc-pro-webhook-list-" + testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebhookDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_webhook" "src" {
						name  = %q
						url   = "https://example.com/list"
						event = "ComputerAdded"
					}
				`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_pro_webhook.src", "id"),
				),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_webhook" "test" {
						provider = jamfplatform

						config {
							filter = {
								name_substring = %q
							}
						}
					}
				`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					// Identity-only list: the client-side name_substring filter
					// narrows to the single record created above.
					querycheck.ExpectLength("jamfplatform_pro_webhook.test", 1),
				},
			},
		},
	})
}
