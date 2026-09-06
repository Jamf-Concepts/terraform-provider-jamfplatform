// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account_group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestLdapServerIDForWrite pins the clear token: a null ldap_server_id must
// reach the wire as -1 (the only value the endpoint accepts as "none"), a
// configured id is sent as-is, and unknown stays off the wire.
func TestLdapServerIDForWrite(t *testing.T) {
	if got := ldapServerIDForWrite(types.Int64Null()); got == nil || *got != ldapServerNone {
		t.Errorf("null must emit %d, got %v", ldapServerNone, got)
	}
	if got := ldapServerIDForWrite(types.Int64Value(7)); got == nil || *got != 7 {
		t.Errorf("configured id must be sent as-is, got %v", got)
	}
	if got := ldapServerIDForWrite(types.Int64Unknown()); got != nil {
		t.Errorf("unknown must be omitted, got %d", *got)
	}
}
