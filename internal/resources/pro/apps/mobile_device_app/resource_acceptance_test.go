// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic /mobiledeviceapplications
// endpoint. Classic has known concurrency issues when multiple writes hit the
// same resource type — keep these tests serial with any future classic
// acceptance work in this package.
//
// Scope happy-paths use all_mobile_devices = true so they need no pre-existing
// tenant target objects, and general.site is left unset. os_type is required on
// every config (the server demands it on a PUT to an in-house app, which is the
// common case). The ungated tests never set vpp.assign_vpp_device_based_licenses
// = true on a non-VPP title (that 409s "App is not available for device
// assignment"); TestAccResource_ProMobileApp_VPPDeviceAssignment409 asserts that
// 409 deliberately.

package mobile_device_app_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const mobileAppResourceAddr = "jamfplatform_pro_mobile_device_app.test"

// ldapGroupEnvVar names a real directory-service group the tenant's LDAP /
// cloud-IdP actually has. directory_service_user_group_names is server-matched
// against real groups ("Problem matching limitation user group" 409 otherwise).
const ldapGroupEnvVar = "JAMFPLATFORM_ACC_ENROLLMENT_GROUP_NAME"

// testAccCheckMobileAppDestroy verifies apps created during the test were
// destroyed. Note the server returns 400 on a successful DELETE (handled in the
// resource), so destroy correctness is itself part of what this asserts.
func testAccCheckMobileAppDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_mobile_device_app" {
				continue
			}
			_, err := c.GetMobileDeviceApplicationByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking Jamf Pro mobile device app %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("Jamf Pro mobile device app %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

// mobileAppGeneralOnlyConfig is the import-stable shape: only the required
// general fields. The importer populates general post-Read but leaves optional
// blocks null, so ImportStateVerify must run against a general-only config.
func mobileAppGeneralOnlyConfig(name, version, deploymentType string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_app" "test" {
			general = {
				name            = %q
				version         = %q
				bundle_id       = "com.example.tfacc.mobileapp"
				os_type         = "iOS"
				deployment_type = %q
			}
		}
	`, name, version, deploymentType)
}

// mobileAppFullConfig adds self_service and an all_mobile_devices scope on top of general.
func mobileAppFullConfig(name, version, buttonText string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_app" "test" {
			general = {
				name            = %q
				version         = %q
				bundle_id       = "com.example.tfacc.mobileapp"
				os_type         = "iOS"
				deployment_type = "Make Available in Self Service"
			}
			scope = {
				all_mobile_devices = true
			}
			self_service = {
				install_button_text        = %q
				after_install_button_text  = "Open"
				self_service_description    = "Managed by Terraform acceptance test."
				feature_on_main_page        = true
				notification_enabled        = true
			}
		}
	`, name, version, buttonText)
}

// TestAccResource_ProMobileApp_Basic exercises create, in-place update, and
// import for the general-only shape. The version/deployment_type change verifies
// the GET-after-Update path (classic UpdateMobileDeviceApplicationByID returns
// 201 empty). Update also re-sends os_type — without it the server 409s on the PUT.
func TestAccResource_ProMobileApp_Basic(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMobileAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: mobileAppGeneralOnlyConfig(name, "1.0", "Make Available in Self Service"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(mobileAppResourceAddr, "id"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "general.name", name),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "general.version", "1.0"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "general.os_type", "iOS"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "general.deployment_type", "Make Available in Self Service"),
				),
			},
			{
				ResourceName:      mobileAppResourceAddr,
				ImportState:       true,
				ImportStateVerify: true,
				// timeouts: not returned by the API.
				// general.os_type: write-mostly — the server requires it on write
				// but does not echo it on GET after a POST create, so an imported
				// app cannot recover it (it stays null until the next apply
				// re-sends the configured value). Not verifiable on import.
				ImportStateVerifyIgnore: []string{"timeouts", "general.os_type"},
			},
			{
				Config: mobileAppGeneralOnlyConfig(name, "2.0", "Install Automatically/Prompt Users to Install"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "general.version", "2.0"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "general.deployment_type", "Install Automatically/Prompt Users to Install"),
				),
			},
			{
				// Rename: guards that an update changing name applies cleanly. The
				// server-derived echo fields (description / internal_app /
				// category_name / site_name) do not change on a rename, so their
				// UseStateForUnknown plan values stay consistent through apply.
				Config: mobileAppGeneralOnlyConfig(name+"-renamed", "2.0", "Install Automatically/Prompt Users to Install"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "general.name", name+"-renamed"),
				),
			},
		},
	})
}

