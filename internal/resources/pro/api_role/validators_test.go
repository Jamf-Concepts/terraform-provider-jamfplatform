// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"context"
	"errors"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type fakePrivilegeLister struct {
	privileges []string
	err        error
	calls      int
}

func (f *fakePrivilegeLister) ListApiRolePrivilegesV1(ctx context.Context) (*pro.ApiRolePrivileges, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &pro.ApiRolePrivileges{Privileges: f.privileges}, nil
}

func privilegeSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	set, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("set build diags: %v", diags)
	}
	return set
}

func TestValidatePrivileges_AllValid(t *testing.T) {
	lister := &fakePrivilegeLister{privileges: []string{"Read Computers", "Create API Roles"}}
	diags := validatePrivileges(context.Background(), lister, privilegeSet(t, "Read Computers"), path.Root("privileges"))
	if diags.HasError() {
		t.Fatalf("expected no error, got %v", diags)
	}
}

func TestValidatePrivileges_Invalid(t *testing.T) {
	lister := &fakePrivilegeLister{privileges: []string{"Read Computers"}}
	diags := validatePrivileges(context.Background(), lister, privilegeSet(t, "Read Computers", "Not Real"), path.Root("privileges"))
	if !diags.HasError() {
		t.Fatalf("expected an error for unknown privilege")
	}
}

func TestValidatePrivileges_TransportErrorWarns(t *testing.T) {
	lister := &fakePrivilegeLister{err: errors.New("boom")}
	diags := validatePrivileges(context.Background(), lister, privilegeSet(t, "Anything"), path.Root("privileges"))
	if diags.HasError() {
		t.Fatalf("transport error must downgrade to warning, got error: %v", diags)
	}
	if diags.WarningsCount() == 0 {
		t.Fatalf("expected a warning on transport error")
	}
}

func TestValidatePrivileges_NilListerNoop(t *testing.T) {
	diags := validatePrivileges(context.Background(), nil, privilegeSet(t, "Anything"), path.Root("privileges"))
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("nil lister must be a no-op, got %v", diags)
	}
}

func TestValidatePrivileges_UnknownSetNoop(t *testing.T) {
	lister := &fakePrivilegeLister{privileges: []string{"Read Computers"}}
	diags := validatePrivileges(context.Background(), lister, types.SetUnknown(types.StringType), path.Root("privileges"))
	if diags.HasError() {
		t.Fatalf("unknown set must be a no-op, got %v", diags)
	}
	if lister.calls != 0 {
		t.Fatalf("unknown set must not call the API, calls=%d", lister.calls)
	}
}
