// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package enrollment_customization_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// TestAccDataSource_ProEnrollmentCustomization_ByID provisions a
// customization via the resource and reads it back via the singular data
// source by ID.
func TestAccDataSource_ProEnrollmentCustomization_ByID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ec-ds-id-" + suffix
	cfg := fmt.Sprintf(`
		resource "jamfplatform_pro_enrollment_customization" "src" {
			display_name = %q
			description  = "tf acc ds by id"
			%s
		}

		data "jamfplatform_pro_enrollment_customization" "lookup" {
			id = jamfplatform_pro_enrollment_customization.src.id
		}
	`, name, configCommon())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnrollmentCustomizationDestroy(t),
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrPair("data.jamfplatform_pro_enrollment_customization.lookup", "display_name", "jamfplatform_pro_enrollment_customization.src", "display_name"),
				resource.TestCheckResourceAttrPair("data.jamfplatform_pro_enrollment_customization.lookup", "description", "jamfplatform_pro_enrollment_customization.src", "description"),
			),
		}},
	})
}

// TestAccDataSource_ProEnrollmentCustomization_ByDisplayName resolves the same
// customization by exact display name.
func TestAccDataSource_ProEnrollmentCustomization_ByDisplayName(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-ec-ds-name-" + suffix
	cfg := fmt.Sprintf(`
		resource "jamfplatform_pro_enrollment_customization" "src" {
			display_name = %q
			description  = "tf acc ds by name"
			%s
		}

		data "jamfplatform_pro_enrollment_customization" "lookup" {
			display_name = jamfplatform_pro_enrollment_customization.src.display_name
		}
	`, name, configCommon())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnrollmentCustomizationDestroy(t),
		Steps: []resource.TestStep{{
			Config: cfg,
			Check:  resource.TestCheckResourceAttrPair("data.jamfplatform_pro_enrollment_customization.lookup", "id", "jamfplatform_pro_enrollment_customization.src", "id"),
		}},
	})
}