// TestAccResource_ProMobileApp_ScopeAndSelfService exercises the scope +
// self_service blocks and an in-place self_service mutation (not a block
// deletion — the classic PUT is partial-merge and will not clear an omitted block).
func TestAccResource_ProMobileApp_ScopeAndSelfService(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-ss-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMobileAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: mobileAppFullConfig(name, "1.0", "Install"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(mobileAppResourceAddr, "id"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.all_mobile_devices", "true"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "self_service.install_button_text", "Install"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "self_service.after_install_button_text", "Open"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "self_service.feature_on_main_page", "true"),
				),
			},
			{
				Config: mobileAppFullConfig(name, "1.0", "Get"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "self_service.install_button_text", "Get"),
				),
			},
		},
	})
}

// TestAccResource_ProMobileApp_AppConfiguration exercises the app_configuration
// block and the CRLF↔LF semantic-equality plan modifier: step 2 reuses the same
// content with CRLF newlines and must produce an empty plan (no diff).
func TestAccResource_ProMobileApp_AppConfiguration(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-appcfg-" + suffix

	cfg := func(prefs string) string {
		return fmt.Sprintf(`
			resource "jamfplatform_pro_mobile_device_app" "test" {
				general = {
					name      = %q
					version   = "1.0"
					bundle_id = "com.example.tfacc.mobileapp.appcfg"
					os_type   = "iOS"
				}
				app_configuration = {
					preferences = %q
				}
			}
		`, name, prefs)
	}

	const lf = "<dict>\n  <key>Server</key>\n  <string>https://example.com</string>\n</dict>"
	const crlf = "<dict>\r\n  <key>Server</key>\r\n  <string>https://example.com</string>\r\n</dict>"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMobileAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: cfg(lf),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(mobileAppResourceAddr, "id"),
					resource.TestCheckResourceAttrSet(mobileAppResourceAddr, "app_configuration.preferences"),
				),
			},
			{
				// Same content, CRLF newlines: the plan modifier treats it as equal.
				Config:   cfg(crlf),
				PlanOnly: true,
			},
		},
	})
}

// TestAccResource_ProMobileApp_InvalidDeploymentType asserts the deployment_type
// OneOf validator rejects an out-of-set value at plan time.
func TestAccResource_ProMobileApp_InvalidDeploymentType(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-bad-dt-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      mobileAppGeneralOnlyConfig(name, "1.0", "Force Install Right Now"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("value must be one of"),
			},
		},
	})
}

// TestAccResource_ProMobileApp_InvalidOsType asserts the os_type OneOf validator
// rejects a value outside {iOS, tvOS} at plan time.
func TestAccResource_ProMobileApp_InvalidOsType(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-bad-os-" + suffix
	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_app" "test" {
			general = {
				name      = %q
				version   = "1.0"
				bundle_id = "com.example.tfacc.mobileapp"
				os_type   = "watchOS"
			}
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("value must be one of"),
			},
		},
	})
}

// TestAccResource_ProMobileApp_AllMobileDevicesConflict asserts the shared scope
// validator rejects all_mobile_devices = true alongside a device target.
func TestAccResource_ProMobileApp_AllMobileDevicesConflict(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-allmd-" + suffix
	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_app" "test" {
			general = {
				name      = %q
				version   = "1.0"
				bundle_id = "com.example.tfacc.mobileapp"
				os_type   = "iOS"
			}
			scope = {
				all_mobile_devices = true
				mobile_device_ids  = ["1"]
			}
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Conflicts with all-flag"),
			},
		},
	})
}

