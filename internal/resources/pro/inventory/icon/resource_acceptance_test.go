// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package icon_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

const (
	iconResourceAddress = "jamfplatform_pro_icon.test"

	// Real public URLs hosted by Jamf Concepts. The two icons differ in
	// content so switching between them exercises the URL→URL replacement
	// path. Both expose strong ETags + Last-Modified headers.
	urlJamfCLI = "https://concepts.jamf.com/icons/jamf-cli.png"
	urlGoSDK   = "https://concepts.jamf.com/icons/go-sdk.png"
)

// sourceHashRegex matches the canonical provider source_hash format.
var sourceHashRegex = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// fixturesDir returns the absolute path to the test_fixtures directory next
// to this test file.
func fixturesDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("could not resolve caller path for fixture lookup")
	}
	return filepath.Join(filepath.Dir(file), "test_fixtures")
}

// fixturePath returns the absolute path to a file under test_fixtures.
func fixturePath(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(fixturesDir(t), name)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("fixture %q not present at %q: %v", name, p, err)
	}
	return p
}

// fixtureBytes returns the bytes of a fixture file.
func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(fixturePath(t, name)) //nolint:gosec // test fixture
	if err != nil {
		t.Fatalf("reading fixture %q: %v", name, err)
	}
	return b
}

// downloadToTempFile downloads url into a tempfile and returns the path.
// The tempfile is cleaned up at test end. Used to materialise URL bytes
// locally so subsequent steps can switch icon_file_source to a path with
// matching content.
func downloadToTempFile(t *testing.T, url string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("downloading %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body for %s: %v", url, err)
	}
	f, err := os.CreateTemp(t.TempDir(), "icon-*.png")
	if err != nil {
		t.Fatalf("creating tempfile: %v", err)
	}
	if _, err := f.Write(body); err != nil {
		_ = f.Close()
		t.Fatalf("writing tempfile: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing tempfile: %v", err)
	}
	return f.Name()
}

// localConfig returns a config block with icon_file_source set to a path.
func localConfig(path string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_icon" "test" {
  icon_file_source = %q
}
`, path)
}

// urlConfig returns a config block with icon_file_source set to a URL.
func urlConfig(url string) string {
	return fmt.Sprintf(`
