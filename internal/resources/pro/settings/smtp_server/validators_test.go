// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package smtp_server

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// --- tftypes object shapes --------------------------------------------------
//
// The validator reads authentication_type plus the four blocks as top-level
// scalars/objects, checking only IsNull/IsUnknown on each block, so a single
// dummy inner attribute per block suffices.

var (
	connInnerType  = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"host": tftypes.String}}
	basicInnerType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"username": tftypes.String}}
	graphInnerType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"client_id": tftypes.String}}
	googInnerType  = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"client_id": tftypes.String}}

	smtpRootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"authentication_type":     tftypes.String,
		"connection_settings":     connInnerType,
		"basic_auth_credentials":  basicInnerType,
		"graph_api_credentials":   graphInnerType,
		"google_mail_credentials": googInnerType,
	}}
)

func minimalSmtpSchema() schema.Schema {
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"authentication_type": schema.StringAttribute{Optional: true},
		"connection_settings": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: map[string]schema.Attribute{"host": schema.StringAttribute{Optional: true}},
		},
		"basic_auth_credentials": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: map[string]schema.Attribute{"username": schema.StringAttribute{Optional: true}},
		},
		"graph_api_credentials": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: map[string]schema.Attribute{"client_id": schema.StringAttribute{Optional: true}},
		},
		"google_mail_credentials": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: map[string]schema.Attribute{"client_id": schema.StringAttribute{Optional: true}},
		},
	}}
}

func strVal(v string) tftypes.Value {
	if v == "" {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, v)
}

func strUnknown() tftypes.Value { return tftypes.NewValue(tftypes.String, tftypes.UnknownValue) }

// block returns a present (non-null) or absent (null) object value for the given
// inner type.
func block(t tftypes.Object, present bool) tftypes.Value {
	if !present {
		return tftypes.NewValue(t, nil)
	}
	vals := map[string]tftypes.Value{}
	for name := range t.AttributeTypes {
		vals[name] = tftypes.NewValue(tftypes.String, "x")
	}
	return tftypes.NewValue(t, vals)
}

func blockUnknown(t tftypes.Object) tftypes.Value { return tftypes.NewValue(t, tftypes.UnknownValue) }

// smtpConfig assembles a minimal tfsdk.Config for the validator.
func smtpConfig(authType, conn, basic, graph, goog tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: minimalSmtpSchema(),
		Raw: tftypes.NewValue(smtpRootObjType, map[string]tftypes.Value{
			"authentication_type":     authType,
			"connection_settings":     conn,
			"basic_auth_credentials":  basic,
			"graph_api_credentials":   graph,
			"google_mail_credentials": goog,
		}),
	}
}

func runValidator(cfg tfsdk.Config) map[string]bool {
	var resp resource.ValidateConfigResponse
	authBlockConfigValidator{}.ValidateResource(
		context.Background(),
		resource.ValidateConfigRequest{Config: cfg},
		&resp,
	)
	paths := make(map[string]bool)
	for _, d := range resp.Diagnostics {
		if dwp, ok := d.(diag.DiagnosticWithPath); ok {
			paths[dwp.Path().String()] = true
		}
	}
	return paths
}

func errCount(cfg tfsdk.Config) int {
	var resp resource.ValidateConfigResponse
	authBlockConfigValidator{}.ValidateResource(
		context.Background(),
		resource.ValidateConfigRequest{Config: cfg},
		&resp,
	)
	n := 0
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError {
			n++
		}
	}
	return n
}

// --- NONE -------------------------------------------------------------------

func TestValidator_None_Valid(t *testing.T) {
	cfg := smtpConfig(strVal(authNone), block(connInnerType, true), block(basicInnerType, false), block(graphInnerType, false), block(googInnerType, false))
	if n := errCount(cfg); n != 0 {
		t.Errorf("NONE + connection only must pass; got %d errors", n)
	}
}

func TestValidator_None_MissingConnection(t *testing.T) {
	cfg := smtpConfig(strVal(authNone), block(connInnerType, false), block(basicInnerType, false), block(graphInnerType, false), block(googInnerType, false))
	if !runValidator(cfg)["connection_settings"] {
		t.Error("NONE without connection_settings must error on connection_settings")
	}
}

