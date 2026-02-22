// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// waitForBenchmarkSync polls until the benchmark reaches a terminal state
// (SYNCED or FAILED) or the provided context is canceled.
func waitForBenchmarkSync(ctx context.Context, c *client.Client, id string, interval time.Duration) (*client.CBEngineBenchmarkV2, error) {
	var synced *client.CBEngineBenchmarkV2
	err := helpers.PollUntil(ctx, interval, func(pollCtx context.Context) (bool, error) {
		benchmarks, err := c.GetCBEngineBenchmarksV2(pollCtx)
		if err != nil {
			tflog.Debug(pollCtx, "polling benchmarks failed", map[string]any{"error": err.Error()})
			return false, fmt.Errorf("failed to poll benchmarks: %w", err)
		}
		for _, b := range benchmarks.Benchmarks {
			if b.ID != id {
				continue
			}
			benchCopy := b
			tflog.Debug(pollCtx, "benchmark syncState", map[string]any{"benchmark_id": id, "sync_state": benchCopy.SyncState})
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
		tflog.Debug(pollCtx, "benchmark not present yet", map[string]any{"benchmark_id": id})
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return synced, nil
}

// waitForBenchmarkDeletion polls until the benchmark is no longer present or
// the context is canceled. Returns nil when the benchmark is absent.
func waitForBenchmarkDeletion(ctx context.Context, c *client.Client, id string, interval time.Duration) error {
	return helpers.PollUntil(ctx, interval, func(pollCtx context.Context) (bool, error) {
		benchmarks, err := c.GetCBEngineBenchmarksV2(pollCtx)
		if err != nil {
			tflog.Debug(pollCtx, "polling benchmarks failed", map[string]any{"error": err.Error()})
			return false, fmt.Errorf("failed to poll benchmarks: %w", err)
		}
		for _, b := range benchmarks.Benchmarks {
			if b.ID != id {
				continue
			}
			tflog.Debug(pollCtx, "benchmark still present during deletion poll", map[string]any{
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
		tflog.Debug(pollCtx, "benchmark absent after delete", map[string]any{"benchmark_id": id})
		return true, nil
	})
}
