// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
)

func assertAttrsPresent(t *testing.T, schema func(context.Context, action.SchemaRequest, *action.SchemaResponse), want []string) {
	t.Helper()
	var resp action.SchemaResponse
	schema(context.Background(), action.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range want {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func assertRequired(t *testing.T, schema func(context.Context, action.SchemaRequest, *action.SchemaResponse), required []string) {
	t.Helper()
	var resp action.SchemaResponse
	schema(context.Background(), action.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range required {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}
}

func assertTypeName(t *testing.T, meta func(context.Context, action.MetadataRequest, *action.MetadataResponse), want string) {
	t.Helper()
	req := action.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp action.MetadataResponse
	meta(context.Background(), req, &resp)
	if resp.TypeName != want {
		t.Errorf("expected type name %q, got %q", want, resp.TypeName)
	}
}

// --- device_lock (canary) ---

func TestDeviceLockAction_Metadata(t *testing.T) {
	assertTypeName(t, NewDeviceLockAction().(*DeviceLockAction).Metadata, "jamfplatform_pro_device_lock")
}

func TestDeviceLockAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewDeviceLockAction().(*DeviceLockAction).Schema,
		[]string{"management_id", "serial_number", "message", "phone_number", "pin"})
}

// --- disable_lost_mode (canary) ---

func TestDisableLostModeAction_Metadata(t *testing.T) {
	assertTypeName(t, NewDisableLostModeAction().(*DisableLostModeAction).Metadata, "jamfplatform_pro_disable_lost_mode")
}

func TestDisableLostModeAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewDisableLostModeAction().(*DisableLostModeAction).Schema,
		[]string{"management_id", "serial_number"})
}

// --- play_lost_mode_sound (canary) ---

func TestPlayLostModeSoundAction_Metadata(t *testing.T) {
	assertTypeName(t, NewPlayLostModeSoundAction().(*PlayLostModeSoundAction).Metadata, "jamfplatform_pro_play_lost_mode_sound")
}

func TestPlayLostModeSoundAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewPlayLostModeSoundAction().(*PlayLostModeSoundAction).Schema,
		[]string{"management_id", "serial_number"})
}

// --- clear_passcode (canary) ---

func TestClearPasscodeAction_Metadata(t *testing.T) {
	assertTypeName(t, NewClearPasscodeAction().(*ClearPasscodeAction).Metadata, "jamfplatform_pro_clear_passcode")
}

func TestClearPasscodeAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewClearPasscodeAction().(*ClearPasscodeAction).Schema,
		[]string{"management_id", "serial_number", "unlock_token"})
}

// --- enable_lost_mode ---

func TestEnableLostModeAction_Metadata(t *testing.T) {
	assertTypeName(t, NewEnableLostModeAction().(*EnableLostModeAction).Metadata, "jamfplatform_pro_enable_lost_mode")
}

func TestEnableLostModeAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewEnableLostModeAction().(*EnableLostModeAction).Schema,
		[]string{"management_id", "serial_number", "lost_mode_message", "lost_mode_footnote", "lost_mode_phone"})
}

// --- enable_remote_desktop ---

func TestEnableRemoteDesktopAction_Metadata(t *testing.T) {
	assertTypeName(t, NewEnableRemoteDesktopAction().(*EnableRemoteDesktopAction).Metadata, "jamfplatform_pro_enable_remote_desktop")
}

func TestEnableRemoteDesktopAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewEnableRemoteDesktopAction().(*EnableRemoteDesktopAction).Schema,
		[]string{"management_id", "serial_number"})
}

// --- disable_remote_desktop ---

func TestDisableRemoteDesktopAction_Metadata(t *testing.T) {
	assertTypeName(t, NewDisableRemoteDesktopAction().(*DisableRemoteDesktopAction).Metadata, "jamfplatform_pro_disable_remote_desktop")
}

func TestDisableRemoteDesktopAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewDisableRemoteDesktopAction().(*DisableRemoteDesktopAction).Schema,
		[]string{"management_id", "serial_number"})
}

// --- clear_restrictions_password ---

func TestClearRestrictionsPasswordAction_Metadata(t *testing.T) {
	assertTypeName(t, NewClearRestrictionsPasswordAction().(*ClearRestrictionsPasswordAction).Metadata, "jamfplatform_pro_clear_restrictions_password")
}

func TestClearRestrictionsPasswordAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewClearRestrictionsPasswordAction().(*ClearRestrictionsPasswordAction).Schema,
		[]string{"management_id", "serial_number"})
}

// --- delete_user ---

func TestDeleteUserAction_Metadata(t *testing.T) {
	assertTypeName(t, NewDeleteUserAction().(*DeleteUserAction).Metadata, "jamfplatform_pro_delete_user")
}

func TestDeleteUserAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewDeleteUserAction().(*DeleteUserAction).Schema,
		[]string{"management_id", "serial_number", "user_name", "delete_all_users", "force_deletion"})
}

// --- log_out_user ---

func TestLogOutUserAction_Metadata(t *testing.T) {
	assertTypeName(t, NewLogOutUserAction().(*LogOutUserAction).Metadata, "jamfplatform_pro_log_out_user")
}

func TestLogOutUserAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewLogOutUserAction().(*LogOutUserAction).Schema,
		[]string{"management_id", "serial_number"})
}

// --- unlock_user_account ---

func TestUnlockUserAccountAction_Metadata(t *testing.T) {
	assertTypeName(t, NewUnlockUserAccountAction().(*UnlockUserAccountAction).Metadata, "jamfplatform_pro_unlock_user_account")
}

func TestUnlockUserAccountAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewUnlockUserAccountAction().(*UnlockUserAccountAction).Schema,
		[]string{"management_id", "serial_number", "user_name"})
}

// --- set_auto_admin_password ---

func TestSetAutoAdminPasswordAction_Metadata(t *testing.T) {
	assertTypeName(t, NewSetAutoAdminPasswordAction().(*SetAutoAdminPasswordAction).Metadata, "jamfplatform_pro_set_auto_admin_password")
}

func TestSetAutoAdminPasswordAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewSetAutoAdminPasswordAction().(*SetAutoAdminPasswordAction).Schema,
		[]string{"management_id", "serial_number", "guid", "password"})
}

// --- send_blank_push ---

func TestSendBlankPushAction_Metadata(t *testing.T) {
	assertTypeName(t, NewSendBlankPushAction().(*SendBlankPushAction).Metadata, "jamfplatform_pro_send_blank_push")
}

func TestSendBlankPushAction_Schema(t *testing.T) {
	assertAttrsPresent(t, NewSendBlankPushAction().(*SendBlankPushAction).Schema,
		[]string{"management_ids", "serial_numbers"})
}

// --- renew_mdm_profile ---

func TestRenewMdmProfileAction_Metadata(t *testing.T) {
	assertTypeName(t, NewRenewMdmProfileAction().(*RenewMdmProfileAction).Metadata, "jamfplatform_pro_renew_mdm_profile")
}

func TestRenewMdmProfileAction_Schema(t *testing.T) {
	schema := NewRenewMdmProfileAction().(*RenewMdmProfileAction).Schema
	assertAttrsPresent(t, schema, []string{"udids"})
	assertRequired(t, schema, []string{"udids"})
}

// --- flush_mdm_commands ---

func TestFlushMdmCommandsAction_Metadata(t *testing.T) {
	assertTypeName(t, NewFlushMdmCommandsAction().(*FlushMdmCommandsAction).Metadata, "jamfplatform_pro_flush_mdm_commands")
}

func TestFlushMdmCommandsAction_Schema(t *testing.T) {
	schema := NewFlushMdmCommandsAction().(*FlushMdmCommandsAction).Schema
	assertAttrsPresent(t, schema, []string{"id_type", "id", "status"})
	assertRequired(t, schema, []string{"id_type", "id", "status"})
}

// --- plan-time device-target validation ---

// TestDeviceTargetValidatorsWired guards that every action whose schema is built
// from targetAttributes also declares the matching ConfigValidators. Without the
// validator, "neither management_id nor serial_number" passes plan and only
// fails part-way through the apply — the schema alone cannot express
// exactly-one-of, so the two halves must be kept in step.
//
// The list is asserted against the schema itself, so an action that gains the
// target selector but forgets ConfigValidators fails here.
func TestDeviceTargetValidatorsWired(t *testing.T) {
	targetActions := map[string]action.Action{
		"clear_passcode":              NewClearPasscodeAction(),
		"clear_restrictions_password": NewClearRestrictionsPasswordAction(),
		"delete_user":                 NewDeleteUserAction(),
		"device_lock":                 NewDeviceLockAction(),
		"disable_lost_mode":           NewDisableLostModeAction(),
		"disable_remote_desktop":      NewDisableRemoteDesktopAction(),
		"enable_lost_mode":            NewEnableLostModeAction(),
		"enable_remote_desktop":       NewEnableRemoteDesktopAction(),
		"log_out_user":                NewLogOutUserAction(),
		"play_lost_mode_sound":        NewPlayLostModeSoundAction(),
		"set_auto_admin_password":     NewSetAutoAdminPasswordAction(),
		"unlock_user_account":         NewUnlockUserAccountAction(),
	}

	for name, a := range targetActions {
		t.Run(name, func(t *testing.T) {
			withValidators, ok := a.(action.ActionWithConfigValidators)
			if !ok {
				t.Fatalf("%s builds its schema from targetAttributes but declares no ConfigValidators", name)
			}
			if len(withValidators.ConfigValidators(context.Background())) == 0 {
				t.Fatalf("%s declares an empty ConfigValidators slice", name)
			}
		})
	}
}

// TestSendBlankPushValidatorsWired covers the at-least-one-of rule: the action
// accepts management_ids and/or serial_numbers, and previously only rejected
// "neither" once the apply was already running.
func TestSendBlankPushValidatorsWired(t *testing.T) {
	a, ok := NewSendBlankPushAction().(action.ActionWithConfigValidators)
	if !ok {
		t.Fatal("send_blank_push declares no ConfigValidators")
	}
	if len(a.ConfigValidators(context.Background())) == 0 {
		t.Fatal("send_blank_push declares an empty ConfigValidators slice")
	}
}
