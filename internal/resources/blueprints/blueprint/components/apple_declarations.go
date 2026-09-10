// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jamf/terraform-provider-jamfplatform/internal/common/appledeclarations"
)

// Wire mapping for this component, kept here rather than in any schema description: the identifier
// is API plumbing and must not reach user-facing text (see STYLE_GUIDE §Attribute names mirror the
// Jamf Pro admin UI). The Terraform attribute is named for what the editor shows.
//
//	Terraform attribute      Jamf Pro editor        wire identifier
//	apple_declarations       "All Declarations"     com.jamf.ddm-strict
//	  declaration[].channel                         declarations[].channelType
//	  declaration[].type                            declarations[].type
//	  declaration[].payload                         declarations[].payload
//	  (derived from type)                           declarations[].kind
//	  (derived from position)                       declarations[].payloadKey
//
// It is the generative-declarations component: Jamf renders a typed form per declaration, generated
// from Apple's own schemas, which is what distinguishes it from the custom-declarations component —
// that one renders an opaque JSON blob instead.
const appleDeclarationsIdentifier = "com.jamf.ddm-strict"

// AppleDeclarationsComponent delivers Apple declarative device management declarations that are
// checked against Apple's published schemas during plan.
//
// The list is ORDERED, and deliberately a list rather than a set: an asset reference inside a
// payload names another declaration by its position in this component through the `$PAYLOAD_n`
// placeholder, so the position is part of the meaning. `payload_key` is derived from the 1-based
// index rather than authored, because it has no other purpose and an authored one could disagree
// with the position it is meant to name.
type AppleDeclarationsComponent struct {
	Declarations []AppleDeclarationModel `tfsdk:"declaration"`
}

// AppleDeclarationModel is one Apple declaration.
//
// There is no `kind` attribute. A declaration's kind is fully determined by its type's
// reverse-domain prefix — `com.apple.configuration.*` is a CONFIGURATION, `com.apple.asset.*` an
// ASSET, and so on — so asking for it only creates a field an author can get wrong. The platform
// accepts any pairing and passes a mismatch through to the device, which makes it a silent failure
// rather than a caught one, so the provider derives the kind instead of trusting it.
type AppleDeclarationModel struct {
	ChannelType types.String `tfsdk:"channel"`
	Payload     types.String `tfsdk:"payload"`
	Type        types.String `tfsdk:"type"`
}

// AppleDeclarationsComponentSchema returns the Terraform schema for the Apple declarations component.
func AppleDeclarationsComponentSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"declaration": schema.ListNestedAttribute{
			MarkdownDescription: "An Apple declaration to deliver. Ordered: a payload may reference another " +
				"declaration in this component with `$PAYLOAD_n`, where `n` is its 1-based position in this list.",
			Optional: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"channel": schema.StringAttribute{
						MarkdownDescription: "The channel the declaration applies to. Valid values are `SYSTEM` (the device channel) and `USER`.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf(blueprints.DeclarationChannelTypeSystem, blueprints.DeclarationChannelTypeUser),
						},
					},
					"type": schema.StringAttribute{
						MarkdownDescription: "The Apple declaration type, for example `com.apple.configuration.passcode.settings`. " +
							"Matched exactly: Jamf Pro delivers nothing for a type spelled differently, including in case.",
						Required: true,
					},
					"payload": schema.StringAttribute{
						MarkdownDescription: "The declaration's payload as a JSON object string, authored with `jsonencode(...)`. " +
							"Keys are Apple's own, spelled as Apple declares them." + AppleDeclarationsBehaviour,
						Required: true,
					},
				},
			},
		},
	}
}

// AppleDeclarationsBehaviour documents what the provider checks and why, appended to the payload
// attribute description. It covers what an author cannot learn from a diagnostic: that the platform
// itself validates none of this, so the check exists only during plan.
const AppleDeclarationsBehaviour = " Jamf Pro stores a payload without validating it and drops any " +
	"key it does not recognise, so a misspelled key never reaches a device. The provider checks each " +
	"payload against Apple's schemas during `plan` and reports an unrecognised or miscased key, a " +
	"wrong value type, a missing required key, a value outside a declared set, and a number outside " +
	"a declared range. The schemas cover Apple's release and current seed branches, so they include " +
	"keys Apple has published but not yet released, and they are embedded in the provider release " +
	"you have installed: a key newer than that release reads as unrecognised until you upgrade the " +
	"provider. To skip the check, use `raw_component`."

// GetIdentifier returns the component identifier.
func (c *AppleDeclarationsComponent) GetIdentifier() string {
	return appleDeclarationsIdentifier
}

// ToRawConfiguration converts the typed component to raw JSON configuration. An empty component
// marshals to `{"declarations":[]}` and never to `{}`, because the platform refuses a configuration
// that is literally `{}` on create but accepts an empty declaration list, which is what an empty
// component means here.
func (c *AppleDeclarationsComponent) ToRawConfiguration() (json.RawMessage, error) {
	declarations := make([]blueprints.CustomDeclaration, 0, len(c.Declarations))
	for idx, declaration := range c.Declarations {
		var payload map[string]any
		if err := json.Unmarshal([]byte(declaration.Payload.ValueString()), &payload); err != nil {
			return nil, err
		}
		declarationType := declaration.Type.ValueString()
		declarations = append(declarations, blueprints.CustomDeclaration{
			ChannelType: declaration.ChannelType.ValueString(),
			Kind:        appledeclarations.KindForType(declarationType),
			Payload:     payload,
			PayloadKey:  idx + 1,
			Type:        declarationType,
		})
	}

	return json.Marshal(blueprints.CustomDeclarationsConfiguration{Declarations: declarations})
}

// FromRawConfiguration populates the typed component from raw JSON configuration.
//
// The stored `kind` and `payloadKey` are deliberately dropped rather than read back: both are
// derived on write, so reading them into state would let a value the platform echoed disagree with
// the position and type that produced it.
func (c *AppleDeclarationsComponent) FromRawConfiguration(raw json.RawMessage) error {
	var config blueprints.CustomDeclarationsConfiguration
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}

	declarations := make([]AppleDeclarationModel, 0, len(config.Declarations))
	for _, declaration := range config.Declarations {
		payloadJSON, err := json.Marshal(declaration.Payload)
		if err != nil {
			return err
		}
		declarations = append(declarations, AppleDeclarationModel{
			ChannelType: types.StringValue(declaration.ChannelType),
			Payload:     types.StringValue(string(payloadJSON)),
			Type:        types.StringValue(declaration.Type),
		})
	}

	c.Declarations = declarations
	return nil
}

// ToClientComponent converts the typed component to the format the Blueprint API client expects.
func (c *AppleDeclarationsComponent) ToClientComponent() (*blueprints.Component, error) {
	cfg, err := c.ToRawConfiguration()
	if err != nil {
		return nil, err
	}
	return &blueprints.Component{
		Identifier:    c.GetIdentifier(),
		Configuration: cfg,
	}, nil
}
