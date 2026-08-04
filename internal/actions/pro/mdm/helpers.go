// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/actionvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	devSDK "github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devices"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// minJamfProVersion is empty: these management commands ride continuously
// deployed Jamf Pro endpoints with no meaningful version floor.
const minJamfProVersion = ""

// secretAttrNote is appended to the description of every action attribute that
// carries a secret the user supplies (a PIN, password or unlock token).
//
// Such an attribute can be given NEITHER protection the framework offers:
//
//   - Sensitive does not exist on action schema attributes. The field is absent,
//     and IsSensitive() is hardcoded to return false ("action schema attributes
//     cannot be Sensitive").
//   - WriteOnly exists on action attributes and compiles, but setting it makes
//     the attribute impossible to use. Action config validation hardcodes
//     WriteOnlyAttributesAllowed: false (fwserver/server_validateactionconfig.go,
//     on the stated grounds that the capability "is only valid for resource
//     schemas"), while the shared SchemaValidate still applies the resource
//     write-only gate (fwserver/attribute_validation.go). Any non-null value for
//     a write-only action attribute therefore fails validation with "WriteOnly
//     Attribute Not Allowed".
//
// Note this is unconditional and version-independent: the capability is a
// hardcoded false, not a negotiated one, so no Terraform upgrade fixes it —
// despite the diagnostic blaming "Terraform 1.11 and later". Observed on
// framework v1.19.0 with Terraform v1.15.8. The framework is internally
// inconsistent here (the field is offered and documented but cannot be used),
// which is worth raising upstream.
//
// So the choice is an attribute that works with its value visible, or one nobody
// can use. We take the former and say so here. Do not re-add WriteOnly: it
// breaks every configuration that sets the attribute, and no device-less test
// catches it. TestActionAttributes_AreNotWriteOnly guards this.
const secretAttrNote = " This value appears in Terraform plan output and should be supplied from a variable or secret store rather than committed."

// batchNote is appended to the description of every action that targets devices
// in bulk, so the single-request behaviour and its all-or-nothing failure mode
// are visible in the rendered docs rather than buried in this package.
const batchNote = "\n\n" +
	"All targeted devices are commanded in a single request. If any one device cannot be " +
	"resolved the whole invocation fails and no device is commanded, so a large batch is " +
	"harder to diagnose than several smaller ones. Jamf Pro publishes no maximum batch size."

// blankPushBatchNote replaces batchNote for send_blank_push. That endpoint is
// the one batch surface here that reports per-device outcomes: it accepts the
// request and names the devices that failed, rather than failing wholesale.
const blankPushBatchNote = "\n\n" +
	"All targeted devices are pushed in a single request. Devices that do not accept the push " +
	"are reported individually as a warning; the invocation itself still succeeds."

// singleTargetNote is appended to the description of the two command actions
// that cannot batch, so the asymmetry with their siblings reads as deliberate.
const singleTargetNote = "\n\n" +
	"This action targets one device at a time, unlike the other management commands, because " +
	"its payload carries a value specific to that device. Use `for_each` to cover several devices."

// batchWarnThreshold is the batch size above which sendCommandBatch warns.
//
// POST /v2/mdm/commands takes clientData as an array and Jamf Pro documents no
// maximum for it: the shipped spec carries no maxItems, and wire probes were
// accepted at 5,000 entries (the sibling blank-push endpoint was accepted at
// 10,000). Where a real cap does exist Jamf documents it — GET /v1/mdm/commands
// states "Limited to 40 UUIDs" and defines a 414 for breaching it — so the
// absence here is meaningful rather than an omission.
//
// The threshold is therefore not a limit but a diagnosability warning: the POST
// is all-or-nothing, and an unresolvable managementId fails the entire request
// with an opaque 500 SYSTEM_EXCEPTION that names no id. The larger the batch,
// the harder that is to attribute.
const batchWarnThreshold = 500

// MDM command type discriminators. The SDK aliases MDMCommandType to a bare
// string and ships no enum constants, so the supported subset is pinned here.
const (
	cmdDeviceLock                = "DEVICE_LOCK"
	cmdEnableLostMode            = "ENABLE_LOST_MODE"
	cmdDisableLostMode           = "DISABLE_LOST_MODE"
	cmdPlayLostModeSound         = "PLAY_LOST_MODE_SOUND"
	cmdEnableRemoteDesktop       = "ENABLE_REMOTE_DESKTOP"
	cmdDisableRemoteDesktop      = "DISABLE_REMOTE_DESKTOP"
	cmdClearRestrictionsPassword = "CLEAR_RESTRICTIONS_PASSWORD"
	cmdDeleteUser                = "DELETE_USER"
	cmdLogOutUser                = "LOG_OUT_USER"
	cmdSetAutoAdminPassword      = "SET_AUTO_ADMIN_PASSWORD"
	cmdUnlockUserAccount         = "UNLOCK_USER_ACCOUNT"
	cmdClearPasscode             = "CLEAR_PASSCODE"
)

