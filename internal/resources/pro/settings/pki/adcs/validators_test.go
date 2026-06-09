// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package adcs

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The validator reads connector_mode + two scalars (adcs_url, api_client_id) and
// two blocks (server_certificate, client_certificate), checking only
// IsNull/IsUnknown on each, so a single dummy inner attribute per block suffices.

var (
	certInnerType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"data_wo": tftypes.String}}

	adcsRootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"connector_mode":     tftypes.String,
		"adcs_url":           tftypes.String,
		"api_client_id":      tftypes.String,
		"server_certificate": certInnerType,
		"client_certificate": certInnerType,
	}}
)

func minimalAdcsSchema() schema.Schema {
	certBlock := schema.SingleNestedAttribute{
		Optional:   true,
		Attributes: map[string]schema.Attribute{"data_wo": schema.StringAttribute{Optional: true}},
	}
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"connector_mode":     schema.StringAttribute{Required: true},
		"adcs_url":           schema.StringAttribute{Optional: true},
		"api_client_id":      schema.StringAttribute{Optional: true},
		"server_certificate": certBlock,
		"client_certificate": certBlock,
	}}
}

func strVal(v string) tftypes.Value {
	if v == "" {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, v)
}
func strUnknown() tftypes.Value { return tftypes.NewValue(tftypes.String, tftypes.UnknownValue) }

func certBlockVal(present bool) tftypes.Value {
	if !present {
		return tftypes.NewValue(certInnerType, nil)
	}
	return tftypes.NewValue(certInnerType, map[string]tftypes.Value{"data_wo": tftypes.NewValue(tftypes.String, "x")})
}
func certBlockUnknown() tftypes.Value { return tftypes.NewValue(certInnerType, tftypes.UnknownValue) }

func adcsConfig(mode, adcsURL, apiClientID tftypes.Value, server, client tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: minimalAdcsSchema(),
		Raw: tftypes.NewValue(adcsRootObjType, map[string]tftypes.Value{
			"connector_mode":     mode,
			"adcs_url":           adcsURL,
			"api_client_id":      apiClientID,
			"server_certificate": server,
			"client_certificate": client,
		}),
	}
}

func validatorPaths(cfg tfsdk.Config) map[string]bool {
	var resp resource.ValidateConfigResponse
	connectorModeConfigValidator{}.ValidateResource(context.Background(), resource.ValidateConfigRequest{Config: cfg}, &resp)
	paths := make(map[string]bool)
	for _, d := range resp.Diagnostics {
		if dwp, ok := d.(diag.DiagnosticWithPath); ok {
			paths[dwp.Path().String()] = true
		}
	}
	return paths
}

func validatorErrCount(cfg tfsdk.Config) int {
	var resp resource.ValidateConfigResponse
	connectorModeConfigValidator{}.ValidateResource(context.Background(), resource.ValidateConfigRequest{Config: cfg}, &resp)
	n := 0
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError {
			n++
		}
	}
	return n
}

// --- INBOUND ---------------------------------------------------------------

func TestValidator_Inbound_Valid(t *testing.T) {
	cfg := adcsConfig(strVal(connectorModeInbound), strVal("connector.example.com"), strVal(""), certBlockVal(true), certBlockVal(true))
	if n := validatorErrCount(cfg); n != 0 {
		t.Errorf("INBOUND with url+certs must pass; got %d errors", n)
	}
}

func TestValidator_Inbound_MissingURL(t *testing.T) {
	cfg := adcsConfig(strVal(connectorModeInbound), strVal(""), strVal(""), certBlockVal(true), certBlockVal(true))
	if !validatorPaths(cfg)["adcs_url"] {
		t.Error("INBOUND without adcs_url must error on adcs_url")
	}
}

func TestValidator_Inbound_MissingServerCert(t *testing.T) {
	cfg := adcsConfig(strVal(connectorModeInbound), strVal("connector.example.com"), strVal(""), certBlockVal(false), certBlockVal(true))
	if !validatorPaths(cfg)["server_certificate"] {
		t.Error("INBOUND without server_certificate must error")
	}
}

func TestValidator_Inbound_ForbidsApiClientID(t *testing.T) {
	cfg := adcsConfig(strVal(connectorModeInbound), strVal("connector.example.com"), strVal("uuid"), certBlockVal(true), certBlockVal(true))
	if !validatorPaths(cfg)["api_client_id"] {
		t.Error("INBOUND with api_client_id must error (forbidden)")
	}
}

// --- OUTBOUND --------------------------------------------------------------

func TestValidator_Outbound_Valid(t *testing.T) {
	cfg := adcsConfig(strVal(connectorModeOutbound), strVal(""), strVal("11111111-2222-3333-4444-555555555555"), certBlockVal(false), certBlockVal(false))
	if n := validatorErrCount(cfg); n != 0 {
		t.Errorf("OUTBOUND with api_client_id must pass; got %d errors", n)
	}
}

func TestValidator_Outbound_MissingApiClientID(t *testing.T) {
	cfg := adcsConfig(strVal(connectorModeOutbound), strVal(""), strVal(""), certBlockVal(false), certBlockVal(false))
	if !validatorPaths(cfg)["api_client_id"] {
		t.Error("OUTBOUND without api_client_id must error")
	}
}

func TestValidator_Outbound_ForbidsURLAndCerts(t *testing.T) {
	cfg := adcsConfig(strVal(connectorModeOutbound), strVal("connector.example.com"), strVal("uuid"), certBlockVal(true), certBlockVal(true))
	p := validatorPaths(cfg)
	for _, want := range []string{"adcs_url", "server_certificate", "client_certificate"} {
		if !p[want] {
			t.Errorf("OUTBOUND with %s must error (forbidden)", want)
		}
	}
}

// --- defer-on-unknown ------------------------------------------------------

func TestValidator_UnknownMode_Defers(t *testing.T) {
	cfg := adcsConfig(strUnknown(), strVal(""), strVal(""), certBlockVal(false), certBlockVal(false))
	if n := validatorErrCount(cfg); n != 0 {
		t.Errorf("unknown connector_mode must defer; got %d errors", n)
	}
}

func TestValidator_UnknownRequiredBlock_Defers(t *testing.T) {
	cfg := adcsConfig(strVal(connectorModeInbound), strVal("connector.example.com"), strVal(""), certBlockUnknown(), certBlockVal(true))
	if validatorPaths(cfg)["server_certificate"] {
		t.Error("unknown server_certificate must defer, not error")
	}
}
