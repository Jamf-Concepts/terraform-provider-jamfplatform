// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_invitation

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// invitationTypeOneOf rebuilds the same OneOf validator the schema attaches to
// invitation_type so the accepted/rejected sets are asserted in isolation.
func invitationTypeOneOf() validator.String {
	return stringvalidator.OneOf(validInvitationTypes...)
}

func TestInvitationTypeOneOf_Accepts(t *testing.T) {
	for _, in := range []string{"USER_INITIATED_URL", "USER_INITIATED_EMAIL"} {
		t.Run(in, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("invitation_type"), ConfigValue: types.StringValue(in)}
			var resp validator.StringResponse
			invitationTypeOneOf().ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected %q accepted, got: %v", in, resp.Diagnostics)
			}
		})
	}
}

func TestInvitationTypeOneOf_Rejects(t *testing.T) {
	// DEP_CUSTOM_ENROLL is observed on the wire but is DEP-system-generated and
	// not user-creatable, so it must be rejected.
	for _, in := range []string{"DEP_CUSTOM_ENROLL", "", "user_initiated_url", "RANDOM"} {
		t.Run(in, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("invitation_type"), ConfigValue: types.StringValue(in)}
			var resp validator.StringResponse
			invitationTypeOneOf().ValidateString(context.Background(), req, &resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected %q rejected", in)
			}
		})
	}
}

func TestInvitationTypeOneOf_SkipsNullAndUnknown(t *testing.T) {
	for name, in := range map[string]types.String{"null": types.StringNull(), "unknown": types.StringUnknown()} {
		t.Run(name, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("invitation_type"), ConfigValue: in}
			var resp validator.StringResponse
			invitationTypeOneOf().ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected no diagnostics for %s, got: %v", name, resp.Diagnostics)
			}
		})
	}
}
