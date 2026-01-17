// Copyright 2026 Jamf Software LLC.

package deviceactions

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// deviceAction shares Configure logic across device action implementations.
type deviceAction struct {
	client *client.Client
}

// configure binds the provider-supplied client to the action.
func (a *deviceAction) configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *client.Client, got %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	a.client = apiClient
}

// ensureClient guarantees Configure completed successfully before Invoke.
func (a *deviceAction) ensureClient(resp *action.InvokeResponse) bool {
	if a.client != nil {
		return true
	}

	resp.Diagnostics.AddError(
		"Provider Not Configured",
		"The Jamf Platform client was not configured. Re-run terraform init/apply so the provider can configure successfully.",
	)
	return false
}

// resolveDeviceIdentifier ensures exactly one device identifier is provided and returns the device ID.
func (a *deviceAction) resolveDeviceIdentifier(ctx context.Context, resp *action.InvokeResponse, deviceIDAttr, serialNumberAttr types.String) (string, bool) {
	hasDeviceID := helpers.IsConfiguredValue(deviceIDAttr)
	hasSerial := helpers.IsConfiguredValue(serialNumberAttr)

	switch {
	case hasDeviceID && hasSerial:
		resp.Diagnostics.AddError(
			"Multiple Device Identifiers Provided",
			"Specify only one of device_id or serial_number when invoking this action.",
		)
		return "", false
	case hasDeviceID:
		return deviceIDAttr.ValueString(), true
	case hasSerial:
		serial := serialNumberAttr.ValueString()
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Resolving serial number %s", serial)})

		deviceID, err := a.lookupDeviceIDBySerial(ctx, serial)
		if err != nil {
			resp.Diagnostics.AddError(
				"Device Lookup Failed",
				fmt.Sprintf("Unable to resolve serial number %s to a device ID: %s", serial, err),
			)
			return "", false
		}

		return deviceID, true
	default:
		resp.Diagnostics.AddError(
			"Missing Device Identifier",
			"Specify either device_id or serial_number to select the device.",
		)
		return "", false
	}
}

func (a *deviceAction) lookupDeviceIDBySerial(ctx context.Context, serial string) (string, error) {
	filter := fmt.Sprintf(`serialNumber=="%s"`, escapeRSQLValue(serial))
	devices, err := a.client.GetDevicesV1(ctx, nil, filter)
	if err != nil {
		return "", fmt.Errorf("failed to query devices by serial number: %w", err)
	}

	switch len(devices) {
	case 0:
		return "", fmt.Errorf("no device found with serial number %s", serial)
	case 1:
		return devices[0].ID, nil
	default:
		return "", fmt.Errorf("multiple devices (%d) returned for serial number %s", len(devices), serial)
	}
}

func escapeRSQLValue(value string) string {
	escaped := strings.ReplaceAll(value, `\\`, `\\\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\\"`)
	return escaped
}
