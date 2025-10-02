// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// WaitForBenchmarkSync polls until the benchmark reaches a terminal state
// (SYNCED or FAILED) or the provided context is canceled. The interval
// controls how often the API is polled.
func waitForBenchmarkSync(ctx context.Context, c *client.Client, id string, interval time.Duration) (*client.CBEngineBenchmark, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
			benchmarks, err := c.GetCBEngineBenchmarks(ctx)
			if err != nil {
				tflog.Debug(ctx, "polling benchmarks failed", map[string]interface{}{"error": err.Error()})
				return nil, fmt.Errorf("failed to poll benchmarks: %w", err)
			}
			var found *client.CBEngineBenchmark
			for _, b := range benchmarks.Benchmarks {
				if b.ID == id {
					found = &b
					break
				}
			}
			if found == nil {
				tflog.Debug(ctx, "benchmark not present yet", map[string]interface{}{"benchmark_id": id})
				continue
			}
			tflog.Debug(ctx, "benchmark syncState", map[string]interface{}{"benchmark_id": id, "sync_state": found.SyncState})
			switch found.SyncState {
			case "PENDING":
				continue
			case "SYNCED":
				return found, nil
			case "FAILED":
				return found, fmt.Errorf("benchmark %s in FAILED state", id)
			default:
				return found, fmt.Errorf("unexpected syncState for benchmark %s: %s", id, found.SyncState)
			}
		}
	}
}

// WaitForBenchmarkDeletion polls until the benchmark is no longer present or
// the context is canceled. Returns nil when the benchmark is absent. If the
// API reports a DELETE_FAILED state an error is returned.
func waitForBenchmarkDeletion(ctx context.Context, c *client.Client, id string, interval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
			benchmarks, err := c.GetCBEngineBenchmarks(ctx)
			if err != nil {
				tflog.Debug(ctx, "polling benchmarks failed", map[string]interface{}{"error": err.Error()})
				return fmt.Errorf("failed to poll benchmarks: %w", err)
			}
			present := false
			for _, b := range benchmarks.Benchmarks {
				if b.ID == id {
					present = true
					tflog.Debug(ctx, "benchmark still present during deletion poll", map[string]interface{}{
						"benchmark_id": b.ID,
						"sync_state":   b.SyncState,
					})
					if b.SyncState == "DELETING" {
						break
					}
					if b.SyncState == "DELETE_FAILED" {
						return fmt.Errorf("benchmark %s deletion failed: syncState=DELETE_FAILED", id)
					}
					return fmt.Errorf("benchmark %s still present after delete, syncState=%s", id, b.SyncState)
				}
			}
			if !present {
				tflog.Debug(ctx, "benchmark absent after delete", map[string]interface{}{"benchmark_id": id})
				return nil
			}
		}
	}
}

// isNotFoundError checks if the error is a 404 not found error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return strings.Contains(errorStr, "status 404") ||
		strings.Contains(errorStr, "was not found") ||
		strings.Contains(errorStr, "NOT_FOUND")
}