func TestValidator_None_ForbidsCredentialBlock(t *testing.T) {
	cfg := smtpConfig(strVal(authNone), block(connInnerType, true), block(basicInnerType, true), block(graphInnerType, false), block(googInnerType, false))
	if !runValidator(cfg)["basic_auth_credentials"] {
		t.Error("NONE with basic_auth_credentials must error on basic_auth_credentials")
	}
}

// --- BASIC ------------------------------------------------------------------

func TestValidator_Basic_Valid(t *testing.T) {
	cfg := smtpConfig(strVal(authBasic), block(connInnerType, true), block(basicInnerType, true), block(graphInnerType, false), block(googInnerType, false))
	if n := errCount(cfg); n != 0 {
		t.Errorf("BASIC + connection + basic must pass; got %d errors", n)
	}
}

func TestValidator_Basic_MissingBasic(t *testing.T) {
	cfg := smtpConfig(strVal(authBasic), block(connInnerType, true), block(basicInnerType, false), block(graphInnerType, false), block(googInnerType, false))
	if !runValidator(cfg)["basic_auth_credentials"] {
		t.Error("BASIC without basic_auth_credentials must error")
	}
}

func TestValidator_Basic_ForbidsGraph(t *testing.T) {
	cfg := smtpConfig(strVal(authBasic), block(connInnerType, true), block(basicInnerType, true), block(graphInnerType, true), block(googInnerType, false))
	if !runValidator(cfg)["graph_api_credentials"] {
		t.Error("BASIC with graph_api_credentials must error")
	}
}

// --- GRAPH_API --------------------------------------------------------------

func TestValidator_Graph_Valid(t *testing.T) {
	cfg := smtpConfig(strVal(authGraphAPI), block(connInnerType, false), block(basicInnerType, false), block(graphInnerType, true), block(googInnerType, false))
	if n := errCount(cfg); n != 0 {
		t.Errorf("GRAPH_API + graph only must pass; got %d errors", n)
	}
}

func TestValidator_Graph_ForbidsConnection(t *testing.T) {
	cfg := smtpConfig(strVal(authGraphAPI), block(connInnerType, true), block(basicInnerType, false), block(graphInnerType, true), block(googInnerType, false))
	if !runValidator(cfg)["connection_settings"] {
		t.Error("GRAPH_API with connection_settings must error (connection forbidden)")
	}
}

func TestValidator_Graph_MissingGraph(t *testing.T) {
	cfg := smtpConfig(strVal(authGraphAPI), block(connInnerType, false), block(basicInnerType, false), block(graphInnerType, false), block(googInnerType, false))
	if !runValidator(cfg)["graph_api_credentials"] {
		t.Error("GRAPH_API without graph_api_credentials must error")
	}
}

// --- GOOGLE_MAIL ------------------------------------------------------------

func TestValidator_Google_Valid(t *testing.T) {
	cfg := smtpConfig(strVal(authGoogleMail), block(connInnerType, false), block(basicInnerType, false), block(graphInnerType, false), block(googInnerType, true))
	if n := errCount(cfg); n != 0 {
		t.Errorf("GOOGLE_MAIL + google only must pass; got %d errors", n)
	}
}

func TestValidator_Google_ForbidsConnection(t *testing.T) {
	cfg := smtpConfig(strVal(authGoogleMail), block(connInnerType, true), block(basicInnerType, false), block(graphInnerType, false), block(googInnerType, true))
	if !runValidator(cfg)["connection_settings"] {
		t.Error("GOOGLE_MAIL with connection_settings must error")
	}
}

// --- defer-on-unknown -------------------------------------------------------

func TestValidator_UnknownAuthType_Defers(t *testing.T) {
	cfg := smtpConfig(strUnknown(), block(connInnerType, false), block(basicInnerType, false), block(graphInnerType, false), block(googInnerType, false))
	if n := errCount(cfg); n != 0 {
		t.Errorf("unknown authentication_type must defer (no errors); got %d", n)
	}
}

func TestValidator_UnknownRequiredBlock_Defers(t *testing.T) {
	// BASIC with an unknown basic block must defer the required-when check.
	cfg := smtpConfig(strVal(authBasic), block(connInnerType, true), blockUnknown(basicInnerType), block(graphInnerType, false), block(googInnerType, false))
	if runValidator(cfg)["basic_auth_credentials"] {
		t.Error("unknown basic_auth_credentials must defer, not error")
	}
}
