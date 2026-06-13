// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package managed_software_updates_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// checkSingletonRecordStillExists verifies the feature-toggle record persists on the tenant
// after Terraform destroys the resource from state — the remote Delete is a no-op.
func checkSingletonRecordStillExists(t *testing.T) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		c := pro.New(testhelpers.NewAcceptanceClient(t))
		got, err := c.GetManagedSoftwareUpdateFeatureToggleV1(context.Background())
		if err != nil {
			return fmt.Errorf("expected Managed Software Updates feature toggle to persist on tenant after destroy, got error: %w", err)
		}
		if got == nil {
			return fmt.Errorf("expected non-nil feature toggle record post-destroy")
		}
		return nil
	}
}

// TestAccResource_ProManagedSoftwareUpdateFeatureToggle_Basic flips the toggle across two
// Update steps against a real tenant and confirms the async poll-to-settle lands the
// requested value each time. It ends with enabled=true so the tenant is left enabled (its
// pre-test state). Singleton resources have no remote Delete, so CheckDestroy verifies the
// record PERSISTS after Terraform stops managing it.
func TestAccResource_ProManagedSoftwareUpdate_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	const addr = "jamfplatform_pro_managed_software_update.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkSingletonRecordStillExists(t),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_managed_software_update" "test" {
						enabled = false
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "id", "singleton"),
					resource.TestCheckResourceAttr(addr, "enabled", "false"),
					// Sub-enables are server-managed; they must be known (Computed), not asserted to a value.
					resource.TestCheckResourceAttrSet(addr, "dss_enabled"),
					resource.TestCheckResourceAttrSet(addr, "recipe_enabled"),
					resource.TestCheckResourceAttrSet(addr, "force_install_local_date_enabled"),
					resource.TestCheckResourceAttrSet(addr, "custom_version_enabled"),
				),
			},
			{
				Config: `
					resource "jamfplatform_pro_managed_software_update" "test" {
						enabled = true
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "enabled", "true"),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateId:     "singleton",
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
				},
			},
		},
	})
}

// TestAccResource_ProManagedSoftwareUpdate_RejectsNonSingletonImport verifies the
// ImportState guard rejects any identifier other than "singleton".
func TestAccResource_ProManagedSoftwareUpdate_RejectsNonSingletonImport(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "jamfplatform_pro_managed_software_update" "test" {}
				`,
			},
			{
				ResourceName:  "jamfplatform_pro_managed_software_update.test",
				ImportState:   true,
				ImportStateId: "not-the-singleton",
				ExpectError:   regexp.MustCompile(`Invalid singleton import identifier`),
			},
		},
	})
}
