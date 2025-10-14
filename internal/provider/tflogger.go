// Copyright 2025 Jamf Software LLC.

package provider

import (
	"context"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure TerraformLogger implements client.Logger interface
var _ client.Logger = (*TerraformLogger)(nil)

// TerraformLogger implements the client.Logger interface using tflog
type TerraformLogger struct{}

// NewTerraformLogger creates a new TerraformLogger
func NewTerraformLogger() *TerraformLogger {
	return &TerraformLogger{}
}

// LogRequest logs HTTP request details using tflog at DEBUG level
func (l *TerraformLogger) LogRequest(ctx context.Context, method, url string, body []byte) {
	fields := map[string]interface{}{
		"method": method,
		"url":    url,
	}

	if len(body) > 0 {
		fields["request_body"] = string(body)
	}

	tflog.Debug(ctx, "HTTP Request", fields)
}

// LogResponse logs HTTP response details using tflog at DEBUG level
func (l *TerraformLogger) LogResponse(ctx context.Context, statusCode int, body []byte) {
	fields := map[string]interface{}{
		"status_code": statusCode,
	}

	if len(body) > 0 {
		bodyStr := string(body)
		if len(bodyStr) > 5000 {
			bodyStr = bodyStr[:5000] + "... (truncated)"
		}
		fields["response_body"] = bodyStr
	}

	tflog.Debug(ctx, "HTTP Response", fields)
}
