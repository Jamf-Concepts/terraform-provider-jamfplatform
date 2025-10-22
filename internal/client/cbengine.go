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

// CBEngineBaselinesResponseV1 represents the response for baselines listing
type CBEngineBaselinesResponseV1 struct {
	Baselines []CBEngineBaselineInfoV1 `json:"baselines,omitempty"`
}

// CBEngineBaselineInfoV1 represents information about a baseline
type CBEngineBaselineInfoV1 struct {
	ID          string `json:"id,omitempty"`
	BaselineID  string `json:"baselineId,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Title       string `json:"title,omitempty"`
	RuleCount   int64  `json:"ruleCount,omitempty"`
}

// CBEngineSourceV1 represents source information
type CBEngineSourceV1 struct {
	Branch   string `json:"branch" validate:"required"`
	Revision string `json:"revision" validate:"required"`
}

// CBEngineTargetV2 represents the target configuration
type CBEngineTargetV2 struct {
	DeviceGroups []string `json:"deviceGroups" validate:"required"`
}

// CBEngine Benchmark Types

// CBEngineBenchmarkRequestV2 represents the request body for creating/updating benchmarks
type CBEngineBenchmarkRequestV2 struct {
	Title            string                  `json:"title" validate:"required,max=100"`
	Description      string                  `json:"description,omitempty" validate:"max=1000"`
	SourceBaselineID string                  `json:"sourceBaselineId" validate:"required"`
	Sources          []CBEngineSourceV1      `json:"sources" validate:"required"`
	Rules            []CBEngineRuleRequestV2 `json:"rules" validate:"required"`
	Target           CBEngineTargetV2        `json:"target" validate:"required"`
	EnforcementMode  string                  `json:"enforcementMode" validate:"required,oneof=MONITOR MONITOR_AND_ENFORCE"`
}

// CBEngineBenchmarkResponseV2 represents the response for benchmark operations
type CBEngineBenchmarkResponseV2 struct {
	BenchmarkID     string               `json:"benchmarkId"`
	TenantID        string               `json:"tenantId"`
	Title           string               `json:"title"`
	Description     string               `json:"description,omitempty"`
	Sources         []CBEngineSourceV1   `json:"sources"`
	Rules           []CBEngineRuleInfoV1 `json:"rules"`
	Target          CBEngineTargetV2     `json:"target"`
	EnforcementMode string               `json:"enforcementMode"`
	Deleted         bool                 `json:"deleted"`
	UpdateAvailable bool                 `json:"updateAvailable"`
	LastUpdatedAt   time.Time            `json:"lastUpdatedAt"`
}

// CBEngineBenchmarksResponseV2 represents the response for listing benchmarks
type CBEngineBenchmarksResponseV2 struct {
	Benchmarks []CBEngineBenchmarkV2 `json:"benchmarks"`
}

// CBEngineBenchmarkV2 represents a benchmark in the list response
type CBEngineBenchmarkV2 struct {
	ID              string           `json:"id"`
	Title           string           `json:"title"`
	Description     string           `json:"description,omitempty"`
	UpdateAvailable bool             `json:"updateAvailable"`
	Target          CBEngineTargetV2 `json:"target"`
	SyncState       string           `json:"syncState"`
}

// CBEngine Rule Types

// CBEngineRuleRequestV2 represents a rule in the request
type CBEngineRuleRequestV2 struct {
	ID      string                `json:"id" validate:"required"`
	Enabled bool                  `json:"enabled" validate:"required"`
	ODV     *CBEngineODVRequestV2 `json:"odv,omitempty"`
}

// CBEngineODVRequestV2 represents an organization-defined value in requests
type CBEngineODVRequestV2 struct {
	Value string `json:"value" validate:"required"`
}

// CBEngineRuleInfoV1 represents detailed rule information in responses
type CBEngineRuleInfoV1 struct {
	ID                 string                                  `json:"id"`
	SectionName        string                                  `json:"sectionName"`
	Enabled            bool                                    `json:"enabled"`
	Title              string                                  `json:"title"`
	References         []string                                `json:"references,omitempty"`
	Description        string                                  `json:"description"`
	ODV                *CBEngineOrganizationDefinedValueV1     `json:"odv,omitempty"`
	SupportedOS        []CBEngineOSInfoV1                      `json:"supportedOs"`
	OSSpecificDefaults map[string]CBEngineOSSpecificRuleInfoV1 `json:"osSpecificDefaults"`
	RuleRelation       *CBEngineRuleRelationV1                 `json:"ruleRelation,omitempty"`
}

// CBEngineOrganizationDefinedValueV1 represents ODV with full details
type CBEngineOrganizationDefinedValueV1 struct {
	Value       string                           `json:"value" validate:"required"`
	Hint        string                           `json:"hint,omitempty"`
	Placeholder string                           `json:"placeholder,omitempty"`
	Type        string                           `json:"type,omitempty"`
	Validation  *CBEngineValidationConstraintsV1 `json:"validation,omitempty"`
}

// CBEngineValidationConstraintsV1 represents validation rules for ODV
type CBEngineValidationConstraintsV1 struct {
	Min        *int     `json:"min,omitempty"`
	Max        *int     `json:"max,omitempty"`
	EnumValues []string `json:"enumValues,omitempty"`
	Regex      string   `json:"regex,omitempty"`
}

// CBEngineOSInfoV1 represents operating system information
type CBEngineOSInfoV1 struct {
	OSType         string `json:"osType"`
	OSVersion      int    `json:"osVersion"`
	ManagementType string `json:"managementType"`
}

// CBEngineOSSpecificRuleInfoV1 represents OS-specific rule details
type CBEngineOSSpecificRuleInfoV1 struct {
	Title       string                       `json:"title"`
	Description string                       `json:"description"`
	ODV         *CBEngineODVRecommendationV1 `json:"odv,omitempty"`
}

// CBEngineODVRecommendationV1 represents ODV recommendation
type CBEngineODVRecommendationV1 struct {
	Value string `json:"value,omitempty"`
	Hint  string `json:"hint,omitempty"`
}

// CBEngineRuleRelationV1 represents rule dependencies
type CBEngineRuleRelationV1 struct {
	DependsOn []string `json:"dependsOn,omitempty"`
}

// CBEngineSourcedRulesV1 represents rules with their sources
type CBEngineSourcedRulesV1 struct {
	Sources []CBEngineSourceV1   `json:"sources"`
	Rules   []CBEngineRuleInfoV1 `json:"rules"`
}

// CBEngine API path constants
const (
	cbEngineV1Prefix = "/api/cb/engine/v1"
	cbEngineV2Prefix = "/api/cb/engine/v2"
)

// CBEngine Baseline operations

// GetCBEngineBaselinesV1 returns list of available mSCP baselines
func (c *Client) GetCBEngineBaselinesV1(ctx context.Context) (*CBEngineBaselinesResponseV1, error) {
	resp, err := c.makeRequest(ctx, "GET", cbEngineV1Prefix+"/baselines", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get baselines: %w", err)
	}

	var result CBEngineBaselinesResponseV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CBEngine Benchmark operations

// CreateCBEngineBenchmarkV2 creates a new benchmark
func (c *Client) CreateCBEngineBenchmarkV2(ctx context.Context, request *CBEngineBenchmarkRequestV2) (*CBEngineBenchmarkResponseV2, error) {
	resp, err := c.makeRequest(ctx, "POST", cbEngineV2Prefix+"/benchmarks", request)
	if err != nil {
		return nil, fmt.Errorf("failed to create benchmark: %w", err)
	}

	var result CBEngineBenchmarkResponseV2
	if err := c.handleAPIResponse(ctx, resp, 202, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetCBEngineBenchmarksV2 retrieves all benchmarks for the tenant
func (c *Client) GetCBEngineBenchmarksV2(ctx context.Context) (*CBEngineBenchmarksResponseV2, error) {
	resp, err := c.makeRequest(ctx, "GET", cbEngineV2Prefix+"/benchmarks", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmarks: %w", err)
	}

	var result CBEngineBenchmarksResponseV2
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetCBEngineBenchmarkByIDV2 retrieves a specific benchmark by ID
func (c *Client) GetCBEngineBenchmarkByIDV2(ctx context.Context, id string) (*CBEngineBenchmarkResponseV2, error) {
	endpoint := fmt.Sprintf("%s/benchmarks/%s", cbEngineV2Prefix, url.PathEscape(id))

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmark %s: %w", id, err)
	}

	var result CBEngineBenchmarkResponseV2
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteCBEngineBenchmarkV1 removes a benchmark by ID
func (c *Client) DeleteCBEngineBenchmarkV1(ctx context.Context, id string) error {
	endpoint := fmt.Sprintf("%s/benchmarks/%s", cbEngineV1Prefix, url.PathEscape(id))

	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete benchmark %s: %w", id, err)
	}

	if err := c.handleAPIResponse(ctx, resp, 204, nil); err != nil {
		return err
	}

	return nil
}

// GetCBEngineBenchmarkByTitleV2 retrieves a specific benchmark by title
func (c *Client) GetCBEngineBenchmarkByTitleV2(ctx context.Context, title string) (*CBEngineBenchmarkResponseV2, error) {
	benchmarks, err := c.GetCBEngineBenchmarksV2(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmarks list: %w", err)
	}

	for _, benchmark := range benchmarks.Benchmarks {
		if benchmark.Title == title {
			return c.GetCBEngineBenchmarkByIDV2(ctx, benchmark.ID)
		}
	}

	return nil, fmt.Errorf("benchmark with title '%s' not found", title)
}

// CBEngine Rule operations

// GetCBEngineRulesV1 returns list of rules for given baseline
func (c *Client) GetCBEngineRulesV1(ctx context.Context, baselineID string) (*CBEngineSourcedRulesV1, error) {
	endpoint := fmt.Sprintf("%s/rules?baselineId=%s", cbEngineV1Prefix, url.QueryEscape(baselineID))

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules for baseline %s: %w", baselineID, err)
	}

	var result CBEngineSourcedRulesV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