resource "jamfplatform_pro_icon" "test" {
  icon_file_source = %q
}
`, url)
}

// TestAccResource_Icon_LocalFile_BasicCRUD creates an icon from a local
// fixture and asserts the canonical source_hash format. The second step
// re-applies the same config and asserts an empty plan.
func TestAccResource_Icon_LocalFile_BasicCRUD(t *testing.T) {
	testhelpers.AccPreCheck(t)

	pathA := fixturePath(t, "icon_a.png")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: localConfig(pathA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(iconResourceAddress, "id"),
					resource.TestCheckResourceAttrSet(iconResourceAddress, "url"),
					resource.TestMatchResourceAttr(iconResourceAddress, "source_hash", sourceHashRegex),
				),
			},
			{
				Config: localConfig(pathA),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccResource_Icon_LocalFile_ContentChange_TriggersReplace switches
// from icon_a.png (red) to icon_b.png (green). Different bytes → provider
// must destroy-before-create and assign a new ID.
func TestAccResource_Icon_LocalFile_ContentChange_TriggersReplace(t *testing.T) {
	testhelpers.AccPreCheck(t)

	pathA := fixturePath(t, "icon_a.png")
	pathB := fixturePath(t, "icon_b.png")

	idCompare := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: localConfig(pathA),
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
			{
				Config: localConfig(pathB),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(iconResourceAddress, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
		},
	})
}

// TestAccResource_Icon_LocalFile_SameContentDifferentPath_NoReplace
// switches the source path to a byte-identical copy. The provider must
// see the same hash and perform an in-place update (icon_file_source
// changes only); the ID must NOT change.
func TestAccResource_Icon_LocalFile_SameContentDifferentPath_NoReplace(t *testing.T) {
	testhelpers.AccPreCheck(t)

	pathA := fixturePath(t, "icon_a.png")
	pathACopy := fixturePath(t, "icon_a_copy.png")

	idCompare := statecheck.CompareValue(compare.ValuesSame())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: localConfig(pathA),
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
			{
				Config: localConfig(pathACopy),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(iconResourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
		},
	})
}

// TestAccResource_Icon_URL_BasicCRUD creates an icon from the real
// concepts.jamf.com URL, asserts state, and re-applies for an empty plan.
// Requires network access during plan + apply.
func TestAccResource_Icon_URL_BasicCRUD(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: urlConfig(urlJamfCLI),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(iconResourceAddress, "id"),
					resource.TestCheckResourceAttrSet(iconResourceAddress, "url"),
					resource.TestMatchResourceAttr(iconResourceAddress, "source_hash", sourceHashRegex),
				),
			},
			{
				Config: urlConfig(urlJamfCLI),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccResource_Icon_URL_ChangeBetweenURLs_TriggersReplace creates from
// the jamf-cli URL and then switches to the go-sdk URL. Different remote
// content → destroy-before-create. ID changes.
func TestAccResource_Icon_URL_ChangeBetweenURLs_TriggersReplace(t *testing.T) {
	testhelpers.AccPreCheck(t)

	idCompare := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: urlConfig(urlJamfCLI),
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
			{
				Config: urlConfig(urlGoSDK),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(iconResourceAddress, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
		},
	})
}

// TestAccResource_Icon_URL_SwitchToLocalSameBytes_NoReplace downloads the
// jamf-cli URL to a tempfile, creates from URL, then switches to that
// tempfile (byte-identical to what was served). Hashes match → in-place
// update only; ID unchanged.
func TestAccResource_Icon_URL_SwitchToLocalSameBytes_NoReplace(t *testing.T) {
	testhelpers.AccPreCheck(t)

	localCopyOfURLBytes := downloadToTempFile(t, urlJamfCLI)

	idCompare := statecheck.CompareValue(compare.ValuesSame())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: urlConfig(urlJamfCLI),
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
			{
				Config: localConfig(localCopyOfURLBytes),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(iconResourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
		},
	})
}

// TestAccResource_Icon_URL_SwitchToLocalDifferentBytes_TriggersReplace
// creates from the jamf-cli URL then switches to icon_b.png on disk.
// Hashes differ → destroy-before-create. ID changes.
func TestAccResource_Icon_URL_SwitchToLocalDifferentBytes_TriggersReplace(t *testing.T) {
	testhelpers.AccPreCheck(t)

	pathB := fixturePath(t, "icon_b.png")

	idCompare := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: urlConfig(urlJamfCLI),
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
			{
				Config: localConfig(pathB),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(iconResourceAddress, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idCompare.AddStateValue(iconResourceAddress, tfjsonpath.New("id")),
				},
			},
		},
	})
}

// TestAccResource_Icon_Import_NoReplace_AfterMatchingLocalConfig pre-creates
// an icon via the SDK, imports it by ID, then applies a config whose
// icon_file_source points at the SERVER-SERVED bytes (downloaded from the
// icon's CDN URL after the import step uploads it). Expected outcome:
// in-place update (icon_file_source goes null → local path) with NO
// replacement — the imported source_hash and the ModifyPlan-computed
// source_hash match because both derive from the SAME server-served bytes.
//
// Why this matters: Jamf Pro re-encodes uploaded PNGs server-side
// (different zlib compression and/or metadata). The bytes served back
// from /icon/{id} or the CDN URL are NOT byte-identical to the original
// upload. Therefore the documented import workflow is:
//
//  1. terraform import jamfplatform_pro_icon.x <id>
//  2. Download the icon from state.url (NOT your original upload file)
//  3. Set icon_file_source to that downloaded copy
//
// This test exercises that contract.
func TestAccResource_Icon_Import_NoReplace_AfterMatchingLocalConfig(t *testing.T) {
	testhelpers.AccPreCheck(t)

	bytesA := fixtureBytes(t, "icon_a.png")

	apiClient := testhelpers.NewAcceptanceClient(t)
	proClient := pro.New(apiClient)

	uploadResp, err := proClient.UploadIconV1(context.Background(), "icon_a.png", bytes.NewReader(bytesA))
	if err != nil {
		t.Fatalf("pre-creating icon via SDK: %v", err)
	}
	preCreatedID := strconv.Itoa(uploadResp.ID)
	t.Logf("pre-created icon id=%s url=%s", preCreatedID, uploadResp.URL)

	// Download the SERVER-SERVED bytes to a tempfile. This emulates the
	// user-facing workflow ("curl -o ./icon.png $url") and ensures the
	// local file we point at in step 2 has the exact bytes Jamf stored.
	serverBytesPath := downloadToTempFile(t, uploadResp.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: import the pre-existing icon. Read's import path
			// downloads the icon bytes via state.url and stores canonical
			// "sha256:<hex>" source_hash. ImportStatePersist=true so the
			// next step inherits this state.
			{
				Config:             localConfig(serverBytesPath),
				ResourceName:       iconResourceAddress,
				ImportState:        true,
				ImportStateId:      preCreatedID,
				ImportStatePersist: true,
				// ImportStateVerify=false because icon_file_source is null
				// immediately after import and is only populated by step 2.
				ImportStateVerify: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(iconResourceAddress, "id", preCreatedID),
					resource.TestMatchResourceAttr(iconResourceAddress, "source_hash", sourceHashRegex),
				),
			},
			// Step 2: apply config with icon_file_source pointing at the
			// server-served bytes (the user's downloaded copy). ModifyPlan
			// recomputes source_hash from the local file. Hash MUST equal
			// the import-path hash so this is an in-place update, not a
			// replacement. ID must remain preCreatedID.
			{
				Config: localConfig(serverBytesPath),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(iconResourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(iconResourceAddress, "id", preCreatedID),
					resource.TestMatchResourceAttr(iconResourceAddress, "source_hash", sourceHashRegex),
				),
			},
		},
	})
}

// TestAccResource_Icon_Import_Replace_WhenLocalIsOriginalUpload demonstrates
// the OTHER side of the server-side-transform reality: if a user imports an
// icon and then points icon_file_source at the ORIGINAL upload file (the
// bytes they had on disk before Jamf re-encoded them), the local hash and
// the import-path hash differ, so the plan correctly shows a REPLACEMENT.
//
// This is expected and correct behaviour — the test exists to document the
// failure mode so a future contributor doesn't try to "fix" it.
func TestAccResource_Icon_Import_Replace_WhenLocalIsOriginalUpload(t *testing.T) {
	testhelpers.AccPreCheck(t)

	pathA := fixturePath(t, "icon_a.png")
	bytesA := fixtureBytes(t, "icon_a.png")

	apiClient := testhelpers.NewAcceptanceClient(t)
	proClient := pro.New(apiClient)

	uploadResp, err := proClient.UploadIconV1(context.Background(), "icon_a.png", bytes.NewReader(bytesA))
	if err != nil {
		t.Fatalf("pre-creating icon via SDK: %v", err)
	}
	preCreatedID := strconv.Itoa(uploadResp.ID)
	t.Logf("pre-created icon id=%s url=%s (Jamf re-encodes; URL bytes != local bytes)", preCreatedID, uploadResp.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             localConfig(pathA),
				ResourceName:       iconResourceAddress,
				ImportState:        true,
				ImportStateId:      preCreatedID,
				ImportStatePersist: true,
				ImportStateVerify:  false,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(iconResourceAddress, "id", preCreatedID),
					resource.TestMatchResourceAttr(iconResourceAddress, "source_hash", sourceHashRegex),
				),
			},
			{
				Config: localConfig(pathA),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(iconResourceAddress, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
}
