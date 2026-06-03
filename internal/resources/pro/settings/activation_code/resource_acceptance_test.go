// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /activationcode endpoint.
//
// IMPORTANT — license-secret handling: the activation `code` is a live license key.
// Writing an INVALID code disables the tenant, and there is no spare valid code to
// rotate to, so these tests DO NOT mutate `code`. The Update round-trip is exercised on
// `organization_name` only. The real, current code is read from the tenant at test
// start and re-used verbatim in every config so the always-full PUT never changes the
// license. The original organization name is restored as the final mutating step.
// (Maintainer-approved exception to the "mutate every non-RequiresReplace attribute"
// acceptance-coverage rule.)

package activation_code_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const resName = "jamfplatform_pro_activation_code.test"

// currentActivationCode reads the tenant's live activation code and organization name so
// the test can re-use the real code (never mutating the license) and restore the org name.
func currentActivationCode(t *testing.T) (org, code string) {
	t.Helper()
	c := proclassic.New(testhelpers.NewAcceptanceClient(t))
	got, err := c.GetActivationCode(context.Background())
	if err != nil {
		t.Fatalf("failed to read current activation code: %v", err)
	}
	if got == nil || got.Code == nil || got.OrganizationName == nil {
		t.Fatalf("tenant returned an incomplete activation code record")
	}
	return *got.OrganizationName, *got.Code
}

// checkSingletonRecordStillExists verifies the activation code record persists on the
// tenant after Terraform destroys the resource from state (the remote Delete is a no-op).
func checkSingletonRecordStillExists(t *testing.T) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		got, err := c.GetActivationCode(context.Background())
		if err != nil {
			return fmt.Errorf("expected activation code record to persist on tenant after destroy, got error: %w", err)
		}
		if got == nil {
			return fmt.Errorf("expected non-nil activation code record post-destroy")
		}
		return nil
	}
}

func configFor(org, code string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_activation_code" "test" {
			organization_name = %q
			code              = %q
		}
	`, org, code)
}

// TestAccResource_ProActivationCode_Basic exercises the Update round-trip on
// organization_name (code held constant at the tenant's real value), then restores the
// original organization name.
func TestAccResource_ProActivationCode_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)

	originalOrg, code := currentActivationCode(t)
	const testOrg = "Terraform Acceptance Test Org"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkSingletonRecordStillExists(t),
		Steps: []resource.TestStep{
			{
				// Step 1: change organization_name; code is the tenant's real code.
				Config: configFor(testOrg, code),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resName, "id", "singleton"),
					resource.TestCheckResourceAttr(resName, "organization_name", testOrg),
					resource.TestCheckResourceAttr(resName, "code", code),
				),
			},
			{
				// Step 2: restore the original organization name (still the real code).
				Config: configFor(originalOrg, code),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resName, "organization_name", originalOrg),
				),
			},
			{
				ResourceName:      resName,
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

// TestAccResource_ProActivationCode_RejectsNonSingletonImport verifies the ImportState
// guard: any identifier other than "singleton" must fail.
func TestAccResource_ProActivationCode_RejectsNonSingletonImport(t *testing.T) {
	testhelpers.AccPreCheck(t)

	org, code := currentActivationCode(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configFor(org, code),
			},
			{
				ResourceName:  resName,
				ImportState:   true,
				ImportStateId: "not-singleton",
				ExpectError:   regexp.MustCompile(`Invalid singleton import identifier`),
			},
		},
	})
}
