// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	devSDK "github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devices"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// --- SDK drift guards: every declared method must exist in its registry ---

func TestClearPasscodeSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, clearPasscodeSDKMethods...); len(missing) > 0 {
		t.Fatalf("clearPasscodeSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestClearRestrictionsPasswordSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, clearRestrictionsPasswordSDKMethods...); len(missing) > 0 {
		t.Fatalf("clearRestrictionsPasswordSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestDeleteUserSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, deleteUserSDKMethods...); len(missing) > 0 {
		t.Fatalf("deleteUserSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestDeviceLockSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, deviceLockSDKMethods...); len(missing) > 0 {
		t.Fatalf("deviceLockSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestDisableLostModeSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, disableLostModeSDKMethods...); len(missing) > 0 {
		t.Fatalf("disableLostModeSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestDisableRemoteDesktopSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, disableRemoteDesktopSDKMethods...); len(missing) > 0 {
		t.Fatalf("disableRemoteDesktopSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestEnableLostModeSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, enableLostModeSDKMethods...); len(missing) > 0 {
		t.Fatalf("enableLostModeSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestEnableRemoteDesktopSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, enableRemoteDesktopSDKMethods...); len(missing) > 0 {
		t.Fatalf("enableRemoteDesktopSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestLogOutUserSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, logOutUserSDKMethods...); len(missing) > 0 {
		t.Fatalf("logOutUserSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestPlayLostModeSoundSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, playLostModeSoundSDKMethods...); len(missing) > 0 {
		t.Fatalf("playLostModeSoundSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestSetAutoAdminPasswordSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, setAutoAdminPasswordSDKMethods...); len(missing) > 0 {
		t.Fatalf("setAutoAdminPasswordSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestUnlockUserAccountSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, unlockUserAccountSDKMethods...); len(missing) > 0 {
		t.Fatalf("unlockUserAccountSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestSendBlankPushSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(resolveSerialMerged, sendBlankPushSDKMethods...); len(missing) > 0 {
		t.Fatalf("sendBlankPushSDKMethods not present in SDK registries (drift): %v", missing)
	}
}

func TestRenewMdmProfileSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, renewMdmProfileSDKMethods...); len(missing) > 0 {
		t.Fatalf("renewMdmProfileSDKMethods not present in pro.Privileges (drift): %v", missing)
	}
}

func TestFlushMdmCommandsSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(proclassic.Privileges, flushMdmCommandsSDKMethods...); len(missing) > 0 {
		t.Fatalf("flushMdmCommandsSDKMethods not present in proclassic.Privileges (drift): %v", missing)
	}
}

// --- file <-> declared-method reachability guards ---
//
// Each MDM command action reaches the SDK two ways: via direct calls on its
// configured client surfaces (a.client.*, a.devices.*, a.classic.*) and via the
// shared helpers in helpers.go (sendCommand -> SendMdmCommandV2;
// resolveManagementID's serial path -> ResolveDeviceIDBySerialNumber, which is a
// resolver wrapper over the /v1/devices list documented as ListDevices;
// resolveUnlockToken -> ListMobileDevicesDetailV2 + GetMobileDeviceDetailV2).
//
// reachableSDKMethods reads an action's own source file, resolves the helper
// indirections it invokes, and returns the full set of SDK privilege-registry
// method names it actually exercises — the honest input to the privileges
// table. The per-action test asserts this equals the declared list.

// helperSDKMethods maps a shared helper to the registry method names it calls.
// ResolveDeviceIDBySerialNumber is intentionally surfaced as "ListDevices": the
// resolver has no registry entry of its own and hits the same /v1/devices list
// endpoint, so its required privilege is that of ListDevices.
var helperSDKMethods = map[string][]string{
	"sendCommand":         {"SendMdmCommandV2"},
	"resolveManagementID": {"ListDevices"}, // serial path -> ResolveDeviceIDBySerialNumber -> /v1/devices
	"resolveUnlockToken":  {"ListMobileDevicesDetailV2", "GetMobileDeviceDetailV2"},
}

// directSDKAliases maps SDK methods callable directly in an action file to their
// registry method names (identity for most; the serial resolver folds onto
// ListDevices for the privilege it actually requires).
var directSDKAliases = map[string]string{
	"ResolveDeviceIDBySerialNumber": "ListDevices",
}

// knownRegistryMethod reports whether name is a method tracked by any of the
// SDK registries this package draws on.
func knownRegistryMethod(name string) bool {
	if _, ok := pro.Privileges[name]; ok {
		return true
	}
	if _, ok := devSDK.Privileges[name]; ok {
		return true
	}
	if _, ok := proclassic.Privileges[name]; ok {
		return true
	}
	return false
}

func reachableSDKMethods(t *testing.T, filename string) []string {
	t.Helper()
	src, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("reading %s: %v", filename, err)
	}
	text := string(src)

	set := map[string]bool{}

	// Direct SDK client calls: a.client.X(, a.devices.X(, a.classic.X(.
	directRe := regexp.MustCompile(`\ba\.(?:client|devices|classic)\.([A-Za-z0-9]+)\(`)
	for _, m := range directRe.FindAllStringSubmatch(text, -1) {
		name := m[1]
		if alias, ok := directSDKAliases[name]; ok {
			name = alias
		}
		if knownRegistryMethod(name) {
			set[name] = true
		}
	}

	// Helper indirections: a.<helper>(.
	helperRe := regexp.MustCompile(`\ba\.([A-Za-z0-9]+)\(`)
	for _, m := range helperRe.FindAllStringSubmatch(text, -1) {
		for _, name := range helperSDKMethods[m[1]] {
			set[name] = true
		}
	}

	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func assertMatch(t *testing.T, filename string, declared []string) {
	t.Helper()
	got := reachableSDKMethods(t, filename)
	want := append([]string(nil), declared...)
	sort.Strings(want)

	gotSet := map[string]bool{}
	for _, m := range got {
		gotSet[m] = true
	}
	wantSet := map[string]bool{}
	for _, m := range want {
		wantSet[m] = true
	}

	var undeclared, unused []string
	for m := range gotSet {
		if !wantSet[m] {
			undeclared = append(undeclared, m)
		}
	}
	for m := range wantSet {
		if !gotSet[m] {
			unused = append(unused, m)
		}
	}
	sort.Strings(undeclared)
	sort.Strings(unused)

	if len(undeclared) > 0 {
		t.Errorf("%s reaches SDK methods missing from declared list: %v", filename, undeclared)
	}
	if len(unused) > 0 {
		t.Errorf("declared list has methods %s does not reach: %v", filename, unused)
	}
}

func TestClearPasscodeSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "clear_passcode.go", clearPasscodeSDKMethods)
}

func TestClearRestrictionsPasswordSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "clear_restrictions_password.go", clearRestrictionsPasswordSDKMethods)
}

func TestDeleteUserSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "delete_user.go", deleteUserSDKMethods)
}

func TestDeviceLockSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "device_lock.go", deviceLockSDKMethods)
}

func TestDisableLostModeSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "disable_lost_mode.go", disableLostModeSDKMethods)
}

func TestDisableRemoteDesktopSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "disable_remote_desktop.go", disableRemoteDesktopSDKMethods)
}

func TestEnableLostModeSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "enable_lost_mode.go", enableLostModeSDKMethods)
}

func TestEnableRemoteDesktopSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "enable_remote_desktop.go", enableRemoteDesktopSDKMethods)
}

func TestLogOutUserSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "log_out_user.go", logOutUserSDKMethods)
}

func TestPlayLostModeSoundSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "play_lost_mode_sound.go", playLostModeSoundSDKMethods)
}

func TestSetAutoAdminPasswordSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "set_auto_admin_password.go", setAutoAdminPasswordSDKMethods)
}

func TestUnlockUserAccountSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "unlock_user_account.go", unlockUserAccountSDKMethods)
}

func TestSendBlankPushSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "send_blank_push.go", sendBlankPushSDKMethods)
}

func TestRenewMdmProfileSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "renew_mdm_profile.go", renewMdmProfileSDKMethods)
}

func TestFlushMdmCommandsSDKMethods_MatchFile(t *testing.T) {
	assertMatch(t, "flush_mdm_commands.go", flushMdmCommandsSDKMethods)
}

// --- rendered-table guards: each section actually produced a privileges block ---

func TestSendCommandPrivileges_Rendered(t *testing.T) {
	// All serial-resolving command actions render the execute command privilege
	// plus the devices read privilege.
	for _, section := range []string{deviceLockPrivileges, clearPasscodePrivileges} {
		if !strings.Contains(section, "execute:pro:computer-commands") {
			t.Fatalf("section did not render the command execute privilege:\n%s", section)
		}
		if !strings.Contains(section, "read:pro:devices") {
			t.Fatalf("section did not render the devices read privilege:\n%s", section)
		}
	}
}

func TestRenewMdmProfilePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(renewMdmProfilePrivileges, "execute:pro:mobile-device-commands") {
		t.Fatalf("renewMdmProfilePrivileges did not render the expected privilege:\n%s", renewMdmProfilePrivileges)
	}
}

func TestFlushMdmCommandsPrivileges_Rendered(t *testing.T) {
	if !strings.Contains(flushMdmCommandsPrivileges, "delete:pro:computer-commands") {
		t.Fatalf("flushMdmCommandsPrivileges did not render the expected privilege:\n%s", flushMdmCommandsPrivileges)
	}
}
