// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/bpcomponents/declarations"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SafariBookmarksComponent represents a strongly-typed Safari bookmarks component.
type SafariBookmarksComponent struct {
	ManagedBookmarks []BookmarkGroupModel `tfsdk:"managed_bookmarks"`
}

// BookmarkGroupModel represents a group of managed bookmarks.
type BookmarkGroupModel struct {
	GroupIdentifier types.String    `tfsdk:"group_identifier"`
	Title           types.String    `tfsdk:"title"`
	Bookmarks       []BookmarkModel `tfsdk:"bookmarks"`
}

// BookmarkModel represents a bookmark item.
type BookmarkModel struct {
	Type   types.String       `tfsdk:"type"`
	Title  types.String       `tfsdk:"title"`
	URL    types.String       `tfsdk:"url"`
	Folder []UrlBookmarkModel `tfsdk:"folder"`
}

// UrlBookmarkModel represents a URL bookmark.
type UrlBookmarkModel struct {
	Title types.String `tfsdk:"title"`
	URL   types.String `tfsdk:"url"`
}

// GetIdentifier returns the component identifier for Safari bookmarks.
func (c *SafariBookmarksComponent) GetIdentifier() string {
	return "com.jamf.ddm.safari-bookmarks"
}

// SafariBookmarksComponentSchema returns the Terraform schema for Safari bookmarks component.
func SafariBookmarksComponentSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"managed_bookmarks": schema.SetNestedAttribute{
			MarkdownDescription: "Set of managed bookmark groups.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"group_identifier": schema.StringAttribute{
						MarkdownDescription: "Unique identifier for this group of managed bookmarks.",
						Required:            true,
					},
					"title": schema.StringAttribute{
						MarkdownDescription: "The name of the bookmarks folder.",
						Required:            true,
					},
					"bookmarks": schema.SetNestedAttribute{
						MarkdownDescription: "Set of bookmarks in this group.",
						Required:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									MarkdownDescription: "Type of bookmark. Valid values are `bookmark` (URL bookmark) or `folder` (bookmark folder).",
									Optional:            true,
									Validators:          []validator.String{stringvalidator.OneOf("bookmark", "folder")},
								},
								"title": schema.StringAttribute{
									MarkdownDescription: "The title of the folder shown in Safari.",
									Required:            true,
								},
								"url": schema.StringAttribute{
									MarkdownDescription: "The URL for direct bookmarks (not used for folders).",
									Optional:            true,
								},
								"folder": schema.SetNestedAttribute{
									MarkdownDescription: "Bookmarks within this folder.",
									Optional:            true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"title": schema.StringAttribute{
												MarkdownDescription: "The title of the bookmark shown in Safari.",
												Required:            true,
											},
											"url": schema.StringAttribute{
												MarkdownDescription: "The URL for the bookmark item.",
												Required:            true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw JSON configuration.
func (c *SafariBookmarksComponent) ToRawConfiguration() (json.RawMessage, error) {
	if len(c.ManagedBookmarks) == 0 {
		return json.Marshal(declarations.SafariBookmarksConfiguration{ManagedBookmarks: []declarations.BookmarkGroup{}})
	}

	groups := make([]declarations.BookmarkGroup, 0, len(c.ManagedBookmarks))
	for _, group := range c.ManagedBookmarks {
		g := declarations.BookmarkGroup{}

		if helpers.IsConfiguredValue(group.GroupIdentifier) {
			g.GroupIdentifier = group.GroupIdentifier.ValueString()
		}
		if helpers.IsConfiguredValue(group.Title) {
			g.Title = group.Title.ValueString()
		}

		bookmarks := make([]any, 0, len(group.Bookmarks))
		for _, bookmark := range group.Bookmarks {
			bm := make(map[string]any)

			if helpers.IsConfiguredValue(bookmark.Type) {
				typeValue := bookmark.Type.ValueString()
				switch typeValue {
				case "bookmark", "url":
					bm["Type"] = "BOOKMARK"
				case "folder":
					bm["Type"] = "FOLDER"
				default:
					bm["Type"] = typeValue
				}
			}
			if helpers.IsConfiguredValue(bookmark.Title) {
				bm["Title"] = bookmark.Title.ValueString()
			}
			if helpers.IsConfiguredValue(bookmark.URL) {
				bm["URL"] = bookmark.URL.ValueString()
			}
			if len(bookmark.Folder) > 0 {
				folder := make([]any, 0, len(bookmark.Folder))
				for _, urlBookmark := range bookmark.Folder {
					ub := map[string]any{"Type": "BOOKMARK"}
					if helpers.IsConfiguredValue(urlBookmark.Title) {
						ub["Title"] = urlBookmark.Title.ValueString()
					}
					if helpers.IsConfiguredValue(urlBookmark.URL) {
						ub["URL"] = urlBookmark.URL.ValueString()
					}
					folder = append(folder, ub)
				}
				bm["Folder"] = folder
			}
			bookmarks = append(bookmarks, bm)
		}
		g.Bookmarks = bookmarks
		groups = append(groups, g)
	}

	cfg := declarations.SafariBookmarksConfiguration{ManagedBookmarks: groups}
	return json.Marshal(cfg)
}

// FromRawConfiguration populates the typed component from raw JSON configuration.
func (c *SafariBookmarksComponent) FromRawConfiguration(raw json.RawMessage) error {
	var cfg declarations.SafariBookmarksConfiguration
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}

	managedBookmarks := make([]BookmarkGroupModel, 0, len(cfg.ManagedBookmarks))
	for _, g := range cfg.ManagedBookmarks {
		group := BookmarkGroupModel{
			GroupIdentifier: types.StringValue(g.GroupIdentifier),
			Title:           types.StringValue(g.Title),
		}

		bookmarks := make([]BookmarkModel, 0, len(g.Bookmarks))
		for _, bRaw := range g.Bookmarks {
			bm := BookmarkModel{}
			if bmMap, ok := bRaw.(map[string]any); ok {
				if t, ok := bmMap["Type"].(string); ok {
					switch t {
					case "BOOKMARK":
						bm.Type = types.StringValue("bookmark")
					case "FOLDER":
						bm.Type = types.StringValue("folder")
					default:
						bm.Type = types.StringValue(t)
					}
				}
				if title, ok := bmMap["Title"].(string); ok {
					bm.Title = types.StringValue(title)
				}
				if url, ok := bmMap["URL"].(string); ok {
					bm.URL = types.StringValue(url)
				}
				if folderRaw, ok := bmMap["Folder"].([]any); ok {
					folder := make([]UrlBookmarkModel, 0, len(folderRaw))
					for _, ubRaw := range folderRaw {
						if ubMap, ok := ubRaw.(map[string]any); ok {
							ub := UrlBookmarkModel{}
							if title, ok := ubMap["Title"].(string); ok {
								ub.Title = types.StringValue(title)
							}
							if url, ok := ubMap["URL"].(string); ok {
								ub.URL = types.StringValue(url)
							}
							folder = append(folder, ub)
						}
					}
					bm.Folder = folder
				}
			}
			bookmarks = append(bookmarks, bm)
		}
		group.Bookmarks = bookmarks
		managedBookmarks = append(managedBookmarks, group)
	}

	c.ManagedBookmarks = managedBookmarks
	return nil
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client.
func (c *SafariBookmarksComponent) ToClientComponent() (*blueprints.Component, error) {
	cfg, err := c.ToRawConfiguration()
	if err != nil {
		return nil, err
	}
	return &blueprints.Component{
		Identifier:    c.GetIdentifier(),
		Configuration: cfg,
	}, nil
}
