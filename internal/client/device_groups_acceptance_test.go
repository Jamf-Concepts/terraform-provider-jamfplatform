// Copyright 2026 Jamf Software LLC.

//go:build acceptance

package client_test

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

func TestAcceptance_DeviceGroup_SmartGroupFixture(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)

	group, err := c.GetDeviceGroupByIDV1(context.Background(), groupID)
	if err != nil {
		t.Fatalf("Failed to read fixture smart group: %v", err)
	}

	if group.GroupType != "SMART" {
		t.Errorf("expected group type 'SMART', got %q", group.GroupType)
	}
	if group.Name != "tf-provider-test-fixture" {
		t.Errorf("expected name 'tf-provider-test-fixture', got %q", group.Name)
	}
	if group.DeviceType != "COMPUTER" {
		t.Errorf("expected device type 'COMPUTER', got %q", group.DeviceType)
	}

	t.Logf("Fixture smart group ID: %s, members: %d, criteria: %d", groupID, group.MemberCount, len(group.Criteria))
}

func TestAcceptance_DeviceGroup_CreateAndDeleteStaticGroup(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	desc := "Acceptance test static group — safe to delete"
	createReq := &client.DeviceGroupCreateRepresentationV1{
		Name:        "tf-acc-static-group",
		Description: &desc,
		DeviceType:  "COMPUTER",
		GroupType:   "STATIC",
		Members:     []string{},
	}

	createResp, err := c.CreateDeviceGroupV1(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateDeviceGroupV1 failed: %v", err)
	}

	t.Cleanup(func() {
		_ = c.DeleteDeviceGroupV1(ctx, createResp.ID)
	})

	group, err := c.GetDeviceGroupByIDV1(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("GetDeviceGroupByIDV1 failed: %v", err)
	}

	if group.Name != "tf-acc-static-group" {
		t.Errorf("expected name 'tf-acc-static-group', got %q", group.Name)
	}
	if group.GroupType != "STATIC" {
		t.Errorf("expected group type 'STATIC', got %q", group.GroupType)
	}
	if group.DeviceType != "COMPUTER" {
		t.Errorf("expected device type 'COMPUTER', got %q", group.DeviceType)
	}

	t.Logf("Created static group ID: %s", createResp.ID)
}

func TestAcceptance_DeviceGroup_UpdateGroup(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	desc := "Acceptance test — safe to delete"
	createReq := &client.DeviceGroupCreateRepresentationV1{
		Name:        "tf-acc-update-group-original",
		Description: &desc,
		DeviceType:  "COMPUTER",
		GroupType:   "STATIC",
		Members:     []string{},
	}

	createResp, err := c.CreateDeviceGroupV1(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateDeviceGroupV1 failed: %v", err)
	}

	t.Cleanup(func() {
		_ = c.DeleteDeviceGroupV1(ctx, createResp.ID)
	})

	updatedDesc := "Updated description"
	updateReq := &client.DeviceGroupUpdateRepresentationV1{
		Name:        "tf-acc-update-group-renamed",
		Description: &updatedDesc,
	}

	err = c.UpdateDeviceGroupV1(ctx, createResp.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateDeviceGroupV1 failed: %v", err)
	}

	group, err := c.GetDeviceGroupByIDV1(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("GetDeviceGroupByIDV1 after update failed: %v", err)
	}

	if group.Name != "tf-acc-update-group-renamed" {
		t.Errorf("expected name 'tf-acc-update-group-renamed', got %q", group.Name)
	}

	t.Logf("Updated device group ID: %s", createResp.ID)
}

func TestAcceptance_DeviceGroup_SmartGroupWithCriteria(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	desc := "Acceptance test smart group with criteria — safe to delete"
	createReq := &client.DeviceGroupCreateRepresentationV1{
		Name:        "tf-acc-smart-criteria",
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

	createResp, err := c.CreateDeviceGroupV1(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateDeviceGroupV1 failed: %v", err)
	}

	t.Cleanup(func() {
		_ = c.DeleteDeviceGroupV1(ctx, createResp.ID)
	})

	group, err := c.GetDeviceGroupByIDV1(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("GetDeviceGroupByIDV1 failed: %v", err)
	}

	if group.GroupType != "SMART" {
		t.Errorf("expected group type 'SMART', got %q", group.GroupType)
	}
	if len(group.Criteria) != 1 {
		t.Errorf("expected 1 criterion, got %d", len(group.Criteria))
	}
	if len(group.Criteria) > 0 && group.Criteria[0].AttributeName != "Serial Number" {
		t.Errorf("expected criterion attribute 'Serial Number', got %q", group.Criteria[0].AttributeName)
	}

	t.Logf("Created smart group with criteria ID: %s, members: %d", createResp.ID, group.MemberCount)
}
