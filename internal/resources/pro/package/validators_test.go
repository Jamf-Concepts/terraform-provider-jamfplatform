// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// hashTypeOneOf rebuilds the OneOf validator the schema attaches to hash_type so
// the accepted and rejected sets are asserted in isolation.
func hashTypeOneOf() validator.String {
	return stringvalidator.OneOf(AllowedHashTypeValues...)
}

// TestHashTypeAllowedValues pins the wire-probed writable set so a regression
// that drops a value the live server returns (notably SHA_512, the pre-upload
// default present on 132 packages in the reference tenant) is caught here.
func TestHashTypeAllowedValues(t *testing.T) {
	want := []string{"MD5", "SHA_256", "SHA_512", "SHA3_512"}
	if len(AllowedHashTypeValues) != len(want) {
		t.Fatalf("expected %d hash types, got %d (%v)", len(want), len(AllowedHashTypeValues), AllowedHashTypeValues)
	}
	for i, v := range want {
		if AllowedHashTypeValues[i] != v {
			t.Errorf("AllowedHashTypeValues[%d] = %q, want %q", i, AllowedHashTypeValues[i], v)
		}
	}
}

func TestHashTypeOneOf_Accepts(t *testing.T) {
	for _, in := range AllowedHashTypeValues {
		t.Run(in, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("hash_type"), ConfigValue: types.StringValue(in)}
			var resp validator.StringResponse
			hashTypeOneOf().ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected %q accepted, got: %v", in, resp.Diagnostics)
			}
		})
	}
}

func TestHashTypeOneOf_Rejects(t *testing.T) {
	for _, in := range []string{"SHA512", "SHA256", "SHA_1", "SHA3_256", "sha_512", ""} {
		t.Run(in, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("hash_type"), ConfigValue: types.StringValue(in)}
			var resp validator.StringResponse
			hashTypeOneOf().ValidateString(context.Background(), req, &resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected %q rejected", in)
			}
		})
	}
}
