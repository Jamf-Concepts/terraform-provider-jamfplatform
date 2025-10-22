// Copyright 2025 Jamf Software LLC.
// Blueprint API client
// https://developer.jamf.com/platform-api/reference/blueprints

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Blueprint API Types

// BlueprintComponentDescriptionV1 describes a component within a blueprint
type BlueprintComponentDescriptionV1 struct {
	Identifier  string          `json:"identifier"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Meta        BlueprintMetaV1 `json:"meta"`
}

// BlueprintMetaV1 describes metadata about a blueprint
type BlueprintMetaV1 struct {
	SupportedOs map[string][]BlueprintSupportedOsV1 `json:"supportedOs"`
}

// BlueprintSupportedOsV1 describes a supported operating system for a blueprint
type BlueprintSupportedOsV1 struct {
	Version string `json:"version"`
}

// BlueprintComponentDescriptionPagedResponseV1 describes a paged response for blueprint components
type BlueprintComponentDescriptionPagedResponseV1 struct {
	Results    []BlueprintComponentDescriptionV1 `json:"results"`
	TotalCount int64                             `json:"totalCount"`
}

// BlueprintComponentV1 describes a component within a blueprint
type BlueprintComponentV1 struct {
	Identifier    string          `json:"identifier"`
	Configuration json.RawMessage `json:"configuration,omitempty"`
}

// BlueprintStepV1 describes a step within a blueprint
type BlueprintStepV1 struct {
	Name       string                 `json:"name"`
	Components []BlueprintComponentV1 `json:"components,omitempty"`
}

// BlueprintCreateScopeV1 defines the scope for creating a blueprint
type BlueprintCreateScopeV1 struct {
	DeviceGroups []string `json:"deviceGroups"`
}

// BlueprintCreateRequestV1 represents a request to create or update a blueprint
type BlueprintCreateRequestV1 struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Scope       BlueprintCreateScopeV1 `json:"scope"`
	Steps       []BlueprintStepV1      `json:"steps,omitempty"`
}

// BlueprintUpdateRequestV1 represents a request to update an existing blueprint
type BlueprintUpdateRequestV1 struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Scope       BlueprintUpdateScopeV1 `json:"scope"`
	Steps       []BlueprintStepV1      `json:"steps,omitempty"`
}

// BlueprintDeploymentV1 describes the deployment status of a blueprint
type BlueprintDeploymentV1 struct {
	Started string `json:"started"`
	State   string `json:"state"`
}

// BlueprintDeploymentStateV1 describes the state of a blueprint deployment
type BlueprintDeploymentStateV1 struct {
	State          string                 `json:"state"`
	LastDeployment *BlueprintDeploymentV1 `json:"lastDeployment"`
}

// BlueprintDetailV1 describes the details of a blueprint
type BlueprintDetailV1 struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Description     string                     `json:"description,omitempty"`
	Scope           BlueprintUpdateScopeV1     `json:"scope,omitempty"`
	Created         string                     `json:"created"`
	Updated         string                     `json:"updated"`
	DeploymentState BlueprintDeploymentStateV1 `json:"deploymentState"`
	Steps           []BlueprintStepV1          `json:"steps"`
}

// BlueprintOverviewV1 describes a summary of a blueprint
type BlueprintOverviewV1 struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Description     string                     `json:"description,omitempty"`
	Created         string                     `json:"created"`
	Updated         string                     `json:"updated"`
	DeploymentState BlueprintDeploymentStateV1 `json:"deploymentState"`
}

// BlueprintUpdateScope defines the scope for updating a blueprint
type BlueprintUpdateScopeV1 struct {
	DeviceGroups []string `json:"deviceGroups"`
}

// BlueprintOverviewPagedResponseV1 describes a paged response for blueprint overviews
type BlueprintOverviewPagedResponseV1 struct {
	Results    []BlueprintOverviewV1 `json:"results"`
	TotalCount int64                 `json:"totalCount"`
}

// BlueprintCreateResponseV1 represents the response for creating a blueprint
type BlueprintCreateResponseV1 struct {
	ID   string `json:"id"`
	Href string `json:"href"`
}

// Blueprint API path constants
const (
	blueprintV1Prefix           = "/api/blueprints/v1/blueprints"
	blueprintComponentsV1Prefix = "/api/blueprints/v1/blueprint-components"
)

// GetBlueprintsV1 returns all blueprints, automatically handling pagination
func (c *Client) GetBlueprintsV1(ctx context.Context, sort []string, search string) ([]BlueprintOverviewV1, error) {
	var allResults []BlueprintOverviewV1
	page := 0
	for {
		params := url.Values{}
		if len(sort) > 0 {
			params.Set("sort", strings.Join(sort, ","))
		}
		params.Set("page", fmt.Sprintf("%d", page))
		if search != "" {
			params.Set("search", search)
		}
		endpoint := blueprintV1Prefix
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
		resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list blueprints: %w", err)
		}
		var result BlueprintOverviewPagedResponseV1
		if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
			return nil, err
		}
		allResults = append(allResults, result.Results...)
		if len(result.Results) < 100 || len(result.Results) == 0 {
			break
		}
		page++
	}
	return allResults, nil
}

// GetBlueprintByIDV1 retrieves a blueprint by ID
func (c *Client) GetBlueprintByIDV1(ctx context.Context, blueprintID string) (*BlueprintDetailV1, error) {
	endpoint := fmt.Sprintf("%s/%s", blueprintV1Prefix, url.PathEscape(blueprintID))
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get blueprint %s: %w", blueprintID, err)
	}
	var result BlueprintDetailV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetBlueprintByNameV1 finds a blueprint by exact name and returns its details
func (c *Client) GetBlueprintByNameV1(ctx context.Context, name string) (*BlueprintDetailV1, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	blueprints, err := c.GetBlueprintsV1(ctx, nil, name)
	if err != nil {
		return nil, fmt.Errorf("error searching for blueprint by name: %w", err)
	}
	for _, bp := range blueprints {
		if bp.Name == name {
			return c.GetBlueprintByIDV1(ctx, bp.ID)
		}
	}
	return nil, fmt.Errorf("blueprint with name '%s' not found", name)
}

// CreateBlueprintV1 creates a new blueprint
func (c *Client) CreateBlueprintV1(ctx context.Context, request *BlueprintCreateRequestV1) (*BlueprintCreateResponseV1, error) {
	endpoint := blueprintV1Prefix
	resp, err := c.makeRequest(ctx, "POST", endpoint, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create blueprint: %w", err)
	}
	var result BlueprintCreateResponseV1
	if err := c.handleAPIResponse(ctx, resp, 201, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateBlueprintV1 updates a blueprint configuration
func (c *Client) UpdateBlueprintV1(ctx context.Context, blueprintID string, request *BlueprintUpdateRequestV1) error {
	endpoint := fmt.Sprintf("%s/%s", blueprintV1Prefix, url.PathEscape(blueprintID))
	resp, err := c.makeRequest(ctx, "PATCH", endpoint, request)
	if err != nil {
		return fmt.Errorf("failed to update blueprint %s: %w", blueprintID, err)
	}
	if err := c.handleAPIResponse(ctx, resp, 204, nil); err != nil {
		return err
	}
	return nil
}

// DeleteBlueprintV1 deletes a blueprint by ID
func (c *Client) DeleteBlueprintV1(ctx context.Context, blueprintID string) error {
	endpoint := fmt.Sprintf("%s/%s", blueprintV1Prefix, url.PathEscape(blueprintID))
	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete blueprint %s: %w", blueprintID, err)
	}
	if err := c.handleAPIResponse(ctx, resp, 204, nil); err != nil {
		return err
	}
	return nil
}

// DeployBlueprintV1 starts deployment of a blueprint
func (c *Client) DeployBlueprintV1(ctx context.Context, blueprintID string) error {
	endpoint := fmt.Sprintf("%s/%s/deploy", blueprintV1Prefix, url.PathEscape(blueprintID))
	resp, err := c.makeRequest(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to deploy blueprint %s: %w", blueprintID, err)
	}
	if err := c.handleAPIResponse(ctx, resp, 202, nil); err != nil {
		return err
	}
	return nil
}

// GetBlueprintComponentsV1 returns all blueprint components, automatically handling pagination
func (c *Client) GetBlueprintComponentsV1(ctx context.Context) ([]BlueprintComponentDescriptionV1, error) {
	var allResults []BlueprintComponentDescriptionV1
	page := 0
	for {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		endpoint := blueprintComponentsV1Prefix
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
		resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list blueprint components: %w", err)
		}
		var result BlueprintComponentDescriptionPagedResponseV1
		if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
			return nil, err
		}
		allResults = append(allResults, result.Results...)
		if len(result.Results) < 100 || len(result.Results) == 0 {
			break
		}
		page++
	}
	return allResults, nil
}

// GetBlueprintComponentByIDV1 gets a blueprint component by identifier
func (c *Client) GetBlueprintComponentByIDV1(ctx context.Context, identifier string) (*BlueprintComponentDescriptionV1, error) {
	endpoint := fmt.Sprintf("%s/%s", blueprintComponentsV1Prefix, url.PathEscape(identifier))
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get blueprint component %s: %w", identifier, err)
	}
	var result BlueprintComponentDescriptionV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