// mdmAction shares Configure logic across the MDM command actions. It holds the
// three client surfaces the package needs: the Jamf Pro client (send-command,
// blank-push, renew-profile, mobile-device lookup), the Platform devices client
// (serial-number resolution, since a Platform device id is the Jamf Pro
// managementId), and the ProClassic client (command-queue flush).
type mdmAction struct {
	client  *pro.Client
	classic *proclassic.Client
	devices *devSDK.Client
}

// configure binds the provider-supplied clients to the action.
func (a *mdmAction) configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *providerdata.Data, got %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "mdm")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	classic, cdiags := providerdata.ConfigureProClassic(ctx, req.ProviderData, minJamfProVersion, "mdm")
	resp.Diagnostics.Append(cdiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	a.client = client
	a.classic = classic
	a.devices = devSDK.New(pd.Client)
}

// ensureClient guarantees the Jamf Pro client was configured before Invoke.
func (a *mdmAction) ensureClient(resp *action.InvokeResponse) bool {
	if a.client != nil {
		return true
	}
	resp.Diagnostics.AddError(
		"Provider Not Configured",
		"The Jamf Platform client was not configured. Re-run terraform init/apply so the provider can configure successfully.",
	)
	return false
}

// ensureClassicClient guarantees the ProClassic client was configured before Invoke.
func (a *mdmAction) ensureClassicClient(resp *action.InvokeResponse) bool {
	if a.classic != nil {
		return true
	}
	resp.Diagnostics.AddError(
		"Provider Not Configured",
		"The Jamf Platform client was not configured. Re-run terraform init/apply so the provider can configure successfully.",
	)
	return false
}

// resolveManagementID ensures exactly one device identifier is provided and
// returns the client management id. A serial number is resolved through the
// Platform devices inventory, whose device id is the Jamf Pro managementId.
func (a *mdmAction) resolveManagementID(ctx context.Context, resp *action.InvokeResponse, managementIDAttr, serialNumberAttr types.String) (string, bool) {
	hasID := helpers.IsConfiguredValue(managementIDAttr)
	hasSerial := helpers.IsConfiguredValue(serialNumberAttr)

	switch {
	case hasID && hasSerial:
		resp.Diagnostics.AddError(
			"Multiple Device Identifiers Provided",
			"Specify only one of management_id or serial_number when invoking this action.",
		)
		return "", false
	case hasID:
		return managementIDAttr.ValueString(), true
	case hasSerial:
		serial := serialNumberAttr.ValueString()
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Resolving serial number %s", serial)})

		id, err := a.devices.ResolveDeviceIDBySerialNumber(ctx, serial)
		if err != nil {
			resp.Diagnostics.AddError(
				"Device Lookup Failed",
				fmt.Sprintf("Unable to resolve serial number %s to a management id: %s", serial, err),
			)
			return "", false
		}
		return id, true
	default:
		resp.Diagnostics.AddError(
			"Missing Device Identifier",
			"Specify either management_id or serial_number to select the device.",
		)
		return "", false
	}
}

// sendCommand posts a single MDM command for one client management id and
// surfaces the queued command id(s). commandData must be a populated *Command
// struct whose CommandType discriminator matches the intended operation.
func (a *mdmAction) sendCommand(ctx context.Context, resp *action.InvokeResponse, managementID string, commandData any, label string) bool {
	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Requesting %s for device %s", label, managementID)})

	request := &pro.MDMCommandRequest{
		ClientData:  &[]pro.MDMCommandClientRequest{{ManagementID: &managementID}},
		CommandData: commandData,
	}

	commands, err := a.client.SendMdmCommandV2(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s Failed", label),
			fmt.Sprintf("Unable to queue %s for device %s: %s", label, managementID, err),
		)
		return false
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("%s accepted for device %s (%d command(s) queued)", label, managementID, len(commands))})
	return true
}

