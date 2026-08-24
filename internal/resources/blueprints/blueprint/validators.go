// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/appleprofiles"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// legacyPayloadSchemaValidator checks each legacy payload against Apple's declared payload keys
// during plan, so the three ways Jamf refuses or rewrites a payload are caught before an apply:
// a value of the wrong type or a missing required key fails the write outright, and a miscased key
// is stored under Apple's spelling, leaving configuration and state permanently apart.
//
// A finding that rests on the embedded schemas being current is a warning, not an error — a key
// Apple added after the snapshot is indistinguishable from one that never existed, and the provider
// must not block a configuration that works. See internal/common/appleprofiles.
type legacyPayloadSchemaValidator struct{}

// blockLegacyPayloadSchemaValidator validates a component block's typed legacy payload list, whose
// settings arrive as JSON object strings.
func blockLegacyPayloadSchemaValidator() validator.List {
	return legacyPayloadSchemaValidator{}
}

// flatLegacyPayloadSchemaValidator validates the deprecated top-level dynamic legacy payload value,
// whose settings arrive as objects rather than strings.
func flatLegacyPayloadSchemaValidator() validator.Dynamic {
	return legacyPayloadSchemaValidator{}
}

func (v legacyPayloadSchemaValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v legacyPayloadSchemaValidator) MarkdownDescription(context.Context) string {
	_, release := appleprofiles.Provenance()
	return fmt.Sprintf("each payload's settings must match Apple's declared keys for its payload type (schemas from %s)", release)
}

// ValidateList checks a block's legacy payload list.
func (v legacyPayloadSchemaValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if !helpers.IsConfiguredValue(req.ConfigValue) {
		return
	}

	var payloads []BlockLegacyPayloadModel
	diags := req.ConfigValue.ElementsAs(ctx, &payloads, false)
	if diags.HasError() {
		// An element still unknown at plan time cannot be validated; the wire will decide.
		return
	}

	for i, payload := range payloads {
		entry := req.Path.AtListIndex(i)
		if !helpers.IsConfiguredValue(payload.PayloadType) {
			continue
		}

		var settings map[string]any
		if helpers.IsConfiguredValue(payload.Settings) && payload.Settings.ValueString() != "" {
			if err := json.Unmarshal([]byte(payload.Settings.ValueString()), &settings); err != nil {
				// collectBlockLegacyPayloads reports unparseable settings; nothing to add here.
				continue
			}
		}

		appendPayloadProblems(&resp.Diagnostics, payload.PayloadType.ValueString(), settings, entry, entry.AtName("payload_type"), entry.AtName("settings"))
	}
}

// ValidateDynamic checks the deprecated top-level dynamic legacy payload value.
func (v legacyPayloadSchemaValidator) ValidateDynamic(_ context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse) {
	if !helpers.IsConfiguredValue(req.ConfigValue) {
		return
	}

	raw, err := helpers.TerraformDynamicToJSON(req.ConfigValue)
	if err != nil {
		return
	}
	items, ok := raw.([]any)
	if !ok {
		// collectLegacyPayloads reports a non-list value; nothing to add here.
		return
	}

	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		payloadType, _ := obj["payload_type"].(string)
		if payloadType == "" {
			continue
		}
		settings, _ := obj["settings"].(map[string]any)

		// A dynamic value carries no traversable schema path for its elements, so every finding is
		// reported against the attribute itself and located by the path in its detail.
		appendPayloadProblems(&resp.Diagnostics, payloadType, settings, req.Path, req.Path, req.Path)
	}
}

// appendPayloadProblems runs one payload through the schema table and turns each problem into a
// diagnostic. A problem that depends on the snapshot being current becomes a warning; the rest
// become errors, because Jamf was observed to refuse or rewrite those writes.
func appendPayloadProblems(diags *diag.Diagnostics, payloadType string, settings map[string]any, payloadPath, typePath, settingsPath path.Path) {
	problems := appleprofiles.Validate(payloadType, settings)
	if len(problems) == 0 {
		return
	}

	_, release := appleprofiles.Provenance()

	for _, problem := range problems {
		target := settingsPath
		summary := "Legacy payload setting does not match Apple's schema"
		switch problem.Kind {
		case appleprofiles.UnknownPayloadType, appleprofiles.MiscasedPayloadType:
			target = typePath
			summary = "Unrecognised legacy payload type"
		case appleprofiles.MissingRequiredKey:
			target = payloadPath
			summary = "Legacy payload is missing a required setting"
		}

		detail := problem.Detail
		if problem.Path != "" {
			detail = problem.Path + ": " + detail
		}
		detail += fmt.Sprintf(" (checked against Apple's schemas as of %s; run `make apple-profiles` if Apple has published newer ones)", release)

		if problem.Advisory() {
			diags.AddAttributeWarning(target, summary, detail)
			continue
		}
		diags.AddAttributeError(target, summary, detail)
	}
}
