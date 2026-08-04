// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package mdmactions_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// These acceptance tests invoke real MDM commands against live enrolled devices.
// They are gated on serial-number environment variables so the operator can
// supply and swap target devices without editing the suite; each test skips
// when its variable is unset. Invoking an action from config requires
// lifecycle.action_trigger (Terraform >= 1.14).
//
//	JAMFPLATFORM_ACC_COMPUTER_SERIAL   — a disposable enrolled computer
//	JAMFPLATFORM_ACC_COMPUTER_SERIAL_2 — a SECOND disposable enrolled computer,
//	                                     needed only by the multi-device batch tests
//	JAMFPLATFORM_ACC_MOBILE_SERIAL     — a disposable enrolled mobile device
//	JAMFPLATFORM_ACC_DESTRUCTIVE=1     — opt-in gate for commands that lock/erase
const (
	envComputerSerial  = "JAMFPLATFORM_ACC_COMPUTER_SERIAL"
	envComputerSerial2 = "JAMFPLATFORM_ACC_COMPUTER_SERIAL_2"
	envMobileSerial    = "JAMFPLATFORM_ACC_MOBILE_SERIAL"
	envDestructive     = "JAMFPLATFORM_ACC_DESTRUCTIVE"
)

// fireConfig wraps an action block with a terraform_data trigger that invokes it
// after the trigger is created, ordering the action correctly behind any
// fixtures. A successful apply means the command was accepted by the server.
func fireConfig(actionBlock, actionRef string) string {
	return fmt.Sprintf(`
%s

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [%s]
    }
  }
}
`, actionBlock, actionRef)
}

// TestAccAction_ProSendBlankPush_Invoke is the benign smoke test: a blank push
// has no device-side effect beyond prompting a check-in.
func TestAccAction_ProSendBlankPush_Invoke(t *testing.T) {
	serial := os.Getenv(envComputerSerial)
	if serial == "" {
		t.Skipf("%s not set; skipping blank push acceptance test", envComputerSerial)
	}
	testhelpers.AccPreCheck(t)

	config := fireConfig(fmt.Sprintf(`
action "jamfplatform_pro_send_blank_push" "push" {
  config {
    serial_numbers = [%q]
  }
}`, serial), "action.jamfplatform_pro_send_blank_push.push")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: config}},
	})
}

// TestAccAction_ProRemoteDesktop_Invoke enables then disables Remote Desktop on a
// computer — a reversible round-trip.
func TestAccAction_ProRemoteDesktop_Invoke(t *testing.T) {
	serial := os.Getenv(envComputerSerial)
	if serial == "" {
		t.Skipf("%s not set; skipping remote desktop acceptance test", envComputerSerial)
	}
	testhelpers.AccPreCheck(t)

	enable := fireConfig(fmt.Sprintf(`
action "jamfplatform_pro_enable_remote_desktop" "rd" {
  config {
    serial_numbers = [%q]
  }
}`, serial), "action.jamfplatform_pro_enable_remote_desktop.rd")

	disable := fireConfig(fmt.Sprintf(`
action "jamfplatform_pro_disable_remote_desktop" "rd" {
  config {
    serial_numbers = [%q]
  }
}`, serial), "action.jamfplatform_pro_disable_remote_desktop.rd")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: enable},
			{Config: disable},
		},
	})
}

// TestAccAction_ProClearRestrictionsPassword_Invoke clears the Screen Time
// passcode on a mobile device (benign, no user data loss).
func TestAccAction_ProClearRestrictionsPassword_Invoke(t *testing.T) {
	serial := os.Getenv(envMobileSerial)
	if serial == "" {
		t.Skipf("%s not set; skipping mobile command acceptance test", envMobileSerial)
	}
	testhelpers.AccPreCheck(t)

	config := fireConfig(fmt.Sprintf(`
action "jamfplatform_pro_clear_restrictions_password" "clear" {
  config {
    serial_numbers = [%q]
  }
}`, serial), "action.jamfplatform_pro_clear_restrictions_password.clear")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: config}},
	})
}

// TestAccAction_ProDeviceLock_Invoke locks a computer. Destructive (the device
// is locked until the PIN is entered), so it carries the extra opt-in gate.
func TestAccAction_ProDeviceLock_Invoke(t *testing.T) {
	serial := os.Getenv(envComputerSerial)
	if serial == "" || os.Getenv(envDestructive) == "" {
		t.Skipf("set %s and %s=1 to run the destructive device lock acceptance test", envComputerSerial, envDestructive)
	}
	testhelpers.AccPreCheck(t)

	config := fireConfig(fmt.Sprintf(`
action "jamfplatform_pro_device_lock" "lock" {
  config {
    serial_numbers = [%q]
    message        = "Acceptance test lock"
    pin            = "123456"
  }
}`, serial), "action.jamfplatform_pro_device_lock.lock")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: config}},
	})
}

