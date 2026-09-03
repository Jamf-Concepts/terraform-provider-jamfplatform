// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package maintenanceactions_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// Invoking an action from config requires lifecycle.action_trigger
// (Terraform >= 1.14).
//
// redeploy_management_framework targets a real computer, gated on a serial so the
// operator can supply and swap it:
//
//	JAMFPLATFORM_ACC_PRO_COMPUTER_SERIAL — a disposable enrolled computer
//
// flush_policy_logs self-provisions its target policy, so it needs no env gate.
const envComputerSerial = "JAMFPLATFORM_ACC_PRO_COMPUTER_SERIAL"

func fireConfig(actionBlock, actionRef string) string {
	return fmt.Sprintf(`
%s

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [%s]
    }
  }
}
`, actionBlock, actionRef)
}

func TestAccAction_ProRedeployManagementFramework_Invoke(t *testing.T) {
	serial := testhelpers.AccEnv(envComputerSerial)
	if serial == "" {
		t.Skipf("%s not set; skipping redeploy management framework acceptance test", envComputerSerial)
	}
	testhelpers.AccPreCheck(t)

	config := fireConfig(fmt.Sprintf(`
action "jamfplatform_pro_redeploy_management_framework" "redeploy" {
  config {
    serial_number = %q
  }
}`, serial), "action.jamfplatform_pro_redeploy_management_framework.redeploy")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: config}},
	})
}

// TestAccAction_ProFlushPolicyLogs_Invoke self-provisions a policy and flushes its
// logs, so it has no fleet impact and needs no env gate. A successful apply means
// the flush was accepted.
func TestAccAction_ProFlushPolicyLogs_Invoke(t *testing.T) {
	testhelpers.AccPreCheck(t)

	name := "tf-acc-flush-policy-" + testhelpers.RunSuffix()

	config := fmt.Sprintf(`
resource "jamfplatform_pro_policy" "test" {
  general = {
    name    = %q
    enabled = false
  }
}

action "jamfplatform_pro_flush_policy_logs" "flush" {
  config {
    policy_id = jamfplatform_pro_policy.test.id
    quantity  = "Six"
    period    = "Months"
  }
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_flush_policy_logs.flush]
    }
  }
}
`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  resource.TestCheckResourceAttrSet("jamfplatform_pro_policy.test", "id"),
		}},
	})
}
