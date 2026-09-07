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
// It exists for the two classic merge contracts that Terraform state cannot
// witness on its own. The omit-retains contract: a resource whose Read gates an
// optional block on prior state drops the block from state when the config
// drops it, the PUT omits the element, and the classic merge is expected to
// leave the server's copy untouched — so the retained value can only be seen
// on the wire. The empty-clears contract (issue #384): a resource that always
// emits an optional scalar sends an empty element when the config drops the
// attribute, the state builder folds the echoed "" back to null, and a null in
// state looks identical whether the server cleared the value or kept it — so
// the clear, too, can only be seen on the wire. If Jamf ever changes an
// endpoint so that an omitted element clears, or an empty one retains, this is
// the test that turns red, and it must turn red before any plan behaviour built
// on the old assumption ships.
//
// get receives the Terraform resource's id; assert receives whatever get
// returned and reports the first field that does not match.
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
			return fmt.Errorf("%s (id %s): server copy does not match the contract under test: %w", addr, rs.Primary.ID, err)
		}
		return nil
	}
}

// RequireEqual reports a mismatch between the expected and live value of one field.
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
