// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure TerraformLogger implements client.Logger interface
var _ jamfplatform.Logger = (*TerraformLogger)(nil)

// TerraformLogger implements the client.Logger interface using tflog
type TerraformLogger struct{}

// NewTerraformLogger creates a new TerraformLogger
func NewTerraformLogger() *TerraformLogger {
	return &TerraformLogger{}
}

// maxLoggedBodyBytes caps the rendered body length before truncation. Applied
// to the pretty-printed string so truncation never lands mid-token.
const maxLoggedBodyBytes = 5000

// LogRequest logs HTTP request details using tflog at DEBUG level. Bodies
// are pretty-printed by content sniff (JSON / XML / opaque).
func (l *TerraformLogger) LogRequest(ctx context.Context, method, url string, body []byte) {
	fields := map[string]any{
		"method": method,
		"url":    url,
	}

	if len(body) > 0 {
		fields["request_body"] = formatBody(body)
	}

	tflog.Debug(ctx, "HTTP Request", fields)
}

// LogResponse logs HTTP response details using tflog at DEBUG level. Bodies
// are pretty-printed by content sniff (JSON / XML / opaque).
func (l *TerraformLogger) LogResponse(ctx context.Context, statusCode int, headers http.Header, body []byte) {
	fields := map[string]any{
		"status_code": statusCode,
	}

	if len(headers) > 0 {
		fields["response_headers"] = formatHeaders(headers)
	}

	if len(body) > 0 {
		fields["response_body"] = formatBody(body)
	}

	tflog.Debug(ctx, "HTTP Response", fields)
}

// formatBody pretty-prints JSON and XML payloads and truncates to keep
// debug logs readable. Falls back to the raw string when sniffing fails or
// the body is neither JSON nor XML.
func formatBody(body []byte) string {
	pretty := prettyPrint(body)
	if len(pretty) > maxLoggedBodyBytes {
		return pretty[:maxLoggedBodyBytes] + "... (truncated)"
	}
	return pretty
}

// prettyPrint sniffs the first non-whitespace byte to pick a formatter.
// '{' or '[' → JSON. '<' → XML. Anything else → raw string.
func prettyPrint(body []byte) string {
	leading := body
	for len(leading) > 0 {
		switch leading[0] {
		case ' ', '\t', '\r', '\n':
			leading = leading[1:]
			continue
		}
		break
	}
	if len(leading) == 0 {
		return string(body)
	}

	switch leading[0] {
	case '{', '[':
		var buf bytes.Buffer
		if err := json.Indent(&buf, body, "", "  "); err == nil {
			return buf.String()
		}
	case '<':
		if out, err := indentXML(body); err == nil {
			return out
		}
	}
	return string(body)
}

// sensitiveHeaders are response/request headers whose values may carry
// credentials or session state. Logged as REDACTED to avoid leaks in
// tflog output. Compared case-insensitively against canonical MIME keys
// (http.Header normalises to canonical form).
var sensitiveHeaders = map[string]struct{}{
	"Authorization":       {},
	"Proxy-Authorization": {},
	"Cookie":              {},
	"Set-Cookie":          {},
	"X-Api-Key":           {},
}

// formatHeaders renders an http.Header as a sorted multi-line block of
// "Key: value" pairs. Multi-value headers are comma-joined. Sensitive
// headers (Authorization, Set-Cookie, etc.) are redacted.
func formatHeaders(h http.Header) string {
	if len(h) == 0 {
		return ""
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k)
		sb.WriteString(": ")
		if _, redact := sensitiveHeaders[http.CanonicalHeaderKey(k)]; redact {
			sb.WriteString("REDACTED")
			continue
		}
		sb.WriteString(strings.Join(h[k], ", "))
	}
	return sb.String()
}

// indentXML re-encodes an XML payload with two-space indentation by streaming
// tokens through xml.Decoder → xml.Encoder. Whitespace-only CharData tokens
// are dropped so the existing wire whitespace does not double up against the
// new indentation.
func indentXML(body []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(body))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if cd, ok := tok.(xml.CharData); ok {
			if strings.TrimSpace(string(cd)) == "" {
				continue
			}
		}
		if err := enc.EncodeToken(tok); err != nil {
			return "", err
		}
	}
	if err := enc.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