// serialFilterBudgetBytes bounds the URL-encoded length of the RSQL filter
// expression built for one bulk serial-resolution request.
//
// The Platform device inventory rejects any request line longer than 8192 bytes
// ("Line exceeds limit of 8192 bytes"). That ceiling covers the whole request
// line — path, tenant id, page and page-size parameters and the encoded filter
// — so the filter itself gets a deliberately conservative share of it.
//
// The limit applies to the ENCODED form, which is why chunking measures encoded
// bytes: every quote and comma costs three bytes (%22, %2C), so budgeting
// against raw serial length would overshoot by roughly half.
const serialFilterBudgetBytes = 4096

// resolveManagementIDs collects every targeted client management id for the
// bulk command actions. Both selectors are additive: management ids are taken
// verbatim and serial numbers are resolved through the Platform devices
// inventory, whose device id IS the Jamf Pro managementId.
//
// Duplicates are preserved rather than collapsed: the config is taken at its
// word, matching send_blank_push.
func (a *mdmAction) resolveManagementIDs(ctx context.Context, resp *action.InvokeResponse, managementIDsAttr, serialNumbersAttr types.List) ([]string, bool) {
	var ids []string

	if !managementIDsAttr.IsNull() && !managementIDsAttr.IsUnknown() {
		resp.Diagnostics.Append(managementIDsAttr.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return nil, false
		}
	}

	if !serialNumbersAttr.IsNull() && !serialNumbersAttr.IsUnknown() {
		var serials []string
		resp.Diagnostics.Append(serialNumbersAttr.ElementsAs(ctx, &serials, false)...)
		if resp.Diagnostics.HasError() {
			return nil, false
		}
		resolved, ok := a.resolveSerialNumbers(ctx, resp, serials)
		if !ok {
			return nil, false
		}
		ids = append(ids, resolved...)
	}

	if len(ids) == 0 {
		resp.Diagnostics.AddError(
			"Missing Device Identifier",
			"Specify at least one of management_ids or serial_numbers to select the devices.",
		)
		return nil, false
	}

	return ids, true
}

// resolveSerialNumbers maps serial numbers to client management ids in bulk,
// preserving the caller's ordering.
//
// The Platform device inventory accepts an RSQL `in=` set on serialNumber, so N
// serials resolve in ceil(N/chunk) requests rather than one request per serial
// — which is what the SDK's ResolveDeviceIDBySerialNumber does, issuing a
// single-equality filtered GET each time. Chunk size is driven by
// serialFilterBudgetBytes; the SDK paginates within a chunk.
//
// Matching is re-applied client-side rather than trusted to the server-side
// filter, so a serial only counts when it comes back exactly. Every serial that
// does not resolve is named, which is a strict improvement on the batched
// command POST itself: that fails opaquely without identifying any device.
func (a *mdmAction) resolveSerialNumbers(ctx context.Context, resp *action.InvokeResponse, serials []string) ([]string, bool) {
	chunks := chunkSerialsByEncodedSize(serials, serialFilterBudgetBytes)

	found := make(map[string]string, len(serials))
	ambiguous := make(map[string]bool)

	for i, chunk := range chunks {
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf(
			"Resolving %d serial number(s) (request %d of %d)", len(chunk), i+1, len(chunks))})

		quoted := make([]string, 0, len(chunk))
		for _, serial := range chunk {
			quoted = append(quoted, quoteRSQLValue(serial))
		}
		filter := "serialNumber=in=(" + strings.Join(quoted, ",") + ")"

		devices, err := a.devices.ListDevices(ctx, nil, filter)
		if err != nil {
			resp.Diagnostics.AddError(
				"Device Lookup Failed",
				fmt.Sprintf("Unable to resolve %d serial number(s) to management ids: %s", len(chunk), err),
			)
			return nil, false
		}

		for _, device := range devices {
			if _, seen := found[device.SerialNumber]; seen {
				ambiguous[device.SerialNumber] = true
				continue
			}
			found[device.SerialNumber] = device.ID
		}
	}

	if len(ambiguous) > 0 {
		resp.Diagnostics.AddError(
			"Ambiguous Serial Number",
			fmt.Sprintf("More than one device reports the serial number(s) %s, so the intended target is unclear. Use management_ids to select the devices explicitly.",
				strings.Join(sortedKeys(ambiguous), ", ")),
		)
		return nil, false
	}

	ids := make([]string, 0, len(serials))
	var missing []string
	for _, serial := range serials {
		id, ok := found[serial]
		if !ok {
			missing = append(missing, serial)
			continue
		}
		ids = append(ids, id)
	}

	if len(missing) > 0 {
		resp.Diagnostics.AddError(
			"Device Not Found",
			fmt.Sprintf("No device matched the serial number(s) %s. Serial numbers are case-sensitive.",
				strings.Join(missing, ", ")),
		)
		return nil, false
	}

	return ids, true
}

