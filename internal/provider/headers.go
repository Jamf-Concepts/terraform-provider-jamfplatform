// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// scopeHeaderNames are the two request headers that carry the API integration
// scope. The provider owns them: they are set from `environment_id` /
// `tenant_id`, and an overriding value is refused by the gateway with the same
// 403 OWNERSHIP_FORBIDDEN a genuinely mismatched credential gives — so a
// silently accepted override is close to undiagnosable from the error alone.
//
// Keyed through http.CanonicalHeaderKey rather than trusted to already be in
// canonical form, because a lookup miss here fails open.
var scopeHeaderNames = map[string]string{
	http.CanonicalHeaderKey("X-Environment-Id"): "environment_id",
	http.CanonicalHeaderKey("X-Tenant-Id"):      "tenant_id",
}

// resolveCustomHeaders turns the `custom_headers` attribute, or the environment
// variable standing in for it, into the header set every outbound request
// carries.
//
// The attribute always takes precedence over the environment variable, which is
// then not consulted at all — the convention the other provider attributes
// follow. A null or empty map from the attribute counts as unset rather than as
// "send nothing", so an operator can leave the block out on a workstation and
// still have a runner fleet inject the proxy's headers from its environment.
//
// Every rejection here is a rejection the underlying client would otherwise make
// silently, or one the wire would make in terms that name the wrong cause. The
// scope headers are the clearest case, but an invalid header name is as bad: it
// surfaces at request time as a transport error with no hint that a provider
// attribute produced it.
//
// Cookie is refused, which is stricter than the client underneath allows.
// Supplied headers replace rather than add, so a Cookie here displaces Jamf
// Cloud's session-pinning cookie, and the consequence — a read taken straight
// after a write landing on a node that has not caught up — surfaces as an
// intermittent inconsistent-result error against some unrelated resource, which
// nobody traces back to a header in provider configuration. The legitimate need
// is thin by comparison: the cookie jar already replays whatever a proxy sets on
// a response, so a static cookie supplied here buys a proxy nothing it cannot
// get by naming a header of its own. Refusing is also the direction that can be
// undone later without breaking a working configuration.
func resolveCustomHeaders(attr types.Map) (http.Header, diag.Diagnostics) {
	var diags diag.Diagnostics

	pairs, ok, parseDiags := customHeaderPairs(attr)
	diags.Append(parseDiags...)
	if !ok || diags.HasError() {
		return nil, diags
	}
	if len(pairs) == 0 {
		return nil, diags
	}

	headers := make(http.Header, len(pairs))
	seen := make(map[string]string, len(pairs))
	for _, name := range sortedKeys(pairs) {
		value := pairs[name]
		canonical := http.CanonicalHeaderKey(name)

		if previous, duplicate := seen[canonical]; duplicate {
			diags.AddError(
				"Duplicate Custom Header",
				fmt.Sprintf("%q and %q are the same header — HTTP header names are case-insensitive — so only one "+
					"of the two values could be sent and the other would be dropped without warning. Keep "+
					"whichever value the proxy expects and remove the other.", previous, name),
			)
			continue
		}
		seen[canonical] = name

		switch {
		case strings.TrimSpace(name) == "":
			diags.AddError(
				"Invalid Custom Header",
				"A custom header name is empty. Remove the entry, or give it the name the proxy expects.",
			)
			continue
		case !validHeaderName(name):
			diags.AddError(
				"Invalid Custom Header",
				fmt.Sprintf("%q is not a usable HTTP header name. Header names may contain letters, digits and "+
					"the characters !#$%%&'*+-.^_`|~ — no spaces, colons or quotes.", name),
			)
			continue
		case !validHeaderValue(value):
			diags.AddError(
				"Invalid Custom Header",
				fmt.Sprintf("The value of %s contains a carriage return, line feed or null byte, which cannot be "+
					"sent in an HTTP header. Check for a stray newline at the end of a value read from a file "+
					"or a secret store.", canonical),
			)
			continue
		}

		if attrName, reserved := scopeHeaderNames[canonical]; reserved {
			diags.AddError(
				"Reserved Custom Header",
				fmt.Sprintf("%s carries the API integration scope and is set by the provider, so it cannot be "+
					"supplied as a custom header. Set `%s` instead — see that attribute for how the two "+
					"scopes differ.", canonical, attrName),
			)
			continue
		}

		if canonical == "Cookie" {
			diags.AddError(
				"Reserved Custom Header",
				"Cookie cannot be supplied as a custom header. A supplied header replaces rather than adds to "+
					"what the provider sends, and the provider sends the session cookie Jamf Cloud uses to "+
					"keep this client on one application node — losing it means a read taken straight after a "+
					"write can miss the change, which surfaces later as an inconsistent-result error or "+
					"phantom drift on an unrelated resource.\n\n"+
					"A proxy that needs a cookie of its own does not need this attribute: the provider keeps a "+
					"cookie jar, so any cookie the proxy sets on a response is sent back on later requests "+
					"automatically. If the proxy instead expects a fixed credential, have it read that "+
					"credential from a header of its own and set `custom_headers` to that name.",
			)
			continue
		}

		if canonical == "User-Agent" {
			diags.AddWarning(
				"Ignored Custom Header",
				"The User-Agent custom header will not be sent: the provider identifies itself to Jamf with its "+
					"own name and version, and that identification is applied after the custom headers. "+
					"Remove the entry to silence this warning.",
			)
			continue
		}

		headers.Set(canonical, value)
	}

	if diags.HasError() {
		return nil, diags
	}
	return headers, diags
}

