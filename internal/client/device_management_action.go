// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Device Management Action API client

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Device Management Action API path constants
const (
	deviceManagementActionsV1Prefix = "/management/actions/v1"
)

// DeviceCommandResponseV1 captures the command metadata returned by device actions.
type DeviceCommandResponseV1 struct {
	DeviceID  string `json:"deviceId"`
	CommandID string `json:"commandId"`
}

// EraseDeviceRequestV1 holds the optional payload properties for erase.
type EraseDeviceRequestV1 struct {
	PreserveDataPlan       *bool   `json:"preserveDataPlan,omitempty"`
	DisallowProximitySetup *bool   `json:"disallowProximitySetup,omitempty"`
	ClearActivationLock    *bool   `json:"clearActivationLock,omitempty"`
	ReturnToService        *bool   `json:"returnToService,omitempty"`
	Pin                    *string `json:"pin,omitempty"`
}

// EraseDeviceV1 requests that the specified device erase its content and settings.
func (c *Client) EraseDeviceV1(ctx context.Context, deviceID string, request *EraseDeviceRequestV1) ([]DeviceCommandResponseV1, error) {
	return c.invokeDeviceManagementActionV1(ctx, deviceID, "erase", request)
}

// RestartDeviceV1 requests that the specified device perform a restart.
func (c *Client) RestartDeviceV1(ctx context.Context, deviceID string) ([]DeviceCommandResponseV1, error) {
	return c.invokeDeviceManagementActionV1(ctx, deviceID, "restart", nil)
}

// ShutdownDeviceV1 requests that the specified device shut down.
func (c *Client) ShutdownDeviceV1(ctx context.Context, deviceID string) ([]DeviceCommandResponseV1, error) {
	return c.invokeDeviceManagementActionV1(ctx, deviceID, "shutdown", nil)
}

// UnmanageDeviceV1 requests that the specified device remove remote management.
func (c *Client) UnmanageDeviceV1(ctx context.Context, deviceID string) ([]DeviceCommandResponseV1, error) {
	return c.invokeDeviceManagementActionV1(ctx, deviceID, "unmanage", nil)
}

// invokeDeviceManagementActionV1 is a helper to call device management action endpoints.
func (c *Client) invokeDeviceManagementActionV1(ctx context.Context, deviceID, action string, payload any) ([]DeviceCommandResponseV1, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("deviceID cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/devices/%s/%s", deviceManagementActionsV1Prefix, url.PathEscape(deviceID), action)
	resp, err := c.makeRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to %s device %s: %w", action, deviceID, err)
	}

	var result []DeviceCommandResponseV1
	if err := c.handleAPIResponse(ctx, resp, http.StatusCreated, &result); err != nil {
		return nil, err
	}

	return result, nil
}