// chunkSerialsByEncodedSize splits serials into groups whose encoded RSQL `in=`
// argument stays within budget. A single serial that exceeds the budget on its
// own still forms a chunk — there is no smaller request to make.
func chunkSerialsByEncodedSize(serials []string, budget int) [][]string {
	var (
		chunks  [][]string
		current []string
		size    int
	)
	const separatorCost = len("%2C") // an encoded comma between arguments

	for _, serial := range serials {
		cost := len(url.QueryEscape(quoteRSQLValue(serial))) + separatorCost
		if len(current) > 0 && size+cost > budget {
			chunks = append(chunks, current)
			current, size = nil, 0
		}
		current = append(current, serial)
		size += cost
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

// quoteRSQLValue renders a value as a quoted RSQL argument. It mirrors the
// escaping the SDK applies in FormatArgument, which lives in an internal
// package and cannot be imported here.
func quoteRSQLValue(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

// sortedKeys returns the map's keys in a stable order so diagnostics do not
// vary between runs.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sendCommandBatch queues one MDM command for every supplied management id in a
// SINGLE request. POST /v2/mdm/commands accepts clientData as an array and
// returns one href per queued command, so batching here is a genuine reduction
// in API calls rather than a client-side loop.
//
// commandData is shared by every device in the batch. That is precisely why the
// two actions carrying per-device command fields stay single-target: clear
// passcode resolves a different unlockToken per device, and set auto admin
// password takes a per-computer account guid. Batching either would apply one
// device's value to all of them.
//
// The request is all-or-nothing — see batchWarnThreshold.
func (a *mdmAction) sendCommandBatch(ctx context.Context, resp *action.InvokeResponse, managementIDs []string, commandData any, label string) bool {
	if len(managementIDs) > batchWarnThreshold {
		resp.Diagnostics.AddWarning(
			"Large Command Batch",
			fmt.Sprintf(
				"%s targets %d devices in a single request. Jamf Pro documents no maximum, but the request is all-or-nothing: if any one device cannot be resolved the whole batch fails with an error that does not identify which. Consider splitting this into smaller invocations.",
				label, len(managementIDs),
			),
		)
	}

	clients := buildClientData(managementIDs)

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Requesting %s for %d device(s)", label, len(managementIDs))})

	request := &pro.MDMCommandRequest{
		ClientData:  &clients,
		CommandData: commandData,
	}

	commands, err := a.client.SendMdmCommandV2(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s Failed", label),
			fmt.Sprintf("Unable to queue %s for %d device(s)%s: %s", label, len(managementIDs), batchTargetSuffix(managementIDs), err),
		)
		return false
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("%s accepted for %d device(s) (%d command(s) queued)", label, len(managementIDs), len(commands))})
	return true
}

// buildClientData turns management ids into the per-device clientData entries
// the batched command request carries.
//
// Each entry holds a POINTER to its id, so every element must reference a
// distinct string. Getting that wrong would not fail loudly: the request would
// stay well-formed and simply command one device N times while leaving the rest
// untouched. Extracted as a pure function so that contract is directly testable.
func buildClientData(managementIDs []string) []pro.MDMCommandClientRequest {
	clients := make([]pro.MDMCommandClientRequest, 0, len(managementIDs))
	for _, id := range managementIDs {
		clients = append(clients, pro.MDMCommandClientRequest{ManagementID: &id})
	}
	return clients
}

// batchTargetSuffix names the targeted devices in an error message when the
// batch is small enough to be useful. Jamf Pro's failure response identifies no
// individual device, so echoing the batch back is the only attribution a user
// gets; beyond a handful of ids it becomes noise rather than help.
func batchTargetSuffix(managementIDs []string) string {
	const maxNamed = 10
	if len(managementIDs) == 0 || len(managementIDs) > maxNamed {
		return ""
	}
	return " (" + strings.Join(managementIDs, ", ") + ")"
}

// resolveUnlockToken looks up the Find My / clear-passcode unlock token for a
// mobile device given its management id. The management id is mapped to the
// Jamf Pro mobile device id, whose inventory detail carries the token. The
// token is only populated for unsupervised devices; supervised devices return
// an empty token (and clear their passcode without one), so an empty result is
// not treated as an error.
func (a *mdmAction) resolveUnlockToken(ctx context.Context, resp *action.InvokeResponse, managementID string) (string, bool) {
	filter := fmt.Sprintf("managementId==%q", managementID)

	matches, err := a.client.ListMobileDevicesDetailV2(ctx, nil, nil, filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Mobile Device Lookup Failed",
			fmt.Sprintf("Unable to look up mobile device %s: %s", managementID, err),
		)
		return "", false
	}
	if len(matches) == 0 || matches[0].IOS == nil || matches[0].IOS.MobileDeviceID == "" {
		resp.Diagnostics.AddError(
			"Mobile Device Not Found",
			fmt.Sprintf("No iOS/iPadOS mobile device matched management id %s.", managementID),
		)
		return "", false
	}

	detail, err := a.client.GetMobileDeviceDetailV2(ctx, matches[0].IOS.MobileDeviceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Mobile Device Detail Failed",
			fmt.Sprintf("Unable to read mobile device detail for management id %s: %s", managementID, err),
		)
		return "", false
	}
	if detail.Ios == nil {
		return "", true
	}
	return detail.Ios.UnlockToken, true
}

