// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package deviceactions

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
)

// --- Erase ---

func TestEraseAction_Metadata(t *testing.T) {
	a := NewEraseAction()
	req := action.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp action.MetadataResponse
	a.(*EraseAction).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_device_erase" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_device_erase", resp.TypeName)
	}
}

func TestEraseAction_Schema(t *testing.T) {
	a := NewEraseAction()
	req := action.SchemaRequest{}
	var resp action.SchemaResponse
	a.(*EraseAction).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	expectedAttrs := []string{"device_id", "serial_number", "preserve_data_plan", "disallow_proximity_setup", "clear_activation_lock", "return_to_service", "pin"}
	for _, name := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

// --- Restart ---

func TestRestartAction_Metadata(t *testing.T) {
	a := NewRestartAction()
	req := action.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp action.MetadataResponse
	a.(*RestartAction).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_device_restart" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_device_restart", resp.TypeName)
	}
}

func TestRestartAction_Schema(t *testing.T) {
	a := NewRestartAction()
	req := action.SchemaRequest{}
	var resp action.SchemaResponse
	a.(*RestartAction).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	expectedAttrs := []string{"device_id", "serial_number"}
	for _, name := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

// --- Shutdown ---

func TestShutdownAction_Metadata(t *testing.T) {
	a := NewShutdownAction()
	req := action.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp action.MetadataResponse
	a.(*ShutdownAction).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_device_shutdown" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_device_shutdown", resp.TypeName)
	}
}

func TestShutdownAction_Schema(t *testing.T) {
	a := NewShutdownAction()
	req := action.SchemaRequest{}
	var resp action.SchemaResponse
	a.(*ShutdownAction).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	expectedAttrs := []string{"device_id", "serial_number"}
	for _, name := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

// --- Unmanage ---

func TestUnmanageAction_Metadata(t *testing.T) {
	a := NewUnmanageAction()
	req := action.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp action.MetadataResponse
	a.(*UnmanageAction).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_device_unmanage" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_device_unmanage", resp.TypeName)
	}
}

func TestUnmanageAction_Schema(t *testing.T) {
	a := NewUnmanageAction()
	req := action.SchemaRequest{}
	var resp action.SchemaResponse
	a.(*UnmanageAction).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	expectedAttrs := []string{"device_id", "serial_number"}
	for _, name := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

// TestDeviceTargetValidatorsWired guards that every device action declares the
// exactly-one-of ConfigValidator alongside the shared device_id / serial_number
// selector. The schema cannot express exactly-one-of on its own, so without the
// validator a config naming NO device passes plan and only fails once the apply
// is already running.
func TestDeviceTargetValidatorsWired(t *testing.T) {
	for name, a := range map[string]action.Action{
		"erase":    NewEraseAction(),
		"restart":  NewRestartAction(),
		"shutdown": NewShutdownAction(),
		"unmanage": NewUnmanageAction(),
	} {
		t.Run(name, func(t *testing.T) {
			withValidators, ok := a.(action.ActionWithConfigValidators)
			if !ok {
				t.Fatalf("%s declares no ConfigValidators", name)
			}
			if len(withValidators.ConfigValidators(context.Background())) == 0 {
				t.Fatalf("%s declares an empty ConfigValidators slice", name)
			}
		})
	}
}
