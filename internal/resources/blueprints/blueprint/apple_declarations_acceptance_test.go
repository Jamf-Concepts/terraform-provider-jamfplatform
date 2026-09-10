// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package blueprint_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/jamf/terraform-provider-jamfplatform/internal/testhelpers"
)

// appleDeclarationsConfig builds a blueprint carrying one apple_declarations component. The caller
// supplies the declaration list body so a step can vary only the declarations.
func appleDeclarationsConfig(scopeSuffix, name, declarations string) string {
	return testBlueprintConfig(smartGroupHCL(scopeSuffix), fmt.Sprintf(`
		resource "jamfplatform_blueprints_blueprint" "test_apple_decl" {
			name          = %q
			description   = "Acceptance test — safe to delete"
			deployed      = false
			device_groups = [jamfplatform_device_group.scope.id]

			component_blocks = [
				{
					name = "Apple Declarations"
					apple_declarations = {
						declaration = %s
					}
				},
			]
		}
	`, name, declarations))
}

// TestAccResource_Blueprint_AppleDeclarations covers create, update and plan stability for the
// component. The second step changes one payload and adds a declaration, so the ordered list is
// exercised rather than just a single entry; the third proves the round-trip settles, which is what
// would break if the platform reformatted a payload or echoed back the derived kind and payloadKey.
func TestAccResource_Blueprint_AppleDeclarations(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-apple-decl-" + suffix

	const oneDeclaration = `[
		{
			channel = "SYSTEM"
			type    = "com.apple.configuration.siri.settings"
			payload = jsonencode({
				Enabled              = true
				ForceProfanityFilter = true
			})
		},
	]`

	const twoDeclarations = `[
		{
			channel = "SYSTEM"
			type    = "com.apple.configuration.siri.settings"
			payload = jsonencode({
				Enabled                   = true
				ForceProfanityFilter      = true
				AllowUserGeneratedContent = false
			})
		},
		{
			channel = "SYSTEM"
			type    = "com.apple.configuration.diskmanagement.settings"
			payload = jsonencode({
				Restrictions = {
					ExternalStorage = "ReadOnly"
					NetworkStorage  = "Allowed"
				}
			})
		},
	]`

	resourceName := "jamfplatform_blueprints_blueprint.test_apple_decl"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: appleDeclarationsConfig("appledecl", name, oneDeclaration),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName,
						"component_blocks.0.apple_declarations.declaration.#", "1"),
					resource.TestCheckResourceAttr(resourceName,
						"component_blocks.0.apple_declarations.declaration.0.type",
						"com.apple.configuration.siri.settings"),
					resource.TestCheckResourceAttr(resourceName,
						"component_blocks.0.apple_declarations.declaration.0.channel", "SYSTEM"),
				),
			},
			{
				Config: appleDeclarationsConfig("appledecl", name, twoDeclarations),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName,
						"component_blocks.0.apple_declarations.declaration.#", "2"),
					// Order is meaningful — $PAYLOAD_n counts positions — so the second entry must
					// come back second rather than wherever a set would have put it.
					resource.TestCheckResourceAttr(resourceName,
						"component_blocks.0.apple_declarations.declaration.1.type",
						"com.apple.configuration.diskmanagement.settings"),
				),
			},
			{
				Config: appleDeclarationsConfig("appledecl", name, twoDeclarations),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccResource_Blueprint_AppleDeclarations_SeedOnlyKeys pins that keys Apple publishes only on
// its pre-release branch are accepted. This is the test that fails if the embedded table is ever
// regenerated from Apple's release branch alone: AllowSiriAI and ForceReduceSensitiveContent are
// offered by the Jamf interface today and absent from release, so a release-only table would reject
// a configuration that works.
func TestAccResource_Blueprint_AppleDeclarations_SeedOnlyKeys(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-apple-decl-seed-" + suffix

	const seedKeys = `[
		{
			channel = "SYSTEM"
			type    = "com.apple.configuration.siri.settings"
			payload = jsonencode({
				Enabled                     = true
				AllowSiriAI                 = false
				ForceReduceSensitiveContent = true
			})
		},
	]`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: appleDeclarationsConfig("appleseed", name, seedKeys),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"jamfplatform_blueprints_blueprint.test_apple_decl", "id"),
				),
			},
		},
	})
}

