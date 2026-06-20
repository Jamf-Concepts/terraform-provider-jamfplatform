// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_request_form_field

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildAppRequestFormFieldInput(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		in := buildAppRequestFormFieldInput(AppRequestFormFieldResourceModel{
			Title:       types.StringValue("Reason"),
			Description: types.StringValue("Why do you need this app?"),
			Priority:    types.Int64Value(3),
		})
		if in.Title != "Reason" {
			t.Errorf("title = %q", in.Title)
		}
		if in.Priority != 3 {
			t.Errorf("priority = %d", in.Priority)
		}
		if in.Description == nil || *in.Description != "Why do you need this app?" {
			t.Errorf("description = %v", in.Description)
		}
		if in.ID != nil {
			t.Errorf("ID must be omitted on write, got %v", in.ID)
		}
	})

	t.Run("omitted description sends nil", func(t *testing.T) {
		in := buildAppRequestFormFieldInput(AppRequestFormFieldResourceModel{
			Title:       types.StringValue("Reason"),
			Description: types.StringNull(),
			Priority:    types.Int64Value(0),
		})
		if in.Description != nil {
			t.Errorf("omitted description must send nil, got %v", *in.Description)
		}
		if in.Priority != 0 {
			t.Errorf("priority 0 must be sent, got %d", in.Priority)
		}
	})

	t.Run("empty-string description round-trips", func(t *testing.T) {
		in := buildAppRequestFormFieldInput(AppRequestFormFieldResourceModel{
			Title:       types.StringValue("Reason"),
			Description: types.StringValue(""),
			Priority:    types.Int64Value(1),
		})
		if in.Description == nil || *in.Description != "" {
			t.Errorf("explicit empty description must send \"\", got %v", in.Description)
		}
	})
}
