// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestGetDeviceGroupByIDV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/device-groups/test-group-id") {
			http.NotFound(w, r)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.DeviceGroupReadRepresentationV1{
			ID:          "test-group-id",
			Name:        "Test Group",
			Description: "A test device group",
			DeviceType:  "COMPUTER",
			GroupType:   "STATIC",
			MemberCount: 5,
		})
	}))

	grp, err := c.GetDeviceGroupByIDV1(context.Background(), "test-group-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if grp.ID != "test-group-id" {
		t.Errorf("expected ID 'test-group-id', got %q", grp.ID)
	}
	if grp.Name != "Test Group" {
		t.Errorf("expected Name 'Test Group', got %q", grp.Name)
	}
	if grp.Description != "A test device group" {
		t.Errorf("expected Description 'A test device group', got %q", grp.Description)
	}
	if grp.DeviceType != "COMPUTER" {
		t.Errorf("expected DeviceType 'COMPUTER', got %q", grp.DeviceType)
	}
	if grp.GroupType != "STATIC" {
		t.Errorf("expected GroupType 'STATIC', got %q", grp.GroupType)
	}
	if grp.MemberCount != 5 {
		t.Errorf("expected MemberCount 5, got %d", grp.MemberCount)
	}
}

func TestGetDeviceGroupByIDV1_NotFound(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.RespondJSON(w, http.StatusNotFound, client.ApiError{
			HTTPStatus: 404,
			Errors: []client.Error{
				{Code: "NOT_FOUND", Description: "Device group not found"},
			},
		})
	}))

	_, err := c.GetDeviceGroupByIDV1(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain '404', got: %v", err)
	}
}

func TestCreateDeviceGroupV1(t *testing.T) {
	var receivedBody client.DeviceGroupCreateRepresentationV1

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		testhelpers.RespondJSON(w, http.StatusCreated, client.DeviceGroupCreateResponseV1{
			ID:   "new-group-id",
			Href: "/management/device-groups/v1/device-groups/new-group-id",
		})
	}))

	desc := "My static group"
	req := &client.DeviceGroupCreateRepresentationV1{
		Name:        "My Group",
		Description: &desc,
		DeviceType:  "COMPUTER",
		GroupType:   "STATIC",
		Members:     []string{"device-1", "device-2"},
	}

	resp, err := c.CreateDeviceGroupV1(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "new-group-id" {
		t.Errorf("expected ID 'new-group-id', got %q", resp.ID)
	}

	if receivedBody.Name != "My Group" {
		t.Errorf("expected request name 'My Group', got %q", receivedBody.Name)
	}
	if receivedBody.DeviceType != "COMPUTER" {
		t.Errorf("expected request deviceType 'COMPUTER', got %q", receivedBody.DeviceType)
	}
	if len(receivedBody.Members) != 2 {
		t.Errorf("expected 2 members in request, got %d", len(receivedBody.Members))
	}
}

func TestCreateDeviceGroupV1_SmartGroup(t *testing.T) {
	var receivedBody client.DeviceGroupCreateRepresentationV1

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		testhelpers.RespondJSON(w, http.StatusCreated, client.DeviceGroupCreateResponseV1{
			ID: "smart-group-id",
		})
	}))

	req := &client.DeviceGroupCreateRepresentationV1{
		Name:       "Smart Group",
		DeviceType: "MOBILE",
		GroupType:  "SMART",
		Criteria: []client.DeviceGroupCriteriaRepresentationV1{
			{
				Order:          0,
				AttributeName:  "Device Name",
				Operator:       "LIKE",
				AttributeValue: "iPad",
				JoinType:       "AND",
			},
		},
	}

	resp, err := c.CreateDeviceGroupV1(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "smart-group-id" {
		t.Errorf("expected ID 'smart-group-id', got %q", resp.ID)
	}
	if len(receivedBody.Criteria) != 1 {
		t.Fatalf("expected 1 criterion in request, got %d", len(receivedBody.Criteria))
	}
	if receivedBody.Criteria[0].AttributeName != "Device Name" {
		t.Errorf("expected criterion attributeName 'Device Name', got %q", receivedBody.Criteria[0].AttributeName)
	}
}

func TestUpdateDeviceGroupV1(t *testing.T) {
	var receivedBody client.DeviceGroupUpdateRepresentationV1
	var receivedMethod string
	var receivedPath string

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.WriteHeader(http.StatusNoContent)
	}))

	desc := "Updated description"
	req := &client.DeviceGroupUpdateRepresentationV1{
		Name:        "Updated Group",
		Description: &desc,
	}

	err := c.UpdateDeviceGroupV1(context.Background(), "group-123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedMethod != http.MethodPatch {
		t.Errorf("expected PATCH method, got %q", receivedMethod)
	}
	if !strings.HasSuffix(receivedPath, "/device-groups/group-123") {
		t.Errorf("expected path to end with '/device-groups/group-123', got %q", receivedPath)
	}
	if receivedBody.Name != "Updated Group" {
		t.Errorf("expected name 'Updated Group', got %q", receivedBody.Name)
	}
}

