// Copyright 2026 Jamf Software LLC.

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

func TestGetBlueprintByIDV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/bp-123") {
			http.NotFound(w, r)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.BlueprintDetailV1{
			ID:          "bp-123",
			Name:        "Test Blueprint",
			Description: "A test blueprint",
			Created:     "2025-01-15T10:00:00Z",
			Updated:     "2025-01-16T12:00:00Z",
			DeploymentState: client.BlueprintDeploymentStateV1{
				State: "ACTIVE",
			},
			Scope: client.BlueprintUpdateScopeV1{
				DeviceGroups: []string{"grp-1", "grp-2"},
			},
			Steps: []client.BlueprintStepV1{
				{
					Name: "Declaration group",
					Components: []client.BlueprintComponentV1{
						{
							Identifier:    "com.jamf.passcode",
							Configuration: json.RawMessage(`{"minLength": 6}`),
						},
					},
				},
			},
		})
	}))

	bp, err := c.GetBlueprintByIDV1(context.Background(), "bp-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bp.ID != "bp-123" {
		t.Errorf("expected ID 'bp-123', got %q", bp.ID)
	}
	if bp.Name != "Test Blueprint" {
		t.Errorf("expected Name 'Test Blueprint', got %q", bp.Name)
	}
	if bp.DeploymentState.State != "ACTIVE" {
		t.Errorf("expected deployment state 'ACTIVE', got %q", bp.DeploymentState.State)
	}
	if len(bp.Scope.DeviceGroups) != 2 {
		t.Errorf("expected 2 device groups, got %d", len(bp.Scope.DeviceGroups))
	}
	if len(bp.Steps) != 1 || len(bp.Steps[0].Components) != 1 {
		t.Error("expected 1 step with 1 component")
	}
}

func TestGetBlueprintByIDV1_NotFound(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.RespondJSON(w, http.StatusNotFound, client.ApiError{
			HTTPStatus: 404,
			Errors: []client.Error{
				{Code: "NOT_FOUND", Description: "Blueprint not found"},
			},
		})
	}))

	_, err := c.GetBlueprintByIDV1(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain '404', got: %v", err)
	}
}

func TestCreateBlueprintV1(t *testing.T) {
	var receivedBody client.BlueprintCreateRequestV1

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		testhelpers.RespondJSON(w, http.StatusCreated, client.BlueprintCreateResponseV1{
			ID:   "new-bp-id",
			Href: "/api/blueprints/v1/blueprints/new-bp-id",
		})
	}))

	req := &client.BlueprintCreateRequestV1{
		Name:        "New Blueprint",
		Description: "Created via test",
		Scope: client.BlueprintCreateScopeV1{
			DeviceGroups: []string{"grp-a"},
		},
		Steps: []client.BlueprintStepV1{
			{
				Name: "Declaration group",
				Components: []client.BlueprintComponentV1{
					{Identifier: "com.jamf.passcode"},
				},
			},
		},
	}

	resp, err := c.CreateBlueprintV1(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "new-bp-id" {
		t.Errorf("expected ID 'new-bp-id', got %q", resp.ID)
	}
	if receivedBody.Name != "New Blueprint" {
		t.Errorf("expected request name 'New Blueprint', got %q", receivedBody.Name)
	}
	if len(receivedBody.Scope.DeviceGroups) != 1 {
		t.Errorf("expected 1 device group in scope, got %d", len(receivedBody.Scope.DeviceGroups))
	}
}

func TestUpdateBlueprintV1(t *testing.T) {
	var receivedMethod string
	var receivedPath string

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))

	req := &client.BlueprintUpdateRequestV1{
		Name: "Updated Blueprint",
		Scope: client.BlueprintUpdateScopeV1{
			DeviceGroups: []string{"grp-b"},
		},
	}

	err := c.UpdateBlueprintV1(context.Background(), "bp-123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedMethod != http.MethodPatch {
		t.Errorf("expected PATCH method, got %q", receivedMethod)
	}
	if !strings.HasSuffix(receivedPath, "/bp-123") {
		t.Errorf("expected path to end with '/bp-123', got %q", receivedPath)
	}
}

func TestDeleteBlueprintV1(t *testing.T) {
	var receivedMethod string

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))

	err := c.DeleteBlueprintV1(context.Background(), "bp-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedMethod != http.MethodDelete {
		t.Errorf("expected DELETE method, got %q", receivedMethod)
	}
}

func TestDeployBlueprintV1(t *testing.T) {
	var receivedPath string

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))

	err := c.DeployBlueprintV1(context.Background(), "bp-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(receivedPath, "/bp-123/deploy") {
		t.Errorf("expected path to end with '/bp-123/deploy', got %q", receivedPath)
	}
}

