// Copyright 2025 Jamf Software LLC.
// https://developer.jamf.com/platform-api/reference/device-groups

package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Device Groups API path constants
const (
	deviceGroupsV1Prefix = "/api/device-groups/v1"
)

// Device group types
type DeviceGroupListReadRepresentationV1 struct {
	ID          string                                `json:"id"`
	Name        string                                `json:"name"`
	Description string                                `json:"description,omitempty"`
	DeviceType  string                                `json:"deviceType"`
	GroupType   string                                `json:"groupType"`
	MemberCount int                                   `json:"memberCount"`
	Criteria    []DeviceGroupCriteriaRepresentationV1 `json:"criteria,omitempty"`
}

type DeviceGroupCriteriaRepresentationV1 struct {
	Order                 int    `json:"order" validate:"required"`
	AttributeName         string `json:"attributeName" validate:"required"`
	Operator              string `json:"operator" validate:"required"`
	AttributeValue        string `json:"attributeValue" validate:"required"`
	JoinType              string `json:"joinType" validate:"required,oneof=AND OR"`
	HasOpeningParenthesis bool   `json:"hasOpeningParenthesis,omitempty"`
	HasClosingParenthesis bool   `json:"hasClosingParenthesis,omitempty"`
}

type DeviceGroupReadRepresentationV1 struct {
	ID          string                                `json:"id"`
	Name        string                                `json:"name"`
	Description string                                `json:"description,omitempty"`
	DeviceType  string                                `json:"deviceType"`
	GroupType   string                                `json:"groupType"`
	MemberCount int                                   `json:"memberCount"`
	Criteria    []DeviceGroupCriteriaRepresentationV1 `json:"criteria,omitempty"`
}

type DeviceGroupCreateRepresentationV1 struct {
	Name        string                                `json:"name" validate:"required"`
	Description string                                `json:"description,omitempty"`
	DeviceType  string                                `json:"deviceType" validate:"required,oneof=COMPUTER MOBILE_DEVICE"`
	GroupType   string                                `json:"groupType" validate:"required,oneof=SMART STATIC"`
	Criteria    []DeviceGroupCriteriaRepresentationV1 `json:"criteria,omitempty" validate:"required_if=GroupType SMART,excluded_with=Members"`
	Members     []string                              `json:"members,omitempty" validate:"required_if=GroupType STATIC,excluded_with=Criteria"`
}

type DeviceGroupCreateResponseV1 struct {
	ID   string `json:"id"`
	Href string `json:"href"`
}

type DeviceGroupUpdateRepresentationV1 struct {
	Name        string                                `json:"name" validate:"required"`
	Description string                                `json:"description,omitempty"`
	Criteria    []DeviceGroupCriteriaRepresentationV1 `json:"criteria,omitempty"`
	DeviceIds   []string                              `json:"deviceIds,omitempty"`
}

type DeviceGroupMemberPatchRepresentationV1 struct {
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
}

type DeviceGroupMemberV1 struct {
	DeviceID string `json:"deviceId" validate:"required"`
}

type ListDeviceGroupMemberReadRepresentation struct {
	Results     []DeviceGroupMemberV1 `json:"results"`
	TotalCount  int                   `json:"totalCount"`
	Page        int                   `json:"page"`
	PageSize    int                   `json:"pageSize"`
	TotalPages  int                   `json:"totalPages"`
	HasNext     bool                  `json:"hasNext"`
	HasPrevious bool                  `json:"hasPrevious"`
}

type DeviceGroupMemberOfRepresentationV1 struct {
	GroupID   string `json:"groupId" validate:"required"`
	GroupName string `json:"groupName" validate:"required"`
}

type ListDeviceGroupMemberOfResponseRepresentationV1 struct {
	Results     []DeviceGroupMemberOfRepresentationV1 `json:"results"`
	TotalCount  int                                   `json:"totalCount"`
	Page        int                                   `json:"page"`
	PageSize    int                                   `json:"pageSize"`
	TotalPages  int                                   `json:"totalPages"`
	HasNext     bool                                  `json:"hasNext"`
	HasPrevious bool                                  `json:"hasPrevious"`
}