// TestAccResource_ProMobileApp_AllJssUsersConflict asserts the shared scope
// validator rejects all_jss_users = true alongside a user target.
func TestAccResource_ProMobileApp_AllJssUsersConflict(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-allu-" + suffix
	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_app" "test" {
			general = {
				name      = %q
				version   = "1.0"
				bundle_id = "com.example.tfacc.mobileapp"
				os_type   = "iOS"
			}
			scope = {
				all_jss_users = true
				user_ids      = ["1"]
			}
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Conflicts with all-flag"),
			},
		},
	})
}

// TestAccResource_ProMobileApp_VPPDeviceAssignment409 asserts the documented VPP
// invariant: assigning device-based licenses to a non-VPP-backed title returns
// HTTP 409 "App is not available for device assignment". Apply-time error.
func TestAccResource_ProMobileApp_VPPDeviceAssignment409(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-vpp409-" + suffix
	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_app" "test" {
			general = {
				name      = %q
				version   = "1.0"
				bundle_id = "com.example.tfacc.mobileapp.vpp"
				os_type   = "iOS"
				is_free   = true
			}
			vpp = {
				assign_vpp_device_based_licenses = true
			}
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMobileAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("(?i)not available for device assignment"),
			},
		},
	})
}

// mobileAppScopeTargetsConfig builds sibling building/department/network_segment
// resources and references their IDs from the app's scope — exercising the scope
// target/limitation/exclusion ID round-trips that the all_mobile_devices
// happy-paths never touch. The mobile-device/user target categories are omitted
// because they need real tenant inventory; everything else is hermetic.
func mobileAppScopeTargetsConfig(suffix string, limitationUserNames string) string {
	return fmt.Sprintf(`
		resource "jamfplatform_pro_building" "b1" {
			name = "tf-acc-mobileapp-bldg-%[1]s"
		}

		resource "jamfplatform_pro_department" "d1" {
			name = "tf-acc-mobileapp-dept-%[1]s"
		}

		resource "jamfplatform_pro_network_segment" "n1" {
			name             = "tf-acc-mobileapp-ns-%[1]s"
			starting_address = "10.124.0.1"
			ending_address   = "10.124.0.254"
		}

		resource "jamfplatform_pro_mobile_device_app" "test" {
			general = {
				name      = "tf-acc-pro-mobileapp-scope-%[1]s"
				version   = "1.0"
				bundle_id = "com.example.tfacc.mobileapp.scope"
				os_type   = "iOS"
			}

			scope = {
				building_ids   = [jamfplatform_pro_building.b1.id]
				department_ids = [jamfplatform_pro_department.d1.id]

				limitations = {
					network_segment_ids                    = [jamfplatform_pro_network_segment.n1.id]
					directory_service_or_local_user_names   = %[2]s
				}

				exclusions = {
					network_segment_ids                    = [jamfplatform_pro_network_segment.n1.id]
					directory_service_or_local_user_names   = ["excluded-user"]
				}
			}
		}
	`, suffix, limitationUserNames)
}

// TestAccResource_ProMobileApp_ScopeTargets exercises the scope target /
// limitation / exclusion ID round-trips against real sibling resources, and
// (step 2) a set-shrink on the limitation directory-user names.
func TestAccResource_ProMobileApp_ScopeTargets(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMobileAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: mobileAppScopeTargetsConfig(suffix, `["alice", "bob"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(mobileAppResourceAddr, "id"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.building_ids.#", "1"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.department_ids.#", "1"),
					resource.TestCheckResourceAttrPair(mobileAppResourceAddr, "scope.building_ids.0", "jamfplatform_pro_building.b1", "id"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.limitations.network_segment_ids.#", "1"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.limitations.directory_service_or_local_user_names.#", "2"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.exclusions.network_segment_ids.#", "1"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.exclusions.directory_service_or_local_user_names.#", "1"),
				),
			},
			{
				Config: mobileAppScopeTargetsConfig(suffix, `["alice"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.limitations.directory_service_or_local_user_names.#", "1"),
				),
			},
		},
	})
}