// TestAccResource_Blueprint_AppleDeclarations_PayloadReferences covers an asset reference resolved
// by position. `$PAYLOAD_1` names the first declaration in the same component, which is why the
// list is ordered and payload_key is derived from the index rather than authored.
func TestAccResource_Blueprint_AppleDeclarations_PayloadReferences(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-apple-decl-ref-" + suffix

	const withReference = `[
		{
			channel = "SYSTEM"
			type    = "com.apple.asset.data"
			payload = jsonencode({
				Reference = {
					DataURL        = "https://cdn.example.com/ddm/sudoers-config.zip"
					ContentType    = "application/zip"
					"Hash-SHA-256" = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
				}
				Authentication = { Type = "MDM" }
			})
		},
		{
			channel = "SYSTEM"
			type    = "com.apple.configuration.services.configuration-files"
			payload = jsonencode({
				ServiceType        = "com.apple.sudo"
				DataAssetReference = "$PAYLOAD_1"
			})
		},
	]`

	resourceName := "jamfplatform_blueprints_blueprint.test_apple_decl"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: appleDeclarationsConfig("appleref", name, withReference),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName,
						"component_blocks.0.apple_declarations.declaration.#", "2"),
					// The asset must stay first, or the reference below it points elsewhere.
					resource.TestCheckResourceAttr(resourceName,
						"component_blocks.0.apple_declarations.declaration.0.type",
						"com.apple.asset.data"),
				),
			},
			{
				Config: appleDeclarationsConfig("appleref", name, withReference),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccResource_Blueprint_AppleDeclarations_SchemaValidation is the plan-time rejection matrix.
// Each step corresponds to a rule established by probing the live service and reading the result
// out of the Jamf Pro editor — the platform accepts every one of these bodies and reports success,
// so a plan error is the only thing standing between the author and a declaration that silently
// never applies.
//
// The expected patterns are kept short on purpose: Terraform wraps diagnostic text at roughly 80
// columns, so a longer pattern can straddle a line break and never match.
func TestAccResource_Blueprint_AppleDeclarations_SchemaValidation(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-apple-decl-schema-" + suffix

	declaration := func(declarationType, payload string) string {
		return fmt.Sprintf(`[
			{
				channel = "SYSTEM"
				type    = %q
				payload = %s
			},
		]`, declarationType, payload)
	}
	config := func(declarationType, payload string) string {
		return appleDeclarationsConfig("appleschema", name, declaration(declarationType, payload))
	}

	const siri = "com.apple.configuration.siri.settings"
	const passcode = "com.apple.configuration.passcode.settings"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				// A key Apple does not declare. The platform discards it silently.
				Config:      config(siri, `jsonencode({ ZzNotAKey = true })`),
				ExpectError: regexp.MustCompile(`does not declare`),
				PlanOnly:    true,
			},
			{
				// Wrong case. Unlike a configuration profile payload, where Jamf restores Apple's
				// spelling, a declaration key is discarded outright.
				Config:      config(siri, `jsonencode({ forceprofanityfilter = true })`),
				ExpectError: regexp.MustCompile(`not spelled the way`),
				PlanOnly:    true,
			},
			{
				// A wrong-cased declaration type renders no card at all in the editor.
				Config:      config("com.apple.configuration.Siri.Settings", `jsonencode({ Enabled = true })`),
				ExpectError: regexp.MustCompile(`Unknown Apple declaration type`),
				PlanOnly:    true,
			},
			{
				Config:      config("com.apple.configuration.not.a.thing", `jsonencode({ Enabled = true })`),
				ExpectError: regexp.MustCompile(`Unknown Apple declaration type`),
				PlanOnly:    true,
			},
			{
				// A string where Apple declares a boolean. The key is recognised, the value dropped.
				Config:      config(siri, `jsonencode({ Enabled = "yes" })`),
				ExpectError: regexp.MustCompile(`declares this as a boolean`),
				PlanOnly:    true,
			},
			{
				// Outside a declared rangelist. The editor leaves the field unpopulated.
				Config: config("com.apple.configuration.diskmanagement.settings",
					`jsonencode({ Restrictions = { ExternalStorage = "Fortnightly" } })`),
				ExpectError: regexp.MustCompile(`not one of the values`),
				PlanOnly:    true,
			},
			{
				// Outside a declared numeric range. Jamf stores this unchanged — the device is what
				// rejects it — which is why it is an error here rather than a note.
				Config:      config(passcode, `jsonencode({ MinimumLength = 999 })`),
				ExpectError: regexp.MustCompile(`outside the range`),
				PlanOnly:    true,
			},
			{
				// A required key omitted, reported on absence rather than on anything present.
				Config: config("com.apple.configuration.math.settings",
					`jsonencode({ Calculator = { InputModes = { UnitConversion = true } } })`),
				ExpectError: regexp.MustCompile(`required`),
				PlanOnly:    true,
			},
			{
				// A status subscription naming an item Apple does not publish. Apple types these as
				// plain strings, so nothing else can catch the typo.
				Config: config("com.apple.configuration.management.status-subscriptions",
					`jsonencode({ StatusItems = [{ Name = "device.identifier.not-a-thing" }] })`),
				ExpectError: regexp.MustCompile(`Unknown status item`),
				PlanOnly:    true,
			},
			{
				// Not JSON at all.
				Config:      config(siri, `"not json"`),
				ExpectError: regexp.MustCompile(`not a JSON object`),
				PlanOnly:    true,
			},
			{
				// A clean declaration plans and applies, so the matrix above cannot be satisfied by
				// a validator that simply rejects everything. A null is tolerated: the platform
				// discards one rather than storing it, so it is never a type error.
				Config: config(siri, `jsonencode({ Enabled = true, ForceProfanityFilter = true, AllowWhileLocked = null })`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"jamfplatform_blueprints_blueprint.test_apple_decl", "id"),
				),
			},
		},
	})
}