// targetListAttributes returns the shared management_ids / serial_numbers
// selector used by every command action that can target devices in bulk.
// deviceNoun tunes the description (e.g. "computer", "mobile device").
//
// These are lists because POST /v2/mdm/commands takes clientData as an array:
// one invocation queues the command for every listed device in a single request.
// The two selectors are additive rather than exclusive, so a configuration may
// mix known management ids with serial numbers that need resolving.
func targetListAttributes(deviceNoun string) map[string]actionschema.Attribute {
	return map[string]actionschema.Attribute{
		"management_ids": actionschema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Jamf Pro Management IDs of the " + deviceNoun + "s to target. These are the `id` values reported by the `jamfplatform_devices`/`jamfplatform_device` data sources. All listed " + deviceNoun + "s are commanded in a single request. Set this and/or `serial_numbers`.",
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
				listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
			},
		},
		"serial_numbers": actionschema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Serial numbers of the " + deviceNoun + "s to target (case-sensitive). Each is looked up to find its Management ID before the command is sent, so `management_ids` avoids that lookup. Set this and/or `management_ids`.",
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
				listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
			},
		},
	}
}

// deviceTargetListConfigValidators is the plan-time counterpart to
// targetListAttributes: at least one of the two lists must select a device, so
// that supplying NO identifier fails at plan time instead of part-way through
// the apply. At-least-one rather than exactly-one, because the two lists are
// additive and may be combined.
func deviceTargetListConfigValidators() []action.ConfigValidator {
	return []action.ConfigValidator{
		actionvalidator.AtLeastOneOf(
			path.MatchRoot("management_ids"),
			path.MatchRoot("serial_numbers"),
		),
	}
}

// targetAttributes returns the shared management_id / serial_number selector
// used by the command actions that must target exactly one device because their
// command payload carries a per-device field (see sendCommandBatch). deviceNoun
// tunes the description (e.g. "device", "computer", "mobile device").
//
// Exactly-one-of is enforced by deviceTargetConfigValidators rather than by
// per-attribute ConflictsWith, so that supplying NO identifier also fails at
// plan time instead of part-way through the apply.
func targetAttributes(deviceNoun string) map[string]actionschema.Attribute {
	return map[string]actionschema.Attribute{
		"management_id": actionschema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Jamf Pro Management ID of the " + deviceNoun + ". This is the `id` reported by the `jamfplatform_devices`/`jamfplatform_device` data sources. Set exactly one of this or `serial_number`.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"serial_number": actionschema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Serial number of the " + deviceNoun + " (case-sensitive). Set exactly one of this or `management_id`.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
	}
}

// deviceTargetConfigValidators is the plan-time counterpart to
// targetAttributes: exactly one of management_id / serial_number selects the
// device. Every action that builds its schema from targetAttributes returns
// this from ConfigValidators.
func deviceTargetConfigValidators() []action.ConfigValidator {
	return []action.ConfigValidator{
		actionvalidator.ExactlyOneOf(
			path.MatchRoot("management_id"),
			path.MatchRoot("serial_number"),
		),
	}
}