func TestUndeployBlueprintV1(t *testing.T) {
	var receivedPath string

	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))

	err := c.UndeployBlueprintV1(context.Background(), "bp-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(receivedPath, "/bp-123/undeploy") {
		t.Errorf("expected path to end with '/bp-123/undeploy', got %q", receivedPath)
	}
}

func TestGetBlueprintsV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		search := r.URL.Query().Get("search")
		if search != "My Blueprint" {
			t.Errorf("expected search 'My Blueprint', got %q", search)
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.BlueprintOverviewPagedResponseV1{
			Results: []client.BlueprintOverviewV1{
				{ID: "bp-1", Name: "My Blueprint"},
			},
			TotalCount: 1,
		})
	}))

	blueprints, err := c.GetBlueprintsV1(context.Background(), nil, "My Blueprint")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blueprints) != 1 {
		t.Errorf("expected 1 blueprint, got %d", len(blueprints))
	}
	if blueprints[0].Name != "My Blueprint" {
		t.Errorf("expected name 'My Blueprint', got %q", blueprints[0].Name)
	}
}

func TestGetBlueprintByNameV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "" {
			testhelpers.RespondJSON(w, http.StatusOK, client.BlueprintOverviewPagedResponseV1{
				Results: []client.BlueprintOverviewV1{
					{ID: "bp-found", Name: "Target Blueprint"},
				},
			})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/bp-found") {
			testhelpers.RespondJSON(w, http.StatusOK, client.BlueprintDetailV1{
				ID:   "bp-found",
				Name: "Target Blueprint",
			})
			return
		}
		http.NotFound(w, r)
	}))

	bp, err := c.GetBlueprintByNameV1(context.Background(), "Target Blueprint")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bp.ID != "bp-found" {
		t.Errorf("expected ID 'bp-found', got %q", bp.ID)
	}
}

func TestGetBlueprintByNameV1_Empty(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not make API call for empty name")
	}))

	_, err := c.GetBlueprintByNameV1(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestGetBlueprintByNameV1_NotFound(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.RespondJSON(w, http.StatusOK, client.BlueprintOverviewPagedResponseV1{
			Results: []client.BlueprintOverviewV1{
				{ID: "bp-other", Name: "Other Blueprint"},
			},
		})
	}))

	_, err := c.GetBlueprintByNameV1(context.Background(), "Missing Blueprint")
	if err == nil {
		t.Fatal("expected error when blueprint not found by name")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestGetBlueprintComponentsV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.BlueprintComponentDescriptionPagedResponseV1{
			TotalCount: 2,
			Results: []client.BlueprintComponentDescriptionV1{
				{Identifier: "com.jamf.ddm.passcode-settings", Name: "Passcode Policy"},
				{Identifier: "com.jamf.ddm.sw-updates", Name: "Software Update"},
			},
		})
	}))

	components, err := c.GetBlueprintComponentsV1(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(components))
	}
	if components[0].Identifier != "com.jamf.ddm.passcode-settings" {
		t.Errorf("expected first identifier 'com.jamf.ddm.passcode-settings', got %q", components[0].Identifier)
	}
	if components[1].Name != "Software Update" {
		t.Errorf("expected second name 'Software Update', got %q", components[1].Name)
	}
}

func TestGetBlueprintComponentByIDV1(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/com.jamf.ddm.passcode-settings") {
			http.NotFound(w, r)
			return
		}
		testhelpers.RespondJSON(w, http.StatusOK, client.BlueprintComponentDescriptionV1{
			Identifier:  "com.jamf.ddm.passcode-settings",
			Name:        "Passcode Policy",
			Description: "Configures device passcode requirements",
		})
	}))

	comp, err := c.GetBlueprintComponentByIDV1(context.Background(), "com.jamf.ddm.passcode-settings")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Identifier != "com.jamf.ddm.passcode-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.passcode-settings', got %q", comp.Identifier)
	}
	if comp.Name != "Passcode Policy" {
		t.Errorf("expected name 'Passcode Policy', got %q", comp.Name)
	}
	if comp.Description != "Configures device passcode requirements" {
		t.Errorf("expected description 'Configures device passcode requirements', got %q", comp.Description)
	}
}

func TestGetBlueprintComponentByIDV1_NotFound(t *testing.T) {
	c := testhelpers.NewMockClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.RespondJSON(w, http.StatusNotFound, client.ApiError{
			HTTPStatus: 404,
			Errors: []client.Error{
				{Code: "NOT_FOUND", Description: "Component not found"},
			},
		})
	}))

	_, err := c.GetBlueprintComponentByIDV1(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain '404', got: %v", err)
	}
}
