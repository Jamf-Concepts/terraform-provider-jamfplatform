// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package benchmark_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckBenchmarkResourcesDestroy verifies that benchmarks and device groups created
// during the test have been destroyed. Benchmark deletion is async, so this polls briefly.
func testAccCheckBenchmarkResourcesDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testhelpers.NewAcceptanceClient(t)
		ctx := context.Background()

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "jamfplatform_cbengine_benchmark":
				deadline := time.Now().Add(30 * time.Second)
				for time.Now().Before(deadline) {
					_, err := c.GetCBEngineBenchmarkByIDV2(ctx, rs.Primary.ID)
					if err != nil {
						if helpers.IsNotFoundError(err) {
							break
						}
						return fmt.Errorf("error checking benchmark %s: %s", rs.Primary.ID, err)
					}
					time.Sleep(2 * time.Second)
				}
			case "jamfplatform_device_group":
				_, err := c.GetDeviceGroupByIDV1(ctx, rs.Primary.ID)
				if err == nil {
					return fmt.Errorf("device group %s still exists after destroy", rs.Primary.ID)
				}
				if !helpers.IsNotFoundError(err) {
					return fmt.Errorf("error checking device group %s: %s", rs.Primary.ID, err)
				}
			}
		}
		return nil
	}
}

// ensureBenchmarkCleanup deletes a benchmark by title and waits for async deletion to complete.
func ensureBenchmarkCleanup(t *testing.T, title string) {
	t.Helper()
	ctx := context.Background()
	c := testhelpers.NewAcceptanceClient(t)
	testhelpers.EnsureBenchmarkDeleted(t, c, ctx, title)
}

func TestAccResource_Benchmark_AllRules_Monitor(t *testing.T) {
	testhelpers.AccPreCheck(t)

	ctx := context.Background()
	c := testhelpers.NewAcceptanceClient(t)
	baselines, err := c.GetCBEngineBaselinesV1(ctx)
	if err != nil {
		t.Fatalf("Failed to check baselines: %v", err)
	}
	if len(baselines.Baselines) == 0 {
		t.Skip("No baselines available — CB Engine may not be enabled")
	}

	baselineID := baselines.Baselines[0].BaselineID
	rules, err := c.GetCBEngineRulesV1(ctx, baselineID)
	if err != nil {
		t.Fatalf("Failed to get rules: %v", err)
	}
	if len(rules.Rules) == 0 {
		t.Skip("No rules found for baseline")
	}

	var ruleBlocks []string
	for _, r := range rules.Rules {
		block := fmt.Sprintf(`{ id = %q, enabled = %t`, r.ID, r.Enabled)
		if r.ODV != nil {
			block += fmt.Sprintf(`, odv_value = %q`, r.ODV.Value)
		}
		block += " }"
		ruleBlocks = append(ruleBlocks, block)
	}

	var sourceBlocks []string
	for _, s := range rules.Sources {
		sourceBlocks = append(sourceBlocks, fmt.Sprintf(`{ branch = %q, revision = %q }`, s.Branch, s.Revision))
	}

	// Clean up any leftover benchmark from a previous test run and wait for async deletion
	ensureBenchmarkCleanup(t, "tf-acc-benchmark-all-rules")

	config := fmt.Sprintf(`
		resource "jamfplatform_device_group" "scope" {
			name        = "tf-acc-benchmark-scope"
			group_type  = "smart"
			device_type = "computer"
			criteria = [{
				criteria = "Serial Number"
				operator = "like"
				value    = ""
			}]
		}

		resource "jamfplatform_cbengine_benchmark" "test_all_rules" {
			title              = "tf-acc-benchmark-all-rules"
			description        = "Acceptance test — safe to delete"
			source_baseline_id = %q

			sources = [%s]
			rules   = [%s]

			target_device_group = jamfplatform_device_group.scope.id
			enforcement_mode    = "MONITOR"
		}
	`, baselineID, strings.Join(sourceBlocks, ",\n"), strings.Join(ruleBlocks, ",\n"))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBenchmarkResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_cbengine_benchmark.test_all_rules", "id"),
					resource.TestCheckResourceAttr("jamfplatform_cbengine_benchmark.test_all_rules", "title", "tf-acc-benchmark-all-rules"),
					resource.TestCheckResourceAttr("jamfplatform_cbengine_benchmark.test_all_rules", "enforcement_mode", "MONITOR"),
				),
			},
		},
	})
}

