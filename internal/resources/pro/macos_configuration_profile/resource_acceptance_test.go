// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Tests in this file talk to the Jamf ProClassic
// /osxconfigurationprofiles endpoint. Classic has known concurrency
// issues when multiple writes hit the same resource type — keep these
// tests serial with any other classic acceptance work.

package macos_configuration_profile_test

import (
	"context"
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
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

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

// checkDestroy verifies profiles created during the test were destroyed.
func checkDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := proclassic.New(testhelpers.NewAcceptanceClient(t))
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "jamfplatform_pro_macos_configuration_profile" {
				continue
			}
			_, err := c.GetOSXConfigurationProfileByID(ctx, rs.Primary.ID)
			if err != nil {
				if helpers.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("checking macOS configuration profile %s: %s", rs.Primary.ID, err)
			}
			return fmt.Errorf("macOS configuration profile %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

func configMinimal(name, payload string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_macos_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
}
`, name, payload)
}

func configWithDistribution(name, payload, dist string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_macos_configuration_profile" "test" {
  general = {
    name                = %q
    distribution_method = %q
    payloads = <<EOF
%sEOF
  }
}
`, name, dist, payload)
}

func configWithDescription(name, payload, desc string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_macos_configuration_profile" "test" {
  general = {
    name        = %q
    description = %q
    payloads = <<EOF
%sEOF
  }
}
`, name, desc, payload)
}

func configWithSelfService(name, payload, dispName string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_macos_configuration_profile" "test" {
  general = {
    name                = %q
    distribution_method = "Make Available in Self Service"
    payloads = <<EOF
%sEOF
  }
  self_service = {
    self_service_display_name     = %q
    install_button_text           = "Install"
    self_service_description      = "Acceptance-test profile"
    ensure_users_view_description = false
    feature_on_main_page          = true
    display_notifications         = true
    notification_location         = "Self Service"
    notification_subject          = "tf-acc"
    notification_message          = "Test notification"
    removal_disallowed            = "Never"
  }
}
`, name, payload, dispName)
}

func configAllComputers(name, payload string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_macos_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
  scope = {
    targets = {
      all_computers = true
    }
  }
}
`, name, payload)
}

func configScopeWithComputerIDs(name, payload, computerID string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_macos_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
  scope = {
    targets = {
      computer_ids = [%q]
    }
  }
}
`, name, payload, computerID)
}

func configScopeWithExclusions(name, payload, computerID string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_macos_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
  scope = {
    targets = {
      all_computers = true
    }
    exclusions = {
      computer_ids = [%q]
    }
  }
}
`, name, payload, computerID)
}

func configScopeAddRemoveJSSUser(name, payload, jssUserID string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_macos_configuration_profile" "test" {
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
`, name, payload, jssUserID)
}

