// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic
// /mobiledeviceconfigurationprofiles endpoint. Classic has known concurrency
// issues when multiple writes hit the same resource type — keep these
// tests serial with any other classic acceptance work.

package mobile_device_configuration_profile_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/configuration_profiles/payloadhelpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// ── Fixture helpers ───────────────────────────────────────────────────────────

func testdataFile(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("could not resolve caller path")
	}
	abs, err := filepath.Abs(filepath.Join(filepath.Dir(file), "testdata", name))
	if err != nil {
		t.Fatalf("resolving testdata path %q: %v", name, err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("testdata fixture %q not present at %q: %v", name, abs, err)
	}
	return abs
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(testdataFile(t, name))
	if err != nil {
		t.Fatalf("reading testdata fixture %q: %v", name, err)
	}
	return string(b)
}

// newUUID generates a random v4 UUID string.
func newUUID(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("generating UUID: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// freshPayload reads a fixture and injects a fresh top-level PayloadUUID +
// PayloadIdentifier so the same on-disk fixture can be used across multiple
// test runs without hitting Jamf Pro's duplicate-UUID rejection. Always
// returns a string ending with \n so Terraform heredoc EOF is on its own line.
func freshPayload(t *testing.T, fixture string) string {
	t.Helper()
	raw := readFixture(t, fixture)
	uuid := newUUID(t)
	out, err := payloadhelpers.InjectTopLevelIdentifierValues([]byte(raw), uuid, uuid)
	if err != nil {
		t.Fatalf("freshPayload(%q): %v", fixture, err)
	}
	s := string(out)
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s
}

// ── SDK helpers ───────────────────────────────────────────────────────────────

func checkDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testhelpers.NewProClassicClient(t)
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_mobile_device_configuration_profile" {
				continue
			}
			_, err := c.GetMobileDeviceConfigurationProfileByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("checking mobile device configuration profile %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("mobile device configuration profile %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

func createDummyUser(t *testing.T, name string) string {
	t.Helper()
	c := testhelpers.NewProClassicClient(t)
	ctx := context.Background()
	got, err := c.CreateUserByID(ctx, "0", &proclassic.UserPost{Name: &name})
	if err != nil || got == nil || got.ID == nil {
		t.Fatalf("CreateUserByID(%q): %v", name, err)
	}
	id := fmt.Sprintf("%d", *got.ID)
	t.Cleanup(func() {
		if err := c.DeleteUserByID(context.Background(), id); err != nil && !helpers.IsNotFoundError(err) {
			t.Logf("cleanup DeleteUserByID(%s): %v", id, err)
		}
	})
	return id
}

func createDummyMobileDeviceGroup(t *testing.T, name string) string {
	t.Helper()
	c := testhelpers.NewProClassicClient(t)
	ctx := context.Background()
	isSmart := false
	got, err := c.CreateMobileDeviceGroupByID(ctx, "0", &proclassic.MobileDeviceGroup{
		Name:    &name,
		IsSmart: &isSmart,
	})
	if err != nil || got == nil || got.ID == nil {
		t.Fatalf("CreateMobileDeviceGroupByID(%q): %v", name, err)
	}
	id := fmt.Sprintf("%d", *got.ID)
	t.Cleanup(func() {
		if err := c.DeleteMobileDeviceGroupByID(context.Background(), id); err != nil && !helpers.IsNotFoundError(err) {
			t.Logf("cleanup DeleteMobileDeviceGroupByID(%s): %v", id, err)
		}
	})
	return id
}

// ── Config builders ───────────────────────────────────────────────────────────

func configMinimal(name, payload string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_mobile_device_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
}
`, name, payload)
}

func configWithDescription(name, payload, desc string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_mobile_device_configuration_profile" "test" {
  general = {
    name        = %q
    description = %q
    payloads = <<EOF
%sEOF
  }
}
`, name, desc, payload)
}

func configWithDistribution(name, payload, dist string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_mobile_device_configuration_profile" "test" {
  general = {
    name                = %q
    distribution_method = %q
    payloads = <<EOF
%sEOF
  }
}
`, name, dist, payload)
}

func configAllMobileDevices(name, payload string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_mobile_device_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
  scope = {
    targets = {
      all_mobile_devices = true
    }
  }
}
`, name, payload)
}

func configScopeWithExclusions(name, payload string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_mobile_device_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
  scope = {
    targets = {
      all_mobile_devices = true
    }
    exclusions = {
      directory_service_or_local_user_names = ["nonexistent-acc-test-user"]
    }
  }
}
`, name, payload)
}

func configScopeWithMobileDeviceGroupIDs(name, payload, groupID string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_mobile_device_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
  scope = {
    targets = {
      mobile_device_group_ids = [%q]
    }
  }
}
`, name, payload, groupID)
}

