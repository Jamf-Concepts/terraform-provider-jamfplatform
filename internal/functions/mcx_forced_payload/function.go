// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package mcx_forced_payload implements the jamfplatform::mcx_forced_payload
// provider-defined function, which encodes a set of application preferences into
// a macOS managed ("forced") preferences payload for use as a configuration
// profile's payload.
package mcx_forced_payload

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Ensure the implementation satisfies the framework interface.
var _ function.Function = &mcxForcedPayloadFunction{}

type mcxForcedPayloadFunction struct{}

// NewFunction returns a new instance of the mcx_forced_payload function for
// registration in the provider's Functions() method.
func NewFunction() function.Function {
	return &mcxForcedPayloadFunction{}
}

func (f *mcxForcedPayloadFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mcx_forced_payload"
}

func (f *mcxForcedPayloadFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Builds a macOS managed (forced) preferences payload for a Custom Settings configuration profile.",
		MarkdownDescription: "Encodes a set of application preferences into a macOS managed (forced) preferences payload. Pass the result to a macOS configuration profile's `payloads` so the settings are delivered to enrolled Macs as non-overridable defaults. Preference values keep their natural types: whole numbers become integers, fractional numbers become reals, booleans and strings map directly, lists become arrays, and nested objects become dictionaries. Whole numbers above 2^53 lose integer precision (the provider carries numbers as floating point); this does not affect normal preference values.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "preference_domain",
				MarkdownDescription: "The application preference domain the settings apply to, e.g. `com.example.app`.",
			},
			function.DynamicParameter{
				Name:                "preferences",
				MarkdownDescription: "An object whose keys are preference names and whose values are the preference values (strings, numbers, booleans, lists, or nested objects).",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *mcxForcedPayloadFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var domain string
	var prefs types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &domain, &prefs))
	if resp.Error != nil {
		return
	}

	raw, err := helpers.TerraformDynamicToJSON(prefs)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, err.Error()))
		return
	}
	prefsMap, ok := raw.(map[string]any)
	if !ok {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "preferences must be an object (map) of preference keys"))
		return
	}

	out, err := renderMCXForcedPayload(domain, prefsMap)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, string(out)))
}
