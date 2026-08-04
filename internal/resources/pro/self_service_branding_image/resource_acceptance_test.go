// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package self_service_branding_image_test

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const imageResourceAddress = "jamfplatform_pro_self_service_branding_image.test"

var imageSourceHashRegex = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// writePNG generates a deterministic w×h PNG at dir/name and returns its path.
// Generated at runtime so no binary fixture is committed to the repo.
func writePNG(t *testing.T, dir, name string, w, h int, tint uint8) string {
	t.Helper()
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			m.Set(x, y, color.RGBA{uint8(x), uint8(y), tint, 255})
		}
	}
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("creating PNG fixture: %v", err)
	}
	if err := png.Encode(f, m); err != nil {
		_ = f.Close()
		t.Fatalf("encoding PNG fixture: %v", err)
	}
	// Close is checked rather than deferred: a dropped Close can hide a flush
	// failure that leaves the fixture truncated, which then fails somewhere far
	// less obvious than here.
	if err := f.Close(); err != nil {
		t.Fatalf("closing PNG fixture: %v", err)
	}
	return p
}

func imageConfig(path string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_self_service_branding_image" "test" {
  image_file_source = %q
}
`, path)
}

// TestAccResource_ProSelfServiceBrandingImage covers create (upload + id
// derivation), a content-change replacement, and import. The import step uses
// ImportStateVerify=false (like jamfplatform_pro_icon): image_file_source, url,
// and source_hash do not round-trip byte-identically on import, so a full
// verify is not meaningful — see the import step comment.
func TestAccResource_ProSelfServiceBrandingImage(t *testing.T) {
	dir := t.TempDir()
	icon := writePNG(t, dir, "icon.png", 180, 180, 128)
	banner := writePNG(t, dir, "banner.png", 1500, 235, 64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: imageConfig(icon),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(imageResourceAddress, "source_hash", imageSourceHashRegex),
					resource.TestMatchResourceAttr(imageResourceAddress, "url", regexp.MustCompile(`/branding-images/download/\d+$`)),
					resource.TestCheckResourceAttrWith(imageResourceAddress, "id", func(v string) error {
						if _, err := strconv.Atoi(v); err != nil {
							return fmt.Errorf("id %q is not numeric", v)
						}
						return nil
					}),
				),
			},
			{
				// Switching to a different image replaces the resource (new id).
				Config: imageConfig(banner),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(imageResourceAddress, "source_hash", imageSourceHashRegex),
				),
			},
			{
				ResourceName: imageResourceAddress,
				ImportState:  true,
				// ImportStateVerify=false (mirrors jamfplatform_pro_icon): on
				// import image_file_source is null (not server-derivable),
				// source_hash is recomputed from the downloaded bytes (Jamf may
				// re-encode), and url is the upload-time tenant jamfcloud URL
				// with no metadata GET to recover it. None round-trip
				// byte-identically, so a full verify is not meaningful.
				ImportStateVerify: false,
			},
		},
	})
}
