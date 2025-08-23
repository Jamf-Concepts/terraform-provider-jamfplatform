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

// BlueprintComponentDescription describes a component within a blueprint
type BlueprintComponentDescription struct {
	Identifier  string        `json:"identifier"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Meta        BlueprintMeta `json:"meta"`
}

// BlueprintMeta describes metadata about a blueprint
type BlueprintMeta struct {
	SupportedOs map[string][]BlueprintSupportedOs `json:"supportedOs"`
}

// BlueprintSupportedOs describes a supported operating system for a blueprint
type BlueprintSupportedOs struct {
	Version string `json:"version"`
}

// BlueprintComponentDescriptionPagedResponse describes a paged response for blueprint components
type BlueprintComponentDescriptionPagedResponse struct {
	Results    []BlueprintComponentDescription `json:"results"`
	TotalCount int64                           `json:"totalCount"`
}

// BlueprintComponent describes a component within a blueprint
type BlueprintComponent struct {
	Identifier    string          `json:"identifier"`
	Configuration json.RawMessage `json:"configuration,omitempty"`
}

// BlueprintStep describes a step within a blueprint
type BlueprintStep struct {
	Name       string               `json:"name"`
	Components []BlueprintComponent `json:"components,omitempty"`
}

// BlueprintCreateScope defines the scope for creating a blueprint
type BlueprintCreateScope struct {
	DeviceGroups []string `json:"deviceGroups"`
}

// BlueprintCreateRequest represents a request to create or update a blueprint
type BlueprintCreateRequest struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Scope       BlueprintCreateScope `json:"scope"`
	Steps       []BlueprintStep      `json:"steps,omitempty"`
}

// BlueprintUpdateRequest represents a request to update an existing blueprint
type BlueprintUpdateRequest struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Scope       BlueprintUpdateScope `json:"scope"`
	Steps       []BlueprintStep      `json:"steps,omitempty"`
}

// BlueprintDeployment describes the deployment status of a blueprint
type BlueprintDeployment struct {
	Started string `json:"started"`
	State   string `json:"state"`
}

// BlueprintDeploymentState describes the state of a blueprint deployment
type BlueprintDeploymentState struct {
	State          string               `json:"state"`
	LastDeployment *BlueprintDeployment `json:"lastDeployment"`
}

// BlueprintDetail describes the details of a blueprint
type BlueprintDetail struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description,omitempty"`
	Scope           BlueprintUpdateScope     `json:"scope,omitempty"`
	Created         string                   `json:"created"`
	Updated         string                   `json:"updated"`
	DeploymentState BlueprintDeploymentState `json:"deploymentState"`
	Steps           []BlueprintStep          `json:"steps"`
}

// BlueprintOverview describes a summary of a blueprint
type BlueprintOverview struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description,omitempty"`
	Created         string                   `json:"created"`
	Updated         string                   `json:"updated"`
	DeploymentState BlueprintDeploymentState `json:"deploymentState"`
}

// BlueprintUpdateScope defines the scope for updating a blueprint
type BlueprintUpdateScope struct {
	DeviceGroups []string `json:"deviceGroups"`
}

// BlueprintOverviewPagedResponse describes a paged response for blueprint overviews
type BlueprintOverviewPagedResponse struct {
	Results    []BlueprintOverview `json:"results"`
	TotalCount int64               `json:"totalCount"`
}

// BlueprintCreateResponse represents the response for creating a blueprint
type BlueprintCreateResponse struct {
	ID   string `json:"id"`
	Href string `json:"href"`
}

// Blueprint API path constants
const (
	blueprintV1Prefix           = "/blueprints/api/v1/blueprints"
	blueprintComponentsV1Prefix = "/blueprints/api/v1/blueprint-components"
)

