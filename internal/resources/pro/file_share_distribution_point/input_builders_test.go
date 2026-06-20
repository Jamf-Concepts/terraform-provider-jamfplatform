// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package file_share_distribution_point

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildInput_SMBRoundTrip covers the SMB path: required scalars and the
// account usernames reach the SDK payload, and the three passwords are threaded
// through verbatim when supplied.
func TestBuildInput_SMBRoundTrip(t *testing.T) {
	plan := FileShareDistributionPointResourceModel{
		Name:                      types.StringValue("dp-smb"),
		ServerName:                types.StringValue("smb.example.com"),
		FileSharingConnectionType: types.StringValue(connectionTypeSMB),
		ShareName:                 types.StringValue("CasperShare"),
		Port:                      types.Int64Value(445),
		Workgroup:                 types.StringValue("WG"),
		ReadWriteUsername:         types.StringValue("rw"),
		ReadOnlyUsername:          types.StringValue("ro"),
		HTTPSEnabled:              types.BoolValue(true),
		HTTPSPort:                 types.Int64Value(443),
		HTTPSSecurityType:         types.StringValue(httpsSecurityUsernamePassword),
		HTTPSUsername:             types.StringValue("hu"),
		BackupDistributionPointID: types.StringValue(noneBackupSentinel),
	}

	rw, ro, hp := new("rwpass"), new("ropass"), new("httpspass")
	got := buildFileShareDistributionPointInput(plan, rw, ro, hp)

	if got.Name != "dp-smb" || got.ServerName != "smb.example.com" || got.FileSharingConnectionType != "SMB" {
		t.Errorf("required scalars wrong: %+v", got)
	}
	if got.ShareName == nil || *got.ShareName != "CasperShare" {
		t.Errorf("ShareName = %v", got.ShareName)
	}
	if got.Port == nil || *got.Port != 445 {
		t.Errorf("Port = %v", got.Port)
	}
	if got.ReadWriteUsername == nil || *got.ReadWriteUsername != "rw" {
		t.Errorf("ReadWriteUsername = %v", got.ReadWriteUsername)
	}
	if got.ReadWritePassword == nil || *got.ReadWritePassword != "rwpass" {
		t.Errorf("ReadWritePassword = %v", got.ReadWritePassword)
	}
	if got.ReadOnlyPassword == nil || *got.ReadOnlyPassword != "ropass" {
		t.Errorf("ReadOnlyPassword = %v", got.ReadOnlyPassword)
	}
	if got.HttpsPassword == nil || *got.HttpsPassword != "httpspass" {
		t.Errorf("HttpsPassword = %v", got.HttpsPassword)
	}
}

// TestBuildInput_OmittedPasswordsAreNil confirms a nil password pointer is
// carried through so the merge update omits it (preserving the stored value).
func TestBuildInput_OmittedPasswordsAreNil(t *testing.T) {
	plan := FileShareDistributionPointResourceModel{
		Name:                      types.StringValue("dp"),
		ServerName:                types.StringValue("s"),
		FileSharingConnectionType: types.StringValue(connectionTypeSMB),
	}
	got := buildFileShareDistributionPointInput(plan, nil, nil, nil)
	if got.ReadWritePassword != nil || got.ReadOnlyPassword != nil || got.HttpsPassword != nil {
		t.Errorf("expected all passwords nil, got rw=%v ro=%v https=%v", got.ReadWritePassword, got.ReadOnlyPassword, got.HttpsPassword)
	}
}

