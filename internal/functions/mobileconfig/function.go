// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobileconfig

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

var _ function.Function = &mobileconfigFunction{}

type mobileconfigFunction struct{}

// NewFunction returns a new instance of the mobileconfig function for
// registration in the provider's Functions() method.
func NewFunction() function.Function {
	return &mobileconfigFunction{}
}

func (f *mobileconfigFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mobileconfig"
}

func (f *mobileconfigFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Builds a complete macOS configuration profile (.mobileconfig) from payload objects.",
		MarkdownDescription: "Assembles a complete macOS configuration profile from one or more payload objects, for use as a configuration profile's `payloads`. Pass an object with a required `identifier` (seeds the deterministic profile and payload UUIDs; distinct profiles must use distinct identifiers to avoid identity collisions) and a `payloads` list; each payload is an object using Apple's real payload keys (for example `PayloadType = \"com.apple.dock\"`, `tilesize = 48`). Optional top-level keys: `display_name`, `organization`, `description`, `scope` (default `System`), and `removal_disallowed` (default `true`). Values keep their natural types: whole numbers become integers, fractional numbers reals, booleans and strings map directly, lists become arrays, and nested objects become dictionaries. Whole numbers above 2^53 lose integer precision (the provider carries numbers as floating point); this does not affect normal configuration-profile values such as sizes, ports, and timeouts.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:                "profile",
				MarkdownDescription: "An object with a required `identifier`, a `payloads` list, and optional top-level metadata (`display_name`, `organization`, `description`, `scope`, `removal_disallowed`).",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *mobileconfigFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var profileArg types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &profileArg))
	if resp.Error != nil {
		return
	}

	raw, err := helpers.TerraformDynamicToJSON(profileArg)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(0, err.Error()))
		return
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(0, "profile must be an object with a payloads list"))
		return
	}

	profile, err := profileFromObject(obj)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(0, err.Error()))
		return
	}

	out, err := Assemble(profile)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, string(out)))
}

// profileFromObject maps a decoded HCL object onto a Profile. The top-level
// metadata keys are snake_case (the provider convention); the payload objects
// themselves use Apple's real payload keys and are passed through verbatim.
func profileFromObject(obj map[string]any) (Profile, error) {
	rawPayloads, ok := obj["payloads"].([]any)
	if !ok {
		return Profile{}, fmt.Errorf("profile must have a payloads list")
	}
	payloads := make([]map[string]any, 0, len(rawPayloads))
	for i, rp := range rawPayloads {
		pm, ok := rp.(map[string]any)
		if !ok {
			return Profile{}, fmt.Errorf("payloads[%d] must be an object", i)
		}
		payloads = append(payloads, pm)
	}

	return Profile{
		DisplayName:       getString(obj, "display_name"),
		Identifier:        getString(obj, "identifier"),
		Organization:      getString(obj, "organization"),
		Description:       getString(obj, "description"),
		Scope:             getString(obj, "scope"),
		RemovalDisallowed: getBoolPtr(obj, "removal_disallowed"),
		Payloads:          payloads,
	}, nil
}

func getString(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func getBoolPtr(m map[string]any, k string) *bool {
	if v, ok := m[k].(bool); ok {
		return &v
	}
	return nil
}
