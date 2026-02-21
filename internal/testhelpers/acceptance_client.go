// Copyright 2026 Jamf Software LLC.

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

// EnsureBenchmarkDeleted removes a CBEngine benchmark by title and polls until it
// is fully deleted. Benchmark deletion is async — the API accepts the delete but the
// benchmark lingers in a DELETING state. This helper waits up to 30 seconds for it
// to disappear before returning.
func EnsureBenchmarkDeleted(t *testing.T, c *client.Client, ctx context.Context, title string) {
	t.Helper()
	existing, err := c.GetCBEngineBenchmarkByTitleV2(ctx, title)
	if err != nil {
		return
	}
	t.Logf("Cleaning up existing benchmark %q (ID: %s)", title, existing.BenchmarkID)
	_ = c.DeleteCBEngineBenchmarkV1(ctx, existing.BenchmarkID)

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
