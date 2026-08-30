// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package aischemas

import (
	"context"
	"fmt"
	"sync"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/aigovernance"
)

// Cache holds the product catalogue and the vendor schemas for one configured provider instance.
//
// Both are read-only server data that cannot change during a plan, and the Claude Code schema alone
// is 184 KB, so fetching per resource per plan would be waste an operator pays for. Failures are not
// cached: a transient blip in the first resource's plan must not silence validation for every later
// one, which is the same rule providerdata applies to the Jamf Pro version fetch.
type Cache struct {
	client *aigovernance.Client

	toolsMu sync.Mutex
	tools   []aigovernance.ToolSummary

	schemaMu sync.Mutex
	schemas  map[string]*Document
}

// NewCache builds a cache over an authenticated client.
func NewCache(base *jamfplatform.Client) *Cache {
	return &Cache{client: aigovernance.New(base)}
}

// Tools returns the product catalogue, fetching it once.
func (c *Cache) Tools(ctx context.Context) ([]aigovernance.ToolSummary, error) {
	if c == nil {
		return nil, nil
	}
	c.toolsMu.Lock()
	defer c.toolsMu.Unlock()

	if c.tools != nil {
		return c.tools, nil
	}
	response, err := c.client.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	c.tools = response.Results
	return c.tools, nil
}

// Tool returns one product from the catalogue, and whether it is in it.
func (c *Cache) Tool(ctx context.Context, toolID string) (aigovernance.ToolSummary, bool, error) {
	tools, err := c.Tools(ctx)
	if err != nil {
		return aigovernance.ToolSummary{}, false, err
	}
	for _, tool := range tools {
		if tool.ID == toolID {
			return tool, true, nil
		}
	}
	return aigovernance.ToolSummary{}, false, nil
}

// Document returns the parsed vendor schema for one product at one schema version, fetching and
// parsing it once.
func (c *Cache) Document(ctx context.Context, toolID, schemaVersion string) (*Document, error) {
	if c == nil {
		return nil, nil
	}
	key := toolID + "@" + schemaVersion

	c.schemaMu.Lock()
	defer c.schemaMu.Unlock()

	if document, ok := c.schemas[key]; ok {
		return document, nil
	}
	response, err := c.client.GetToolSchema(ctx, toolID, schemaVersion)
	if err != nil {
		return nil, err
	}
	document, err := Parse(toolID, schemaVersion, response.Schema)
	if err != nil {
		return nil, fmt.Errorf("parse %s schema %s: %w", toolID, schemaVersion, err)
	}
	if c.schemas == nil {
		c.schemas = map[string]*Document{}
	}
	c.schemas[key] = document
	return document, nil
}
