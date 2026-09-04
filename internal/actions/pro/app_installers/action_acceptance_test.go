// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package appinstalleractions_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// titleName is a Jamf-published App Catalog title, chosen for the same reason as
// the app_installer resource fixture: Jamf owns the catalog entry, so it stays
// healthy.
const titleName = "Jamf Composer"

// deploymentFixture is a disabled, unscoped SELF_SERVICE/MANUAL draft. It
// installs nothing on any computer — the smart group is left unset, so the
// deployment targets no devices — which is what makes exercising the retry and
// version-update actions against it safe.
func deploymentFixture(suffix string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_app_installer" "test" {
  name            = "tf-acc-ai-action-%[1]s"
  app_title_name  = %[2]q
  deployment_type = "SELF_SERVICE"
  update_behavior = "MANUAL"
}
`, suffix, titleName)
}

// TestAccAction_ProRetryAppInstallerInstallations_Invoke retries a deployment
// with no failed installations. Jamf Pro answers 404 with an empty errors array,
// which the action reports as a warning rather than an error — so the apply must
// succeed. That is the behaviour under test: a retry wired to an action_trigger
// fires after every apply, and a healthy fleet must not fail the workspace.
func TestAccAction_ProRetryAppInstallerInstallations_Invoke(t *testing.T) {
	testhelpers.AccPreCheck(t)

	suffix := testhelpers.RunSuffix()
	config := deploymentFixture(suffix) + `
action "jamfplatform_pro_retry_app_installer_installations" "retry" {
  config {
    deployment_id = jamfplatform_pro_app_installer.test.id
  }
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_retry_app_installer_installations.retry]
    }
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  resource.TestCheckResourceAttrSet("jamfplatform_pro_app_installer.test", "id"),
		}},
	})
}

// TestAccAction_ProRetryAppInstallerInstallations_PerComputer takes the
// per-computer branch, which issues one request per ID because Jamf Pro exposes
// no batch form. The computer ID is deliberately one that does not exist: every
// request 404s, the action must skip each and still complete, and the apply must
// succeed. A real computer ID cannot be used here — the suite has no enrolled
// device it may install software on.
func TestAccAction_ProRetryAppInstallerInstallations_PerComputer(t *testing.T) {
	testhelpers.AccPreCheck(t)

	suffix := testhelpers.RunSuffix()
	config := deploymentFixture(suffix) + `
action "jamfplatform_pro_retry_app_installer_installations" "retry" {
  config {
    deployment_id = jamfplatform_pro_app_installer.test.id
    computer_ids  = ["999999", "999998"]
  }
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_retry_app_installer_installations.retry]
    }
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  resource.TestCheckResourceAttrSet("jamfplatform_pro_app_installer.test", "id"),
		}},
	})
}

// TestAccAction_ProRetryAllAppInstallerInstallations_Invoke exercises the
// tenant-wide retry. It takes no arguments and needs no fixture; with nothing to
// retry Jamf Pro answers the same empty 404, warned rather than raised.
func TestAccAction_ProRetryAllAppInstallerInstallations_Invoke(t *testing.T) {
	testhelpers.AccPreCheck(t)

	config := `
action "jamfplatform_pro_retry_all_app_installer_installations" "retry_all" {
  config {}
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_retry_all_app_installer_installations.retry_all]
    }
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  resource.TestCheckResourceAttrSet("terraform_data.trigger", "id"),
		}},
	})
}

// TestAccAction_ProUpdateAppInstallerVersion_RefusesCurrentVersion pins the
// forward-only rule that is the whole reason this is an action and not a
// writable attribute. A freshly created MANUAL deployment sits on the title's
// current version, so asking to move to that same version is refused with
// "current version 'X' cannot be updated to the new version 'X'". The apply must
// fail with the action's own diagnostic.
//
// The success path is not covered: it needs a title with a newer version than
// the deployment's, which means creating a deployment pinned to an older version
// — and the create endpoint carries no version field, so the fixture cannot be
// built. Jamf Pro decides the version at create time.
func TestAccAction_ProUpdateAppInstallerVersion_RefusesCurrentVersion(t *testing.T) {
	testhelpers.AccPreCheck(t)

	suffix := testhelpers.RunSuffix()
	config := deploymentFixture(suffix) + `
action "jamfplatform_pro_update_app_installer_version" "move" {
  config {
    deployment_id = jamfplatform_pro_app_installer.test.id
    version       = jamfplatform_pro_app_installer.test.selected_version
  }
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.jamfplatform_pro_update_app_installer_version.move]
    }
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      config,
			ExpectError: regexp.MustCompile(`Update App Installer Version Failed`),
		}},
	})
}
