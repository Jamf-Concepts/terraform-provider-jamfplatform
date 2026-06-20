// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// webhookTimeoutAttributeTypes defines the timeout attribute types for the
// webhook resource operations.
var webhookTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// Supported enum values, wire-probed against a live tenant (WEBHOOK_SPIKE.md §3).

// webhookEvents is the set of accepted <event> values (23). UI labels differ in
// case (e.g. UI "ComputerCheckin" ⇒ wire "ComputerCheckIn").
var webhookEvents = []string{
	"ComputerAdded",
	"ComputerCheckIn",
	"ComputerInventoryCompleted",
	"ComputerPatchPolicyCompleted",
	"ComputerPolicyFinished",
	"ComputerPushCapabilityChanged",
	"DeviceAddedToDEP",
	"DeviceRateLimited",
	"JSSShutdown",
	"JSSStartup",
	"MobileDeviceCheckIn",
	"MobileDeviceCommandCompleted",
	"MobileDeviceEnrolled",
	"MobileDeviceInventoryCompleted",
	"MobileDevicePushSent",
	"MobileDeviceUnEnrolled",
	"PatchSoftwareTitleUpdated",
	"PushSent",
	"RestAPIOperation",
	"SCEPChallenge",
	"SmartGroupComputerMembershipChange",
	"SmartGroupMobileDeviceMembershipChange",
	"SmartGroupUserMembershipChange",
}

// smartGroupEvents is the subset of webhookEvents for which <smart_group_id> is
// meaningful. The server 409s when smart_group_id is supplied for any other
// event, so the cross-field validator mirrors this set.
var smartGroupEvents = map[string]struct{}{
	"SmartGroupComputerMembershipChange":     {},
	"SmartGroupMobileDeviceMembershipChange": {},
	"SmartGroupUserMembershipChange":         {},
}

// isSmartGroupEvent reports whether the event keys a smart group object and so
// accepts <smart_group_id>.
func isSmartGroupEvent(event string) bool {
	_, ok := smartGroupEvents[event]
	return ok
}

// webhookAuthTypes is the set of accepted <authentication_type> values exposed
// by this provider. The Jamf UI also offers "Mutual TLS Authentication"
// (wire: MTLS), but its certificate material is settable only through the
// legacy admin web form (HAR-proven) — not via any supported API — so a
// Terraform-managed MTLS webhook would be non-functional. MTLS is therefore
// intentionally excluded from the writable enum. (Reading/listing a
// pre-existing MTLS webhook still works: OneOf gates config, not state.)
var webhookAuthTypes = []string{
	"NONE",
	"BASIC",
	"HEADER",
	"HASH_SIGNATURE",
}

// webhookContentTypes is the accepted <content_type> set. The server silently
// coerces any other value to text/xml, so the OneOf prevents that silent drift.
var webhookContentTypes = []string{
	"application/json",
	"text/xml",
}

// webhookHashAlgorithms is the accepted <hash_algorithm> set (HASH_SIGNATURE).
var webhookHashAlgorithms = []string{
	"SHA256",
	"SHA512",
}

const (
	authTypeNone          = "NONE"
	authTypeBasic         = "BASIC"
	authTypeHeader        = "HEADER"
	authTypeHashSignature = "HASH_SIGNATURE"
)
