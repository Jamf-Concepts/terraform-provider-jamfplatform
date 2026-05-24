// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMultipleOfInt64_AcceptsMultiples(t *testing.T) {
	t.Parallel()
	v := MultipleOfInt64(1440)
	for _, val := range []int64{0, 1440, 2880, 14400} {
		req := validator.Int64Request{
			Path:        path.Root("allow_deferral_minutes"),
			ConfigValue: types.Int64Value(val),
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected %d to be accepted, got diagnostics %v", val, resp.Diagnostics)
		}
	}
}

func TestMultipleOfInt64_RejectsNonMultiples(t *testing.T) {
	t.Parallel()
	v := MultipleOfInt64(1440)
	for _, val := range []int64{1, 60, 1441, 2881} {
		req := validator.Int64Request{
			Path:        path.Root("allow_deferral_minutes"),
			ConfigValue: types.Int64Value(val),
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatalf("expected %d to be rejected, got no error", val)
		}
	}
}

func TestMultipleOfInt64_SkipsNullAndUnknown(t *testing.T) {
	t.Parallel()
	v := MultipleOfInt64(1440)
	for name, value := range map[string]types.Int64{
		"null":    types.Int64Null(),
		"unknown": types.Int64Unknown(),
	} {
		req := validator.Int64Request{
			Path:        path.Root("allow_deferral_minutes"),
			ConfigValue: value,
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected %s to be skipped, got diagnostics %v", name, resp.Diagnostics)
		}
	}
}
