// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSetBoolField_Configured(t *testing.T) {
	result := setBoolField(types.BoolValue(true), false)
	if result["Enabled"] != true {
		t.Errorf("expected Enabled true, got %v", result["Enabled"])
	}
	if result["Included"] != true {
		t.Errorf("expected Included true, got %v", result["Included"])
	}
}

func TestSetBoolField_ConfiguredFalse(t *testing.T) {
	result := setBoolField(types.BoolValue(false), true)
	if result["Enabled"] != false {
		t.Errorf("expected Enabled false, got %v", result["Enabled"])
	}
	if result["Included"] != true {
		t.Errorf("expected Included true for configured value, got %v", result["Included"])
	}
}

func TestSetBoolField_Null(t *testing.T) {
	result := setBoolField(types.BoolNull(), true)
	if result["Enabled"] != true {
		t.Errorf("expected default Enabled true, got %v", result["Enabled"])
	}
	if result["Included"] != false {
		t.Errorf("expected Included false for null, got %v", result["Included"])
	}
}

func TestSetBoolField_Unknown(t *testing.T) {
	result := setBoolField(types.BoolUnknown(), false)
	if result["Enabled"] != false {
		t.Errorf("expected default Enabled false, got %v", result["Enabled"])
	}
	if result["Included"] != false {
		t.Errorf("expected Included false for unknown, got %v", result["Included"])
	}
}

func TestSetStringField_Configured(t *testing.T) {
	result := setStringField(types.StringValue("custom"), "default")
	if result["Value"] != "custom" {
		t.Errorf("expected Value 'custom', got %v", result["Value"])
	}
	if result["Included"] != true {
		t.Errorf("expected Included true, got %v", result["Included"])
	}
}

func TestSetStringField_Null(t *testing.T) {
	result := setStringField(types.StringNull(), "fallback")
	if result["Value"] != "fallback" {
		t.Errorf("expected Value 'fallback', got %v", result["Value"])
	}
	if result["Included"] != false {
		t.Errorf("expected Included false for null, got %v", result["Included"])
	}
}

func TestSetStringField_Unknown(t *testing.T) {
	result := setStringField(types.StringUnknown(), "default")
	if result["Value"] != "default" {
		t.Errorf("expected Value 'default', got %v", result["Value"])
	}
	if result["Included"] != false {
		t.Errorf("expected Included false for unknown, got %v", result["Included"])
	}
}

func TestSetBoolFieldWithKey_CustomKey(t *testing.T) {
	result := setBoolFieldWithKey(types.BoolValue(true), "AddSquareRoot", false)
	if result["AddSquareRoot"] != true {
		t.Errorf("expected AddSquareRoot true, got %v", result["AddSquareRoot"])
	}
	if result["Included"] != true {
		t.Errorf("expected Included true, got %v", result["Included"])
	}
}

func TestSetBoolFieldWithKey_NullCustomKey(t *testing.T) {
	result := setBoolFieldWithKey(types.BoolNull(), "AddSquareRoot", true)
	if result["AddSquareRoot"] != true {
		t.Errorf("expected default AddSquareRoot true, got %v", result["AddSquareRoot"])
	}
	if result["Included"] != false {
		t.Errorf("expected Included false for null, got %v", result["Included"])
	}
}

func TestSetValueField(t *testing.T) {
	result := setValueField("test", true)
	if result["Value"] != "test" {
		t.Errorf("expected Value 'test', got %v", result["Value"])
	}
	if result["Included"] != true {
		t.Errorf("expected Included true, got %v", result["Included"])
	}
}

func TestSetValueField_NotIncluded(t *testing.T) {
	result := setValueField(42, false)
	if result["Value"] != 42 {
		t.Errorf("expected Value 42, got %v", result["Value"])
	}
	if result["Included"] != false {
		t.Errorf("expected Included false, got %v", result["Included"])
	}
}
