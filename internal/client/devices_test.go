// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestGetDeviceByIDV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/devices/device-1") {
			http.NotFound(w, r)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.DeviceReadRepresentationV1{
			ID:   "device-1",
			Name: "Test Mac",
			Hardware: &client.DeviceHardwareReadRepresentationV1{
				Model:           "MacBook Pro",
				ModelIdentifier: "Mac14,5",
				SerialNumber:    "ABC123",
			},
			OperatingSystem: &client.DeviceOperatingSystemReadRepresentationV1{
				Version: "15.2",
			},
		})
	}))

	device, err := c.GetDeviceByIDV1(context.Background(), "device-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if device.ID != "device-1" {
		t.Errorf("expected ID 'device-1', got %q", device.ID)
	}
	if device.Name != "Test Mac" {
		t.Errorf("expected Name 'Test Mac', got %q", device.Name)
	}
	if device.Hardware.SerialNumber != "ABC123" {
		t.Errorf("expected SerialNumber 'ABC123', got %q", device.Hardware.SerialNumber)
	}
	if device.OperatingSystem.Version != "15.2" {
		t.Errorf("expected OS Version '15.2', got %q", device.OperatingSystem.Version)
	}
}

func TestGetDeviceByIDV1_NotFound(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.RespondJSON(w, http.StatusNotFound, client.ApiError{
			HTTPStatus: 404,
			Errors: []client.Error{
				{Code: "NOT_FOUND", Description: "Device not found"},
			},
		})
	}))

	_, err := c.GetDeviceByIDV1(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain '404', got: %v", err)
	}
}

func TestGetDevicesV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.PaginatedDeviceResponseRepresentation{
			PaginatedResponseRepresentation: client.PaginatedResponseRepresentation{
				TotalCount: 2,
				HasNext:    false,
			},
			Results: []client.DeviceListReadRepresentationV1{
				{ID: "d1", Name: "Device 1", SerialNumber: "SN1"},
				{ID: "d2", Name: "Device 2", SerialNumber: "SN2"},
			},
		})
	}))

	devices, err := c.GetDevicesV1(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].ID != "d1" {
		t.Errorf("expected first device ID 'd1', got %q", devices[0].ID)
	}
	if devices[1].SerialNumber != "SN2" {
		t.Errorf("expected second device SerialNumber 'SN2', got %q", devices[1].SerialNumber)
	}
}

func TestGetDevicesV1_WithFilter(t *testing.T) {
	var capturedFilter string
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedFilter = r.URL.Query().Get("filter")
		testhelpers.RespondJSON(w, http.StatusOK, client.PaginatedDeviceResponseRepresentation{
			PaginatedResponseRepresentation: client.PaginatedResponseRepresentation{HasNext: false},
			Results:                         []client.DeviceListReadRepresentationV1{},
		})
	}))

	_, err := c.GetDevicesV1(context.Background(), []string{"name:asc"}, "name==Test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilter != "name==Test" {
		t.Errorf("expected filter 'name==Test', got %q", capturedFilter)
	}
}

func TestUpdateDeviceV1(t *testing.T) {
	var capturedBody []byte
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))

	name := "Updated Name"
	err := c.UpdateDeviceV1(context.Background(), "device-1", &client.DeviceUpdateRepresentationV1{
		Name: &name,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if payload["name"] != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %v", payload["name"])
	}
}

func TestDeleteDeviceV1(t *testing.T) {
	var capturedPath string
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))

	err := c.DeleteDeviceV1(context.Background(), "device-to-delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedPath, "device-to-delete") {
		t.Errorf("expected path to contain 'device-to-delete', got %q", capturedPath)
	}
}

func TestGetDeviceInstalledApplicationsV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/devices/device-1/applications") {
			http.NotFound(w, r)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.PaginatedDeviceInstalledApplicationReadRepresentationV1{
			PaginatedResponseRepresentation: client.PaginatedResponseRepresentation{HasNext: false},
			Results: []client.DeviceInstalledApplicationReadRepresentationV1{
				{Name: "Safari", Version: "17.0"},
				{Name: "Xcode", Version: "15.2"},
			},
		})
	}))

	apps, err := c.GetDeviceInstalledApplicationsV1(context.Background(), "device-1", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}
	if apps[0].Name != "Safari" {
		t.Errorf("expected first app 'Safari', got %q", apps[0].Name)
	}
}

func TestGetDevicesForUserV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/users/user-1/devices") {
			http.NotFound(w, r)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.PaginatedDeviceResponseRepresentation{
			PaginatedResponseRepresentation: client.PaginatedResponseRepresentation{HasNext: false},
			Results: []client.DeviceListReadRepresentationV1{
				{ID: "d1", Name: "User's Mac"},
			},
		})
	}))

	devices, err := c.GetDevicesForUserV1(context.Background(), "user-1", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if devices[0].Name != "User's Mac" {
		t.Errorf("expected device name 'User's Mac', got %q", devices[0].Name)
	}
}

func TestNullableString_MarshalJSON_Value(t *testing.T) {
	ns := client.NewNullableString("hello")
	data, err := json.Marshal(ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `"hello"` {
		t.Errorf("expected '\"hello\"', got %s", string(data))
	}
}

func TestNullableString_MarshalJSON_Null(t *testing.T) {
	ns := client.NewNullableStringNull()
	data, err := json.Marshal(ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("expected 'null', got %s", string(data))
	}
}
