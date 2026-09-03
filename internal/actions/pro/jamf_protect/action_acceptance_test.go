// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package jamfprotectactions_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccAction_ProJamfProtectPlansSync_Invoke registers Jamf Protect (from
// env-var credentials) and fires the plans-sync action after the registration
// exists. Gated on the same Protect credentials as the
// jamfplatform_pro_jamf_protect resource acceptance test — the sync endpoint
// requires a live registration.
//
// NOTE: this test overwrites any existing Jamf Protect registration and leaves
// the tenant unregistered at the end (a pre-existing foreign registration
// cannot be restored — its password is write-only). Invoking an action from
// config requires lifecycle.action_trigger (Terraform >= 1.14). A successful
// apply means the sync POST was accepted.
func TestAccAction_ProJamfProtectPlansSync_Invoke(t *testing.T) {
	protectURL := strings.TrimSpace(testhelpers.AccEnv("JAMFPLATFORM_ACC_PRO_PROTECT_URL"))
	clientID := strings.TrimSpace(testhelpers.AccEnv("JAMFPLATFORM_ACC_PRO_PROTECT_CLIENT_ID"))
	password := strings.TrimSpace(testhelpers.AccEnv("JAMFPLATFORM_ACC_PRO_PROTECT_PASSWORD"))
	if protectURL == "" || clientID == "" || password == "" {
		t.Skip("JAMFPLATFORM_ACC_PRO_PROTECT_{URL,CLIENT_ID,PASSWORD} not all set — Jamf Protect plans sync test needs a Jamf Protect API client")
	}
	if !strings.HasSuffix(protectURL, "/graphql") {
		protectURL = strings.TrimRight(protectURL, "/") + "/graphql"
	}
	testhelpers.AccPreCheck(t)
	t.Log("warning: this test overwrites any existing Jamf Protect registration and leaves the tenant unregistered; a pre-existing registration cannot be restored (password is write-only)")

	config := fmt.Sprintf(`
resource "jamfplatform_pro_jamf_protect" "test" {
  api_url             = %q
  client_id           = %q
  password            = %q
  password_wo_version = 1
}

action "jamfplatform_pro_jamf_protect_plans_sync" "sync" {
  config {}
}

resource "terraform_data" "trigger" {
  depends_on = [jamfplatform_pro_jamf_protect.test]

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_jamf_protect_plans_sync.sync]
    }
  }
}
`, protectURL, clientID, password)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  resource.TestCheckResourceAttr("jamfplatform_pro_jamf_protect.test", "id", "singleton"),
		}},
	})
}

// TestAccAction_ProJamfProtectDeploymentRetry_ConflictingTargets verifies the
// exactly-one-target ConfigValidator rejects two target modes at plan time. No
// tenant interaction — validation runs before apply, so no env gate beyond the
// standard pre-check.
func TestAccAction_ProJamfProtectDeploymentRetry_ConflictingTargets(t *testing.T) {
	testhelpers.AccPreCheck(t)

	config := `
action "jamfplatform_pro_jamf_protect_deployment_retry" "bad" {
  config {
    deployment_id = "24a7bb2a-9871-4895-9009-d1be07ed31b1"
    serial_number = "C02XXXXXXXXX"
    all_failed    = true
  }
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_jamf_protect_deployment_retry.bad]
    }
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      config,
			ExpectError: regexp.MustCompile(`Conflicting Retry Targets`),
		}},
	})
}

// TestAccAction_ProJamfProtectDeploymentRetry_Invoke retries a computer's failed
// Jamf Protect install task(s) end-to-end. Heavily gated: it needs a real Jamf
// Protect deployment that has a task for the target computer, so it is skipped
// unless the operator supplies the deployment UUID and a serial:
//
//	JAMFPLATFORM_ACC_PRO_PROTECT_DEPLOYMENT_ID — a deployment (plan) UUID
//	JAMFPLATFORM_ACC_PRO_COMPUTER_SERIAL       — an enrolled computer scoped to that plan
//
// A successful apply means the retry POST was accepted (204). Retrying re-queues
// the Protect install command for that computer — real but low-risk fleet impact.
func TestAccAction_ProJamfProtectDeploymentRetry_Invoke(t *testing.T) {
	deploymentID := strings.TrimSpace(testhelpers.AccEnv("JAMFPLATFORM_ACC_PRO_PROTECT_DEPLOYMENT_ID"))
	serial := strings.TrimSpace(testhelpers.AccEnv("JAMFPLATFORM_ACC_PRO_COMPUTER_SERIAL"))
	if deploymentID == "" || serial == "" {
		t.Skip("JAMFPLATFORM_ACC_PRO_PROTECT_DEPLOYMENT_ID and JAMFPLATFORM_ACC_PRO_COMPUTER_SERIAL not both set — deployment retry test needs a live Protect deployment and a target computer")
	}
	testhelpers.AccPreCheck(t)

	config := fmt.Sprintf(`
action "jamfplatform_pro_jamf_protect_deployment_retry" "retry" {
  config {
    deployment_id = %q
    serial_number = %q
  }
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_jamf_protect_deployment_retry.retry]
    }
  }
}
`, deploymentID, serial)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: config}},
	})
}
