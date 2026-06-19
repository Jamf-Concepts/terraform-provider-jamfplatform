// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/ldapgroups"
)

// envDSGroupValue is the optional acceptance oracle: when set, ResolveDSGroupWireValue
// cross-checks the provider's encoded directory-service group value against it.
const envDSGroupValue = "JAMFPLATFORM_ACC_CRITERIA_DS_GROUP_VALUE"

// NewProClassicClient builds a ProClassic SDK client from the acceptance client.
func NewProClassicClient(t *testing.T) *proclassic.Client {
	t.Helper()
	return proclassic.New(NewAcceptanceClient(t))
}

// RequireSingletonStillExists returns a CheckDestroy that asserts a Pro singleton
// record still exists on the tenant after Terraform destroy — the convention for
// update-only singleton resources whose Delete is a no-op. label names the record
// in failure messages; get performs the resource-specific SDK read. A nil record
// (including a typed-nil pointer) or a read error fails the check.
func RequireSingletonStillExists(t *testing.T, label string, get func(context.Context) (any, error)) resource.TestCheckFunc {
	t.Helper()
	return func(_ *terraform.State) error {
		got, err := get(context.Background())
		if err != nil {
			return fmt.Errorf("expected %s record to persist on tenant after destroy, got error: %w", label, err)
		}
		if got == nil {
			return fmt.Errorf("expected non-nil %s record post-destroy", label)
		}
		if rv := reflect.ValueOf(got); rv.Kind() == reflect.Ptr && rv.IsNil() {
			return fmt.Errorf("expected non-nil %s record post-destroy", label)
		}
		return nil
	}
}

// ResolveDSGroupWireValue resolves a directory-service group by name to its encoded
// wire value (uuid + ldap server id). Fails the test unless the name resolves to
// exactly one group with a uuid mapping. When envDSGroupValue is set, the resolved
// value is cross-checked against it as an independent oracle.
func ResolveDSGroupWireValue(t *testing.T, name string) string {
	t.Helper()
	groups, err := ldapgroups.ResolveByName(context.Background(), pro.New(NewAcceptanceClient(t)), name)
	if err != nil {
		t.Fatalf("resolving %q: %v", name, err)
	}
	if len(groups) != 1 {
		t.Fatalf("group %q must resolve to exactly one directory group (got %d) — pick an unambiguous name", name, len(groups))
	}
	if groups[0].UUID == "" {
		t.Fatalf("group %q resolved with no uuid mapping", name)
	}
	wire := criteria.EncodeDSGroupValue(groups[0].UUID, strconv.Itoa(groups[0].LdapServerID))
	if want := os.Getenv(envDSGroupValue); want != "" && wire != want {
		t.Fatalf("provider resolved %q to %q, but %s=%q — name/value mismatch or an encoding bug", name, wire, envDSGroupValue, want)
	}
	return wire
}
