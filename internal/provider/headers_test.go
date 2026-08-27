// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// headerMap builds the attribute value the provider schema produces for
// custom_headers.
func headerMap(t *testing.T, pairs map[string]string) types.Map {
	t.Helper()
	elements := make(map[string]attr.Value, len(pairs))
	for k, v := range pairs {
		elements[k] = types.StringValue(v)
	}
	value, diags := types.MapValue(types.StringType, elements)
	if diags.HasError() {
		t.Fatalf("MapValue() diagnostics = %v", diags)
	}
	return value
}

func TestResolveCustomHeaders(t *testing.T) {
	tests := []struct {
		name         string
		attr         types.Map
		env          string
		want         http.Header
		wantErr      string
		wantWarning  string
		wantNoHeader string
	}{
		{
			name: "unset sends nothing",
			attr: types.MapNull(types.StringType),
		},
		{
			name: "unknown attribute falls through to the environment",
			attr: types.MapUnknown(types.StringType),
			env:  "X-Proxy-Route: eu-west",
			want: http.Header{"X-Proxy-Route": []string{"eu-west"}},
		},
		{
			// An empty map is the shape an operator writing `custom_headers = {}`
			// produces, and means the same thing as leaving it out.
			name: "empty map sends nothing",
			attr: headerMap(t, map[string]string{}),
		},
		{
			name: "attribute pairs are canonicalised",
			attr: headerMap(t, map[string]string{"x-proxy-route": "eu-west"}),
			want: http.Header{"X-Proxy-Route": []string{"eu-west"}},
		},
		{
			// The arrangement the feature exists for: the proxy's own credential
			// in Authorization, Jamf's moved elsewhere.
			name: "authorization is allowed",
			attr: headerMap(t, map[string]string{"Authorization": "Basic c2VydmljZTpwYXNz"}),
			want: http.Header{"Authorization": []string{"Basic c2VydmljZTpwYXNz"}},
		},
		{
			name: "attribute overrides environment",
			attr: headerMap(t, map[string]string{"X-Proxy-Route": "us-east"}),
			env:  "X-Proxy-Route: eu-west",
			want: http.Header{"X-Proxy-Route": []string{"us-east"}},
		},
		{
			// A set attribute means the environment is never read, so even a
			// malformed value there is harmless.
			name: "attribute suppresses an unparseable environment value",
			attr: headerMap(t, map[string]string{"X-Proxy-Route": "us-east"}),
			env:  "no colon here",
			want: http.Header{"X-Proxy-Route": []string{"us-east"}},
		},
		{
			name: "environment pairs one per line",
			attr: types.MapNull(types.StringType),
			env:  "X-Proxy-Route: eu-west\nX-Trace-Id: abc123\n",
			want: http.Header{"X-Proxy-Route": []string{"eu-west"}, "X-Trace-Id": []string{"abc123"}},
		},
		{
			// Split on the FIRST colon: a bearer token or a URL routinely
			// contains colons of its own.
			name: "environment value may contain colons",
			attr: types.MapNull(types.StringType),
			env:  "Authorization: Bearer a:b:c",
			want: http.Header{"Authorization": []string{"Bearer a:b:c"}},
		},
		{
			name: "blank environment lines are skipped",
			attr: types.MapNull(types.StringType),
			env:  "\n  \nX-Proxy-Route: eu-west\n\n",
			want: http.Header{"X-Proxy-Route": []string{"eu-west"}},
		},
		{
			name: "whitespace-only environment value is unset",
			attr: types.MapNull(types.StringType),
			env:  "   \n  ",
		},
		{
			// Dropping it silently would surface much later as a proxy rejection.
			name:    "environment line without a colon errors",
			attr:    types.MapNull(types.StringType),
			env:     "X-Proxy-Route eu-west",
			wantErr: "no colon separating the header name",
		},
		{
			name:    "environment scope header is refused",
			attr:    types.MapNull(types.StringType),
			env:     "X-Environment-Id: 00000000-0000-0000-0000-000000000000",
			wantErr: "environment_id",
		},
		{
			name:    "tenant scope header is refused",
			attr:    headerMap(t, map[string]string{"X-Tenant-Id": "00000000-0000-0000-0000-000000000000"}),
			wantErr: "tenant_id",
		},
		{
			// The guard keys through http.CanonicalHeaderKey, so a spelling Go
			// canonicalises differently must still be caught rather than fail open.
			name:    "scope header is refused whatever its casing",
			attr:    headerMap(t, map[string]string{"x-tenant-ID": "00000000-0000-0000-0000-000000000000"}),
			wantErr: "tenant_id",
		},
		{
			name:    "cookie is refused",
			attr:    headerMap(t, map[string]string{"Cookie": "proxysession=abc"}),
			wantErr: "Cookie cannot be supplied as a custom header",
		},
		{
			// Warned, not refused: the header is simply dropped, and nothing
			// about the request stops working.
			name:        "user agent is dropped with a warning",
			attr:        headerMap(t, map[string]string{"User-Agent": "mine/1.0", "X-Proxy-Route": "eu-west"}),
			want:        http.Header{"X-Proxy-Route": []string{"eu-west"}},
			wantWarning: "User-Agent custom header will not be sent",
		},
		{
			name:    "invalid header name is refused",
			attr:    headerMap(t, map[string]string{"X Proxy Route": "eu-west"}),
			wantErr: "not a usable HTTP header name",
		},
		{
			name:    "empty header name is refused",
			attr:    headerMap(t, map[string]string{"": "eu-west"}),
			wantErr: "name is empty",
		},
		{
			// The commonest real cause is a trailing newline on a value read out
			// of a file or a secret store.
			name:    "newline in a value is refused",
			attr:    headerMap(t, map[string]string{"X-Proxy-Route": "eu-west\nX-Injected: yes"}),
			wantErr: "carriage return, line feed or null byte",
		},
		{
			// Both spellings canonicalise to one header, so one value would be
			// dropped without warning.
			name:    "case-insensitive duplicates are refused",
			attr:    headerMap(t, map[string]string{"X-Proxy-Route": "eu-west", "x-proxy-route": "us-east"}),
			wantErr: "are the same header",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Always stamped, so a variable set in the ambient environment cannot
			// change the result of the unset cases.
			t.Setenv(envCustomHeaders, tc.env)

			got, diags := resolveCustomHeaders(tc.attr)

			if tc.wantErr != "" {
				if !diags.HasError() {
					t.Fatalf("resolveCustomHeaders() diagnostics = %v, want an error containing %q", diags, tc.wantErr)
				}
				if !diagsContain(diags.Errors(), tc.wantErr) {
					t.Fatalf("resolveCustomHeaders() errors = %v, want one containing %q", diags.Errors(), tc.wantErr)
				}
				if got != nil {
					t.Fatalf("resolveCustomHeaders() = %v, want no headers alongside an error", got)
				}
				return
			}
			if diags.HasError() {
				t.Fatalf("resolveCustomHeaders() unexpected errors = %v", diags.Errors())
			}
			if tc.wantWarning != "" && !diagsContain(diags.Warnings(), tc.wantWarning) {
				t.Fatalf("resolveCustomHeaders() warnings = %v, want one containing %q", diags.Warnings(), tc.wantWarning)
			}
			if tc.wantWarning == "" && len(diags.Warnings()) > 0 {
				t.Fatalf("resolveCustomHeaders() unexpected warnings = %v", diags.Warnings())
			}
			if !headersEqual(got, tc.want) {
				t.Fatalf("resolveCustomHeaders() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveAuthorizationHeaderName(t *testing.T) {
	tests := []struct {
		name    string
		attr    types.String
		env     string
		headers http.Header
		want    string
		wantErr string
	}{
		{
			name: "unset leaves the credential where it is",
			attr: types.StringNull(),
		},
		{
			name: "attribute is canonicalised",
			attr: types.StringValue("x-jamf-authorization"),
			want: "X-Jamf-Authorization",
		},
		{
			name: "surrounding whitespace is trimmed",
			attr: types.StringValue("  X-Jamf-Authorization  "),
			want: "X-Jamf-Authorization",
		},
		{
			name: "environment set",
			attr: types.StringNull(),
			env:  "X-Jamf-Authorization",
			want: "X-Jamf-Authorization",
		},
		{
			name: "unknown attribute falls through to the environment",
			attr: types.StringUnknown(),
			env:  "X-Jamf-Authorization",
			want: "X-Jamf-Authorization",
		},
		{
			name: "attribute overrides environment",
			attr: types.StringValue("X-Chosen"),
			env:  "X-Ignored",
			want: "X-Chosen",
		},
		{
			// The whole arrangement: the proxy's credential stays in
			// Authorization, Jamf's moves out of the way.
			name:    "coexists with an authorization custom header",
			attr:    types.StringValue("X-Jamf-Authorization"),
			headers: http.Header{"Authorization": []string{"Basic c2VydmljZTpwYXNz"}},
			want:    "X-Jamf-Authorization",
		},
		{
			// Relocating a header onto itself removes it, and every call would
			// then answer 401 exactly as a wrong client secret does.
			name:    "authorization is refused",
			attr:    types.StringValue("Authorization"),
			wantErr: "cannot be set to Authorization itself",
		},
		{
			name:    "authorization is refused whatever its casing",
			attr:    types.StringValue("authorization"),
			wantErr: "cannot be set to Authorization itself",
		},
		{
			name:    "scope header is refused",
			attr:    types.StringValue("X-Tenant-Id"),
			wantErr: "tenant_id",
		},
		{
			// The relocation runs first and the custom headers are applied over
			// it, so the credential would be replaced by the custom value.
			name:    "a name also used as a custom header is refused",
			attr:    types.StringValue("X-Jamf-Authorization"),
			headers: http.Header{"X-Jamf-Authorization": []string{"something else"}},
			wantErr: "would replace the credential",
		},
		{
			name:    "clash is detected whatever the casing",
			attr:    types.StringValue("x-jamf-authorization"),
			headers: http.Header{"X-Jamf-Authorization": []string{"something else"}},
			wantErr: "would replace the credential",
		},
		{
			name:    "invalid header name is refused",
			attr:    types.StringValue("X Jamf Authorization"),
			wantErr: "not a usable HTTP header name",
		},
		{
			name: "whitespace-only value is unset",
			attr: types.StringValue("   "),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envAuthorizationHeaderName, tc.env)

			got, diags := resolveAuthorizationHeaderName(tc.attr, tc.headers)

			if tc.wantErr != "" {
				if !diagsContain(diags.Errors(), tc.wantErr) {
					t.Fatalf("resolveAuthorizationHeaderName() errors = %v, want one containing %q", diags.Errors(), tc.wantErr)
				}
				if got != "" {
					t.Fatalf("resolveAuthorizationHeaderName() = %q, want no name alongside an error", got)
				}
				return
			}
			if diags.HasError() {
				t.Fatalf("resolveAuthorizationHeaderName() unexpected errors = %v", diags.Errors())
			}
			if got != tc.want {
				t.Fatalf("resolveAuthorizationHeaderName() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestCredentialHeaderName pins the configure-time log field, which is the only
// place an operator can read back where the credential actually went.
func TestCredentialHeaderName(t *testing.T) {
	if got := credentialHeaderName(""); got != "Authorization" {
		t.Fatalf("credentialHeaderName(\"\") = %q, want Authorization", got)
	}
	if got := credentialHeaderName("X-Jamf-Authorization"); got != "X-Jamf-Authorization" {
		t.Fatalf("credentialHeaderName() = %q, want X-Jamf-Authorization", got)
	}
}

// TestSortedHeaderNames pins that the configure-time log line carries header
// names only. Values are a credential as often as not.
func TestSortedHeaderNames(t *testing.T) {
	headers := http.Header{
		"X-Proxy-Route": []string{"eu-west"},
		"Authorization": []string{"Basic c2VydmljZTpwYXNz"},
	}
	got := sortedHeaderNames(headers)
	want := []string{"Authorization", "X-Proxy-Route"}
	if len(got) != len(want) {
		t.Fatalf("sortedHeaderNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortedHeaderNames() = %v, want %v", got, want)
		}
	}
}

// diagsContain reports whether any diagnostic's summary or detail contains want.
func diagsContain[T interface {
	Summary() string
	Detail() string
}](diagnostics []T, want string) bool {
	for _, d := range diagnostics {
		if strings.Contains(d.Summary(), want) || strings.Contains(d.Detail(), want) {
			return true
		}
	}
	return false
}

// headersEqual compares two header sets, treating nil and empty as equal.
func headersEqual(got, want http.Header) bool {
	if len(got) != len(want) {
		return false
	}
	for name, wantValues := range want {
		gotValues := got.Values(name)
		if len(gotValues) != len(wantValues) {
			return false
		}
		for i := range wantValues {
			if gotValues[i] != wantValues[i] {
				return false
			}
		}
	}
	return true
}
