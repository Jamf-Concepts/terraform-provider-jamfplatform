// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package client_test

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestAcceptance_Benchmark_CreateAllRules(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	baselines, err := c.GetCBEngineBaselinesV1(ctx)
	if err != nil {
		t.Fatalf("GetCBEngineBaselinesV1 failed: %v", err)
	}
	if len(baselines.Baselines) == 0 {
		t.Skip("No baselines available — cannot create benchmark")
	}

	baseline := baselines.Baselines[0]
	rules, err := c.GetCBEngineRulesV1(ctx, baseline.BaselineID)
	if err != nil {
		t.Fatalf("GetCBEngineRulesV1 failed: %v", err)
	}
	if len(rules.Rules) == 0 {
		t.Skip("No rules found for baseline — cannot create benchmark")
	}

	benchmarkRules := make([]client.CBEngineRuleRequestV2, 0, len(rules.Rules))
	for _, r := range rules.Rules {
		rr := client.CBEngineRuleRequestV2{
			ID:      r.ID,
			Enabled: r.Enabled,
		}
		if r.ODV != nil {
			rr.ODV = &client.CBEngineODVRequestV2{Value: r.ODV.Value}
		}
		benchmarkRules = append(benchmarkRules, rr)
	}

	title := "tf-acc-benchmark-all-rules"
	testhelpers.EnsureBenchmarkDeleted(t, c, ctx, title)

	createReq := &client.CBEngineBenchmarkRequestV2{
		Title:            title,
		Description:      "Acceptance test — all rules from first baseline — safe to delete",
		SourceBaselineID: baseline.BaselineID,
		Sources:          rules.Sources,
		Rules:            benchmarkRules,
		Target:           client.CBEngineTargetV2{DeviceGroups: []string{groupID}},
		EnforcementMode:  "MONITOR",
	}

	benchmark, err := c.CreateCBEngineBenchmarkV2(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateCBEngineBenchmarkV2 failed: %v", err)
	}

	t.Cleanup(func() {
		testhelpers.EnsureBenchmarkDeletedByID(t, c, ctx, benchmark.BenchmarkID)
	})

	fetched, err := c.GetCBEngineBenchmarkByIDV2(ctx, benchmark.BenchmarkID)
	if err != nil {
		t.Fatalf("GetCBEngineBenchmarkByIDV2 failed: %v", err)
	}

	if fetched.Title != title {
		t.Errorf("expected title %q, got %q", title, fetched.Title)
	}
	if fetched.EnforcementMode != "MONITOR" {
		t.Errorf("expected enforcement mode 'MONITOR', got %q", fetched.EnforcementMode)
	}
	if len(fetched.Target.DeviceGroups) != 1 || fetched.Target.DeviceGroups[0] != groupID {
		t.Errorf("expected target group %q, got %v", groupID, fetched.Target.DeviceGroups)
	}

	t.Logf("Created benchmark ID: %s, baseline: %s, rules: %d", benchmark.BenchmarkID, baseline.BaselineID, len(fetched.Rules))
}

