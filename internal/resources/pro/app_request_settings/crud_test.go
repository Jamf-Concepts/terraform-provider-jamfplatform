// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_request_settings

import (
	"errors"
	"strings"
	"testing"
)

func TestAppRequestWriteErrorDiagnostic(t *testing.T) {
	t.Parallel()

	t.Run("missing form field is named", func(t *testing.T) {
		t.Parallel()

		err := errors.New("UpdateAppRequestSettingsV1: API request failed with status 400, traceId abc (method=PUT, url=https://example.jamfcloud.com/pro/v1/app-request/settings): [INVALID_SIZE] formInputFields: At least one form field is required")
		summary, detail := appRequestWriteErrorDiagnostic("Error updating Jamf Pro App Request settings", err)

		if !strings.Contains(summary, "form field") {
			t.Errorf("summary does not name the prerequisite: %s", summary)
		}
		if !strings.Contains(detail, "jamfplatform_pro_app_request_form_field") {
			t.Errorf("detail does not name the resource that satisfies the prerequisite: %s", detail)
		}
		if !strings.Contains(detail, err.Error()) {
			t.Errorf("detail drops the underlying error: %s", detail)
		}
	})

	t.Run("any other failure passes through", func(t *testing.T) {
		t.Parallel()

		err := errors.New("API request failed with status 400: [INVALID_ID] static group not found [445]")
		summary, detail := appRequestWriteErrorDiagnostic("Error updating Jamf Pro App Request settings", err)

		if summary != "Error updating Jamf Pro App Request settings" {
			t.Errorf("summary was rewritten: %s", summary)
		}
		if detail != err.Error() {
			t.Errorf("detail was rewritten: %s", detail)
		}
	})
}
