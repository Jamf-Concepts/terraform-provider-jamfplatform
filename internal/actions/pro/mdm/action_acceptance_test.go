// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package mdmactions_test

import (
	"fmt"
	"os"
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
//	JAMFPLATFORM_ACC_COMPUTER_SERIAL — a disposable enrolled computer
//	JAMFPLATFORM_ACC_MOBILE_SERIAL   — a disposable enrolled mobile device
//	JAMFPLATFORM_ACC_DESTRUCTIVE=1   — opt-in gate for commands that lock/erase
const (
	envComputerSerial = "JAMFPLATFORM_ACC_COMPUTER_SERIAL"
	envMobileSerial   = "JAMFPLATFORM_ACC_MOBILE_SERIAL"
	envDestructive    = "JAMFPLATFORM_ACC_DESTRUCTIVE"
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
    serial_number = %q
  }
}`, serial), "action.jamfplatform_pro_enable_remote_desktop.rd")

	disable := fireConfig(fmt.Sprintf(`
action "jamfplatform_pro_disable_remote_desktop" "rd" {
  config {
    serial_number = %q
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
    serial_number = %q
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
    serial_number = %q
    message       = "Acceptance test lock"
    pin           = "123456"
  }
}`, serial), "action.jamfplatform_pro_device_lock.lock")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: config}},
	})
}
