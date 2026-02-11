// Copyright 2025 Jamf Software LLC.
// https://developer.jamf.com/platform-api/reference/devices

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Device Inventory API path constants.
const (
	deviceInventoryV1Prefix = "/management/devices/v1"
)

// DeviceListReadRepresentationV1 represents a device in a list response.
type DeviceListReadRepresentationV1 struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Model                   string  `json:"model"`
	ModelIdentifier         string  `json:"modelIdentifier"`
	SerialNumber            string  `json:"serialNumber"`
	LastInventoryUpdateTime string  `json:"lastInventoryUpdateTime"`
	LastCheckInTime         *string `json:"lastCheckInTime,omitempty"`
	OperatingSystemVersion  string  `json:"operatingSystemVersion"`
	UserID                  *string `json:"userId,omitempty"`
	EnrollmentType          string  `json:"enrollmentType"`
	LastEnrollmentTime      string  `json:"lastEnrollmentTime"`
}

// DeviceReadRepresentationV1 represents the full device payload.
type DeviceReadRepresentationV1 struct {
	ID                      string                                     `json:"id"`
	Name                    string                                     `json:"name"`
	LastInventoryUpdateTime string                                     `json:"lastInventoryUpdateTime"`
	LastCheckInTime         *string                                    `json:"lastCheckInTime,omitempty"`
	UserID                  *string                                    `json:"userId,omitempty"`
	EnrollmentType          string                                     `json:"enrollmentType"`
	LastEnrollmentTime      string                                     `json:"lastEnrollmentTime"`
	Managed                 bool                                       `json:"managed"`
	Supervised              bool                                       `json:"supervised"`
	Hardware                *DeviceHardwareReadRepresentationV1        `json:"hardware,omitempty"`
	Network                 *DeviceNetworkReadRepresentationV1         `json:"network,omitempty"`
	OperatingSystem         *DeviceOperatingSystemReadRepresentationV1 `json:"operatingSystem,omitempty"`
	Security                *DeviceSecurityReadRepresentationV1        `json:"security,omitempty"`
}

// DeviceHardwareReadRepresentationV1 represents the hardware section of a device.
type DeviceHardwareReadRepresentationV1 struct {
	Make            string `json:"make"`
	Model           string `json:"model"`
	ModelIdentifier string `json:"modelIdentifier"`
	UDID            string `json:"udid"`
	SerialNumber    string `json:"serialNumber"`
	BatteryHealth   string `json:"batteryHealth"`
	MacAddress      string `json:"macAddress"`
	StorageCapacity int    `json:"storageCapacity"`
	StorageUsed     int    `json:"storageUsed"`
}

// DeviceOperatingSystemReadRepresentationV1 represents operating system information.
type DeviceOperatingSystemReadRepresentationV1 struct {
	Name                     string  `json:"name"`
	Version                  string  `json:"version"`
	Build                    string  `json:"build"`
	SupplementalBuildVersion *string `json:"supplementalBuildVersion,omitempty"`
	RapidSecurityResponse    *string `json:"rapidSecurityResponse,omitempty"`
}

// DeviceSecurityReadRepresentationV1 represents security information for a device.
type DeviceSecurityReadRepresentationV1 struct {
	BootstrapTokenEscrowedStatus string `json:"bootstrapTokenEscrowedStatus"`
	HardwareEncryption           *bool  `json:"hardwareEncryption,omitempty"`
	PasscodePresent              *bool  `json:"passcodePresent,omitempty"`
	PasscodeCompliant            *bool  `json:"passcodeCompliant,omitempty"`
	LostModeEnabled              *bool  `json:"lostModeEnabled,omitempty"`
}

// DeviceNetworkReadRepresentationV1 represents network information for a device.
type DeviceNetworkReadRepresentationV1 struct {
	LastIPAddress           *string `json:"lastIpAddress,omitempty"`
	LastReportedIPv4Address *string `json:"lastReportedIpV4Address,omitempty"`
	LastReportedIPv6Address *string `json:"lastReportedIpV6Address,omitempty"`
}

// DeviceInstalledApplicationReadRepresentationV1 represents an installed application on a device.
type DeviceInstalledApplicationReadRepresentationV1 struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// DeviceUpdateRepresentationV1 represents the payload used to update a device.
type DeviceUpdateRepresentationV1 struct {
	Name   *string         `json:"name,omitempty"`
	UserID *NullableString `json:"userId,omitempty"`
}

// PaginatedResponseRepresentation captures pagination metadata shared by multiple endpoints.
type PaginatedResponseRepresentation struct {
	Page        int   `json:"page"`
	PageSize    int   `json:"pageSize"`
	TotalCount  int64 `json:"totalCount"`
	TotalPages  int   `json:"totalPages"`
	HasNext     bool  `json:"hasNext"`
	HasPrevious bool  `json:"hasPrevious"`
}

// PaginatedDeviceResponseRepresentation represents a paginated list of devices.
type PaginatedDeviceResponseRepresentation struct {
	PaginatedResponseRepresentation
	Results []DeviceListReadRepresentationV1 `json:"results"`
}

// PaginatedDeviceInstalledApplicationReadRepresentationV1 represents a paginated list of installed applications for a device.
type PaginatedDeviceInstalledApplicationReadRepresentationV1 struct {
	PaginatedResponseRepresentation
	Results []DeviceInstalledApplicationReadRepresentationV1 `json:"results"`
}

// NullableString helps differentiate between explicit null and omitted string fields.
type NullableString struct {
	Value  string
	IsNull bool
}

