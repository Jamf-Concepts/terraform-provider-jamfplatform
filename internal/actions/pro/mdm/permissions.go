// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	devSDK "github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devices"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// Each MDM command action declares the SDK methods it reaches at Invoke time.
// Most actions route their API calls through the shared helpers in helpers.go
// (sendCommand -> SendMdmCommandV2; resolveManagementID's serial-number path ->
// ResolveDeviceIDBySerialNumber, which hits the same /v1/devices list endpoint
// as ListDevices; resolveUnlockToken -> ListMobileDevicesDetailV2 +
// GetMobileDeviceDetailV2). A few actions call the SDK directly. The declared
// list is the full reachable set so the rendered "Required Jamf privileges"
// table is honest; permissions_test.go asserts each list stays in sync with the
// methods reachable from the action's own file (direct calls plus the helpers
// it invokes) and with the SDK privilege registries.
//
// ResolveDeviceIDBySerialNumber is a resolver wrapper with no registry entry of
// its own; its required privilege is that of the underlying /v1/devices list
// (devices.ListDevices), so the serial-number-capable actions merge in the
// devices registry and declare ListDevices.

// resolveSerialMerged is the registry for actions that resolve a serial number
// through the Platform devices inventory in addition to issuing a Pro command.
var resolveSerialMerged = permissions.Merge(pro.Privileges, devSDK.Privileges)

// clearPasscode additionally looks up the unlock token from mobile-device
// inventory before issuing the clear-passcode command.
var clearPasscodeSDKMethods = []string{
	"SendMdmCommandV2",
	"ListMobileDevicesDetailV2",
	"GetMobileDeviceDetailV2",
	"ListDevices",
}
var clearPasscodePrivileges = permissions.Section(resolveSerialMerged, clearPasscodeSDKMethods...)

var clearRestrictionsPasswordSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var clearRestrictionsPasswordPrivileges = permissions.Section(resolveSerialMerged, clearRestrictionsPasswordSDKMethods...)

var deleteUserSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var deleteUserPrivileges = permissions.Section(resolveSerialMerged, deleteUserSDKMethods...)

var deviceLockSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var deviceLockPrivileges = permissions.Section(resolveSerialMerged, deviceLockSDKMethods...)

var disableLostModeSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var disableLostModePrivileges = permissions.Section(resolveSerialMerged, disableLostModeSDKMethods...)

var disableRemoteDesktopSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var disableRemoteDesktopPrivileges = permissions.Section(resolveSerialMerged, disableRemoteDesktopSDKMethods...)

var enableLostModeSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var enableLostModePrivileges = permissions.Section(resolveSerialMerged, enableLostModeSDKMethods...)

var enableRemoteDesktopSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var enableRemoteDesktopPrivileges = permissions.Section(resolveSerialMerged, enableRemoteDesktopSDKMethods...)

var triggerEnhancedLogCollectionSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var triggerEnhancedLogCollectionPrivileges = permissions.Section(resolveSerialMerged, triggerEnhancedLogCollectionSDKMethods...)

var cancelEnhancedLogCollectionSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var cancelEnhancedLogCollectionPrivileges = permissions.Section(resolveSerialMerged, cancelEnhancedLogCollectionSDKMethods...)

var logOutUserSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var logOutUserPrivileges = permissions.Section(resolveSerialMerged, logOutUserSDKMethods...)

var playLostModeSoundSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var playLostModeSoundPrivileges = permissions.Section(resolveSerialMerged, playLostModeSoundSDKMethods...)

var setAutoAdminPasswordSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var setAutoAdminPasswordPrivileges = permissions.Section(resolveSerialMerged, setAutoAdminPasswordSDKMethods...)

var unlockUserAccountSDKMethods = []string{
	"SendMdmCommandV2",
	"ListDevices",
}
var unlockUserAccountPrivileges = permissions.Section(resolveSerialMerged, unlockUserAccountSDKMethods...)

// sendBlankPush issues a blank-push command and resolves any serial numbers
// directly in its own file.
var sendBlankPushSDKMethods = []string{
	"SendMdmBlankPushV2",
	"ListDevices",
}
var sendBlankPushPrivileges = permissions.Section(resolveSerialMerged, sendBlankPushSDKMethods...)

// renewMdmProfile calls the Pro renew-profile endpoint directly.
var renewMdmProfileSDKMethods = []string{
	"RenewMdmProfileV1",
}
var renewMdmProfilePrivileges = permissions.Section(pro.Privileges, renewMdmProfileSDKMethods...)

// flushMdmCommands calls the Classic command-flush endpoint directly.
var flushMdmCommandsSDKMethods = []string{
	"DeleteCommandFlushByIDTypeIDStatus",
}
var flushMdmCommandsPrivileges = permissions.Section(proclassic.Privileges, flushMdmCommandsSDKMethods...)
