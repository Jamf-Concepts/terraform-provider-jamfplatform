// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package enrollment_customization_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccListResource_ProEnrollmentCustomization_Basic exercises the list
// resource via the `terraform query` workflow.
func TestAccListResource_ProEnrollmentCustomization_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ec-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnrollmentCustomizationDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_enrollment_customization" "src" {
						display_name = %q
						description  = "tf acc list"
						%s

						text_panes = [
							{
								display_name         = "Welcome"
								rank                 = 0
								title                = "Welcome"
								body                 = "list hydration fixture"
								previous_button_text = "Back"
								next_button_text     = "Next"
							},
						]
					}
				`, name, configCommon()),
				Check: resource.TestCheckResourceAttrSet("jamfplatform_pro_enrollment_customization.src", "id"),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_enrollment_customization" "test" {
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
					querycheck.ExpectLength("jamfplatform_pro_enrollment_customization.test", 1),
					querycheck.ExpectResourceKnownValues(
						"jamfplatform_pro_enrollment_customization.test",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("display_name"), KnownValue: knownvalue.StringExact(name)},
							// The list endpoint carries the parent record only — panes
							// live on their own endpoints. Asserting display_name alone
							// is what let a list resource leaving them null pass, while
							// `terraform query -generate-config-out` wrote a
							// customization with no panes, and applying that back deleted
							// them. This pins the pane hydration.
							{Path: tfjsonpath.New("text_panes").AtSliceIndex(0).AtMapKey("title"), KnownValue: knownvalue.StringExact("Welcome")},
							{Path: tfjsonpath.New("text_panes").AtSliceIndex(0).AtMapKey("body"), KnownValue: knownvalue.StringExact("list hydration fixture")},
						},
					),
				},
			},
		},
	})
}
