// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package permissions

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite testdata/catalogue.golden from the current catalogue")

const goldenPath = "testdata/catalogue.golden"

// catalogueGolden renders every transcribed row as one "category | name | slug"
// line, in the order a table would print them: section name first, then
// permission name. Rendering through rowKey rather than sorting by slug keeps
// the golden's order the rendered order, so a reviewer reads the diff in the
// same sequence an operator reads the table.
func catalogueGolden() string {
	slugs := make([]string, 0, len(catalogue))
	for slug := range catalogue {
		slugs = append(slugs, slug)
	}
	sort.Slice(slugs, func(i, j int) bool {
		ci, ni, _ := rowKey(requirement{capability: slugs[i]})
		cj, nj, _ := rowKey(requirement{capability: slugs[j]})
		if ci != cj {
			return ci < cj
		}
		if ni != nj {
			return ni < nj
		}
		return slugs[i] < slugs[j]
	})
	var b strings.Builder
	for _, slug := range slugs {
		e := catalogue[slug]
		fmt.Fprintf(&b, "%s | %s | %s\n", e.category, e.name, slug)
	}
	return b.String()
}

// TestCatalogueGolden pins what every catalogue row SAYS, which no other test
// in this package does: TestCatalogueCoversEverySDKCapability asserts only that
// a required capability has a row, so a wrong section or a wrong permission
// name passes it. The golden cannot tell us a row is right — nothing in any
// repo can, since Jamf's permissions map is prose (see catalogue.go) — but it
// turns an edited row into a reviewable diff instead of a silent change to 300+
// published pages. A deliberate re-transcription updates the golden in the same
// commit: go test ./internal/common/permissions/ -update-golden
func TestCatalogueGolden(t *testing.T) {
	got := catalogueGolden()

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("creating testdata dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("writing %s: %v", goldenPath, err)
		}
		t.Logf("wrote %s (%d rows)", goldenPath, len(catalogue))
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading %s (regenerate with -update-golden): %v", goldenPath, err)
	}
	want := string(wantBytes)
	if got == want {
		return
	}

	gotLines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	wantLines := strings.Split(strings.TrimRight(want, "\n"), "\n")
	inWant := make(map[string]bool, len(wantLines))
	for _, l := range wantLines {
		inWant[l] = true
	}
	inGot := make(map[string]bool, len(gotLines))
	for _, l := range gotLines {
		inGot[l] = true
	}
	for _, l := range gotLines {
		if !inWant[l] {
			t.Errorf("catalogue row not in golden: %s", l)
		}
	}
	for _, l := range wantLines {
		if !inGot[l] {
			t.Errorf("golden row no longer in catalogue: %s", l)
		}
	}
	t.Errorf("catalogue.go and %s disagree — if the change is a deliberate re-transcription, "+
		"re-read Jamf's permissions map (URL in catalogue.go) and rerun with -update-golden", goldenPath)
}