// GetBlueprints returns all blueprints, automatically handling pagination
func (c *Client) GetBlueprints(ctx context.Context, sort []string, search string) ([]BlueprintOverview, error) {
	var allResults []BlueprintOverview
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
		var result BlueprintOverviewPagedResponse
		if err := c.handleAPIResponse(resp, 200, &result); err != nil {
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

// GetBlueprintByID retrieves a blueprint by ID
func (c *Client) GetBlueprintByID(ctx context.Context, blueprintID string) (*BlueprintDetail, error) {
	endpoint := fmt.Sprintf("%s/%s", blueprintV1Prefix, url.PathEscape(blueprintID))
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get blueprint %s: %w", blueprintID, err)
	}
	var result BlueprintDetail
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetBlueprintByName finds a blueprint by exact name and returns its details
func (c *Client) GetBlueprintByName(ctx context.Context, name string) (*BlueprintDetail, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	blueprints, err := c.GetBlueprints(ctx, nil, name)
	if err != nil {
		return nil, fmt.Errorf("error searching for blueprint by name: %w", err)
	}
	for _, bp := range blueprints {
		if bp.Name == name {
			return c.GetBlueprintByID(ctx, bp.ID)
		}
	}
	return nil, fmt.Errorf("blueprint with name '%s' not found", name)
}

// CreateBlueprint creates a new blueprint
func (c *Client) CreateBlueprint(ctx context.Context, request *BlueprintCreateRequest) (*BlueprintCreateResponse, error) {
	endpoint := blueprintV1Prefix
	resp, err := c.makeRequest(ctx, "POST", endpoint, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create blueprint: %w", err)
	}
	var result BlueprintCreateResponse
	if err := c.handleAPIResponse(resp, 201, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateBlueprint updates a blueprint configuration
func (c *Client) UpdateBlueprint(ctx context.Context, blueprintID string, request *BlueprintUpdateRequest) error {
	endpoint := fmt.Sprintf("%s/%s", blueprintV1Prefix, url.PathEscape(blueprintID))
	resp, err := c.makeRequest(ctx, "PATCH", endpoint, request)
	if err != nil {
		return fmt.Errorf("failed to update blueprint %s: %w", blueprintID, err)
	}
	if err := c.handleAPIResponse(resp, 204, nil); err != nil {
		return err
	}
	return nil
}

// DeleteBlueprint deletes a blueprint by ID
func (c *Client) DeleteBlueprint(ctx context.Context, blueprintID string) error {
	endpoint := fmt.Sprintf("%s/%s", blueprintV1Prefix, url.PathEscape(blueprintID))
	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete blueprint %s: %w", blueprintID, err)
	}
	if err := c.handleAPIResponse(resp, 204, nil); err != nil {
		return err
	}
	return nil
}

// DeployBlueprint starts deployment of a blueprint
func (c *Client) DeployBlueprint(ctx context.Context, blueprintID string) error {
	endpoint := fmt.Sprintf("%s/%s/deploy", blueprintV1Prefix, url.PathEscape(blueprintID))
	resp, err := c.makeRequest(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to deploy blueprint %s: %w", blueprintID, err)
	}
	if err := c.handleAPIResponse(resp, 202, nil); err != nil {
		return err
	}
	return nil
}

// GetBlueprintComponents returns all blueprint components, automatically handling pagination
func (c *Client) GetBlueprintComponents(ctx context.Context) ([]BlueprintComponentDescription, error) {
	var allResults []BlueprintComponentDescription
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
		var result BlueprintComponentDescriptionPagedResponse
		if err := c.handleAPIResponse(resp, 200, &result); err != nil {
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

// GetBlueprintComponentByID gets a blueprint component by identifier
func (c *Client) GetBlueprintComponentByID(ctx context.Context, identifier string) (*BlueprintComponentDescription, error) {
	endpoint := fmt.Sprintf("%s/%s", blueprintComponentsV1Prefix, url.PathEscape(identifier))
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get blueprint component %s: %w", identifier, err)
	}
	var result BlueprintComponentDescription
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
