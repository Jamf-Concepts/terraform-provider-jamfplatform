// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestConfigureWarnsAboutTheNamespaceMigration asserts that every Configure
// call carries the migration notice, including one the provider rejects for
// missing credentials — an operator whose configuration cannot authenticate
// still needs to hear that the provider address is changing.
func TestConfigureWarnsAboutTheNamespaceMigration(t *testing.T) {
	t.Setenv(envBaseURL, "")
	t.Setenv(envClientID, "")
	t.Setenv(envClientSecret, "")
	t.Setenv(envEnvironmentID, "")
	t.Setenv(envTenantID, "")

	server := providerserver.NewProtocol6(New("test")())()

	schemaResp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}

	configType := schemaResp.Provider.ValueType()
	config, err := tfprotov6.NewDynamicValue(configType, tftypes.NewValue(configType, nil))
	if err != nil {
		t.Fatalf("NewDynamicValue: %v", err)
	}

	resp, err := server.ConfigureProvider(context.Background(), &tfprotov6.ConfigureProviderRequest{Config: &config})
	if err != nil {
		t.Fatalf("ConfigureProvider: %v", err)
	}

	var found bool
	for _, d := range resp.Diagnostics {
		if d.Severity != tfprotov6.DiagnosticSeverityWarning || d.Summary != namespaceMigrationSummary {
			continue
		}
		found = true
		for _, want := range []string{
			"jamf/jamfplatform",
			"jamf-concepts/jamfplatform",
			"terraform state replace-provider",
			"terraform init -upgrade",
			"tofu",
		} {
			if !strings.Contains(d.Detail, want) {
				t.Errorf("migration notice detail is missing %q:\n%s", want, d.Detail)
			}
		}
	}

	if !found {
		t.Fatalf("Configure did not raise the %q warning; diagnostics: %+v", namespaceMigrationSummary, resp.Diagnostics)
	}
}

// TestNamespaceMigrationDetailCarriesNoMarkdown guards the plain-text contract:
// Terraform prints diagnostic detail verbatim, so a backtick or an asterisk
// reaches the operator as itself rather than as formatting.
func TestNamespaceMigrationDetailCarriesNoMarkdown(t *testing.T) {
	for _, char := range []string{"`", "*"} {
		if strings.Contains(namespaceMigrationDetail, char) {
			t.Errorf("namespaceMigrationDetail contains markdown %q; diagnostics render as plain text", char)
		}
	}
}
