// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// acceptanceClientOnce ensures the acceptance client is initialized only once across all tests.
var acceptanceClientOnce sync.Once

// acceptanceClient holds the singleton acceptance client instance.
var acceptanceClient *client.Client

// acceptanceClientErr captures any error during acceptance client initialization.
var acceptanceClientErr error

// NewAcceptanceClient returns a live Jamf Platform API client for acceptance testing.
// It reads credentials from JAMFPLATFORM_BASE_URL, JAMFPLATFORM_CLIENT_ID, and
// JAMFPLATFORM_CLIENT_SECRET environment variables, and skips the test if they are not set.
// The client is created once and reused across all tests in a run.
func NewAcceptanceClient(t *testing.T) *client.Client {
	t.Helper()

	acceptanceClientOnce.Do(func() {
		baseURL := os.Getenv("JAMFPLATFORM_BASE_URL")
		clientID := os.Getenv("JAMFPLATFORM_CLIENT_ID")
		clientSecret := os.Getenv("JAMFPLATFORM_CLIENT_SECRET")

		if baseURL == "" || clientID == "" || clientSecret == "" {
			acceptanceClientErr = fmt.Errorf("missing required environment variables (JAMFPLATFORM_BASE_URL, JAMFPLATFORM_CLIENT_ID, JAMFPLATFORM_CLIENT_SECRET)")
			return
		}

		c := client.NewClient(baseURL, clientID, clientSecret)
		if err := c.ValidateCredentials(context.Background()); err != nil {
			acceptanceClientErr = fmt.Errorf("failed to validate credentials: %w", err)
			return
		}

		acceptanceClient = c
	})

	if acceptanceClientErr != nil {
		t.Skipf("Skipping acceptance test: %v", acceptanceClientErr)
	}

	return acceptanceClient
}

// smartGroupFixtureOnce ensures the fixture smart group is created only once.
var smartGroupFixtureOnce sync.Once

// smartGroupID holds the ID of the fixture smart group.
var smartGroupID string

// smartGroupErr captures any error during fixture creation.
var smartGroupErr error

// smartGroupFixtureName is the name used for the shared smart group fixture.
const smartGroupFixtureName = "tf-provider-test-fixture"

// RequireSmartGroupFixture returns the ID of a smart device group for acceptance tests
// that need a live smart group to target, such as benchmarks and blueprints. If the group
// already exists in the tenant it is reused; otherwise a new one is created.
func RequireSmartGroupFixture(t *testing.T) string {
	t.Helper()

	c := NewAcceptanceClient(t)

	smartGroupFixtureOnce.Do(func() {
		ctx := context.Background()

		groups, err := c.GetDeviceGroupsV1(ctx, nil, fmt.Sprintf("name==%q", smartGroupFixtureName))
		if err != nil {
			smartGroupErr = fmt.Errorf("failed to look up fixture smart group: %w", err)
			return
		}

		for _, g := range groups {
			if g.Name == smartGroupFixtureName {
				smartGroupID = g.ID
				return
			}
		}

		desc := "Terraform provider acceptance test fixture — safe to delete"
		req := &client.DeviceGroupCreateRepresentationV1{
			Name:        smartGroupFixtureName,
			Description: &desc,
			DeviceType:  "COMPUTER",
			GroupType:   "SMART",
			Criteria: []client.DeviceGroupCriteriaRepresentationV1{
				{
					Order:          0,
					AttributeName:  "Serial Number",
					Operator:       "LIKE",
					AttributeValue: "",
					JoinType:       "AND",
				},
			},
		}

		resp, err := c.CreateDeviceGroupV1(ctx, req)
		if err != nil {
			smartGroupErr = fmt.Errorf("failed to create fixture smart group: %w", err)
			return
		}

		smartGroupID = resp.ID
	})

	if smartGroupErr != nil {
		t.Fatalf("Smart group fixture failed: %v", smartGroupErr)
	}

	return smartGroupID
}

