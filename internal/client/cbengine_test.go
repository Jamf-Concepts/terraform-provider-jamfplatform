// Copyright 2026 Jamf Software LLC.

package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestGetCBEngineBaselinesV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.CBEngineBaselinesResponseV1{
			Baselines: []client.CBEngineBaselineInfoV1{
				{
					ID:          "baseline-1",
					BaselineID:  "bl-1",
					Name:        "macOS 15 Security",
					Description: "Security baseline for macOS 15",
					Version:     "1.0",
					Title:       "macOS 15 Security Baseline",
					RuleCount:   150,
				},
			},
		})
	}))

	resp, err := c.GetCBEngineBaselinesV1(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Baselines) != 1 {
		t.Fatalf("expected 1 baseline, got %d", len(resp.Baselines))
	}
	if resp.Baselines[0].Name != "macOS 15 Security" {
		t.Errorf("expected Name 'macOS 15 Security', got %q", resp.Baselines[0].Name)
	}
	if resp.Baselines[0].RuleCount != 150 {
		t.Errorf("expected RuleCount 150, got %d", resp.Baselines[0].RuleCount)
	}
}

func TestCreateCBEngineBenchmarkV2(t *testing.T) {
	var receivedBody client.CBEngineBenchmarkRequestV2

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		testhelpers.RespondJSON(w, http.StatusAccepted, client.CBEngineBenchmarkResponseV2{
			BenchmarkID:     "bench-new",
			Title:           "Test Benchmark",
			EnforcementMode: "AUDIT_ONLY",
			LastUpdatedAt:   time.Now(),
		})
	}))

	req := &client.CBEngineBenchmarkRequestV2{
		Title:            "Test Benchmark",
		SourceBaselineID: "baseline-1",
		Sources: []client.CBEngineSourceV1{
			{Branch: "main", Revision: "abc123"},
		},
		Rules: []client.CBEngineRuleRequestV2{
			{ID: "rule-1", Enabled: true},
			{ID: "rule-2", Enabled: false},
		},
		Target: client.CBEngineTargetV2{
			DeviceGroups: []string{"grp-1"},
		},
		EnforcementMode: "AUDIT_ONLY",
	}

	resp, err := c.CreateCBEngineBenchmarkV2(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.BenchmarkID != "bench-new" {
		t.Errorf("expected BenchmarkID 'bench-new', got %q", resp.BenchmarkID)
	}
	if receivedBody.Title != "Test Benchmark" {
		t.Errorf("expected request title 'Test Benchmark', got %q", receivedBody.Title)
	}
	if len(receivedBody.Rules) != 2 {
		t.Errorf("expected 2 rules in request, got %d", len(receivedBody.Rules))
	}
	if receivedBody.EnforcementMode != "AUDIT_ONLY" {
		t.Errorf("expected enforcement mode 'AUDIT_ONLY', got %q", receivedBody.EnforcementMode)
	}
}

func TestGetCBEngineBenchmarkByIDV2(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/benchmarks/bench-123") {
			http.NotFound(w, r)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.CBEngineBenchmarkResponseV2{
			BenchmarkID:     "bench-123",
			Title:           "Existing Benchmark",
			EnforcementMode: "ENFORCE",
			Rules: []client.CBEngineRuleInfoV1{
				{
					ID:      "rule-1",
					Title:   "Test Rule",
					Enabled: true,
				},
			},
			Target: client.CBEngineTargetV2{
				DeviceGroups: []string{"grp-a"},
			},
		})
	}))

	resp, err := c.GetCBEngineBenchmarkByIDV2(context.Background(), "bench-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.BenchmarkID != "bench-123" {
		t.Errorf("expected BenchmarkID 'bench-123', got %q", resp.BenchmarkID)
	}
	if resp.Title != "Existing Benchmark" {
		t.Errorf("expected Title 'Existing Benchmark', got %q", resp.Title)
	}
	if len(resp.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(resp.Rules))
	}
}

