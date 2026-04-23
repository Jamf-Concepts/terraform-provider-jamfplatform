// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CustomDeclarationsComponent represents a strongly-typed custom DDM declarations component.
type CustomDeclarationsComponent struct {
	Declarations []CustomDeclarationModel `tfsdk:"declaration"`
}

// CustomDeclarationModel represents a single custom DDM declaration.
type CustomDeclarationModel struct {
	ChannelType types.String `tfsdk:"channel"`
	Kind        types.String `tfsdk:"kind"`
	Payload     types.String `tfsdk:"payload"`
	Type        types.String `tfsdk:"type"`
}

// CustomDeclarationsComponentSchema returns the Terraform schema for custom declarations component.
func CustomDeclarationsComponentSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"declaration": schema.SetNestedAttribute{
			MarkdownDescription: "Custom DDM declaration.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"channel": schema.StringAttribute{
						MarkdownDescription: "The channel type for the declaration. Valid values are `SYSTEM`, `USER`.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.OneOf("SYSTEM", "USER")},
					},
					"kind": schema.StringAttribute{
						MarkdownDescription: "The kind of declaration. Valid values are `CONFIGURATION`, `ASSET`.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.OneOf("CONFIGURATION", "ASSET")},
					},
					"payload": schema.StringAttribute{
						MarkdownDescription: "JSON-encoded payload object for the declaration.",
						Required:            true,
					},
					"type": schema.StringAttribute{
						MarkdownDescription: "The declaration type identifier (e.g., `com.apple.configuration.softwareupdate.settings`).",
						Required:            true,
					},
				},
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw JSON configuration.
func (c *CustomDeclarationsComponent) ToRawConfiguration() (json.RawMessage, error) {
	config := make(map[string]any)

	if len(c.Declarations) > 0 {
		declarations := make([]any, 0, len(c.Declarations))

		for idx, declaration := range c.Declarations {
			declarationMap := make(map[string]any)

			if helpers.IsConfiguredValue(declaration.ChannelType) {
				declarationMap["channelType"] = declaration.ChannelType.ValueString()
			}
			if helpers.IsConfiguredValue(declaration.Kind) {
				declarationMap["kind"] = declaration.Kind.ValueString()
			}
			if helpers.IsConfiguredValue(declaration.Payload) {
				var payloadObj any
				if err := json.Unmarshal([]byte(declaration.Payload.ValueString()), &payloadObj); err != nil {
					return nil, err
				}
				declarationMap["payload"] = payloadObj
			}
			declarationMap["payloadKey"] = idx + 1
			if helpers.IsConfiguredValue(declaration.Type) {
				declarationMap["type"] = declaration.Type.ValueString()
			}

			declarations = append(declarations, declarationMap)
		}

		config["declarations"] = declarations
	}

	return json.Marshal(config)
}

// FromRawConfiguration populates the typed component from raw JSON configuration.
func (c *CustomDeclarationsComponent) FromRawConfiguration(raw json.RawMessage) error {
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}

	if declarationsRaw, exists := config["declarations"]; exists {
		if declarationsSlice, ok := declarationsRaw.([]any); ok {
			declarations := make([]CustomDeclarationModel, 0, len(declarationsSlice))

			for _, declarationRaw := range declarationsSlice {
				if declarationMap, ok := declarationRaw.(map[string]any); ok {
					declaration := CustomDeclarationModel{}

					if channelType, exists := declarationMap["channelType"]; exists {
						if channelStr, ok := channelType.(string); ok {
							declaration.ChannelType = types.StringValue(channelStr)
						}
					}
					if kind, exists := declarationMap["kind"]; exists {
						if kindStr, ok := kind.(string); ok {
							declaration.Kind = types.StringValue(kindStr)
						}
					}
					if payload, exists := declarationMap["payload"]; exists {
						payloadJSON, err := json.Marshal(payload)
						if err != nil {
							return err
						}
						declaration.Payload = types.StringValue(string(payloadJSON))
					}
					if declType, exists := declarationMap["type"]; exists {
						if typeStr, ok := declType.(string); ok {
							declaration.Type = types.StringValue(typeStr)
						}
					}

					declarations = append(declarations, declaration)
				}
			}

			c.Declarations = declarations
		}
	}

	return nil
}

// GetIdentifier returns the component identifier for custom declarations.
func (c *CustomDeclarationsComponent) GetIdentifier() string {
	return "com.jamf.ddm.custom-declarations"
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client.
func (c *CustomDeclarationsComponent) ToClientComponent() (*blueprints.Component, error) {
	cfg, err := c.ToRawConfiguration()
	if err != nil {
		return nil, err
	}
	return &blueprints.Component{
		Identifier:    c.GetIdentifier(),
		Configuration: cfg,
	}, nil
}
