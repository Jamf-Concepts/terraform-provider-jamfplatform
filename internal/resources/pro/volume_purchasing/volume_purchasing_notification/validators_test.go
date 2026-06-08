// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package volume_purchasing_notification

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// triggersValidators pulls the element-level OneOf validators off the live schema,
// so the test exercises exactly what ships (not a re-declared copy).
func triggersValidators(t *testing.T) []validator.Set {
	t.Helper()
	r := NewVolumePurchasingNotificationResource()
	var resp resource.SchemaResponse
	r.(*VolumePurchasingNotificationResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	attr, ok := resp.Schema.Attributes["triggers"].(schema.SetAttribute)
	if !ok {
		t.Fatalf("triggers attribute is not a SetAttribute")
	}
	if len(attr.Validators) == 0 {
		t.Fatal("triggers attribute has no validators (enum OneOf missing)")
	}
	return attr.Validators
}

func triggerSetHasError(t *testing.T, set types.Set) bool {
	t.Helper()
	for _, v := range triggersValidators(t) {
		req := validator.SetRequest{ConfigValue: set}
		var resp validator.SetResponse
		v.ValidateSet(context.Background(), req, &resp)
		if resp.Diagnostics.HasError() {
			return true
		}
	}
	return false
}

func TestTriggers_OneOf_RejectsUnknownValue(t *testing.T) {
	if !triggerSetHasError(t, mustStringSet(t, "BOGUS_TRIGGER")) {
		t.Error("expected a validation error for an unknown trigger value, got none")
	}
}

func TestTriggers_OneOf_AcceptsKnownValues(t *testing.T) {
	if triggerSetHasError(t, mustStringSet(t, triggerNoMoreLicenses, triggerRemovedFromAppStore)) {
		t.Error("expected no validation errors for the accepted trigger set")
	}
}
