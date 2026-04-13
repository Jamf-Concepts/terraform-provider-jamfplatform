// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
)

// GetBenchmarkByTitle retrieves a benchmark by title by listing all benchmarks and filtering by title.
func GetBenchmarkByTitle(ctx context.Context, client *jamfplatform.Client, title string) (*jamfplatform.BenchmarkResponseV2, error) {
	resp, err := client.ListBenchmarks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list benchmarks: %w", err)
	}
	for _, b := range resp.Benchmarks {
		if b.Title == title {
			return client.GetBenchmark(ctx, b.ID)
		}
	}
	return nil, fmt.Errorf("benchmark with title %q not found", title)
}