// TestBuildInput_HTTPSGating confirms the HTTPS sub-fields are omitted when
// https_enabled is false (server rejects httpsPort alongside httpsEnabled=false),
// and emitted when true.
func TestBuildInput_HTTPSGating(t *testing.T) {
	// HTTPS off: even with stale values in the plan, sub-fields must not be sent.
	off := FileShareDistributionPointResourceModel{
		Name:                      types.StringValue("dp"),
		ServerName:                types.StringValue("s"),
		FileSharingConnectionType: types.StringValue(connectionTypeSMB),
		HTTPSEnabled:              types.BoolValue(false),
		HTTPSPort:                 types.Int64Value(443),
		HTTPSSecurityType:         types.StringValue(httpsSecurityNone),
		HTTPSContext:              types.StringValue("ctx"),
		HTTPSUsername:             types.StringValue("u"),
	}
	hp := "x"
	got := buildFileShareDistributionPointInput(off, nil, nil, &hp)
	if got.HttpsPort != nil || got.HttpsContext != nil || got.HttpsSecurityType != nil || got.HttpsUsername != nil || got.HttpsPassword != nil {
		t.Errorf("HTTPS sub-fields must be omitted when https disabled: %+v", got)
	}
	if got.HttpsEnabled == nil || *got.HttpsEnabled {
		t.Errorf("HttpsEnabled must still be sent as false")
	}

	// HTTPS on: sub-fields are emitted.
	on := off
	on.HTTPSEnabled = types.BoolValue(true)
	got = buildFileShareDistributionPointInput(on, nil, nil, &hp)
	if got.HttpsPort == nil || *got.HttpsPort != 443 || got.HttpsPassword == nil {
		t.Errorf("HTTPS sub-fields must be emitted when enabled: %+v", got)
	}
}

// TestBuildInput_FileSharingGating confirms the file-sharing fields are omitted
// when the connection type is NONE (server rejects them: "port should be blank
// when fileSharingConnectionType is NONE"), and emitted for AFP/SMB.
func TestBuildInput_FileSharingGating(t *testing.T) {
	// NONE: file-sharing fields (incl. passwords) must not be sent even if the
	// plan still carries stale values.
	none := FileShareDistributionPointResourceModel{
		Name:                      types.StringValue("dp"),
		ServerName:                types.StringValue("s"),
		FileSharingConnectionType: types.StringValue(connectionTypeNone),
		ShareName:                 types.StringValue("stale"),
		Port:                      types.Int64Value(445),
		Workgroup:                 types.StringValue("WG"),
		ReadWriteUsername:         types.StringValue("rw"),
		ReadOnlyUsername:          types.StringValue("ro"),
		HTTPSEnabled:              types.BoolValue(true),
		HTTPSPort:                 types.Int64Value(443),
	}
	rw := "x"
	got := buildFileShareDistributionPointInput(none, &rw, &rw, nil)
	if got.ShareName != nil || got.Port != nil || got.Workgroup != nil ||
		got.ReadWriteUsername != nil || got.ReadOnlyUsername != nil ||
		got.ReadWritePassword != nil || got.ReadOnlyPassword != nil {
		t.Errorf("file-sharing fields must be omitted when type is NONE: %+v", got)
	}

	// SMB: file-sharing fields are emitted.
	smb := none
	smb.FileSharingConnectionType = types.StringValue(connectionTypeSMB)
	got = buildFileShareDistributionPointInput(smb, &rw, &rw, nil)
	if got.Port == nil || *got.Port != 445 || got.ShareName == nil || got.ReadWritePassword == nil {
		t.Errorf("file-sharing fields must be emitted for SMB: %+v", got)
	}
}

// TestPasswordOnRotation covers the rotation gate: a changed *_wo_version
// re-sends the configured password; an unchanged version omits it.
func TestPasswordOnRotation(t *testing.T) {
	cfg := types.StringValue("secret")

	// Unchanged version → nil (omit, preserve stored value).
	if got := passwordOnRotation(types.Int64Value(1), types.Int64Value(1), cfg); got != nil {
		t.Errorf("unchanged version must omit password, got %v", *got)
	}
	// Both null (never set) → unchanged → nil.
	if got := passwordOnRotation(types.Int64Null(), types.Int64Null(), cfg); got != nil {
		t.Errorf("null==null version must omit password, got %v", *got)
	}
	// Bumped version → send configured password.
	if got := passwordOnRotation(types.Int64Value(2), types.Int64Value(1), cfg); got == nil || *got != "secret" {
		t.Errorf("bumped version must send password, got %v", got)
	}
	// First-time set (null → 1) → send.
	if got := passwordOnRotation(types.Int64Value(1), types.Int64Null(), cfg); got == nil || *got != "secret" {
		t.Errorf("first set must send password, got %v", got)
	}
}
