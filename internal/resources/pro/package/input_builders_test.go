// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildPackageInput_PureMetadata(t *testing.T) {
	plan := PackageResourceModel{
		DisplayName: types.StringValue("MyApp"),
		FileName:    types.StringValue("MyApp.pkg"),
	}
	got := buildPackageInput(plan)

	if got.PackageName != "MyApp" {
		t.Errorf("PackageName = %q, want %q", got.PackageName, "MyApp")
	}
	if got.FileName != "MyApp.pkg" {
		t.Errorf("FileName = %q, want %q", got.FileName, "MyApp.pkg")
	}
	if got.CategoryID != CategoryIDDefault {
		t.Errorf("CategoryID = %q, want %q (None sentinel)", got.CategoryID, CategoryIDDefault)
	}
	if got.Priority != PriorityDefault {
		t.Errorf("Priority = %d, want %d", got.Priority, PriorityDefault)
	}
	if got.FillUserTemplate {
		t.Errorf("FillUserTemplate must default to false, got true")
	}

	// All Optional+Computed strings must emit nil pointers — server returns
	// "" when null.
	for _, c := range []struct {
		name string
		got  *string
	}{
		{"Info", got.Info},
		{"Notes", got.Notes},
		{"OsRequirements", got.OsRequirements},
		{"HashType", got.HashType},
		{"HashValue", got.HashValue},
		{"Sha3512", got.Sha3512},
		{"Sha256", got.Sha256},
		{"Md5", got.Md5},
		{"Size", got.Size}, // never emitted, must stay nil
	} {
		if c.got != nil {
			t.Errorf("%s must be nil for pure-metadata mode, got %v", c.name, *c.got)
		}
	}
}

func TestBuildPackageInput_JCDSMode_OmitsHashes(t *testing.T) {
	// JCDS mode: PackageFileSource is set. ConflictsWith already prevents
	// the user from supplying hashes at plan time, so the model values are
	// null/unconfigured for those attrs. The input builder must NOT emit
	// hashes — the server populates them post-upload.
	plan := PackageResourceModel{
		DisplayName:       types.StringValue("MyApp"),
		FileName:          types.StringValue("MyApp.pkg"),
		PackageFileSource: types.StringValue("/path/to/MyApp.pkg"),
	}
	got := buildPackageInput(plan)

	if got.HashType != nil {
		t.Errorf("JCDS mode: HashType must be nil, got %q", *got.HashType)
	}
	if got.HashValue != nil {
		t.Errorf("JCDS mode: HashValue must be nil, got %q", *got.HashValue)
	}
	if got.Md5 != nil {
		t.Errorf("JCDS mode: Md5 must be nil, got %q", *got.Md5)
	}
}

func TestBuildPackageInput_FSDPWithHashes(t *testing.T) {
	plan := PackageResourceModel{
		DisplayName: types.StringValue("MyApp"),
		FileName:    types.StringValue("MyApp.pkg"),
		HashType:    types.StringValue("MD5"),
		HashValue:   types.StringValue("d41d8cd98f00b204e9800998ecf8427e"),
		Md5:         types.StringValue("d41d8cd98f00b204e9800998ecf8427e"),
		Sha256:      types.StringValue("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
	}
	got := buildPackageInput(plan)

	if got.HashType == nil || *got.HashType != "MD5" {
		t.Errorf("HashType = %v, want %q", got.HashType, "MD5")
	}
	if got.HashValue == nil || *got.HashValue != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Errorf("HashValue not emitted correctly: %v", got.HashValue)
	}
	if got.Md5 == nil {
		t.Errorf("Md5 must be emitted in FSDP-with-hashes mode")
	}
	if got.Sha256 == nil {
		t.Errorf("Sha256 must be emitted in FSDP-with-hashes mode")
	}
	// Sha3512 was not supplied — must remain nil.
	if got.Sha3512 != nil {
		t.Errorf("Sha3512 must stay nil when not supplied: %v", *got.Sha3512)
	}
}

func TestBuildPackageInput_RespectsUserPriority(t *testing.T) {
	plan := PackageResourceModel{
		DisplayName: types.StringValue("X"),
		FileName:    types.StringValue("X.pkg"),
		Priority:    types.Int64Value(5),
	}
	got := buildPackageInput(plan)
	if got.Priority != 5 {
		t.Errorf("Priority = %d, want 5", got.Priority)
	}
}

func TestBuildPackageInput_RespectsUserCategory(t *testing.T) {
	plan := PackageResourceModel{
		DisplayName: types.StringValue("X"),
		FileName:    types.StringValue("X.pkg"),
		CategoryID:  types.StringValue("42"),
	}
	got := buildPackageInput(plan)
	if got.CategoryID != "42" {
		t.Errorf("CategoryID = %q, want %q", got.CategoryID, "42")
	}
}