// EnsureBenchmarkDeleted removes a CBEngine benchmark by title. It waits for the
// benchmark to reach a stable sync state (SYNCED or FAILED) before issuing the
// delete — deleting while still in PENDING causes the benchmark to get stuck in a
// DELETING state. After the delete is issued it polls until the benchmark disappears.
func EnsureBenchmarkDeleted(t *testing.T, c *client.Client, ctx context.Context, title string) {
	t.Helper()
	existing, err := c.GetCBEngineBenchmarkByTitleV2(ctx, title)
	if err != nil {
		return
	}
	t.Logf("Cleaning up existing benchmark %q (ID: %s)", title, existing.BenchmarkID)

	waitForBenchmarkSyncState(t, c, ctx, existing.BenchmarkID)

	if err := c.DeleteCBEngineBenchmarkV1(ctx, existing.BenchmarkID); err != nil {
		t.Logf("Warning: failed to delete benchmark %q: %v", title, err)
		return
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		_, err := c.GetCBEngineBenchmarkByTitleV2(ctx, title)
		if err != nil {
			t.Logf("Benchmark %q fully deleted", title)
			return
		}
	}
	t.Logf("Warning: benchmark %q still exists after 30s — proceeding anyway", title)
}

// EnsureBenchmarkDeletedByID removes a CBEngine benchmark by ID. It waits for the
// benchmark to reach a stable sync state before deleting, then polls until the
// benchmark is fully removed.
func EnsureBenchmarkDeletedByID(t *testing.T, c *client.Client, ctx context.Context, benchmarkID string) {
	t.Helper()

	waitForBenchmarkSyncState(t, c, ctx, benchmarkID)

	if err := c.DeleteCBEngineBenchmarkV1(ctx, benchmarkID); err != nil {
		t.Logf("Warning: failed to delete benchmark %s: %v", benchmarkID, err)
		return
	}
	t.Logf("Delete issued for benchmark %s", benchmarkID)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		if _, found := benchmarkSyncState(c, ctx, benchmarkID); !found {
			t.Logf("Benchmark %s fully deleted", benchmarkID)
			return
		}
	}
	t.Logf("Warning: benchmark %s still present after 30s", benchmarkID)
}

// waitForBenchmarkSyncState polls until the benchmark reaches SYNCED or FAILED,
// or until a 2-minute timeout expires.
func waitForBenchmarkSyncState(t *testing.T, c *client.Client, ctx context.Context, benchmarkID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		state, found := benchmarkSyncState(c, ctx, benchmarkID)
		if !found {
			t.Logf("Benchmark %s not found in list, may already be deleted", benchmarkID)
			return
		}
		if state == "SYNCED" || state == "FAILED" {
			t.Logf("Benchmark %s reached state %s", benchmarkID, state)
			return
		}
		t.Logf("Benchmark %s in state %q, waiting for SYNCED", benchmarkID, state)
		time.Sleep(3 * time.Second)
	}
	t.Logf("Warning: benchmark %s did not reach SYNCED after 2m — proceeding anyway", benchmarkID)
}

// benchmarkSyncState returns the sync state of a benchmark by ID from the list endpoint.
func benchmarkSyncState(c *client.Client, ctx context.Context, benchmarkID string) (string, bool) {
	benchmarks, err := c.GetCBEngineBenchmarksV2(ctx)
	if err != nil {
		return "", false
	}
	for _, b := range benchmarks.Benchmarks {
		if b.ID == benchmarkID {
			return b.SyncState, true
		}
	}
	return "", false
}

// CleanupSmartGroupFixture deletes the shared smart group fixture if it was created.
// Call this from TestMain after all tests in the package have completed.
func CleanupSmartGroupFixture() {
	if acceptanceClient != nil && smartGroupID != "" {
		_ = acceptanceClient.DeleteDeviceGroupV1(context.Background(), smartGroupID)
	}
}
