// Copyright 2025 Jamf Software LLC.
// https://developer.jamf.com/platform-api/reference/getbaselines
// https://developer.jamf.com/platform-api/reference/getbenchmark
// https://developer.jamf.com/platform-api/reference/gettenantbenchmarks
// https://developer.jamf.com/platform-api/reference/postbenchmark
// https://developer.jamf.com/platform-api/reference/deletebenchmark

package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// CBEngine Baseline Types

// CBEngineBaselinesResponse represents the response for baselines listing
type CBEngineBaselinesResponse struct {
	Baselines []CBEngineBaselineInfo `json:"baselines,omitempty"`
}

// CBEngineBaselineInfo represents information about a baseline
type CBEngineBaselineInfo struct {
	ID          string `json:"id,omitempty"`
	BaselineID  string `json:"baselineId,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Title       string `json:"title,omitempty"`
	RuleCount   int64  `json:"ruleCount,omitempty"`
}

// CBEngineSource represents source information
type CBEngineSource struct {
	Branch   string `json:"branch" validate:"required"`
	Revision string `json:"revision" validate:"required"`
}

// CBEngineTarget represents the target configuration
type CBEngineTarget struct {
	DeviceGroups []string `json:"deviceGroups" validate:"required"`
}

// CBEngine Benchmark Types

// CBEngineBenchmarkRequest represents the request body for creating/updating benchmarks
type CBEngineBenchmarkRequest struct {
	Title            string                `json:"title" validate:"required,max=100"`
	Description      string                `json:"description,omitempty" validate:"max=1000"`
	SourceBaselineID string                `json:"sourceBaselineId" validate:"required"`
	Sources          []CBEngineSource      `json:"sources" validate:"required"`
	Rules            []CBEngineRuleRequest `json:"rules" validate:"required"`
	Target           CBEngineTarget        `json:"target" validate:"required"`
	EnforcementMode  string                `json:"enforcementMode" validate:"required,oneof=MONITOR MONITOR_AND_ENFORCE"`
}

// CBEngineBenchmarkResponse represents the response for benchmark operations
type CBEngineBenchmarkResponse struct {
	BenchmarkID     string             `json:"benchmarkId"`
	TenantID        string             `json:"tenantId"`
	Title           string             `json:"title"`
	Description     string             `json:"description,omitempty"`
	Sources         []CBEngineSource   `json:"sources"`
	Rules           []CBEngineRuleInfo `json:"rules"`
	Target          CBEngineTarget     `json:"target"`
	EnforcementMode string             `json:"enforcementMode"`
	Deleted         bool               `json:"deleted"`
	UpdateAvailable bool               `json:"updateAvailable"`
	LastUpdatedAt   time.Time          `json:"lastUpdatedAt"`
}

// CBEngineBenchmarksResponse represents the response for listing benchmarks
type CBEngineBenchmarksResponse struct {
	Benchmarks []CBEngineBenchmark `json:"benchmarks"`
}

// CBEngineBenchmark represents a benchmark in the list response
type CBEngineBenchmark struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	Description     string         `json:"description,omitempty"`
	UpdateAvailable bool           `json:"updateAvailable"`
	Target          CBEngineTarget `json:"target"`
	SyncState       string         `json:"syncState"`
}

// CBEngine Rule Types

// CBEngineRuleRequest represents a rule in the request
type CBEngineRuleRequest struct {
	ID      string              `json:"id" validate:"required"`
	Enabled bool                `json:"enabled" validate:"required"`
	ODV     *CBEngineODVRequest `json:"odv,omitempty"`
}

// CBEngineODVRequest represents an organization-defined value in requests
type CBEngineODVRequest struct {
	Value string `json:"value" validate:"required"`
}

// CBEngineRuleInfo represents detailed rule information in responses
type CBEngineRuleInfo struct {
	ID                 string                                `json:"id"`
	SectionName        string                                `json:"sectionName"`
	Enabled            bool                                  `json:"enabled"`
	Title              string                                `json:"title"`
	References         []string                              `json:"references,omitempty"`
	Description        string                                `json:"description"`
	ODV                *CBEngineOrganizationDefinedValue     `json:"odv,omitempty"`
	SupportedOS        []CBEngineOSInfo                      `json:"supportedOs"`
	OSSpecificDefaults map[string]CBEngineOSSpecificRuleInfo `json:"osSpecificDefaults"`
	RuleRelation       *CBEngineRuleRelation                 `json:"ruleRelation,omitempty"`
}

// CBEngineOrganizationDefinedValue represents ODV with full details
type CBEngineOrganizationDefinedValue struct {
	Value       string                         `json:"value" validate:"required"`
	Hint        string                         `json:"hint,omitempty"`
	Placeholder string                         `json:"placeholder,omitempty"`
	Type        string                         `json:"type,omitempty"`
	Validation  *CBEngineValidationConstraints `json:"validation,omitempty"`
}

// CBEngineValidationConstraints represents validation rules for ODV
type CBEngineValidationConstraints struct {
	Min        *int     `json:"min,omitempty"`
	Max        *int     `json:"max,omitempty"`
	EnumValues []string `json:"enumValues,omitempty"`
	Regex      string   `json:"regex,omitempty"`
}

// CBEngineOSInfo represents operating system information
type CBEngineOSInfo struct {
	OSType         string `json:"osType"`
	OSVersion      int    `json:"osVersion"`
	ManagementType string `json:"managementType"`
}

// CBEngineOSSpecificRuleInfo represents OS-specific rule details
type CBEngineOSSpecificRuleInfo struct {
	Title       string                     `json:"title"`
	Description string                     `json:"description"`
	ODV         *CBEngineODVRecommendation `json:"odv,omitempty"`
}

// CBEngineODVRecommendation represents ODV recommendation
type CBEngineODVRecommendation struct {
	Value string `json:"value,omitempty"`
	Hint  string `json:"hint,omitempty"`
}

// CBEngineRuleRelation represents rule dependencies
type CBEngineRuleRelation struct {
	DependsOn []string `json:"dependsOn,omitempty"`
}

// CBEngineSourcedRules represents rules with their sources
type CBEngineSourcedRules struct {
	Sources []CBEngineSource   `json:"sources"`
	Rules   []CBEngineRuleInfo `json:"rules"`
}

// CBEngine Baseline operations

// GetCBEngineBaselines returns list of available mSCP baselines
func (c *Client) GetCBEngineBaselines(ctx context.Context) (*CBEngineBaselinesResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/v1/baselines", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get baselines: %w", err)
	}

	var result CBEngineBaselinesResponse
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CBEngine Benchmark operations

// CreateCBEngineBenchmark creates a new benchmark
func (c *Client) CreateCBEngineBenchmark(ctx context.Context, request *CBEngineBenchmarkRequest) (*CBEngineBenchmarkResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/v2/benchmarks", request)
	if err != nil {
		return nil, fmt.Errorf("failed to create benchmark: %w", err)
	}

	var result CBEngineBenchmarkResponse
	if err := c.handleAPIResponse(resp, 202, &result); err != nil {
		return nil, err
	}

	const pollInterval = 5
	const maxRetries = 24
	retries := 0
	for retries < maxRetries {
		time.Sleep(pollInterval * time.Second)
		benchmarks, err := c.GetCBEngineBenchmarks(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to poll benchmarks after create: %w", err)
		}
		var found *CBEngineBenchmark
		for _, b := range benchmarks.Benchmarks {
			if b.ID == result.BenchmarkID {
				found = &b
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("benchmark %s not found during polling after create", result.BenchmarkID)
		}
		switch found.SyncState {
		case "PENDING":
			retries++
			continue
		case "SYNCED":
			return &result, nil
		case "FAILED":
			_ = c.DeleteCBEngineBenchmark(ctx, found.ID)
			return c.CreateCBEngineBenchmark(ctx, request)
		default:
			return nil, fmt.Errorf("unexpected syncState after create: %s", found.SyncState)
		}
	}
	return nil, fmt.Errorf("timed out waiting for benchmark creation to complete after %d retries", maxRetries)
}

// GetCBEngineBenchmarks retrieves all benchmarks for the tenant
func (c *Client) GetCBEngineBenchmarks(ctx context.Context) (*CBEngineBenchmarksResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/v2/benchmarks", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmarks: %w", err)
	}

	var result CBEngineBenchmarksResponse
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetCBEngineBenchmarkByID retrieves a specific benchmark by ID
func (c *Client) GetCBEngineBenchmarkByID(ctx context.Context, id string) (*CBEngineBenchmarkResponse, error) {
	endpoint := fmt.Sprintf("/v2/benchmarks/%s", url.PathEscape(id))

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmark %s: %w", id, err)
	}

	var result CBEngineBenchmarkResponse
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteCBEngineBenchmark removes a benchmark by ID
func (c *Client) DeleteCBEngineBenchmark(ctx context.Context, id string) error {
	endpoint := fmt.Sprintf("/v1/benchmarks/%s", url.PathEscape(id))

	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete benchmark %s: %w", id, err)
	}

	if err := c.handleAPIResponse(resp, 204, nil); err != nil {
		return err
	}

	const pollInterval = 5
	const maxRetries = 24
	retries := 0
	for retries < maxRetries {
		time.Sleep(pollInterval * time.Second)

		benchmarks, err := c.GetCBEngineBenchmarks(ctx)
		if err != nil {
			return fmt.Errorf("failed to poll benchmarks after delete: %w", err)
		}

		found := false
		for _, b := range benchmarks.Benchmarks {
			if b.ID == id {
				found = true
				if b.SyncState == "DELETING" {
					_ = c.DeleteCBEngineBenchmark(ctx, id)
					retries++
					break
				} else if b.SyncState == "DELETE_FAILED" {
					return fmt.Errorf("benchmark %s deletion failed: syncState=DELETE_FAILED", id)
				} else {
					return fmt.Errorf("benchmark %s still present after delete, syncState=%s", id, b.SyncState)
				}
			}
		}
		if !found {
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for benchmark deletion to complete after %d retries", maxRetries)
}

// GetCBEngineBenchmarkByTitle retrieves a specific benchmark by title
func (c *Client) GetCBEngineBenchmarkByTitle(ctx context.Context, title string) (*CBEngineBenchmarkResponse, error) {
	benchmarks, err := c.GetCBEngineBenchmarks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmarks list: %w", err)
	}

	for _, benchmark := range benchmarks.Benchmarks {
		if benchmark.Title == title {
			return c.GetCBEngineBenchmarkByID(ctx, benchmark.ID)
		}
	}

	return nil, fmt.Errorf("benchmark with title '%s' not found", title)
}

// CBEngine Rule operations

// GetCBEngineRules returns list of rules for given baseline
func (c *Client) GetCBEngineRules(ctx context.Context, baselineID string) (*CBEngineSourcedRules, error) {
	endpoint := fmt.Sprintf("/v1/rules?baselineId=%s", url.QueryEscape(baselineID))

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules for baseline %s: %w", baselineID, err)
	}

	var result CBEngineSourcedRules
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