type DeviceGroupPagedResponseV1 struct {
	Results     []DeviceGroupListReadRepresentationV1 `json:"results"`
	TotalCount  int64                                 `json:"totalCount"`
	Page        int                                   `json:"page"`
	PageSize    int                                   `json:"pageSize"`
	TotalPages  int                                   `json:"totalPages"`
	HasNext     bool                                  `json:"hasNext"`
	HasPrevious bool                                  `json:"hasPrevious"`
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

		resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list device groups: %w", err)
		}

		var result DeviceGroupPagedResponseV1
		if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
			return nil, err
		}

		allResults = append(allResults, result.Results...)
		if len(result.Results) < pageSize || len(result.Results) == 0 {
			break
		}
		page++
	}
	return allResults, nil
}

// GetDeviceGroupByIDV1 retrieves a device group by ID
func (c *Client) GetDeviceGroupByIDV1(ctx context.Context, id string) (*DeviceGroupReadRepresentationV1, error) {
	endpoint := fmt.Sprintf("%s/%s", deviceGroupsV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get device group %s: %w", id, err)
	}
	var result DeviceGroupReadRepresentationV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateDeviceGroupV1 creates a new device group
func (c *Client) CreateDeviceGroupV1(ctx context.Context, request *DeviceGroupCreateRepresentationV1) (*DeviceGroupCreateResponseV1, error) {
	endpoint := deviceGroupsV1Prefix + "/device-groups"
	resp, err := c.makeRequest(ctx, "POST", endpoint, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create device group: %w", err)
	}
	var result DeviceGroupCreateResponseV1
	if err := c.handleAPIResponse(ctx, resp, 201, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateDeviceGroupV1 updates a device group
func (c *Client) UpdateDeviceGroupV1(ctx context.Context, id string, request *DeviceGroupUpdateRepresentationV1) error {
	endpoint := fmt.Sprintf("%s/device-groups/%s", deviceGroupsV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, "PATCH", endpoint, request)
	if err != nil {
		return fmt.Errorf("failed to update device group %s: %w", id, err)
	}
	if err := c.handleAPIResponse(ctx, resp, 204, nil); err != nil {
		return err
	}
	return nil
}

// DeleteDeviceGroupV1 deletes a device group by ID
func (c *Client) DeleteDeviceGroupV1(ctx context.Context, id string) error {
	endpoint := fmt.Sprintf("%s/device-groups/%s", deviceGroupsV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete device group %s: %w", id, err)
	}
	if err := c.handleAPIResponse(ctx, resp, 204, nil); err != nil {
		return err
	}
	return nil
}

// GetDeviceGroupMembersV1 returns the full paginated response with metadata
func (c *Client) GetDeviceGroupMembersV1(ctx context.Context, id string, page, pageSize int) (*ListDeviceGroupMemberReadRepresentation, error) {
	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("page-size", fmt.Sprintf("%d", pageSize))

	endpoint := fmt.Sprintf("%s/device-groups/%s/members?%s", deviceGroupsV1Prefix, url.PathEscape(id), params.Encode())
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get members for device group %s: %w", id, err)
	}
	var result ListDeviceGroupMemberReadRepresentation
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateDeviceGroupMembersV1 patches the members of a static device group
func (c *Client) UpdateDeviceGroupMembersV1(ctx context.Context, id string, patch *DeviceGroupMemberPatchRepresentationV1) error {
	endpoint := fmt.Sprintf("%s/device-groups/%s/members", deviceGroupsV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, "PATCH", endpoint, patch)
	if err != nil {
		return fmt.Errorf("failed to update members for device group %s: %w", id, err)
	}
	if err := c.handleAPIResponse(ctx, resp, 204, nil); err != nil {
		return err
	}
	return nil
}

// GetDeviceGroupsForDeviceV1 returns the device groups that a device belongs to with full pagination
func (c *Client) GetDeviceGroupsForDeviceV1(ctx context.Context, deviceID string, page, pageSize int) (*ListDeviceGroupMemberOfResponseRepresentationV1, error) {
	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("page-size", fmt.Sprintf("%d", pageSize))

	endpoint := fmt.Sprintf("/devices/%s/device-groups?%s", url.PathEscape(deviceID), params.Encode())
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get device groups for device %s: %w", deviceID, err)
	}
	var result ListDeviceGroupMemberOfResponseRepresentationV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
