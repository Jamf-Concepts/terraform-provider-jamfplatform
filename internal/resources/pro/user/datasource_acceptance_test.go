// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package user_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// createInventoryUserFixture creates a throwaway Jamf Pro inventory user via the
// SDK (Pro v1 /users — the same endpoint the data source reads) and registers
// its deletion, returning the username. The provider ships no user resource, so
// the data-source test self-provisions its subject this way rather than
// depending on a pre-existing tenant user named by an env var.
func createInventoryUserFixture(t *testing.T) string {
	t.Helper()
	c := pro.New(testhelpers.NewAcceptanceClient(t))
	ctx := context.Background()
	username := "tf-acc-user-" + testhelpers.RunSuffix()
	email := username + "@example.invalid"
	created, err := c.CreateUserV1(ctx, &pro.UserInventory{Username: &username, Email: &email}, false)
	if err != nil {
		t.Fatalf("creating inventory user fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := c.DeleteUserV1(ctx, created.ID); err != nil {
			t.Errorf("deleting inventory user fixture %s: %v", created.ID, err)
		}
	})
	return username
}

// TestAccDataSource_ProUser_ByUsernameThenID looks a user up by username, then
// re-reads the resolved ID through the same data source and asserts the records
// agree. Self-provisions its subject via createInventoryUserFixture.
func TestAccDataSource_ProUser_ByUsernameThenID(t *testing.T) {
	testhelpers.AccPreCheck(t)
	username := createInventoryUserFixture(t)

	cfg := fmt.Sprintf(`
		data "jamfplatform_pro_user" "by_name" {
			username = %q
		}
		data "jamfplatform_pro_user" "by_id" {
			id = data.jamfplatform_pro_user.by_name.id
		}
	`, username)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.jamfplatform_pro_user.by_name", "id"),
				resource.TestCheckResourceAttr("data.jamfplatform_pro_user.by_name", "username", username),
				// The id-keyed read must resolve the same record.
				resource.TestCheckResourceAttrPair(
					"data.jamfplatform_pro_user.by_id", "username",
					"data.jamfplatform_pro_user.by_name", "username",
				),
				resource.TestCheckResourceAttrPair(
					"data.jamfplatform_pro_user.by_id", "id",
					"data.jamfplatform_pro_user.by_name", "id",
				),
			),
		}},
	})
}
