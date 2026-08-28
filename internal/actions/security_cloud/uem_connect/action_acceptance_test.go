// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package uemconnectactions_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// The action's fixtures are the resource suite's, since the action operates on the
// integration that suite creates. Kept as literals rather than shared so this
// package does not depend on another test package's unexported names.
const envPlatformTenantID = "JAMFPLATFORM_ACC_UEM_CONNECT_PLATFORM_TENANT_ID"

func securityCloudClient(t *testing.T) *securitycloud.Client {
	t.Helper()
	return securitycloud.New(testhelpers.NewAcceptanceClient(t))
}

// requireNoExistingIntegration skips when the tenant already holds a UEM Connect
// integration, because this test creates one and a tenant holds only one. It
// refuses to remove someone else's to make room — that would take their device
// sync down.
func requireNoExistingIntegration(t *testing.T) {
	t.Helper()

	page, err := securityCloudClient(t).ListUemConnectorsV1(context.Background())
	if err != nil {
		t.Skipf("could not check for an existing UEM Connect integration: %v", err)
	}
	if page != nil && len(page.Results) > 0 {
		t.Skipf("this tenant already holds a UEM Connect integration (%s); remove it first to run this test", page.Results[0].ID)
	}
}

func platformTenantIDOrSkip(t *testing.T) string {
	t.Helper()

	tenant := os.Getenv(envPlatformTenantID)
	if tenant == "" {
		t.Skipf("%s must be set to the platform tenant ID of a Jamf Pro instance for this test", envPlatformTenantID)
	}
	return tenant
}

// TestAccAction_SecurityCloudUEMConnectSynchronize_Invoke triggers a sync on an
// integration the same configuration creates.
//
// What it can assert is deliberately limited. The action is fire-once over an
// asynchronous operation: Jamf Security Cloud accepts the request and runs the sync
// in the background, so a passing apply means "the request was accepted", which is
// the whole contract. Asserting on the sync's outcome would be asserting on a race,
// and the run history is not surfaced by this provider by design.
//
// Referencing the resource's id in the action config is also the dependency edge
// that makes the action run after the integration exists — the ergonomics the
// optional attribute is there for.
func TestAccAction_SecurityCloudUEMConnectSynchronize_Invoke(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	requireNoExistingIntegration(t)
	tenant := platformTenantIDOrSkip(t)

	config := fmt.Sprintf(`
resource "jamfplatform_security_cloud_uem_connect" "test" {
  uem_vendor = "JAMF_PRO"

  platform_tenant = {
    tenant_id = %q
  }
}

action "jamfplatform_security_cloud_uem_connect_synchronize" "now" {
  config {
    uem_connect_id = jamfplatform_security_cloud_uem_connect.test.id
  }
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_security_cloud_uem_connect_synchronize.now]
    }
  }
}
`, tenant)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  resource.TestCheckResourceAttrSet("jamfplatform_security_cloud_uem_connect.test", "id"),
		}},
	})
}

// TestAccAction_SecurityCloudUEMConnectSynchronize_WithoutID covers the form the
// example leads with: no uem_connect_id at all, resolved from the tenant's only
// integration.
//
// Worth its own test rather than a variation, because the resolution path is a
// second read the explicit form never exercises — and it is the path a caller who
// has not read the docs will take.
func TestAccAction_SecurityCloudUEMConnectSynchronize_WithoutID(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	requireNoExistingIntegration(t)
	tenant := platformTenantIDOrSkip(t)

	config := fmt.Sprintf(`
resource "jamfplatform_security_cloud_uem_connect" "test" {
  uem_vendor = "JAMF_PRO"

  platform_tenant = {
    tenant_id = %q
  }
}

action "jamfplatform_security_cloud_uem_connect_synchronize" "now" {
  config {}
}

# depends_on rather than a config reference: with no uem_connect_id there is nothing
# to create the edge, so the trigger has to carry it or the sync can run before the
# integration exists.
resource "terraform_data" "trigger" {
  depends_on = [jamfplatform_security_cloud_uem_connect.test]

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_security_cloud_uem_connect_synchronize.now]
    }
  }
}
`, tenant)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  resource.TestCheckResourceAttrSet("jamfplatform_security_cloud_uem_connect.test", "id"),
		}},
	})
}

// TestAccAction_SecurityCloudUEMConnectSynchronize_DisabledIntegration pins the one
// failure worth a named diagnostic: Jamf Security Cloud refuses to synchronize a
// disabled integration, and its message names the integration's identifier rather
// than the setting that has to change.
func TestAccAction_SecurityCloudUEMConnectSynchronize_DisabledIntegration(t *testing.T) {
	testhelpers.AccPreCheckSecurityCloud(t)
	requireNoExistingIntegration(t)
	tenant := platformTenantIDOrSkip(t)

	config := fmt.Sprintf(`
resource "jamfplatform_security_cloud_uem_connect" "test" {
  uem_vendor = "JAMF_PRO"
  enabled    = false

  platform_tenant = {
    tenant_id = %q
  }
}

action "jamfplatform_security_cloud_uem_connect_synchronize" "now" {
  config {
    uem_connect_id = jamfplatform_security_cloud_uem_connect.test.id
  }
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_security_cloud_uem_connect_synchronize.now]
    }
  }
}
`, tenant)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			// Anchored on no-space tokens and with \s+ for the gaps: Terraform wraps
			// error output at ~80 columns and where the wrap lands shifts with the
			// message around it.
			ExpectError: regexp.MustCompile(`disabled,\s+so\s+it\s+cannot\s+synchronize`),
		}},
	})
}
