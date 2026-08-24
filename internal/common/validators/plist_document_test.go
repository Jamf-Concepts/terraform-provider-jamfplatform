// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/validators"
)

// payloadBody is the settings dict from issue #326, shared by the wrapped and
// unwrapped cases so the wrapper is the only difference between them.
const payloadBody = `<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadType</key>
			<string>com.apple.ManagedClient.preferences</string>
		</dict>
	</array>
</dict>`

func validatePlistDocument(t *testing.T, value types.String) validator.StringResponse {
	t.Helper()
	req := validator.StringRequest{
		Path:        path.Root("general").AtName("payloads"),
		ConfigValue: value,
	}
	resp := validator.StringResponse{}
	validators.PlistDocument().ValidateString(context.Background(), req, &resp)
	return resp
}

func TestPlistDocument_AcceptsCompleteDocuments(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		// Wire-probed 2026-08-24: Jamf Pro accepts all three forms.
		"root element only":             "<plist version=\"1.0\">\n" + payloadBody + "\n</plist>",
		"declaration and root":          "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<plist version=\"1.0\">\n" + payloadBody + "\n</plist>",
		"declaration, doctype and root": "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1\">\n" + payloadBody + "\n</plist>",
		"leading whitespace":            "\n\t<plist version=\"1.0\">" + payloadBody + "</plist>",
		"comment before root":           "<!-- built by templatefile -->\n<plist version=\"1.0\">" + payloadBody + "</plist>",
	}

	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if resp := validatePlistDocument(t, types.StringValue(payload)); resp.Diagnostics.HasError() {
				t.Fatalf("expected no error, got: %v", resp.Diagnostics.Errors())
			}
		})
	}
}

func TestPlistDocument_RejectsFragment(t *testing.T) {
	t.Parallel()

	resp := validatePlistDocument(t, types.StringValue(payloadBody))
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for a bare <dict> fragment, got none")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	for _, want := range []string{"<dict>", "<plist version=\"1.0\">"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail should mention %q, got:\n%s", want, detail)
		}
	}
	withPath, ok := resp.Diagnostics.Errors()[0].(diag.DiagnosticWithPath)
	if !ok {
		t.Fatal("expected an attribute-scoped diagnostic")
	}
	if got := withPath.Path().String(); got != "general.payloads" {
		t.Errorf("expected the error on general.payloads, got %s", got)
	}
}

func TestPlistDocument_RejectsEmpty(t *testing.T) {
	t.Parallel()

	for name, payload := range map[string]string{"empty": "", "whitespace": " \n\t "} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp := validatePlistDocument(t, types.StringValue(payload))
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected an error for an empty payload, got none")
			}
			if summary := resp.Diagnostics.Errors()[0].Summary(); summary != "Empty payload" {
				t.Errorf("unexpected summary: %s", summary)
			}
		})
	}
}

// TestPlistDocument_DefersWhenUnresolvable covers everything the validator must
// hand to Jamf Pro rather than reject itself: values not yet known at validate
// time, and payloads it cannot tokenise. False-rejecting a payload Jamf Pro
// would accept is worse than letting an unparseable one reach the API.
func TestPlistDocument_DefersWhenUnresolvable(t *testing.T) {
	t.Parallel()

	cases := map[string]types.String{
		"null":                  types.StringNull(),
		"unknown":               types.StringUnknown(),
		"binary plist":          types.StringValue("bplist00\xd1\x01\x02"),
		"no element at all":     types.StringValue("not xml at all"),
		"declaration only":      types.StringValue("<?xml version=\"1.0\" encoding=\"UTF-8\"?>"),
		"non-utf8 declaration":  types.StringValue("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><plist version=\"1.0\">" + payloadBody + "</plist>"),
		"unterminated fragment": types.StringValue("<plist version=\"1.0\"><dict>"),
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if resp := validatePlistDocument(t, value); resp.Diagnostics.HasError() {
				t.Fatalf("expected no error, got: %v", resp.Diagnostics.Errors())
			}
		})
	}
}