func TestDeleteDeviceGroupV1(t *testing.T) {
	var receivedMethod string

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))

	err := c.DeleteDeviceGroupV1(context.Background(), "group-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedMethod != http.MethodDelete {
		t.Errorf("expected DELETE method, got %q", receivedMethod)
	}
}

func TestGetDeviceGroupMembersV1(t *testing.T) {
	callCount := 0

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		testhelpers.RespondJSON(w, http.StatusOK, client.ListDeviceGroupMemberReadRepresentationV1{
			Results:    []string{"device-1", "device-2", "device-3"},
			TotalCount: 3,
			Page:       0,
			PageSize:   100,
			HasNext:    false,
		})
	}))

	members, err := c.GetDeviceGroupMembersV1(context.Background(), "group-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members, got %d", len(members))
	}
	if members[0] != "device-1" {
		t.Errorf("expected first member 'device-1', got %q", members[0])
	}
}

func TestGetDeviceGroupMembersV1_Paginated(t *testing.T) {
	callCount := 0

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		callCount++

		switch page {
		case "0":
			results := make([]string, 100)
			for i := range results {
				results[i] = "device-" + strings.Repeat("a", i+1)
			}
			testhelpers.RespondJSON(w, http.StatusOK, client.ListDeviceGroupMemberReadRepresentationV1{
				Results:    results,
				TotalCount: 150,
				Page:       0,
				PageSize:   100,
				HasNext:    true,
			})
		case "1":
			results := make([]string, 50)
			for i := range results {
				results[i] = "device-page2-" + strings.Repeat("b", i+1)
			}
			testhelpers.RespondJSON(w, http.StatusOK, client.ListDeviceGroupMemberReadRepresentationV1{
				Results:    results,
				TotalCount: 150,
				Page:       1,
				PageSize:   100,
				HasNext:    false,
			})
		default:
			t.Errorf("unexpected page: %s", page)
		}
	}))

	members, err := c.GetDeviceGroupMembersV1(context.Background(), "group-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 150 {
		t.Errorf("expected 150 members, got %d", len(members))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
}

func TestUpdateDeviceGroupMembersV1(t *testing.T) {
	var receivedBody client.DeviceGroupMemberPatchRepresentationV1

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusNoContent)
	}))

	patch := &client.DeviceGroupMemberPatchRepresentationV1{
		Added:   []string{"device-new"},
		Removed: []string{"device-old"},
	}

	err := c.UpdateDeviceGroupMembersV1(context.Background(), "group-123", patch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(receivedBody.Added) != 1 || receivedBody.Added[0] != "device-new" {
		t.Errorf("expected added=['device-new'], got %v", receivedBody.Added)
	}
	if len(receivedBody.Removed) != 1 || receivedBody.Removed[0] != "device-old" {
		t.Errorf("expected removed=['device-old'], got %v", receivedBody.Removed)
	}
}

func TestGetDeviceGroupsV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter := r.URL.Query().Get("filter")
		if filter != "name==\"Test\"" {
			t.Errorf("expected filter 'name==\"Test\"', got %q", filter)
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.DeviceGroupPagedResponseV1{
			Results: []client.DeviceGroupListReadRepresentationV1{
				{ID: "grp-1", Name: "Test Group 1", DeviceType: "COMPUTER", GroupType: "STATIC"},
				{ID: "grp-2", Name: "Test Group 2", DeviceType: "MOBILE", GroupType: "SMART"},
			},
		})
	}))

	groups, err := c.GetDeviceGroupsV1(context.Background(), nil, "name==\"Test\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].ID != "grp-1" {
		t.Errorf("expected first group ID 'grp-1', got %q", groups[0].ID)
	}
}

func TestCreateDeviceGroupV1_APIError(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.RespondJSON(w, http.StatusBadRequest, client.ApiError{
			HTTPStatus: 400,
			TraceID:    "trace-123",
			Errors: []client.Error{
				{Code: "VALIDATION_ERROR", Field: "name", Description: "Name is required"},
			},
		})
	}))

	req := &client.DeviceGroupCreateRepresentationV1{
		DeviceType: "COMPUTER",
		GroupType:  "STATIC",
	}

	_, err := c.CreateDeviceGroupV1(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "VALIDATION_ERROR") {
		t.Errorf("expected error to contain 'VALIDATION_ERROR', got: %v", err)
	}
}