func configScopeClearJSSUsers(name, payload string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_macos_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
  scope = {
    targets = {
      user_ids = []
    }
  }
}
`, name, payload)
}

func createDummyComputer(t *testing.T, name string) string {
	t.Helper()
	c := testhelpers.NewProClassicClient(t)
	ctx := context.Background()
	if err := c.CreateComputerByID(ctx, "0", &proclassic.ComputerPost{
		General: &proclassic.ComputerPostGeneral{Name: &name},
	}); err != nil {
		t.Fatalf("CreateComputerByID(%q): %v", name, err)
	}
	got, err := c.GetComputerByName(ctx, name)
	if err != nil || got == nil || got.General == nil || got.General.ID == nil {
		t.Fatalf("GetComputerByName(%q) after create: %v", name, err)
	}
	id := fmt.Sprintf("%d", *got.General.ID)
	t.Cleanup(func() {
		if err := c.DeleteComputerByID(context.Background(), id); err != nil && !helpers.IsNotFoundError(err) {
			t.Logf("cleanup DeleteComputerByID(%s): %v", id, err)
		}
	})
	return id
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

// TestAccResource_MacOSConfigurationProfile_Minimal — minimal create / rename /
// import path. Uses a small managed-login-items mobileconfig from the corpus
// because it parses cleanly through Jamf Pro and exercises both top-level
// PayloadUUID rewrite and a non-trivial PayloadContent shape.
func TestAccResource_MacOSConfigurationProfile_Minimal(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-min-" + suffix
	renamed := name + "-renamed"
	payload := readFixture(t, "1Password__managed_login_items_profile.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configMinimal(name, payload),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("general").AtMapKey("name"),
						knownvalue.StringExact(name),
					),
				},
			},
			{
				Config: configMinimal(renamed, payload),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("general").AtMapKey("name"),
						knownvalue.StringExact(renamed),
					),
				},
			},
			{
				// Re-apply identical config → mask must suppress every diff,
				// producing an empty plan. This is the "no ghost profile"
				// test on the Update path.
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

// TestAccResource_MacOSConfigurationProfile_PayloadByteDifferentSemanticallyEqual
// — payload bytes change (whitespace re-indent) but semantics don't. Diff
// suppression must produce no resource update.
func TestAccResource_MacOSConfigurationProfile_PayloadByteDifferentSemanticallyEqual(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-sem-" + suffix
	payload := readFixture(t, "1Password__managed_login_items_profile.mobileconfig")

	// Re-indent: tabs → 4 spaces. Keeps line breaks (heredoc EOF still
	// resolves) and the `<?xml` preamble starts the string (plist parsers
	// reject leading whitespace before the declaration). The whitespace
	// re-indent is byte-different but semantically identical — the mask
	// must neutralise it.
	reformatted := strings.ReplaceAll(payload, "\t", "    ")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configMinimal(name, payload),
			},
			{
				Config: configMinimal(name, reformatted),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"jamfplatform_pro_macos_configuration_profile.test",
							plancheck.ResourceActionNoop,
						),
					},
				},
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_RealPayloadChangeProducesPlan
// — modifying a non-masked field inside PayloadContent (the Rules[].Comment)
// must NOT be suppressed. The plan must show a non-empty change.
func TestAccResource_MacOSConfigurationProfile_RealPayloadChangeProducesPlan(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-real-" + suffix
	payload := readFixture(t, "1Password__managed_login_items_profile.mobileconfig")
	// Replace the Comment string of the first Rules entry with a distinct
	// value. This survives the mask (Rules.Comment is not server-rewritten)
	// and must surface as drift.
	tampered := strings.Replace(payload,
		"<string>Allow 1Password Launch Item</string>",
		"<string>tampered comment</string>",
		1,
	)
	if tampered == payload {
		t.Skip("expected to be able to tamper with Rules[0].Comment but pattern not found")
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

// TestAccResource_MacOSConfigurationProfile_DescriptionChange — sanity check
// that envelope-level fields (general.description) produce a normal plan
// when changed. This is a non-payload diff and should never be suppressed.
func TestAccResource_MacOSConfigurationProfile_DescriptionChange(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-desc-" + suffix
	payload := readFixture(t, "1Password__notifications_profile.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{Config: configWithDescription(name, payload, "v1")},
			{
				Config: configWithDescription(name, payload, "v2"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("general").AtMapKey("description"),
						knownvalue.StringExact("v2"),
					),
				},
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_AllComputersScope — exercises
// the scope sub-block validator (all_computers=true forbidding per-computer
// IDs) and roundtrips through state.
func TestAccResource_MacOSConfigurationProfile_AllComputersScope(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-all-" + suffix
	payload := readFixture(t, "DuckDuckGo__content_filter_profile.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configAllComputers(name, payload),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_computers"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_SelfService — exercises the
// Self Service sub-block including the dual <notification> wire elements.
func TestAccResource_MacOSConfigurationProfile_SelfService(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-ss-" + suffix
	payload := readFixture(t, "1Password__notifications_profile.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configWithSelfService(name, payload, "Test SS Display Name"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("self_service").AtMapKey("display_notifications"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("self_service").AtMapKey("notification_location"),
						knownvalue.StringExact("Self Service"),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("self_service").AtMapKey("removal_disallowed"),
						knownvalue.StringExact("Never"),
					),
				},
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_DistributionMethodChange —
// flips distribution_method between Install Automatically and Make Available
// in Self Service to confirm the wire-symmetric attribute round-trips.
func TestAccResource_MacOSConfigurationProfile_DistributionMethodChange(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-dm-" + suffix
	payload := readFixture(t, "1Password__screen_recording_profile.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{Config: configWithDistribution(name, payload, "Install Automatically")},
			{
				Config: configWithDistribution(name, payload, "Make Available in Self Service"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("general").AtMapKey("distribution_method"),
						knownvalue.StringExact("Make Available in Self Service"),
					),
				},
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_ScopeWithComputerIDs — pins a
// specific per-computer target via classic ID. Asserts the ID round-trips;
// the undeclared all_computers toggle stays null (owned outside Terraform).
func TestAccResource_MacOSConfigurationProfile_ScopeWithComputerIDs(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-comp-" + suffix
	computerID := createDummyComputer(t, "tf-acc-mcp-fixture-comp-"+suffix)
	payload := readFixture(t, "1Password__notifications_profile.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configScopeWithComputerIDs(name, payload, computerID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("computer_ids"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact(computerID)}),
					),
				},
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_ScopeWithExclusions — all_computers
// combined with a per-computer exclusion. Exercises the exclusion sub-block
// wiring.
func TestAccResource_MacOSConfigurationProfile_ScopeWithExclusions(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-excl-" + suffix
	computerID := createDummyComputer(t, "tf-acc-mcp-fixture-excl-"+suffix)
	payload := readFixture(t, "1Password__screen_recording_profile.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configScopeWithExclusions(name, payload, computerID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("all_computers"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("exclusions").AtMapKey("computer_ids"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact(computerID)}),
					),
				},
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_ScopeJSSUserAddRemove — add a Jamf
// Pro user to the scope on first apply, clear the category with a declared `[]`
// on the second (omitting it would leave the user scoped, as configured outside
// Terraform), then release the whole scope block on the third (state → null;
// the live scope is left untouched).
func TestAccResource_MacOSConfigurationProfile_ScopeJSSUserAddRemove(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-jssu-" + suffix
	jssUserID := createDummyUser(t, "tf-acc-mcp-jssuser-"+suffix)
	payload := readFixture(t, "1Password__managed_login_items_profile.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configScopeAddRemoveJSSUser(name, payload, jssUserID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("user_ids"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact(jssUserID)}),
					),
				},
			},
			{
				// Declared [] clears the category on the wire.
				Config: configScopeClearJSSUsers(name, payload),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"jamfplatform_pro_macos_configuration_profile.test",
						tfjsonpath.New("scope").AtMapKey("targets").AtMapKey("user_ids"),
						knownvalue.SetExact([]knownvalue.Check{}),
					),
				},
			},
			{
				Config: configMinimal(name, payload),
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_ImportState — import by ID
// without ImportStateVerify: import hydrates every wire-present optional
// section (scope with every category, self_service), while this minimal
// config declares none of them, so a verify would legitimately diff.
func TestAccResource_MacOSConfigurationProfile_ImportState(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-import-" + suffix
	payload := readFixture(t, "KarabinerElements__system_extension_profile.mobileconfig")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{Config: configMinimal(name, payload)},
			{
				ResourceName:                         "jamfplatform_pro_macos_configuration_profile.test",
				ImportState:                          true,
				ImportStateVerify:                    false,
				ImportStateVerifyIdentifierAttribute: "id",
			},
		},
	})
}

// expectGeneralAttrKnown asserts that general.<attr> is NOT planned as
// unknown ("known after apply"). plancheck.ExpectKnownValue cannot express
// this: it reads Change.After, where an unknown value is serialised as null
// and is therefore indistinguishable from a known null. This check reads the
// parallel Change.AfterUnknown tree (the same source ExpectUnknownValue uses)
// and fails when the attribute is flagged unknown.
type expectGeneralAttrKnown struct {
	resourceAddress string
	attr            string
}

func (e expectGeneralAttrKnown) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, rc := range req.Plan.ResourceChanges {
		if rc.Address != e.resourceAddress {
			continue
		}
		unknown, err := tfjsonpath.Traverse(rc.Change.AfterUnknown, tfjsonpath.New("general").AtMapKey(e.attr))
		if err != nil {
			// Path absent from the AfterUnknown tree ⇒ not flagged unknown ⇒ known.
			return
		}
		if isUnknown, ok := unknown.(bool); ok && isUnknown {
			resp.Error = fmt.Errorf(
				"%s: general.%s is planned unknown (\"known after apply\") on the first post-import plan; "+
					"it must stay known. Regression of the §886 derived-name restore in ModifyPlan's two-way "+
					"fallback (0bfb64b follow-up).",
				e.resourceAddress, e.attr)
		}
		return
	}
	resp.Error = fmt.Errorf("%s - resource not found in plan ResourceChanges", e.resourceAddress)
}

// TestAccResource_MacOSConfigurationProfile_ImportThenPlan_DerivedNamesStayKnown
// pins the fix for the phantom in-place update seen on the first plan after a
// fresh import. On import the payload is stored in the server-canonical form,
// which is byte-different from (but semantically equal to) the user's HCL, so
// the next plan proposes an update on `general`; that marks category_name /
// site_name Unknown (they are Computed without UseStateForUnknown per §886).
// Import leaves the three-way payload private-state refs empty, so ModifyPlan
// takes its two-way fallback — the branch that, before this fix, suppressed the
// payload diff but forgot to restore the derived names, leaving them Unknown
// and surfacing a spurious update (which for a config profile issues a PUT that
// can re-deploy the profile). The names must stay known.
func TestAccResource_MacOSConfigurationProfile_ImportThenPlan_DerivedNamesStayKnown(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-import-noop-" + suffix
	fixture := readFixture(t, "KarabinerElements__system_extension_profile.mobileconfig")
	const addr = "jamfplatform_pro_macos_configuration_profile.test"

	// Create the object out-of-band so import is the first Terraform action, as
	// a user importing a pre-existing profile does. A Terraform-managed create
	// step would populate the three-way payload private-state refs and route the
	// next plan through the three-way compare (already correct since 0bfb64b) —
	// the regression lives on the import-only two-way fallback path. The config
	// uses the payload read back from the server so it is semantically equal to
	// (but byte-different from) the form import canonicalises into state, which
	// is what drives ModifyPlan into that fallback's suppression branch.
	profileID, serverPayload := createOOBProfile(t, name, fixture)
	cfg := configMinimal(name, serverPayload)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			// 1. Import the out-of-band object and persist it into the working
			//    state. Import's Read never writes payload_last_input /
			//    payload_last_canonical, so the next plan takes the two-way
			//    fallback.
			{
				Config:             cfg,
				ResourceName:       addr,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateVerify:  false,
				ImportStateIdFunc:  func(*terraform.State) (string, error) { return profileID, nil },
			},
			// 2. Plan the imported state against the same HCL. The pre-apply plan
			//    is legitimately non-empty — import hydrates scope / self_service
			//    (includeUnmanaged) that this minimal config omits — but the two
			//    derived names in `general` must NOT churn to "known after apply".
			//    The plan checks run on that pre-apply plan; the apply then
			//    reconciles the hydrated sections and the framework's built-in
			//    post-apply idempotency check confirms the resource settles.
			{
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						expectGeneralAttrKnown{addr, "category_name"},
						expectGeneralAttrKnown{addr, "site_name"},
					},
				},
			},
		},
	})
}

// createOOBProfile creates a minimal macOS configuration profile directly via
// the SDK (no category, no site), then reads it back and returns its ID plus
// the server-stored payload. A t.Cleanup deletes it so the object never leaks
// if the import step fails before Terraform takes over management.
func createOOBProfile(t *testing.T, name, payload string) (id, serverPayload string) {
	t.Helper()
	c := proclassic.New(testhelpers.NewAcceptanceClient(t))
	ctx := context.Background()
	pl := proclassic.PayloadsXMLText(payload)
	created, err := c.CreateOSXConfigurationProfileByID(ctx, "0", &proclassic.OsXConfigurationProfile{
		General: &proclassic.OsXConfigurationProfileGeneral{Name: &name, Payloads: &pl},
	})
	if err != nil {
		t.Fatalf("out-of-band create of macOS configuration profile %q: %v", name, err)
	}
	switch {
	case created != nil && created.ID != nil:
		id = fmt.Sprintf("%d", *created.ID)
	case created != nil && created.General != nil && created.General.ID != nil:
		id = fmt.Sprintf("%d", *created.General.ID)
	}
	if id == "" {
		t.Fatalf("out-of-band create of macOS configuration profile %q returned no ID", name)
	}
	t.Cleanup(func() { _ = c.DeleteOSXConfigurationProfileByID(context.Background(), id) })

	got, err := c.GetOSXConfigurationProfileByID(ctx, id)
	if err != nil {
		t.Fatalf("reading back out-of-band macOS configuration profile %s: %v", id, err)
	}
	if got != nil && got.General != nil && got.General.Payloads != nil {
		serverPayload = string(*got.General.Payloads)
	}
	if serverPayload == "" {
		t.Fatalf("out-of-band macOS configuration profile %s returned an empty payload on read-back", id)
	}
	// The server returns the payload as a single line with no trailing newline;
	// configMinimal embeds it in a <<EOF heredoc, which needs the closing EOF on
	// its own line.
	if !strings.HasSuffix(serverPayload, "\n") {
		serverPayload += "\n"
	}
	return id, serverPayload
}

// mutatePPPCProfileChangeIdentifier simulates an out-of-band admin UI
// edit by fetching a PPPC profile, parsing its mobileconfig payload, and
// changing the Identifier of the first existing TCC service entry. The
// modified payload is then PUT back via the SDK. Mutating an existing
// service value rather than adding a new service avoids Jamf's
// invalid-service sanitisation pass (Jamf silently strips unknown or
// malformed service entries — see the DEVONthink Location case
// documented in helpers.go), so the modification is guaranteed to
// survive the round-trip and surface as drift on the next plan.
func mutatePPPCProfileChangeIdentifier(t *testing.T, profileID, newIdentifierValue string) {
	t.Helper()
	c := testhelpers.NewProClassicClient(t)
	ctx := context.Background()

	got, err := c.GetOSXConfigurationProfileByID(ctx, profileID)
	if err != nil {
		t.Fatalf("GetOSXConfigurationProfileByID(%s): %v", profileID, err)
	}
	if got == nil || got.General == nil || got.General.Payloads == nil {
		t.Fatalf("profile %s missing payload in GET response", profileID)
	}
	currentPayload := []byte(string(*got.General.Payloads))

	parsed, _, err := plisthelpers.ParsePlist(currentPayload)
	if err != nil {
		t.Fatalf("ParsePlist for profile %s: %v", profileID, err)
	}
	pcAny, ok := parsed["PayloadContent"]
	if !ok {
		t.Fatalf("profile %s payload missing PayloadContent", profileID)
	}
	pc, ok := pcAny.([]any)
	if !ok || len(pc) == 0 {
		t.Fatalf("profile %s PayloadContent not a non-empty array", profileID)
	}
	first, ok := pc[0].(map[string]any)
	if !ok {
		t.Fatalf("profile %s PayloadContent[0] not a dict", profileID)
	}
	servicesAny, ok := first["Services"]
	if !ok {
		t.Fatalf("profile %s PayloadContent[0] missing Services (not a PPPC payload)", profileID)
	}
	services, ok := servicesAny.(map[string]any)
	if !ok {
		t.Fatalf("profile %s PayloadContent[0].Services not a dict", profileID)
	}
	mutated := false
	for serviceKey, entriesAny := range services {
		entries, ok := entriesAny.([]any)
		if !ok || len(entries) == 0 {
			continue
		}
		entry, ok := entries[0].(map[string]any)
		if !ok {
			continue
		}
		entry["Identifier"] = newIdentifierValue
		t.Logf("admin-UI simulation: mutated Services[%q][0].Identifier → %q", serviceKey, newIdentifierValue)
		mutated = true
		break
	}
	if !mutated {
		t.Fatalf("profile %s had no Services[*] entries to mutate", profileID)
	}

	newPayloadBytes, err := plisthelpers.MarshalPlist(parsed)
	if err != nil {
		t.Fatalf("MarshalPlist for profile %s: %v", profileID, err)
	}

	newName := ""
	if got.General.Name != nil {
		newName = *got.General.Name
	}
	pxt := proclassic.PayloadsXMLText(newPayloadBytes)
	update := &proclassic.OsXConfigurationProfile{
		General: &proclassic.OsXConfigurationProfileGeneral{
			Name:     &newName,
			Payloads: &pxt,
		},
	}
	if err := c.UpdateOSXConfigurationProfileByID(ctx, profileID, update); err != nil {
		t.Fatalf("UpdateOSXConfigurationProfileByID(%s) during admin-UI simulation: %v", profileID, err)
	}
}

// TestAccResource_MacOSConfigurationProfile_AdminUIEdit_SurfacesAsDrift —
// the three-way payload compare must catch an out-of-band UI edit: the
// admin adds a service to the PPPC profile's Services dict via the Jamf
// Pro UI (simulated here via a direct SDK PUT). The next terraform plan
// against unchanged HCL must produce a non-empty plan so the drift is
// surfaced and the next apply re-aligns the server with the user's HCL.
//
// Before the three-way ModifyPlan landed, the legacy two-way compare
// with intersection semantics dropped the asymmetric Services key and
// silently reported "no changes" — the bug this test pins.
func TestAccResource_MacOSConfigurationProfile_AdminUIEdit_SurfacesAsDrift(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-drift-" + suffix
	payload := readFixture(t, "1Password__screen_recording_profile.mobileconfig")

	var profileID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configMinimal(name, payload),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["jamfplatform_pro_macos_configuration_profile.test"]
						if !ok {
							return fmt.Errorf("resource not found in state after Create")
						}
						profileID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				// Re-apply identical config immediately after Create: with the
				// three-way private-state references freshly populated, the
				// plan must be empty.
				Config: configMinimal(name, payload),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				// Out-of-band admin UI edit between this step's plan-refresh
				// and plan: the Identifier of a TCC service entry is mutated
				// server-side. The plan must report drift (non-empty), which
				// the immediately-following apply step will then reconcile
				// back to the HCL baseline.
				PreConfig: func() {
					if profileID == "" {
						t.Fatal("profileID empty; Step 1 Check should have captured it")
					}
					mutatePPPCProfileChangeIdentifier(t, profileID, "com.acceptance.injected-by-ui-simulation")
				},
				Config: configMinimal(name, payload),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
			},
		},
	})
}

// mutatePPPCProfileAddValidService is the add-direction counterpart to
// mutatePPPCProfileChangeIdentifier. It injects a real-shape entry under
// a known-valid TCC service key. Jamf's invalid-service sanitisation
// pass is *key-name-driven* — well-known TCC service keys
// (Accessibility, Reminders, SpeechRecognition, …) survive the round
// trip when the entry dict is well-formed.
func mutatePPPCProfileAddValidService(t *testing.T, profileID, serviceKey string) {
	t.Helper()
	c := testhelpers.NewProClassicClient(t)
	ctx := context.Background()

	got, err := c.GetOSXConfigurationProfileByID(ctx, profileID)
	if err != nil {
		t.Fatalf("GetOSXConfigurationProfileByID(%s): %v", profileID, err)
	}
	currentPayload := []byte(string(*got.General.Payloads))
	parsed, _, err := plisthelpers.ParsePlist(currentPayload)
	if err != nil {
		t.Fatalf("ParsePlist: %v", err)
	}
	services := parsed["PayloadContent"].([]any)[0].(map[string]any)["Services"].(map[string]any)
	services[serviceKey] = []any{
		map[string]any{
			"Authorization":   "Allow",
			"Identifier":      "com.acceptance.added-via-ui",
			"CodeRequirement": "anchor apple generic",
			"IdentifierType":  "bundleID",
		},
	}
	newPayloadBytes, err := plisthelpers.MarshalPlist(parsed)
	if err != nil {
		t.Fatalf("MarshalPlist: %v", err)
	}
	newName := ""
	if got.General.Name != nil {
		newName = *got.General.Name
	}
	pxt := proclassic.PayloadsXMLText(newPayloadBytes)
	if err := c.UpdateOSXConfigurationProfileByID(ctx, profileID, &proclassic.OsXConfigurationProfile{
		General: &proclassic.OsXConfigurationProfileGeneral{Name: &newName, Payloads: &pxt},
	}); err != nil {
		t.Fatalf("UpdateOSXConfigurationProfileByID add-service: %v", err)
	}
	t.Logf("admin-UI simulation: added Services[%q]", serviceKey)
}

// mutatePPPCProfileRemoveFirstService removes the first service entry
// from PayloadContent[0].Services.
func mutatePPPCProfileRemoveFirstService(t *testing.T, profileID string) {
	t.Helper()
	c := testhelpers.NewProClassicClient(t)
	ctx := context.Background()

	got, err := c.GetOSXConfigurationProfileByID(ctx, profileID)
	if err != nil {
		t.Fatalf("GetOSXConfigurationProfileByID(%s): %v", profileID, err)
	}
	currentPayload := []byte(string(*got.General.Payloads))
	parsed, _, err := plisthelpers.ParsePlist(currentPayload)
	if err != nil {
		t.Fatalf("ParsePlist: %v", err)
	}
	services := parsed["PayloadContent"].([]any)[0].(map[string]any)["Services"].(map[string]any)
	if len(services) < 2 {
		t.Fatalf("profile %s has fewer than 2 services; remove-test needs >= 2 to leave a non-empty Services dict", profileID)
	}
	for k := range services {
		t.Logf("admin-UI simulation: removed Services[%q]", k)
		delete(services, k)
		break
	}
	newPayloadBytes, err := plisthelpers.MarshalPlist(parsed)
	if err != nil {
		t.Fatalf("MarshalPlist: %v", err)
	}
	newName := ""
	if got.General.Name != nil {
		newName = *got.General.Name
	}
	pxt := proclassic.PayloadsXMLText(newPayloadBytes)
	if err := c.UpdateOSXConfigurationProfileByID(ctx, profileID, &proclassic.OsXConfigurationProfile{
		General: &proclassic.OsXConfigurationProfileGeneral{Name: &newName, Payloads: &pxt},
	}); err != nil {
		t.Fatalf("UpdateOSXConfigurationProfileByID remove-service: %v", err)
	}
}

// TestAccResource_MacOSConfigurationProfile_AdminUIAdd_SurfacesAsDrift —
// admin adds a new service to a PPPC profile via the UI; plan must
// produce non-empty.
func TestAccResource_MacOSConfigurationProfile_AdminUIAdd_SurfacesAsDrift(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-add-" + suffix
	payload := readFixture(t, "1Password__screen_recording_profile.mobileconfig")

	var profileID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configMinimal(name, payload),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["jamfplatform_pro_macos_configuration_profile.test"]
						if !ok {
							return fmt.Errorf("resource not found in state after Create")
						}
						profileID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				PreConfig: func() {
					if profileID == "" {
						t.Fatal("profileID empty")
					}
					mutatePPPCProfileAddValidService(t, profileID, "Accessibility")
				},
				Config: configMinimal(name, payload),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_AdminUIRemove_SurfacesAsDrift —
// admin removes a service from a PPPC profile via the UI; plan must
// produce non-empty.
func TestAccResource_MacOSConfigurationProfile_AdminUIRemove_SurfacesAsDrift(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-mcp-rm-" + suffix
	// DEVONthink fixture carries multiple services so removing one still
	// leaves a non-empty Services dict on the wire.
	payload := readFixture(t, "DEVONthink__pppcp_profile.mobileconfig")

	var profileID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: configMinimal(name, payload),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["jamfplatform_pro_macos_configuration_profile.test"]
						if !ok {
							return fmt.Errorf("resource not found in state after Create")
						}
						profileID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				PreConfig: func() {
					if profileID == "" {
						t.Fatal("profileID empty")
					}
					mutatePPPCProfileRemoveFirstService(t, profileID)
				},
				Config: configMinimal(name, payload),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccResource_MacOSConfigurationProfile_ScopeLimitationsClearWithEmptySet
// verifies that a declared-but-empty category clears its members. Under
// granular per-category scope ownership a declared `[]` is the clear gesture:
// the build emits an explicit empty element, and the update's read-merge-write
// re-emits every undeclared category so only the declared one changes.
// Omitting the category instead would leave it as configured outside
// Terraform. Uses a network-segment fixture (no LDAP needed).
func TestAccResource_MacOSConfigurationProfile_ScopeLimitationsClearWithEmptySet(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-macoscp-limclear-" + suffix
	seg := "tf-acc-netseg-macoscp-" + suffix
	payload := readFixture(t, "1Password__notifications_profile.mobileconfig")
	cfg := func(segs string) string {
		return fmt.Sprintf(`
resource "jamfplatform_pro_network_segment" "fixture" {
  name             = %q
  starting_address = "10.96.0.0"
  ending_address   = "10.96.0.255"
}

resource "jamfplatform_pro_macos_configuration_profile" "test" {
  general = {
    name     = %q
    payloads = <<EOF
%sEOF
  }
  scope = {
    targets = {
      all_computers = true
    }
    limitations = {
      network_segment_ids = [%s]
    }
  }
}
`, seg, name, payload, segs)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: cfg(`jamfplatform_pro_network_segment.fixture.id`),
				Check:  resource.TestCheckResourceAttr("jamfplatform_pro_macos_configuration_profile.test", "scope.limitations.network_segment_ids.#", "1"),
			},
			{
				// Clear to [] — the declared-but-empty category must be emitted
				// as an explicit empty element so the subtree replace clears it.
				// Implicit post-step empty-plan enforces the round-trip.
				Config: cfg(``),
				Check:  resource.TestCheckResourceAttr("jamfplatform_pro_macos_configuration_profile.test", "scope.limitations.network_segment_ids.#", "0"),
			},
		},
	})
}
