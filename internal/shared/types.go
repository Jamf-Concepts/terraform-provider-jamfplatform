// Copyright 2025 Jamf Software LLC.

package shared

import "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"

// ProviderClients holds both API clients for use by data sources/resources.
type ProviderClients struct {
	CBEngine  *client.Client
	Inventory *client.Client
}