// TestAccResource_Blueprint_CustomDeclarations_SchemaValidation pins that the same checks apply to
// the older component. It delivers identical declarations to the same devices with the same absence
// of server-side validation, so leaving it unchecked would only move the silent failure.
//
// The diagnostic names the declaration type rather than an element index: custom_declarations backs
// its declarations with a set, which has no stable position to address.
func TestAccResource_Blueprint_CustomDeclarations_SchemaValidation(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-custom-decl-schema-" + suffix

	config := func(payload string) string {
		return testBlueprintConfig(smartGroupHCL("customschema"), fmt.Sprintf(`
			resource "jamfplatform_blueprints_blueprint" "test_custom_schema" {
				name          = %q
				description   = "Acceptance test — safe to delete"
				deployed      = false
				device_groups = [jamfplatform_device_group.scope.id]

				component_blocks = [
					{
						name = "Custom Declarations"
						custom_declarations = {
							declaration = [{
								channel = "SYSTEM"
								kind    = "CONFIGURATION"
								type    = "com.apple.configuration.siri.settings"
								payload = %s
							}]
						}
					},
				]
			}
		`, name, payload))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config:      config(`jsonencode({ ZzNotAKey = true })`),
				ExpectError: regexp.MustCompile(`does not declare`),
				PlanOnly:    true,
			},
			{
				// The finding must identify which declaration it came from, since the path cannot.
				Config:      config(`jsonencode({ Enabled = "yes" })`),
				ExpectError: regexp.MustCompile(`In declaration com\.apple\.configuration`),
				PlanOnly:    true,
			},
			{
				Config: config(`jsonencode({ Enabled = true })`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"jamfplatform_blueprints_blueprint.test_custom_schema", "id"),
				),
			},
		},
	})
}

// TestAccResource_Blueprint_RawComponentSkipsSchemaValidation pins the escape hatch, which is what
// makes erroring on an unrecognised name defensible: there is no switch to soften a finding, so
// there has to be somewhere to put a declaration the tables are wrong about. The payload below
// would fail every check in the matrix above.
//
// It also covers the nesting idiom, which is easy to get wrong: raw_component's configuration is a
// map of strings, and a value is nested by JSON-encoding it. That round-trips byte-identically,
// which the second step proves.
func TestAccResource_Blueprint_RawComponentSkipsSchemaValidation(t *testing.T) {
	testhelpers.AccPreCheck(t)
	suffix := testhelpers.RunSuffix()
	name := "tf-acc-raw-decl-" + suffix

	config := testBlueprintConfig(smartGroupHCL("rawdecl"), fmt.Sprintf(`
		resource "jamfplatform_blueprints_blueprint" "test_raw_decl" {
			name          = %q
			description   = "Acceptance test — safe to delete"
			deployed      = false
			device_groups = [jamfplatform_device_group.scope.id]

			component_blocks = [
				{
					name = "Unchecked"
					raw_component = [
						{
							identifier = "com.jamf.ddm-strict"
							configuration = {
								declarations = jsonencode([
									{
										channelType = "SYSTEM"
										kind        = "CONFIGURATION"
										type        = "com.apple.configuration.siri.settings"
										payload = {
											ZzNotAKey = true
											Enabled   = "yes"
										}
									},
								])
							}
						},
					]
				},
			]
		}
	`, name))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlueprintResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"jamfplatform_blueprints_blueprint.test_raw_decl", "id"),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