// resolveAuthorizationHeaderName resolves `authorization_header_name`, the header
// the Jamf bearer credential is moved into, and checks it against the custom
// headers it has to coexist with.
//
// The three refusals all describe the same failure from different directions: a
// request that reaches Jamf carrying no usable credential, and answers 401 or 403
// in terms that point at the client secret or the scope rather than at this
// attribute.
//
//   - `Authorization` is where the bearer already is, and relocating a header
//     onto itself removes it.
//   - A scope header would have its scope overwritten by the bearer.
//   - A name that is also a custom header would have the bearer overwritten by
//     that header's value, since the relocation happens first and the custom
//     headers are applied over the result.
func resolveAuthorizationHeaderName(attr types.String, headers http.Header) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	name := strings.TrimSpace(attr.ValueString())
	if attr.IsNull() || attr.IsUnknown() {
		name = strings.TrimSpace(getenv(envAuthorizationHeaderName))
	}
	if name == "" {
		return "", diags
	}

	canonical := http.CanonicalHeaderKey(name)

	if !validHeaderName(name) {
		diags.AddError(
			"Invalid Authorization Header Name",
			fmt.Sprintf("%q is not a usable HTTP header name. Header names may contain letters, digits and the "+
				"characters !#$%%&'*+-.^_`|~ — no spaces, colons or quotes.", name),
		)
		return "", diags
	}

	if canonical == "Authorization" {
		diags.AddError(
			"Invalid Authorization Header Name",
			"`authorization_header_name` moves the Jamf credential out of the Authorization header, so it cannot "+
				"be set to Authorization itself — doing so would leave the request with no credential at all, "+
				"and Jamf would answer 401 exactly as a wrong client secret does. Remove the attribute to "+
				"leave the credential where it is.",
		)
		return "", diags
	}

	if attrName, reserved := scopeHeaderNames[canonical]; reserved {
		diags.AddError(
			"Invalid Authorization Header Name",
			fmt.Sprintf("The Jamf credential cannot be moved into %s: that header carries the API integration "+
				"scope set from `%s`, and overwriting it is refused by Jamf with 403 OWNERSHIP_FORBIDDEN. "+
				"Choose the header name the proxy expects the credential under.", canonical, attrName),
		)
		return "", diags
	}

	if len(headers.Values(canonical)) > 0 {
		diags.AddError(
			"Conflicting Authorization Header Name",
			fmt.Sprintf("%s is set as a custom header and is also where `authorization_header_name` moves the "+
				"Jamf credential, so the custom value would replace the credential and every request would "+
				"fail authentication. Point `authorization_header_name` at the header the proxy expects the "+
				"Jamf credential under, and keep the proxy's own credential under a different name.", canonical),
		)
		return "", diags
	}

	return canonical, diags
}