// TestAccDataSource_ProMobileApp resolves a created app through the singular data
// source by ID and by name, asserting the flat read-only projection.
func TestAccDataSource_ProMobileApp(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-ds-" + suffix

	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_app" "test" {
			general = {
				name      = %q
				version   = "3.1"
				bundle_id = "com.example.tfacc.mobileapp.ds"
				os_type   = "iOS"
				is_free   = true
			}
		}

		data "jamfplatform_pro_mobile_device_app" "by_id" {
			id = jamfplatform_pro_mobile_device_app.test.id
		}

		data "jamfplatform_pro_mobile_device_app" "by_name" {
			name = jamfplatform_pro_mobile_device_app.test.general.name
		}
	`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMobileAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_app.by_id", "name", mobileAppResourceAddr, "general.name"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mobile_device_app.by_id", "version", "3.1"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mobile_device_app.by_id", "bundle_id", "com.example.tfacc.mobileapp.ds"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mobile_device_app.by_id", "is_free", "true"),
					resource.TestCheckResourceAttrPair("data.jamfplatform_pro_mobile_device_app.by_name", "id", mobileAppResourceAddr, "id"),
					resource.TestCheckResourceAttr("data.jamfplatform_pro_mobile_device_app.by_name", "name", name),
				),
			},
		},
	})
}

// TestAccListResource_ProMobileApp exercises the list resource via the
// `terraform query` workflow, filtering to the unique created app by name.
func TestAccListResource_ProMobileApp(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-list-" + suffix

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMobileAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "jamfplatform_pro_mobile_device_app" "test" {
						general = {
							name      = %q
							version   = "1.0"
							bundle_id = "com.example.tfacc.mobileapp.list"
							os_type   = "iOS"
						}
					}
				`, name),
				Check: resource.TestCheckResourceAttrSet(mobileAppResourceAddr, "id"),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
					provider "jamfplatform" {}

					list "jamfplatform_pro_mobile_device_app" "test" {
						provider = jamfplatform

						config {
							filter = {
								name_substring = %q
							}
						}
					}
				`, name),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("jamfplatform_pro_mobile_device_app.test", 1),
				},
			},
		},
	})
}

// TestAccResource_ProMobileApp_ScopeLdapGroup exercises a limitation that
// references a real directory-service user group, which the server validates
// against the configured LDAP / cloud-IdP. Gated on
// JAMFPLATFORM_ACC_ENROLLMENT_GROUP_NAME; also serves as the live check that the
// plan-time DS-group preflight accepts a real group rather than rejecting it.
func TestAccResource_ProMobileApp_ScopeLdapGroup(t *testing.T) {
	group := os.Getenv(ldapGroupEnvVar)
	if group == "" {
		t.Skipf("%s not set; skipping directory-service group scope test", ldapGroupEnvVar)
	}
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-pro-mobileapp-ldap-" + suffix

	config := fmt.Sprintf(`
		resource "jamfplatform_pro_mobile_device_app" "test" {
			general = {
				name      = %q
				version   = "1.0"
				bundle_id = "com.example.tfacc.mobileapp.ldap"
				os_type   = "iOS"
			}

			scope = {
				all_mobile_devices = true

				limitations = {
					directory_service_user_group_names = [%q]
				}
			}
		}
	`, name, group)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMobileAppDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(mobileAppResourceAddr, "id"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.limitations.directory_service_user_group_names.#", "1"),
					resource.TestCheckResourceAttr(mobileAppResourceAddr, "scope.limitations.directory_service_user_group_names.0", group),
				),
			},
		},
	})
}
