// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package self_service_branding_image_test

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/files"
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

// pngBytes returns the bytes of a generated fixture, for asserting a hash the
// provider should have arrived at independently.
func pngBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // test fixture written by this test
	if err != nil {
		t.Fatalf("reading PNG fixture %q: %v", path, err)
	}
	return b
}

func imageConfig(path string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_self_service_branding_image" "test" {
  image_file_source = %q
}
`, path)
}

// unstableImageServer answers every request for one path with a different PNG,
// the way a CDN can answer a fixed URL. The returned counter records how many
// times the provider fetched it, so a test can prove the provider read the
// source once rather than once per plan.
//
// The provider runs in this process, so a loopback server is reachable from it.
func unstableImageServer(t *testing.T) (string, *atomic.Int64) {
	t.Helper()
	dir := t.TempDir()
	images := [][]byte{
		pngBytes(t, writePNG(t, dir, "a.png", 180, 180, 10)),
		pngBytes(t, writePNG(t, dir, "b.png", 180, 180, 200)),
	}

	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(images[int(n-1)%len(images)])
	}))
	t.Cleanup(server.Close)

	return server.URL + "/branding.png", &hits
}

// TestAccResource_ProSelfServiceBrandingImage covers create (upload + id
// derivation), a content-change replacement, and import.
//
// Every create and replacement plan must leave source_hash unknown: it is
// resolved in Create from the bytes uploaded, so that a source whose two reads
// differ cannot plan one value and apply another (issue #373).
//
// The import step keeps ImportStateVerify=false, but for a narrower reason than
// it used to claim. image_file_source is not server-derivable and url has no
// metadata GET behind it, so neither round-trips. source_hash does: this store
// returns an image byte for byte as it was sent, wire-verified 2026-09-04, so
// the imported hash is asserted against the local file's own hash rather than
// only matched against the format.
func TestAccResource_ProSelfServiceBrandingImage(t *testing.T) {
	dir := t.TempDir()
	icon := writePNG(t, dir, "icon.png", 180, 180, 128)
	banner := writePNG(t, dir, "banner.png", 1500, 235, 64)

	iconHash := files.ComputeContentSHA256(pngBytes(t, icon))
	bannerHash := files.ComputeContentSHA256(pngBytes(t, banner))

	idCompare := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: imageConfig(icon),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue(imageResourceAddress, tfjsonpath.New("source_hash")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(imageResourceAddress, "source_hash", iconHash),
					resource.TestMatchResourceAttr(imageResourceAddress, "url", regexp.MustCompile(`/branding-images/download/\d+$`)),
					resource.TestCheckResourceAttrWith(imageResourceAddress, "id", func(v string) error {
						if _, err := strconv.Atoi(v); err != nil {
							return fmt.Errorf("id %q is not numeric", v)
						}
						return nil
					}),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(imageResourceAddress, tfjsonpath.New("id")),
				},
			},
			{
				// Switching to a different image replaces the resource, because
				// the id is derived from the upload URL and there is no update
				// endpoint. The new id proves the replacement happened.
				Config: imageConfig(banner),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(imageResourceAddress, plancheck.ResourceActionDestroyBeforeCreate),
						plancheck.ExpectUnknownValue(imageResourceAddress, tfjsonpath.New("source_hash")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(imageResourceAddress, "source_hash", bannerHash),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(imageResourceAddress, tfjsonpath.New("id")),
				},
			},
			{
				ResourceName: imageResourceAddress,
				ImportState:  true,
				// image_file_source is null on import (not server-derivable) and
				// url has no metadata GET to recover it, so neither round-trips
				// and a full verify would assert nothing useful.
				ImportStateVerify: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					// The imported hash comes from the downloaded bytes, and this
					// store returns them as they were sent, so it must equal the
					// hash of the file that was uploaded.
					resource.TestCheckResourceAttr(imageResourceAddress, "source_hash", bannerHash),
				),
			},
		},
	})
}

// TestAccResource_ProSelfServiceBrandingImage_UnstableURL is the acceptance
// cover for issue #373 on this resource, reproduced against a live tenant on
// 2026-09-04.
//
// The condition is a URL whose bytes differ between two requests, which no
// public URL can be relied on to do on demand, so the test serves it. The apply
// has to succeed on the first run, the re-apply has to plan nothing, and the
// provider has to have read the URL exactly once across both: before the fix it
// read it on every plan, so the count is the assertion that would still catch a
// plan-time fetch reintroduced somewhere the hash comparison no longer visits.
func TestAccResource_ProSelfServiceBrandingImage_UnstableURL(t *testing.T) {
	url, hits := unstableImageServer(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: imageConfig(url),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue(imageResourceAddress, tfjsonpath.New("source_hash")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(imageResourceAddress, "id"),
					resource.TestMatchResourceAttr(imageResourceAddress, "source_hash", imageSourceHashRegex),
				),
			},
			{
				Config: imageConfig(url),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})

	if got := hits.Load(); got != 1 {
		t.Fatalf("the provider fetched the image URL %d times, want 1: only the upload in Create may read it, or an unstable source plans one hash and applies another", got)
	}
}
