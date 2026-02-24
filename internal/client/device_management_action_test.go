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

func TestEraseDeviceV1(t *testing.T) {
	var capturedPath string
	var capturedBody []byte
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		testhelpers.RespondJSON(w, http.StatusCreated, []client.DeviceCommandResponseV1{
			{DeviceID: "device-1", CommandID: "cmd-1"},
		})
	}))

	preserveDataPlan := true
	result, err := c.EraseDeviceV1(context.Background(), "device-1", &client.EraseDeviceRequestV1{
		PreserveDataPlan: &preserveDataPlan,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedPath, "/devices/device-1/erase") {
		t.Errorf("expected path to contain '/devices/device-1/erase', got %q", capturedPath)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].CommandID != "cmd-1" {
		t.Errorf("expected CommandID 'cmd-1', got %q", result[0].CommandID)
	}

	var payload map[string]any
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if payload["preserveDataPlan"] != true {
		t.Errorf("expected preserveDataPlan true, got %v", payload["preserveDataPlan"])
	}
}

func TestEraseDeviceV1_EmptyDeviceID(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called with empty device ID")
	}))

	_, err := c.EraseDeviceV1(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty device ID")
	}
	if !strings.Contains(err.Error(), "deviceID cannot be empty") {
		t.Errorf("expected 'deviceID cannot be empty' error, got: %v", err)
	}
}

func TestRestartDeviceV1(t *testing.T) {
	var capturedPath string
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		testhelpers.RespondJSON(w, http.StatusCreated, []client.DeviceCommandResponseV1{
			{DeviceID: "device-2", CommandID: "cmd-2"},
		})
	}))

	result, err := c.RestartDeviceV1(context.Background(), "device-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedPath, "/devices/device-2/restart") {
		t.Errorf("expected path to contain '/devices/device-2/restart', got %q", capturedPath)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].DeviceID != "device-2" {
		t.Errorf("expected DeviceID 'device-2', got %q", result[0].DeviceID)
	}
}

func TestShutdownDeviceV1(t *testing.T) {
	var capturedPath string
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		testhelpers.RespondJSON(w, http.StatusCreated, []client.DeviceCommandResponseV1{
			{DeviceID: "device-3", CommandID: "cmd-3"},
		})
	}))

	result, err := c.ShutdownDeviceV1(context.Background(), "device-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedPath, "/devices/device-3/shutdown") {
		t.Errorf("expected path to contain '/devices/device-3/shutdown', got %q", capturedPath)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
}

func TestUnmanageDeviceV1(t *testing.T) {
	var capturedPath string
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		testhelpers.RespondJSON(w, http.StatusCreated, []client.DeviceCommandResponseV1{
			{DeviceID: "device-4", CommandID: "cmd-4"},
		})
	}))

	result, err := c.UnmanageDeviceV1(context.Background(), "device-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedPath, "/devices/device-4/unmanage") {
		t.Errorf("expected path to contain '/devices/device-4/unmanage', got %q", capturedPath)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].CommandID != "cmd-4" {
		t.Errorf("expected CommandID 'cmd-4', got %q", result[0].CommandID)
	}
}

func TestDeviceAction_ServerError(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.RespondJSON(w, http.StatusInternalServerError, client.ApiError{
			HTTPStatus: 500,
			Errors: []client.Error{
				{Code: "INTERNAL_ERROR", Description: "Something went wrong"},
			},
		})
	}))

	_, err := c.RestartDeviceV1(context.Background(), "device-1")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain '500', got: %v", err)
	}
}