func TestAcceptance_Benchmark_CreateCustomRules(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	baselines, err := c.GetCBEngineBaselinesV1(ctx)
	if err != nil {
		t.Fatalf("GetCBEngineBaselinesV1 failed: %v", err)
	}
	if len(baselines.Baselines) == 0 {
		t.Skip("No baselines available — cannot create benchmark")
	}

	baseline := baselines.Baselines[0]
	rules, err := c.GetCBEngineRulesV1(ctx, baseline.BaselineID)
	if err != nil {
		t.Fatalf("GetCBEngineRulesV1 failed: %v", err)
	}
	if len(rules.Rules) < 2 {
		t.Skip("Need at least 2 rules to create custom benchmark")
	}

	customRules := make([]client.CBEngineRuleRequestV2, 0, 2)
	for i := 0; i < 2 && i < len(rules.Rules); i++ {
		r := rules.Rules[i]
		rr := client.CBEngineRuleRequestV2{
			ID:      r.ID,
			Enabled: true,
		}
		if r.ODV != nil {
			rr.ODV = &client.CBEngineODVRequestV2{Value: r.ODV.Value}
		}
		customRules = append(customRules, rr)
	}

	title := "tf-acc-benchmark-custom-rules"
	testhelpers.EnsureBenchmarkDeleted(t, c, ctx, title)

	createReq := &client.CBEngineBenchmarkRequestV2{
		Title:            title,
		Description:      "Acceptance test — custom subset of rules — safe to delete",
		SourceBaselineID: baseline.BaselineID,
		Sources:          rules.Sources,
		Rules:            customRules,
		Target:           client.CBEngineTargetV2{DeviceGroups: []string{groupID}},
		EnforcementMode:  "MONITOR_AND_ENFORCE",
	}

	benchmark, err := c.CreateCBEngineBenchmarkV2(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateCBEngineBenchmarkV2 failed: %v", err)
	}

	t.Cleanup(func() {
		testhelpers.EnsureBenchmarkDeletedByID(t, c, ctx, benchmark.BenchmarkID)
	})

	fetched, err := c.GetCBEngineBenchmarkByIDV2(ctx, benchmark.BenchmarkID)
	if err != nil {
		t.Fatalf("GetCBEngineBenchmarkByIDV2 failed: %v", err)
	}

	if fetched.Title != title {
		t.Errorf("expected title %q, got %q", title, fetched.Title)
	}
	if fetched.EnforcementMode != "MONITOR_AND_ENFORCE" {
		t.Errorf("expected enforcement mode 'MONITOR_AND_ENFORCE', got %q", fetched.EnforcementMode)
	}
	if len(fetched.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(fetched.Rules))
	}

	t.Logf("Created custom benchmark ID: %s, rules: %d", benchmark.BenchmarkID, len(fetched.Rules))
}

func TestAcceptance_Benchmark_GetByTitle(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	baselines, err := c.GetCBEngineBaselinesV1(ctx)
	if err != nil {
		t.Fatalf("GetCBEngineBaselinesV1 failed: %v", err)
	}
	if len(baselines.Baselines) == 0 {
		t.Skip("No baselines available — cannot create benchmark")
	}

	baseline := baselines.Baselines[0]
	rules, err := c.GetCBEngineRulesV1(ctx, baseline.BaselineID)
	if err != nil {
		t.Fatalf("GetCBEngineRulesV1 failed: %v", err)
	}
	if len(rules.Rules) == 0 {
		t.Skip("No rules found — cannot create benchmark")
	}

	rr := client.CBEngineRuleRequestV2{
		ID:      rules.Rules[0].ID,
		Enabled: true,
	}
	if rules.Rules[0].ODV != nil {
		rr.ODV = &client.CBEngineODVRequestV2{Value: rules.Rules[0].ODV.Value}
	}

	title := "tf-acc-benchmark-find-by-title"
	testhelpers.EnsureBenchmarkDeleted(t, c, ctx, title)

	createReq := &client.CBEngineBenchmarkRequestV2{
		Title:            title,
		Description:      "Acceptance test — safe to delete",
		SourceBaselineID: baseline.BaselineID,
		Sources:          rules.Sources,
		Rules:            []client.CBEngineRuleRequestV2{rr},
		Target:           client.CBEngineTargetV2{DeviceGroups: []string{groupID}},
		EnforcementMode:  "MONITOR",
	}

	benchmark, err := c.CreateCBEngineBenchmarkV2(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateCBEngineBenchmarkV2 failed: %v", err)
	}

	t.Cleanup(func() {
		testhelpers.EnsureBenchmarkDeletedByID(t, c, ctx, benchmark.BenchmarkID)
	})

	found, err := c.GetCBEngineBenchmarkByTitleV2(ctx, title)
	if err != nil {
		t.Fatalf("GetCBEngineBenchmarkByTitleV2 failed: %v", err)
	}

	if found.BenchmarkID != benchmark.BenchmarkID {
		t.Errorf("expected benchmark ID %q, got %q", benchmark.BenchmarkID, found.BenchmarkID)
	}

	t.Logf("Found benchmark by title: ID %s", found.BenchmarkID)
}
