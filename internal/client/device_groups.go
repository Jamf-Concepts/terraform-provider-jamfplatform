// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// https://developer.jamf.com/platform-api/reference/device-groups

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Device Groups API path constants
const (
	deviceGroupsV1Prefix = "/management/device-groups/v1"
)

// Device Groups Types

// DeviceGroupListReadRepresentationV1 represents a device group in a list response
type DeviceGroupListReadRepresentationV1 struct {
	ID          string                                `json:"id"`
	Name        string                                `json:"name"`
	Description string                                `json:"description,omitempty"`
	DeviceType  string                                `json:"deviceType"`
	GroupType   string                                `json:"groupType"`
	MemberCount int                                   `json:"memberCount"`
	Criteria    []DeviceGroupCriteriaRepresentationV1 `json:"criteria,omitempty"`
}

// DeviceGroupCriteriaRepresentationV1 represents a criterion for device groups
type DeviceGroupCriteriaRepresentationV1 struct {
	Order                 int    `json:"order"`
	AttributeName         string `json:"attributeName"`
	Operator              string `json:"operator"`
	AttributeValue        string `json:"attributeValue"`
	JoinType              string `json:"joinType"`
	HasOpeningParenthesis bool   `json:"hasOpeningParenthesis,omitempty"`
	HasClosingParenthesis bool   `json:"hasClosingParenthesis,omitempty"`
}

// DeviceGroupReadRepresentationV1 represents a device group in a single-read response.
type DeviceGroupReadRepresentationV1 struct {
	ID          string                                `json:"id"`
	Name        string                                `json:"name"`
	Description string                                `json:"description,omitempty"`
	DeviceType  string                                `json:"deviceType"`
	GroupType   string                                `json:"groupType"`
	MemberCount int                                   `json:"memberCount"`
	Criteria    []DeviceGroupCriteriaRepresentationV1 `json:"criteria,omitempty"`
}

// DeviceGroupCreateRepresentationV1 represents the payload to create a device group
type DeviceGroupCreateRepresentationV1 struct {
	Name        string                                `json:"name"`
	Description *string                               `json:"description,omitempty"`
	DeviceType  string                                `json:"deviceType"`
	GroupType   string                                `json:"groupType"`
	Criteria    []DeviceGroupCriteriaRepresentationV1 `json:"criteria,omitempty"`
	Members     []string                              `json:"members,omitempty"`
}

// DeviceGroupCreateResponseV1 represents the response after creating a device group
type DeviceGroupCreateResponseV1 struct {
	ID   string `json:"id"`
	Href string `json:"href"`
}

// DeviceGroupUpdateRepresentationV1 represents the payload to update a device group
type DeviceGroupUpdateRepresentationV1 struct {
	Name        string                                `json:"name"`
	Description *string                               `json:"description,omitempty"`
	Criteria    []DeviceGroupCriteriaRepresentationV1 `json:"criteria,omitempty"`
	DeviceIds   []string                              `json:"deviceIds,omitempty"`
}

// DeviceGroupMemberPatchRepresentationV1 represents the payload to patch device group members
type DeviceGroupMemberPatchRepresentationV1 struct {
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
}

// DeviceGroupMemberV1 represents a device group member
type DeviceGroupMemberV1 struct {
	DeviceID string `json:"deviceId"`
}

// ListDeviceGroupMemberReadRepresentationV1 represents the paginated response for device group members
type ListDeviceGroupMemberReadRepresentationV1 struct {
	Results     []string `json:"results"`
	TotalCount  int      `json:"totalCount"`
	Page        int      `json:"page"`
	PageSize    int      `json:"pageSize"`
	TotalPages  int      `json:"totalPages"`
	HasNext     bool     `json:"hasNext"`
	HasPrevious bool     `json:"hasPrevious"`
}

// DeviceGroupMemberOfRepresentationV1 represents a device group that a device belongs to
type DeviceGroupMemberOfRepresentationV1 struct {
	GroupID   string `json:"groupId"`
	GroupName string `json:"groupName"`
}

// ListDeviceGroupMemberOfResponseRepresentationV1 represents the paginated response for device groups a device belongs to
type ListDeviceGroupMemberOfResponseRepresentationV1 struct {
	Results     []DeviceGroupMemberOfRepresentationV1 `json:"results"`
	TotalCount  int                                   `json:"totalCount"`
	Page        int                                   `json:"page"`
	PageSize    int                                   `json:"pageSize"`
	TotalPages  int                                   `json:"totalPages"`
	HasNext     bool                                  `json:"hasNext"`
	HasPrevious bool                                  `json:"hasPrevious"`
}

// DeviceGroupPagedResponseV1 represents a paginated response for device groups.
type DeviceGroupPagedResponseV1 struct {
	PaginatedResponseRepresentation
	Results []DeviceGroupListReadRepresentationV1 `json:"results"`
}

