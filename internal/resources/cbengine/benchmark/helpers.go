// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// WaitForBenchmarkSync polls until the benchmark reaches a terminal state
// (SYNCED or FAILED) or the provided context is canceled. The interval
// controls how often the API is polled.
func waitForBenchmarkSync(ctx context.Context, c *client.Client, id string, interval time.Duration) (*client.CBEngineBenchmarkV2, error) {
	var synced *client.CBEngineBenchmarkV2
	err := helpers.PollUntil(ctx, interval, func(pollCtx context.Context) (bool, error) {
		benchmarks, err := c.GetCBEngineBenchmarksV2(pollCtx)
		if err != nil {
			tflog.Debug(pollCtx, "polling benchmarks failed", map[string]interface{}{"error": err.Error()})
			return false, fmt.Errorf("failed to poll benchmarks: %w", err)
		}
		for _, b := range benchmarks.Benchmarks {
			if b.ID != id {
				continue
			}
			benchCopy := b
			tflog.Debug(pollCtx, "benchmark syncState", map[string]interface{}{"benchmark_id": id, "sync_state": benchCopy.SyncState})
			switch benchCopy.SyncState {
			case "PENDING":
				return false, nil
			case "SYNCED":
				synced = &benchCopy
				return true, nil
			case "FAILED":
				return false, fmt.Errorf("benchmark %s in FAILED state", id)
			default:
				return false, fmt.Errorf("unexpected syncState for benchmark %s: %s", id, benchCopy.SyncState)
			}
		}
		tflog.Debug(pollCtx, "benchmark not present yet", map[string]interface{}{"benchmark_id": id})
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return synced, nil
}

// WaitForBenchmarkDeletion polls until the benchmark is no longer present or
// the context is canceled. Returns nil when the benchmark is absent. If the
// API reports a DELETE_FAILED state an error is returned.
func waitForBenchmarkDeletion(ctx context.Context, c *client.Client, id string, interval time.Duration) error {
	return helpers.PollUntil(ctx, interval, func(pollCtx context.Context) (bool, error) {
		benchmarks, err := c.GetCBEngineBenchmarksV2(pollCtx)
		if err != nil {
			tflog.Debug(pollCtx, "polling benchmarks failed", map[string]interface{}{"error": err.Error()})
			return false, fmt.Errorf("failed to poll benchmarks: %w", err)
		}
		for _, b := range benchmarks.Benchmarks {
			if b.ID != id {
				continue
			}
			tflog.Debug(pollCtx, "benchmark still present during deletion poll", map[string]interface{}{
				"benchmark_id": b.ID,
				"sync_state":   b.SyncState,
			})
			switch b.SyncState {
			case "DELETING":
				return false, nil
			case "DELETE_FAILED":
				return false, fmt.Errorf("benchmark %s deletion failed: syncState=DELETE_FAILED", id)
			default:
				return false, fmt.Errorf("benchmark %s still present after delete, syncState=%s", id, b.SyncState)
			}
		}
		tflog.Debug(pollCtx, "benchmark absent after delete", map[string]interface{}{"benchmark_id": id})
		return true, nil
	})
}