// TestAccAction_ProRemoteDesktop_MultiDeviceBatch is the acceptance counterpart
// to the plural selectors: two computers commanded by ONE request, then reversed.
// Remote Desktop is the right vehicle because it is benign and reversible.
//
// Requires a second computer serial; skips otherwise, since a one-element list
// would not actually exercise batching.
func TestAccAction_ProRemoteDesktop_MultiDeviceBatch(t *testing.T) {
	first, second := os.Getenv(envComputerSerial), os.Getenv(envComputerSerial2)
	if first == "" || second == "" {
		t.Skipf("set %s and %s to run the multi-device batch acceptance test", envComputerSerial, envComputerSerial2)
	}
	testhelpers.AccPreCheck(t)

	enable := fireConfig(fmt.Sprintf(`
action "jamfplatform_pro_enable_remote_desktop" "rd" {
  config {
    serial_numbers = [%q, %q]
  }
}`, first, second), "action.jamfplatform_pro_enable_remote_desktop.rd")

	disable := fireConfig(fmt.Sprintf(`
action "jamfplatform_pro_disable_remote_desktop" "rd" {
  config {
    serial_numbers = [%q, %q]
  }
}`, first, second), "action.jamfplatform_pro_disable_remote_desktop.rd")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: enable},
			{Config: disable},
		},
	})
}

// TestAccAction_ProSendBlankPush_MixedSelectors covers the additive path: a
// management id and a serial number in one invocation must both be targeted,
// rather than one selector silently winning.
func TestAccAction_ProSendBlankPush_MixedSelectors(t *testing.T) {
	first, second := os.Getenv(envComputerSerial), os.Getenv(envComputerSerial2)
	if first == "" || second == "" {
		t.Skipf("set %s and %s to run the mixed-selector acceptance test", envComputerSerial, envComputerSerial2)
	}
	testhelpers.AccPreCheck(t)

	// Resolve the second serial through the provider itself so the config
	// exercises management_ids alongside serial_numbers.
	config := fireConfig(fmt.Sprintf(`
data "jamfplatform_devices" "targets" {}

locals {
  second_id = one([
    for d in data.jamfplatform_devices.targets.devices : d.id
    if d.serial_number == %q
  ])
}

action "jamfplatform_pro_send_blank_push" "push" {
  config {
    serial_numbers = [%q]
    management_ids = [local.second_id]
  }
}`, second, first), "action.jamfplatform_pro_send_blank_push.push")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: config}},
	})
}

// TestAccAction_ProCommand_NoTargetFails asserts the plan-time guard: with both
// selectors omitted the run must fail during validation, not part-way through an
// apply that has already commanded nothing.
func TestAccAction_ProCommand_NoTargetFails(t *testing.T) {
	testhelpers.AccPreCheck(t)

	config := fireConfig(`
action "jamfplatform_pro_enable_remote_desktop" "rd" {
  config {}
}`, "action.jamfplatform_pro_enable_remote_desktop.rd")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		// actionvalidator.AtLeastOneOf reports "Missing Attribute Configuration"
		// and names the selectors as a bracketed, space-free list. Both anchors
		// are short enough that Terraform's ~80-column wrapping cannot split
		// them, so neither straddles a line break.
		Steps: []resource.TestStep{{
			Config:      config,
			ExpectError: regexp.MustCompile(`(?s)Missing Attribute Configuration.*\[management_ids,serial_numbers\]`),
		}},
	})
}

// TestAccAction_ProCommand_EmptyListFails asserts the SizeAtLeast(1) validator:
// an empty list is not a valid way to say "no devices". Jamf Pro rejects an
// empty id list outright, so catching it at plan time is the better failure.
func TestAccAction_ProCommand_EmptyListFails(t *testing.T) {
	testhelpers.AccPreCheck(t)

	config := fireConfig(`
action "jamfplatform_pro_enable_remote_desktop" "rd" {
  config {
    serial_numbers = []
  }
}`, "action.jamfplatform_pro_enable_remote_desktop.rd")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      config,
			ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value.*at least 1 element`),
		}},
	})
}

// TestAccAction_ProCommand_UnknownSerialFails covers the bulk resolver's error
// path: an unmatched serial must be named. This is the one place a bad target is
// attributable — the command POST itself fails without identifying any device.
func TestAccAction_ProCommand_UnknownSerialFails(t *testing.T) {
	testhelpers.AccPreCheck(t)

	config := fireConfig(`
action "jamfplatform_pro_enable_remote_desktop" "rd" {
  config {
    serial_numbers = ["NOSUCHSERIAL0001"]
  }
}`, "action.jamfplatform_pro_enable_remote_desktop.rd")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      config,
			ExpectError: regexp.MustCompile(`NOSUCHSERIAL0001`),
		}},
	})
}