// customHeaderPairs reads the header pairs from the attribute, falling back to
// the environment variable. The second return reports whether a usable value was
// found at all, separating "nothing configured" from "configured and rejected".
//
// The environment form is one `Name: value` pair per line, splitting on the first
// colon so a value may contain colons of its own — which a bearer token or a URL
// routinely does. A line with no colon is an error rather than a skipped line:
// the operator meant to send something, and dropping it would show up much later
// as a proxy rejection.
func customHeaderPairs(attr types.Map) (map[string]string, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !attr.IsNull() && !attr.IsUnknown() {
		elements := attr.Elements()
		pairs := make(map[string]string, len(elements))
		for name, element := range elements {
			value, ok := element.(types.String)
			if !ok || value.IsUnknown() {
				diags.AddError(
					"Invalid Custom Header",
					fmt.Sprintf("The value of the %q custom header is not known at plan time. Header values must "+
						"be resolvable before the provider is configured, so they cannot come from a "+
						"resource attribute created in the same run.", name),
				)
				continue
			}
			if value.IsNull() {
				diags.AddError(
					"Invalid Custom Header",
					fmt.Sprintf("The %q custom header has a null value. Give it the value the proxy expects, or "+
						"remove the entry.", name),
				)
				continue
			}
			pairs[name] = value.ValueString()
		}
		return pairs, true, diags
	}

	raw := getenv(envCustomHeaders)
	if strings.TrimSpace(raw) == "" {
		return nil, false, diags
	}

	pairs := map[string]string{}
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, value, found := strings.Cut(line, ":")
		if !found {
			diags.AddError(
				fmt.Sprintf("Invalid %s", envCustomHeaders),
				fmt.Sprintf("The line %q has no colon separating the header name from its value. Set one "+
					"`Name: value` pair per line, for example:\n\n  X-Proxy-Route: eu-west\n  "+
					"Authorization: Basic %s", line, "abc123"),
			)
			continue
		}
		pairs[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}
	return pairs, true, diags
}

// validHeaderName reports whether name is an RFC 9110 field name — one or more
// token characters. Checked here rather than left to the transport, which reports
// a malformed name as a bare request error naming neither the header nor the
// attribute it came from.
func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	const tokenPunctuation = "!#$%&'*+-.^_`|~"
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			strings.ContainsRune(tokenPunctuation, r):
		default:
			return false
		}
	}
	return true
}

// validHeaderValue reports whether value can be sent as a field value. Only the
// characters that would terminate or split the header are rejected; anything else
// is the proxy's business, and a value read from a secret store may legitimately
// hold non-ASCII bytes.
func validHeaderValue(value string) bool {
	return !strings.ContainsAny(value, "\r\n\x00")
}

// sortedKeys returns the map's keys in a stable order, so a configuration with
// more than one rejected header produces the same diagnostics every run.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sortedHeaderNames returns the configured custom header names, for the
// configure-time log line. Names only: a header configured for a proxy routinely
// holds a credential, so the values must not be logged, but knowing which
// headers are in play is most of what makes a proxy misconfiguration readable.
func sortedHeaderNames(headers http.Header) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// credentialHeaderName reports which header the Jamf credential is sent in, for
// the configure-time log line.
func credentialHeaderName(relocated string) string {
	if relocated == "" {
		return "Authorization"
	}
	return relocated
}