func TestAccResource_Benchmark_CustomRules_MonitorAndEnforce(t *testing.T) {
	testhelpers.AccPreCheck(t)

	ctx := context.Background()
	c := testhelpers.NewAcceptanceClient(t)
	baselines, err := c.GetCBEngineBaselinesV1(ctx)
	if err != nil {
		t.Fatalf("Failed to check baselines: %v", err)
	}
	if len(baselines.Baselines) == 0 {
		t.Skip("No baselines available — CB Engine may not be enabled")
	}

	baselineID := baselines.Baselines[0].BaselineID
	rules, err := c.GetCBEngineRulesV1(ctx, baselineID)
	if err != nil {
		t.Fatalf("Failed to get rules: %v", err)
	}
	if len(rules.Rules) < 2 {
		t.Skip("Need at least 2 rules to create custom benchmark")
	}

	var ruleBlocks []string
	for i := 0; i < 2; i++ {
		r := rules.Rules[i]
		block := fmt.Sprintf(`{ id = %q, enabled = true`, r.ID)
		if r.ODV != nil {
			block += fmt.Sprintf(`, odv_value = %q`, r.ODV.Value)
		}
		block += " }"
		ruleBlocks = append(ruleBlocks, block)
	}

	var sourceBlocks []string
	for _, s := range rules.Sources {
		sourceBlocks = append(sourceBlocks, fmt.Sprintf(`{ branch = %q, revision = %q }`, s.Branch, s.Revision))
	}

	// Clean up any leftover benchmark from a previous test run and wait for async deletion
	ensureBenchmarkCleanup(t, "tf-acc-benchmark-custom-rules")

	config := fmt.Sprintf(`
		resource "jamfplatform_device_group" "scope" {
			name        = "tf-acc-benchmark-scope-custom"
			group_type  = "smart"
			device_type = "computer"
			criteria = [{
				criteria = "Serial Number"
				operator = "like"
				value    = ""
			}]
		}

		resource "jamfplatform_cbengine_benchmark" "test_custom" {
			title              = "tf-acc-benchmark-custom-rules"
			description        = "Acceptance test custom rules — safe to delete"
			source_baseline_id = %q

			sources = [%s]
			rules   = [%s]

			target_device_group = jamfplatform_device_group.scope.id
			enforcement_mode    = "MONITOR_AND_ENFORCE"
		}
	`, baselineID, strings.Join(sourceBlocks, ",\n"), strings.Join(ruleBlocks, ",\n"))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBenchmarkResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("jamfplatform_cbengine_benchmark.test_custom", "id"),
					resource.TestCheckResourceAttr("jamfplatform_cbengine_benchmark.test_custom", "title", "tf-acc-benchmark-custom-rules"),
					resource.TestCheckResourceAttr("jamfplatform_cbengine_benchmark.test_custom", "enforcement_mode", "MONITOR_AND_ENFORCE"),
				),
			},
		},
	})
}

func TestAccDataSource_Baselines(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBenchmarkResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_cbengine_baselines" "all" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_cbengine_baselines.all", "baselines.#"),
				),
			},
		},
	})
}

func TestAccDataSource_Benchmarks(t *testing.T) {
	testhelpers.AccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBenchmarkResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: `
					data "jamfplatform_cbengine_benchmarks" "all" {}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_cbengine_benchmarks.all", "benchmarks.#"),
				),
			},
		},
	})
}

func TestAccDataSource_Rules(t *testing.T) {
	testhelpers.AccPreCheck(t)

	ctx := context.Background()
	c := testhelpers.NewAcceptanceClient(t)
	baselines, err := c.GetCBEngineBaselinesV1(ctx)
	if err != nil {
		t.Fatalf("Failed to check baselines: %v", err)
	}
	if len(baselines.Baselines) == 0 {
		t.Skip("No baselines available")
	}

	config := fmt.Sprintf(`
		data "jamfplatform_cbengine_rules" "test" {
			baseline_id = %q
		}
	`, baselines.Baselines[0].BaselineID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBenchmarkResourcesDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jamfplatform_cbengine_rules.test", "rules.#"),
					resource.TestCheckResourceAttrSet("data.jamfplatform_cbengine_rules.test", "sources.#"),
				),
			},
		},
	})
}
