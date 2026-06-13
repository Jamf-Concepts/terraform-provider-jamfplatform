// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package managed_software_updates_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccAction_ProManagedSoftwareUpdatePlan_Invoke exercises the group plan action
// end-to-end against a real tenant. It self-provisions a 0-member static computer group and
// feeds its Computed jamf_pro_id as group_id, so the POST mints zero device-plans and has no
// real fleet impact. The Managed Software Updates feature is enabled first (the POST 503s
// when the toggle is off); the terraform_data trigger fires the action after both the toggle
// and the group exist, which orders the action correctly.
//
// NOTE: invoking an action from config requires `lifecycle.action_trigger`, which needs
// Terraform >= 1.14. Run with a recent Terraform binary (TF_ACC_TERRAFORM_PATH / the
// harness default). This test self-provisions its fixtures — no env-gate.
func TestAccAction_ProManagedSoftwareUpdatePlan_Invoke(t *testing.T) {
	testhelpers.AccPreCheck(t)

	name := "tf-acc-msu-target-" + testhelpers.RunSuffix()

	config := fmt.Sprintf(`
		resource "jamfplatform_pro_managed_software_update" "msu" {
			enabled = true
		}

		resource "jamfplatform_device_group" "target" {
			name        = %q
			description = "Acceptance test — safe to delete"
			group_type  = "static"
			device_type = "computer"
			depends_on  = [jamfplatform_pro_managed_software_update.msu]
		}

		action "jamfplatform_pro_managed_software_update_plan" "deploy" {
			config {
				group_id      = jamfplatform_device_group.target.jamf_pro_id
				object_type   = "COMPUTER_GROUP"
				update_action = "DOWNLOAD_ONLY"
				version_type  = "LATEST_ANY"
			}
		}

		resource "terraform_data" "trigger" {
			input = jamfplatform_device_group.target.jamf_pro_id

			lifecycle {
				action_trigger {
					events  = [after_create]
					actions = [action.jamfplatform_pro_managed_software_update_plan.deploy]
				}
			}
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// A successful apply means the action was invoked and the group plan POST
				// returned 201 (0 device-plans, since the group has no members).
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_device_group.target", "jamf_pro_id"),
				),
			},
		},
	})
}

// TestAccAction_ProManagedSoftwareUpdatePlan_RequiresSpecificVersion verifies the plan-time
// ConfigValidator rejects version_type=SPECIFIC_VERSION without specific_version. Validation
// runs at plan time, so no resources are created.
func TestAccAction_ProManagedSoftwareUpdatePlan_RequiresSpecificVersion(t *testing.T) {
	testhelpers.AccPreCheck(t)

	config := `
		action "jamfplatform_pro_managed_software_update_plan" "bad" {
			config {
				group_id      = "1"
				object_type   = "COMPUTER_GROUP"
				update_action = "DOWNLOAD_INSTALL"
				version_type  = "SPECIFIC_VERSION"
			}
		}

		resource "terraform_data" "trigger" {
			lifecycle {
				action_trigger {
					events  = [after_create]
					actions = [action.jamfplatform_pro_managed_software_update_plan.bad]
				}
			}
		}
	`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Missing specific_version`),
			},
		},
	})
}
