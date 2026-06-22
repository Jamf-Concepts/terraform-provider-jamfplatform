// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"
	"fmt"

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

// targetAttributes returns the shared management_id / serial_number selector
// used by every command action that targets a single device. deviceNoun tunes
// the description (e.g. "device", "computer", "mobile device").
func targetAttributes(deviceNoun string) map[string]actionschema.Attribute {
	return map[string]actionschema.Attribute{
		"management_id": actionschema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Jamf Pro Management ID of the " + deviceNoun + ". This is the `id` reported by the `jamfplatform_devices`/`jamfplatform_device` data sources. Provide this or `serial_number`.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
				stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("serial_number")),
			},
		},
		"serial_number": actionschema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Serial number of the " + deviceNoun + " (case-sensitive). Provide this or `management_id`.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
				stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("management_id")),
			},
		},
	}
}