func configScopeJSSUser(name, payload, userID string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_mobile_device_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
  scope = {
    targets = {
      user_ids = [%q]
    }
  }
}
`, name, payload, userID)
}

func configWithSelfServiceAuthPassword(name, payload, password string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_mobile_device_configuration_profile" "test" {
  general = {
    name                = %q
    distribution_method = "Make Available in Self Service"
    payloads = <<EOF
%sEOF
  }
  self_service = {
    self_service_description = "Acceptance-test mobile profile auth"
    feature_on_main_page     = true
    removal_disallowed       = "With Authorization"
    authorization_password   = %q
  }
}
`, name, payload, password)
}

func configWithSelfService(name, payload string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_mobile_device_configuration_profile" "test" {
  general = {
    name                = %q
    distribution_method = "Make Available in Self Service"
    payloads = <<EOF
%sEOF
  }
  self_service = {
    self_service_description = "Acceptance-test mobile profile"
    feature_on_main_page     = true
    removal_disallowed       = "Never"
  }
}
`, name, payload)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestAccResource_MobileDeviceConfigurationProfile_Minimal — minimal create /
// rename / idempotent re-apply. Uses a real Exchange ActiveSync payload
// (profile_50) to exercise PayloadUUID injection on update and verify the
// no-ghost-profile path.
func TestAccResource_MobileDeviceConfigurationProfile_Minimal(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-min-" + suffix
	renamed := name + "-renamed"
	payload := freshPayload(t, "profile_50.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configMinimal(name, payload),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("general").AtMapKey("name"),
						knownvalue.StringExact(name),
					),
				},
			},
			{
				Config: configMinimal(renamed, payload),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("general").AtMapKey("name"),
						knownvalue.StringExact(renamed),
					),
				},
			},
			{
				// Re-apply identical config → must produce empty plan.
				Config: configMinimal(renamed, payload),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_PayloadByteDifferentSemanticallyEqual
// — compact XML reformatted by inserting newlines between tags. Bytes differ
// but semantics don't; diff suppression must produce ResourceActionNoop.
func TestAccResource_MobileDeviceConfigurationProfile_PayloadByteDifferentSemanticallyEqual(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-sem-" + suffix
	payload := freshPayload(t, "profile_48.mobileconfig")

	// Compact XML has no whitespace between tags. Insert newlines at every
	// ><  boundary — byte-different, semantically identical to a plist parser.
	reformatted := strings.ReplaceAll(payload, "><", ">\n<")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{Config: configMinimal(name, payload)},
			{
				Config: configMinimal(name, reformatted),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"jamfplatform_pro_mobile_device_configuration_profile.test",
							plancheck.ResourceActionNoop,
						),
					},
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_RealPayloadChangeProducesPlan
// — increments maxFailedAttempts in profile_48 (Passcode policy). The field
// is not in the masked set so the change must surface as drift and produce
// a non-empty plan. Uses a small payload to avoid Jamf PUT timeouts.
func TestAccResource_MobileDeviceConfigurationProfile_RealPayloadChangeProducesPlan(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-real-" + suffix
	payload := freshPayload(t, "profile_48.mobileconfig")

	// freshPayload marshals via the plist library; use its output format.
	// Skip guard catches format drift if the fixture value changes.
	const before = "<key>maxFailedAttempts</key>\n\t\t\t\t<integer>2</integer>"
	const after = "<key>maxFailedAttempts</key>\n\t\t\t\t<integer>10</integer>"
	tampered := strings.Replace(payload, before, after, 1)
	if tampered == payload {
		t.Skip("expected to tamper with maxFailedAttempts in profile_48 but pattern not found")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{Config: configMinimal(name, payload)},
			{
				Config: configMinimal(name, tampered),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_DescriptionChange — envelope
// field change (general.description) must produce a normal plan and round-trip
// through state. Uses the Notifications payload (profile_44).
func TestAccResource_MobileDeviceConfigurationProfile_DescriptionChange(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-desc-" + suffix
	payload := freshPayload(t, "profile_44.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{Config: configWithDescription(name, payload, "v1")},
			{
				Config: configWithDescription(name, payload, "v2"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("general").AtMapKey("description"),
						knownvalue.StringExact("v2"),
					),
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_AllMobileDevicesScope —
// exercises all_mobile_devices=true and verifies the flag round-trips through
// state. Uses the Wi-Fi payload (profile_70).
func TestAccResource_MobileDeviceConfigurationProfile_AllMobileDevicesScope(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-all-" + suffix
	payload := freshPayload(t, "profile_70.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configAllMobileDevices(name, payload),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_mobile_devices"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_ScopeWithExclusions —
// all_mobile_devices + a directory-user name exclusion. Exercises the
// exclusion sub-block wiring without requiring an enrolled device.
func TestAccResource_MobileDeviceConfigurationProfile_ScopeWithExclusions(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-excl-" + suffix
	payload := freshPayload(t, "profile_70.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configScopeWithExclusions(name, payload),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_mobile_devices"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("directory_service_or_local_user_names"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact("nonexistent-acc-test-user")}),
					),
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_ScopeWithMobileDeviceGroupIDs
// — creates a static mobile device group, pins it in the profile scope, and
// verifies the group ID round-trips through state.
func TestAccResource_MobileDeviceConfigurationProfile_ScopeWithMobileDeviceGroupIDs(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-grp-" + suffix
	groupID := createDummyMobileDeviceGroup(t, "tf-acc-mdcp-fixture-grp-"+suffix)
	payload := freshPayload(t, "profile_70.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configScopeWithMobileDeviceGroupIDs(name, payload, groupID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("mobile_device_group_ids"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact(groupID)}),
					),
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_ScopeJSSUserAddRemove —
// adds a Jamf Pro user to scope, then removes it. Exercises the Set→null
// teardown path. Uses the Exchange ActiveSync payload (profile_50).
func TestAccResource_MobileDeviceConfigurationProfile_ScopeJSSUserAddRemove(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-jssu-" + suffix
	jssUserID := createDummyUser(t, "tf-acc-mdcp-jssuser-"+suffix)
	payload := freshPayload(t, "profile_50.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configScopeJSSUser(name, payload, jssUserID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("user_ids"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact(jssUserID)}),
					),
				},
			},
			{
				Config: configMinimal(name, payload),
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_SelfService — exercises the
// self_service sub-block (mobile variant: no notification fields). Uses the
// Notifications payload (profile_44) with Make Available in Self Service.
func TestAccResource_MobileDeviceConfigurationProfile_SelfService(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-ss-" + suffix
	payload := freshPayload(t, "profile_44.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configWithSelfService(name, payload),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("self_service").AtMapKey("feature_on_main_page"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("self_service").AtMapKey("removal_disallowed"),
						knownvalue.StringExact("Never"),
					),
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_SelfServiceAuthorizationPassword —
// exercises self_service.security end-to-end: round-trip of plaintext
// `authorization_password` (Jamf Pro echoes it on read) plus a step that
// rotates the password. Also exercises the
// payloadhelpers.serverInjectedPayloadTypes path: Jamf injects a
// com.apple.profileRemovalPassword PayloadContent entry into the stored
// mobileconfig when authorization_password is set, and the mask filter must
// drop it so PayloadsSemanticallyEqual treats plan and server as equal and
// the self-healing payload branch in flattenGeneral keeps the user-authored
// payload bytes in state.
func TestAccResource_MobileDeviceConfigurationProfile_SelfServiceAuthorizationPassword(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-ssauth-" + suffix
	payload := freshPayload(t, "profile_44.mobileconfig")

	const pw1 = "tf-acc-secret-aaa"
	const pw2 = "tf-acc-secret-bbb"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configWithSelfServiceAuthPassword(name, payload, pw1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("self_service").AtMapKey("removal_disallowed"),
						knownvalue.StringExact("With Authorization"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("self_service").AtMapKey("authorization_password"),
						knownvalue.StringExact(pw1),
					),
				},
			},
			{
				Config: configWithSelfServiceAuthPassword(name, payload, pw2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("self_service").AtMapKey("authorization_password"),
						knownvalue.StringExact(pw2),
					),
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_DistributionMethodChange —
// flips distribution_method; wire uses deployment_method but TF exposes
// distribution_method (UI-canonical). Confirms symmetric round-trip.
// Uses the Passcode policy payload (profile_48).
func TestAccResource_MobileDeviceConfigurationProfile_DistributionMethodChange(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-dm-" + suffix
	payload := freshPayload(t, "profile_48.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{Config: configWithDistribution(name, payload, "Install Automatically")},
			{
				Config: configWithDistribution(name, payload, "Make Available in Self Service"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_mobile_device_configuration_profile.test",
						tfjsonpath.New("general").AtMapKey("distribution_method"),
						knownvalue.StringExact("Make Available in Self Service"),
					),
				},
			},
		},
	})
}

// TestAccResource_MobileDeviceConfigurationProfile_ImportState — import by ID.
// No ImportStateVerify: importer only populates general; scope and self_service
// stay null until authored (per feedback_acc_import_optional_sections.md).
// Uses the Wi-Fi payload (profile_70).
func TestAccResource_MobileDeviceConfigurationProfile_ImportState(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mdcp-import-" + suffix
	payload := freshPayload(t, "profile_70.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{Config: configMinimal(name, payload)},
			{
				ResourceName:                         "jamfplatform_pro_mobile_device_configuration_profile.test",
				ImportState:                          true,
				ImportStateVerify:                    false,
				ImportStateVerifyIdentifierAttribute: "id",
			},
		},
	})
}
