// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// CheckLiveObject returns a TestCheckFunc that fetches the server's copy of the
// resource at addr and hands it to assert.
//
// It exists for the omit-retains contract tests on the classic resources whose
// Read gates an optional block on prior state: a config that drops such a block
// plans it as removed, the PUT omits the element, and the classic merge is
// expected to leave the server's copy untouched. Terraform state cannot witness
// that — the block is gone from state by design — so the assertion has to be
// made against the wire object. If Jamf ever changes an endpoint so that an
// omitted element clears, this is the test that turns red, and it must turn
// red before any plan suppression built on the same assumption ships.
//
// get receives the Terraform resource's id; assert receives whatever get
// returned and reports the first block that no longer matches.
func CheckLiveObject[T any](addr string, get func(ctx context.Context, id string) (T, error), assert func(T) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", addr)
		}
		got, err := get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("reading live object for %s (id %s): %w", addr, rs.Primary.ID, err)
		}
		if err := assert(got); err != nil {
			return fmt.Errorf("%s (id %s): omitted block was not retained on the server: %w", addr, rs.Primary.ID, err)
		}
		return nil
	}
}

// RequireEqual reports a retained-value mismatch for one field.
func RequireEqual[V comparable](field string, want, got V) error {
	if want != got {
		return fmt.Errorf("%s: want %v, got %v", field, want, got)
	}
	return nil
}

// Deref returns the pointee or the zero value, so a nil pointer in a classic
// response compares as "absent" rather than panicking.
func Deref[V any](p *V) V {
	var zero V
	if p == nil {
		return zero
	}
	return *p
}
