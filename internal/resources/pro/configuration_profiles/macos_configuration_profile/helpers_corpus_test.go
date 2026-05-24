// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build profile_corpus

// Run with: go test -tags profile_corpus ./internal/resources/pro/configuration_profiles/macos_configuration_profile/...
//
// This is the regression net for "Jamf changed a mutation we didn't catch".
// It runs the mask + lenient-compare across every file in the full
// 200-profile roundtrip corpus under testing/profile_roundtrip/. The
// directory is gitignored and only present on developer machines that
// have run /tmp/sample_titles.py and /tmp/roundtrip.py.
//
// Run path is opt-in via build tag so make test stays fast and CI doesn't
// fail when the corpus is absent.

package macos_configuration_profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaskPayload_FullCorpus(t *testing.T) {
	corpusDir := "../../../../../testing/profile_roundtrip"
	mcDir := filepath.Join(corpusDir, "mobileconfigs")
	respDir := filepath.Join(corpusDir, "classic_response")
	entries, err := os.ReadDir(mcDir)
	if err != nil {
		t.Skipf("corpus directory %s not present — run /tmp/sample_titles.py and /tmp/roundtrip.py to populate", mcDir)
	}
	if len(entries) == 0 {
		t.Skipf("corpus directory %s is empty", mcDir)
	}

	var (
		processed int
		failed    []string
	)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".mobileconfig") {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), ".mobileconfig")
		mcPath := filepath.Join(mcDir, e.Name())
		respPath := filepath.Join(respDir, stem+".create_response.xml")
		mc, err := os.ReadFile(mcPath)
		if err != nil {
			t.Fatalf("reading %s: %v", mcPath, err)
		}
		resp, err := os.ReadFile(respPath)
		if err != nil {
			t.Logf("skip %s: no response file", stem)
			continue
		}
		srv, err := extractServerPayloadFromGeneral(resp)
		if err != nil {
			t.Logf("skip %s: extract failed: %v", stem, err)
			continue
		}
		ok, err := payloadsSemanticallyEqual(mc, srv)
		if err != nil {
			t.Logf("skip %s: compare error: %v", stem, err)
			continue
		}
		if !ok {
			failed = append(failed, stem)
		}
		processed++
	}
	t.Logf("processed %d corpus pairs; %d failed", processed, len(failed))
	if len(failed) > 0 {
		for _, s := range failed {
			t.Errorf("mask did not neutralise diff for %s", s)
		}
	}
}