// GetDeviceGroupsV1 returns all device groups, automatically handling pagination
func (c *Client) GetDeviceGroupsV1(ctx context.Context, sort []string, filter string) ([]DeviceGroupListReadRepresentationV1, error) {
	var allResults []DeviceGroupListReadRepresentationV1
	page := 0
	pageSize := 100
	for {
		params := url.Values{}
		if len(sort) > 0 {
			params.Set("sort", strings.Join(sort, ","))
		}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("page-size", fmt.Sprintf("%d", pageSize))
		if filter != "" {
			params.Set("filter", filter)
		}

		endpoint := deviceGroupsV1Prefix + "/device-groups"
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}

		resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list device groups: %w", err)
		}

		var result DeviceGroupPagedResponseV1
		if err := c.handleAPIResponse(ctx, resp, http.StatusOK, &result); err != nil {
			return nil, err
		}

		allResults = append(allResults, result.Results...)
		if !result.HasNext {
			break
		}
		page++
	}
	return allResults, nil
}

// GetDeviceGroupByIDV1 retrieves a device group by ID
func (c *Client) GetDeviceGroupByIDV1(ctx context.Context, id string) (*DeviceGroupReadRepresentationV1, error) {
	endpoint := fmt.Sprintf("%s/%s", deviceGroupsV1Prefix+"/device-groups", url.PathEscape(id))
	resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get device group %s: %w", id, err)
	}
	var result DeviceGroupReadRepresentationV1
	if err := c.handleAPIResponse(ctx, resp, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateDeviceGroupV1 creates a new device group
func (c *Client) CreateDeviceGroupV1(ctx context.Context, request *DeviceGroupCreateRepresentationV1) (*DeviceGroupCreateResponseV1, error) {
	endpoint := deviceGroupsV1Prefix + "/device-groups"
	resp, err := c.makeRequest(ctx, http.MethodPost, endpoint, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create device group: %w", err)
	}
	var result DeviceGroupCreateResponseV1
	if err := c.handleAPIResponse(ctx, resp, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateDeviceGroupV1 updates a device group
func (c *Client) UpdateDeviceGroupV1(ctx context.Context, id string, request *DeviceGroupUpdateRepresentationV1) error {
	endpoint := fmt.Sprintf("%s/device-groups/%s", deviceGroupsV1Prefix, url.PathEscape(id))
	resp, err := c.doRequest(ctx, http.MethodPatch, endpoint, request, "application/json")
	if err != nil {
		return fmt.Errorf("failed to update device group %s: %w", id, err)
	}
	if err := c.handleAPIResponse(ctx, resp, http.StatusNoContent, nil); err != nil {
		return err
	}
	return nil
}

// DeleteDeviceGroupV1 deletes a device group by ID
func (c *Client) DeleteDeviceGroupV1(ctx context.Context, id string) error {
	endpoint := fmt.Sprintf("%s/device-groups/%s", deviceGroupsV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete device group %s: %w", id, err)
	}
	if err := c.handleAPIResponse(ctx, resp, http.StatusNoContent, nil); err != nil {
		return err
	}
	return nil
}

// GetDeviceGroupMembersV1 returns all member IDs for a device group, handling pagination internally.
func (c *Client) GetDeviceGroupMembersV1(ctx context.Context, id string) ([]string, error) {
	var allMembers []string
	page := 0
	pageSize := 100

	for {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("page-size", fmt.Sprintf("%d", pageSize))

		endpoint := fmt.Sprintf("%s/device-groups/%s/members?%s", deviceGroupsV1Prefix, url.PathEscape(id), params.Encode())
		resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get members for device group %s: %w", id, err)
		}

		var result ListDeviceGroupMemberReadRepresentationV1
		if err := c.handleAPIResponse(ctx, resp, http.StatusOK, &result); err != nil {
			return nil, err
		}

		if len(result.Results) == 0 {
			break
		}

		allMembers = append(allMembers, result.Results...)
		if len(result.Results) < pageSize || !result.HasNext {
			break
		}
		page++
	}

	return allMembers, nil
}

// UpdateDeviceGroupMembersV1 patches the members of a static device group
func (c *Client) UpdateDeviceGroupMembersV1(ctx context.Context, id string, patch *DeviceGroupMemberPatchRepresentationV1) error {
	endpoint := fmt.Sprintf("%s/device-groups/%s/members", deviceGroupsV1Prefix, url.PathEscape(id))
	resp, err := c.doRequest(ctx, http.MethodPatch, endpoint, patch, "application/json")
	if err != nil {
		return fmt.Errorf("failed to update members for device group %s: %w", id, err)
	}
	if err := c.handleAPIResponse(ctx, resp, http.StatusNoContent, nil); err != nil {
		return err
	}
	return nil
}

// GetDeviceGroupsForDeviceV1 returns all device groups a device belongs to, handling pagination internally.
func (c *Client) GetDeviceGroupsForDeviceV1(ctx context.Context, deviceID string) ([]DeviceGroupMemberOfRepresentationV1, error) {
	var allGroups []DeviceGroupMemberOfRepresentationV1
	page := 0
	pageSize := 100

	for {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("page-size", fmt.Sprintf("%d", pageSize))

		endpoint := fmt.Sprintf("%s/devices/%s/device-groups?%s", deviceGroupsV1Prefix, url.PathEscape(deviceID), params.Encode())
		resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get device groups for device %s: %w", deviceID, err)
		}

		var result ListDeviceGroupMemberOfResponseRepresentationV1
		if err := c.handleAPIResponse(ctx, resp, http.StatusOK, &result); err != nil {
			return nil, err
		}

		if len(result.Results) == 0 {
			break
		}

		allGroups = append(allGroups, result.Results...)
		if len(result.Results) < pageSize || !result.HasNext {
			break
		}
		page++
	}

	return allGroups, nil
}
