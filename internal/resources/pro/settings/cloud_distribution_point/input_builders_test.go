// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_distribution_point

import (
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildInput_JamfCloud(t *testing.T) {
	plan := CloudDistributionPointResourceModel{
		CdnType: types.StringValue("JAMF_CLOUD"),
		Master:  types.BoolValue(false),
		// Optional+Computed left unknown (typical for an unset JCDS config).
		Username:                types.StringNull(),
		Directory:               types.StringNull(),
		UploadURL:               types.StringNull(),
		DownloadURL:             types.StringNull(),
		RequireSignedURLs:       types.BoolNull(),
		KeyPairID:               types.StringNull(),
		ExpirationSeconds:       types.Int64Null(),
		SecondaryAuthRequired:   types.BoolNull(),
		SecondaryAuthTimeToLive: types.Int64Null(),
	}
	cfg := CloudDistributionPointResourceModel{
		Password:   types.StringNull(),
		PrivateKey: types.StringNull(),
	}

	in, err := buildCloudDistributionPointInput(plan, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if in.CdnType != "JAMF_CLOUD" {
		t.Errorf("CdnType = %q, want JAMF_CLOUD", in.CdnType)
	}
	// cdnType is always emitted (API-mandatory on every write).
	if in.Username != "" {
		t.Errorf("Username = %q, want empty string for unset JCDS", in.Username)
	}
	if in.Password != "" {
		t.Errorf("Password = %q, want empty string", in.Password)
	}
	if in.PrivateKey != nil {
		t.Errorf("PrivateKey = %v, want nil (omitted)", in.PrivateKey)
	}
	if in.Master == nil || *in.Master {
		t.Errorf("Master = %v, want pointer to false", in.Master)
	}
	// Unset optional pointers must be nil so the field is omitted on the wire.
	if in.Directory != nil || in.ExpirationSeconds != nil || in.RequireSignedUrls != nil {
		t.Errorf("unset optionals must be nil; got directory=%v expiration=%v signed=%v", in.Directory, in.ExpirationSeconds, in.RequireSignedUrls)
	}
}

func TestBuildInput_AmazonS3SignedURLs(t *testing.T) {
	keyB64 := base64.StdEncoding.EncodeToString([]byte("PEM-PRIVATE-KEY"))
	plan := CloudDistributionPointResourceModel{
		CdnType:           types.StringValue("AMAZON_S3"),
		Username:          types.StringValue("AKIAEXAMPLE"),
		RequireSignedURLs: types.BoolValue(true),
		KeyPairID:         types.StringValue("K1R8C5EXAMPLE"),
		ExpirationSeconds: types.Int64Value(3600),
	}
	cfg := CloudDistributionPointResourceModel{
		Password:   types.StringValue("secretAccessKey"),
		PrivateKey: types.StringValue(keyB64),
	}

	in, err := buildCloudDistributionPointInput(plan, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in.Username != "AKIAEXAMPLE" || in.Password != "secretAccessKey" {
		t.Errorf("creds not threaded: username=%q password=%q", in.Username, in.Password)
	}
	if in.RequireSignedUrls == nil || !*in.RequireSignedUrls {
		t.Errorf("RequireSignedUrls = %v, want pointer to true", in.RequireSignedUrls)
	}
	if in.PrivateKey == nil || string(*in.PrivateKey) != "PEM-PRIVATE-KEY" {
		t.Errorf("PrivateKey not decoded from base64: %v", in.PrivateKey)
	}
	if in.ExpirationSeconds == nil || *in.ExpirationSeconds != 3600 {
		t.Errorf("ExpirationSeconds = %v, want 3600", in.ExpirationSeconds)
	}
}

func TestBuildInput_InvalidPrivateKeyBase64(t *testing.T) {
	plan := CloudDistributionPointResourceModel{CdnType: types.StringValue("AMAZON_S3")}
	cfg := CloudDistributionPointResourceModel{
		Password:   types.StringValue("x"),
		PrivateKey: types.StringValue("!!!not base64!!!"),
	}
	if _, err := buildCloudDistributionPointInput(plan, cfg); err == nil {
		t.Fatal("expected error for invalid base64 private_key, got nil")
	}
}
