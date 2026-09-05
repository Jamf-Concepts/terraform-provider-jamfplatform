// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package smtp_server

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// senderModel builds a sender_settings model from raw values, where a nil string
// pointer means the attribute is unknown (the shape a first apply produces for
// display_name) and an empty string means it is set to "" (the shape a tenant
// that has never set up mail reads back).
func senderModel(email, display *string) *smtpSenderSettingsModel {
	value := func(v *string) types.String {
		if v == nil {
			return types.StringUnknown()
		}
		return types.StringValue(*v)
	}
	return &smtpSenderSettingsModel{EmailAddress: value(email), DisplayName: value(display)}
}

func TestValidateSenderSettingsWhenEnabled(t *testing.T) {
	empty := ""
	address := "notifications@example.com"
	name := "Jamf Pro"

	tests := []struct {
		name         string
		enabled      types.Bool
		sender       *smtpSenderSettingsModel
		wantEmailErr bool
		wantNameErr  bool
	}{
		{
			name:    "disabled tolerates both empty",
			enabled: types.BoolValue(false),
			sender:  senderModel(&empty, &empty),
		},
		{
			name:    "unknown enabled defers",
			enabled: types.BoolUnknown(),
			sender:  senderModel(&empty, &empty),
		},
		{
			name:    "null enabled defers",
			enabled: types.BoolNull(),
			sender:  senderModel(&empty, &empty),
		},
		{
			name:    "nil sender block defers",
			enabled: types.BoolValue(true),
			sender:  nil,
		},
		{
			name:    "enabled with both set passes",
			enabled: types.BoolValue(true),
			sender:  senderModel(&address, &name),
		},
		{
			name:         "enabled with empty address is refused",
			enabled:      types.BoolValue(true),
			sender:       senderModel(&empty, &name),
			wantEmailErr: true,
		},
		{
			name:        "enabled with empty display name is refused",
			enabled:     types.BoolValue(true),
			sender:      senderModel(&address, &empty),
			wantNameErr: true,
		},
		{
			name:         "enabled with both empty names both attributes",
			enabled:      types.BoolValue(true),
			sender:       senderModel(&empty, &empty),
			wantEmailErr: true,
			wantNameErr:  true,
		},
		{
			name:    "enabled with unknown display name defers to the server",
			enabled: types.BoolValue(true),
			sender:  senderModel(&address, nil),
		},
		{
			name:    "enabled with unknown address defers to the server",
			enabled: types.BoolValue(true),
			sender:  senderModel(nil, &name),
		},
	}

	emailPath := path.Root("sender_settings").AtName("email_address")
	namePath := path.Root("sender_settings").AtName("display_name")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateSenderSettingsWhenEnabled(tc.enabled, tc.sender)

			var gotEmail, gotName bool
			for _, d := range diags.Errors() {
				withPath, ok := d.(diag.DiagnosticWithPath)
				if !ok {
					t.Fatalf("diagnostic %q carries no attribute path", d.Summary())
				}
				switch {
				case withPath.Path().Equal(emailPath):
					gotEmail = true
				case withPath.Path().Equal(namePath):
					gotName = true
				default:
					t.Fatalf("unexpected diagnostic at %s: %s", withPath.Path(), d.Summary())
				}
			}

			if gotEmail != tc.wantEmailErr {
				t.Errorf("email_address error = %v, want %v", gotEmail, tc.wantEmailErr)
			}
			if gotName != tc.wantNameErr {
				t.Errorf("display_name error = %v, want %v", gotName, tc.wantNameErr)
			}
			if len(diags.Warnings()) != 0 {
				t.Errorf("expected no warnings, got %d", len(diags.Warnings()))
			}
		})
	}
}

func TestSmtpServerWriteErrorDiagnostic(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantSummary string
	}{
		{
			name:        "both fields named",
			err:         errors.New(`400: [INVALID_DISPLAY_NAME] senderSettings.displayName: Invalid display name; [INVALID_EMAIL] senderSettings.emailAddress: Invalid email address`),
			wantSummary: "Sender email address and display name required to enable the SMTP server",
		},
		{
			name:        "display name only",
			err:         errors.New(`400: [INVALID_DISPLAY_NAME] senderSettings.displayName: Invalid display name; please ensure this field is in not empty`),
			wantSummary: "Sender display name required to enable the SMTP server",
		},
		{
			name:        "email only",
			err:         errors.New(`400: [INVALID_EMAIL] senderSettings.emailAddress: Invalid email address; please ensure this field is in a proper email address format`),
			wantSummary: "Sender email address rejected by Jamf Pro",
		},
		{
			name:        "unrelated failure passes through",
			err:         errors.New(`400: [INVALID_HOSTNAME] connectionSettings.host: Invalid hostname`),
			wantSummary: "Error updating Jamf Pro SMTP Server settings",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			summary, detail := smtpServerWriteErrorDiagnostic("Error updating Jamf Pro SMTP Server settings", tc.err)
			if summary != tc.wantSummary {
				t.Errorf("summary = %q, want %q", summary, tc.wantSummary)
			}
			if !strings.Contains(detail, tc.err.Error()) {
				t.Errorf("detail must quote what Jamf Pro reported, got %q", detail)
			}
		})
	}
}