func TestGetCBEngineBenchmarksV2(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.RespondJSON(w, http.StatusOK, client.CBEngineBenchmarksResponseV2{
			Benchmarks: []client.CBEngineBenchmarkV2{
				{ID: "bench-1", Title: "Benchmark One", SyncState: "SYNCED"},
				{ID: "bench-2", Title: "Benchmark Two", SyncState: "SYNCING"},
			},
		})
	}))

	resp, err := c.GetCBEngineBenchmarksV2(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Benchmarks) != 2 {
		t.Errorf("expected 2 benchmarks, got %d", len(resp.Benchmarks))
	}
	if resp.Benchmarks[0].SyncState != "SYNCED" {
		t.Errorf("expected first benchmark SyncState 'SYNCED', got %q", resp.Benchmarks[0].SyncState)
	}
}

func TestDeleteCBEngineBenchmarkV1(t *testing.T) {
	var receivedMethod string
	var receivedPath string

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))

	err := c.DeleteCBEngineBenchmarkV1(context.Background(), "bench-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedMethod != http.MethodDelete {
		t.Errorf("expected DELETE method, got %q", receivedMethod)
	}
	if !strings.Contains(receivedPath, "/v1/benchmarks/bench-123") {
		t.Errorf("expected v1 benchmark path, got %q", receivedPath)
	}
}

func TestGetCBEngineRulesV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		baselineID := r.URL.Query().Get("baselineId")
		if baselineID != "bl-1" {
			t.Errorf("expected baselineId 'bl-1', got %q", baselineID)
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.CBEngineSourcedRulesV1{
			Sources: []client.CBEngineSourceV1{
				{Branch: "main", Revision: "abc123"},
			},
			Rules: []client.CBEngineRuleInfoV1{
				{
					ID:          "rule-1",
					Title:       "Require Passcode",
					SectionName: "Authentication",
					Enabled:     true,
					References:  []string{"CIS 1.1", "NIST AC-1"},
				},
			},
		})
	}))

	resp, err := c.GetCBEngineRulesV1(context.Background(), "bl-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(resp.Sources))
	}
	if len(resp.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(resp.Rules))
	}
	if resp.Rules[0].Title != "Require Passcode" {
		t.Errorf("expected rule title 'Require Passcode', got %q", resp.Rules[0].Title)
	}
	if len(resp.Rules[0].References) != 2 {
		t.Errorf("expected 2 references, got %d", len(resp.Rules[0].References))
	}
}

func TestGetCBEngineBenchmarkByTitleV2(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/benchmarks") && r.Method == http.MethodGet {
			testhelpers.RespondJSON(w, http.StatusOK, client.CBEngineBenchmarksResponseV2{
				Benchmarks: []client.CBEngineBenchmarkV2{
					{ID: "bench-target", Title: "Target Benchmark"},
					{ID: "bench-other", Title: "Other Benchmark"},
				},
			})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/bench-target") {
			testhelpers.RespondJSON(w, http.StatusOK, client.CBEngineBenchmarkResponseV2{
				BenchmarkID: "bench-target",
				Title:       "Target Benchmark",
			})
			return
		}
		http.NotFound(w, r)
	}))

	resp, err := c.GetCBEngineBenchmarkByTitleV2(context.Background(), "Target Benchmark")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.BenchmarkID != "bench-target" {
		t.Errorf("expected BenchmarkID 'bench-target', got %q", resp.BenchmarkID)
	}
}

func TestGetCBEngineBenchmarkByTitleV2_NotFound(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.RespondJSON(w, http.StatusOK, client.CBEngineBenchmarksResponseV2{
			Benchmarks: []client.CBEngineBenchmarkV2{
				{ID: "bench-1", Title: "Other Benchmark"},
			},
		})
	}))

	_, err := c.GetCBEngineBenchmarkByTitleV2(context.Background(), "Missing Benchmark")
	if err == nil {
		t.Fatal("expected error for missing benchmark title")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}
