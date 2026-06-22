// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package patchactions_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// Patch software title fixture constants — a real available title on the tenant
// (mirrors the patch_policy resource acceptance fixture). 8x8 Work, source 1.
const (
	accTitleNameID   = "285"
	accTitleSourceID = 1
	accTitleVersion  = "8.33.2.2"
)

// TestAccAction_ProRetryPatchPolicyLogs_Invoke self-provisions the package +
// patch software title + patch policy chain, then retries the (empty) failed-log
// set for that policy. No env gate. Invoking an action from config requires
// lifecycle.action_trigger (Terraform >= 1.14).
func TestAccAction_ProRetryPatchPolicyLogs_Invoke(t *testing.T) {
	testhelpers.AccPreCheck(t)

	suffix := testhelpers.RunSuffix()

	config := fmt.Sprintf(`
resource "jamfplatform_pro_package" "pkg" {
  display_name = "tf-acc-retry-pkg-%[1]s"
  file_name    = "tf-acc-retry-pkg-%[1]s.pkg"
}

resource "jamfplatform_pro_patch_software_title" "title" {
  name      = "tf-acc retry 8x8 Work %[1]s"
  name_id   = %[2]q
  source_id = %[3]d

  version_packages = {
    %[4]q = jamfplatform_pro_package.pkg.id
  }
}

resource "jamfplatform_pro_patch_policy" "test" {
  software_title_configuration_id = jamfplatform_pro_patch_software_title.title.id
  name                            = "tf-acc-retry-%[1]s"
  target_version                  = %[4]q
  enabled                         = false
  distribution_method             = "selfservice"
}

action "jamfplatform_pro_retry_patch_policy_logs" "retry" {
  config {
    patch_policy_id = jamfplatform_pro_patch_policy.test.id
  }
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_retry_patch_policy_logs.retry]
    }
  }
}
`, suffix, accTitleNameID, accTitleSourceID, accTitleVersion)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  resource.TestCheckResourceAttrSet("jamfplatform_pro_patch_policy.test", "id"),
		}},
	})
}