// MarshalJSON implements json.Marshaler to emit either the string value or null.
func (ns NullableString) MarshalJSON() ([]byte, error) {
	if ns.IsNull {
		return []byte("null"), nil
	}
	return json.Marshal(ns.Value)
}

// NewNullableString returns a NullableString with a concrete value.
func NewNullableString(value string) *NullableString {
	return &NullableString{Value: value}
}

// NewNullableStringNull returns a NullableString that marshals to JSON null.
func NewNullableStringNull() *NullableString {
	return &NullableString{IsNull: true}
}

// GetDevicesV1 returns all devices, automatically handling pagination.
func (c *Client) GetDevicesV1(ctx context.Context, sort []string, filter string) ([]DeviceListReadRepresentationV1, error) {
	var allDevices []DeviceListReadRepresentationV1
	page := 0
	pageSize := 100

	for {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("page-size", fmt.Sprintf("%d", pageSize))
		if len(sort) > 0 {
			params.Set("sort", strings.Join(sort, ","))
		}
		if filter != "" {
			params.Set("filter", filter)
		}

		endpoint := deviceInventoryV1Prefix + "/devices"
		if encoded := params.Encode(); encoded != "" {
			endpoint += "?" + encoded
		}

		resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list devices: %w", err)
		}

		var result PaginatedDeviceResponseRepresentation
		if err := c.handleAPIResponse(ctx, resp, http.StatusOK, &result); err != nil {
			return nil, err
		}

		if len(result.Results) == 0 {
			break
		}

		allDevices = append(allDevices, result.Results...)
		if !result.HasNext {
			break
		}

		page++
	}

	return allDevices, nil
}

// GetDeviceByIDV1 retrieves a device by ID.
func (c *Client) GetDeviceByIDV1(ctx context.Context, id string) (*DeviceReadRepresentationV1, error) {
	endpoint := fmt.Sprintf("%s/devices/%s", deviceInventoryV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get device %s: %w", id, err)
	}

	var device DeviceReadRepresentationV1
	if err := c.handleAPIResponse(ctx, resp, http.StatusOK, &device); err != nil {
		return nil, err
	}

	return &device, nil
}

// UpdateDeviceV1 updates an existing device.
func (c *Client) UpdateDeviceV1(ctx context.Context, id string, payload *DeviceUpdateRepresentationV1) error {
	endpoint := fmt.Sprintf("%s/devices/%s", deviceInventoryV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, http.MethodPatch, endpoint, payload)
	if err != nil {
		return fmt.Errorf("failed to update device %s: %w", id, err)
	}

	if err := c.handleAPIResponse(ctx, resp, http.StatusNoContent, nil); err != nil {
		return err
	}

	return nil
}

// DeleteDeviceV1 deletes a device by ID.
func (c *Client) DeleteDeviceV1(ctx context.Context, id string) error {
	endpoint := fmt.Sprintf("%s/devices/%s", deviceInventoryV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete device %s: %w", id, err)
	}

	if err := c.handleAPIResponse(ctx, resp, http.StatusNoContent, nil); err != nil {
		return err
	}

	return nil
}

// GetDeviceInstalledApplicationsV1 returns all installed applications for a device, handling pagination internally.
func (c *Client) GetDeviceInstalledApplicationsV1(ctx context.Context, deviceID string, sort []string, filter string) ([]DeviceInstalledApplicationReadRepresentationV1, error) {
	var allApps []DeviceInstalledApplicationReadRepresentationV1
	page := 0
	pageSize := 100

	for {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("page-size", fmt.Sprintf("%d", pageSize))
		if len(sort) > 0 {
			params.Set("sort", strings.Join(sort, ","))
		}
		if filter != "" {
			params.Set("filter", filter)
		}

		endpoint := fmt.Sprintf("%s/devices/%s/applications", deviceInventoryV1Prefix, url.PathEscape(deviceID))
		if encoded := params.Encode(); encoded != "" {
			endpoint += "?" + encoded
		}

		resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list installed applications for device %s: %w", deviceID, err)
		}

		var result PaginatedDeviceInstalledApplicationReadRepresentationV1
		if err := c.handleAPIResponse(ctx, resp, http.StatusOK, &result); err != nil {
			return nil, err
		}

		if len(result.Results) == 0 {
			break
		}

		allApps = append(allApps, result.Results...)
		if !result.HasNext {
			break
		}

		page++
	}

	return allApps, nil
}

// GetDevicesForUserV1 returns all devices assigned to the specified user, handling pagination internally.
func (c *Client) GetDevicesForUserV1(ctx context.Context, userID string, sort []string, filter string) ([]DeviceListReadRepresentationV1, error) {
	var devices []DeviceListReadRepresentationV1
	page := 0
	pageSize := 100

	for {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("page-size", fmt.Sprintf("%d", pageSize))
		if len(sort) > 0 {
			params.Set("sort", strings.Join(sort, ","))
		}
		if filter != "" {
			params.Set("filter", filter)
		}

		endpoint := fmt.Sprintf("%s/users/%s/devices", deviceInventoryV1Prefix, url.PathEscape(userID))
		if encoded := params.Encode(); encoded != "" {
			endpoint += "?" + encoded
		}

		resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list devices for user %s: %w", userID, err)
		}

		var result PaginatedDeviceResponseRepresentation
		if err := c.handleAPIResponse(ctx, resp, http.StatusOK, &result); err != nil {
			return nil, err
		}

		if len(result.Results) == 0 {
			break
		}

		devices = append(devices, result.Results...)
		if !result.HasNext {
			break
		}

		page++
	}

	return devices, nil
}
