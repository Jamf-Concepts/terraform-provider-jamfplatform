// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestDeviceGroupSchemaV0_HasCriteriaBlock(t *testing.T) {
	s := deviceGroupSchemaV0()
	if s == nil {
		t.Fatal("v0 schema is nil")
	}

	if _, ok := s.Blocks["criteria"]; !ok {
		t.Error("v0 schema should have 'criteria' block")
	}

	if _, ok := s.Attributes["criteria"]; ok {
		t.Error("v0 schema should NOT have 'criteria' as an attribute")
	}
}

func TestDeviceGroupSchemaV0_RequiredFields(t *testing.T) {
	s := deviceGroupSchemaV0()

	requiredAttrs := []string{"name", "device_type", "group_type"}
	for _, name := range requiredAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if sa, ok := attr.(schema.StringAttribute); ok && !sa.Required {
			t.Errorf("attribute %q should be required", name)
		}
	}
}

func TestDeviceGroupSchemaV0_CriteriaBlockStructure(t *testing.T) {
	s := deviceGroupSchemaV0()

	block, ok := s.Blocks["criteria"]
	if !ok {
		t.Fatal("missing criteria block")
	}

	listBlock, ok := block.(schema.ListNestedBlock)
	if !ok {
		t.Fatal("criteria should be a ListNestedBlock")
	}

	expectedAttrs := []string{"order", "criteria", "operator", "value", "and_or", "has_opening_parenthesis", "has_closing_parenthesis"}
	for _, name := range expectedAttrs {
		if _, ok := listBlock.NestedObject.Attributes[name]; !ok {
			t.Errorf("criteria block missing attribute %q", name)
		}
	}
}
