// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SafariBookmarksComponent represents a strongly-typed Safari bookmarks component
type SafariBookmarksComponent struct {
	ManagedBookmarks []BookmarkGroupModel `tfsdk:"managed_bookmarks"`
}

// BookmarkGroupModel represents a group of managed bookmarks
type BookmarkGroupModel struct {
	GroupIdentifier types.String    `tfsdk:"group_identifier"`
	Title           types.String    `tfsdk:"title"`
	Bookmarks       []BookmarkModel `tfsdk:"bookmarks"`
}

// BookmarkModel represents a bookmark item
type BookmarkModel struct {
	Type   types.String       `tfsdk:"type"`
	Title  types.String       `tfsdk:"title"`
	URL    types.String       `tfsdk:"url"`
	Folder []UrlBookmarkModel `tfsdk:"folder"`
}

// UrlBookmarkModel represents a URL bookmark
type UrlBookmarkModel struct {
	Title types.String `tfsdk:"title"`
	URL   types.String `tfsdk:"url"`
}

// GetIdentifier returns the component identifier for Safari bookmarks
func (c *SafariBookmarksComponent) GetIdentifier() string {
	return "com.jamf.ddm.safari-bookmarks"
}

// SafariBookmarksComponentSchema returns the Terraform schema for Safari bookmarks component
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
						Optional:            true,
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

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI SafariBookmarksConfiguration schema
func (c *SafariBookmarksComponent) ToRawConfiguration() (map[string]any, error) {
	config := make(map[string]any)

	if len(c.ManagedBookmarks) > 0 {
		managedBookmarks := make([]any, 0, len(c.ManagedBookmarks))

		for _, group := range c.ManagedBookmarks {
			groupMap := make(map[string]any)

			if helpers.IsConfiguredValue(group.GroupIdentifier) {
				groupMap["GroupIdentifier"] = group.GroupIdentifier.ValueString()
			}

			if helpers.IsConfiguredValue(group.Title) {
				groupMap["Title"] = group.Title.ValueString()
			}

			if len(group.Bookmarks) > 0 {
				bookmarks := make([]any, 0, len(group.Bookmarks))

				for _, bookmark := range group.Bookmarks {
					bookmarkMap := make(map[string]any)

					if helpers.IsConfiguredValue(bookmark.Type) {
						typeValue := bookmark.Type.ValueString()
						switch typeValue {
						case "bookmark", "url":
							bookmarkMap["Type"] = "BOOKMARK"
						case "folder":
							bookmarkMap["Type"] = "FOLDER"
						default:
							bookmarkMap["Type"] = typeValue
						}
					}

					if helpers.IsConfiguredValue(bookmark.Title) {
						bookmarkMap["Title"] = bookmark.Title.ValueString()
					}

					if helpers.IsConfiguredValue(bookmark.URL) {
						bookmarkMap["URL"] = bookmark.URL.ValueString()
					}

					if len(bookmark.Folder) > 0 {
						folder := make([]any, 0, len(bookmark.Folder))
						for _, urlBookmark := range bookmark.Folder {
							urlBookmarkMap := make(map[string]any)
							urlBookmarkMap["Type"] = "BOOKMARK"
							if helpers.IsConfiguredValue(urlBookmark.Title) {
								urlBookmarkMap["Title"] = urlBookmark.Title.ValueString()
							}
							if helpers.IsConfiguredValue(urlBookmark.URL) {
								urlBookmarkMap["URL"] = urlBookmark.URL.ValueString()
							}
							folder = append(folder, urlBookmarkMap)
						}
						bookmarkMap["Folder"] = folder
					}

					bookmarks = append(bookmarks, bookmarkMap)
				}

				groupMap["Bookmarks"] = bookmarks
			}

			managedBookmarks = append(managedBookmarks, groupMap)
		}

		config["ManagedBookmarks"] = managedBookmarks
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *SafariBookmarksComponent) FromRawConfiguration(raw map[string]any) error {
	if managedBookmarksRaw, exists := raw["ManagedBookmarks"]; exists {
		if managedBookmarksSlice, ok := managedBookmarksRaw.([]any); ok {
			managedBookmarks := make([]BookmarkGroupModel, 0, len(managedBookmarksSlice))

			for _, groupRaw := range managedBookmarksSlice {
				if groupMap, ok := groupRaw.(map[string]any); ok {
					group := BookmarkGroupModel{}

					if groupIdentifier, exists := groupMap["GroupIdentifier"]; exists {
						if groupIdentifierStr, ok := groupIdentifier.(string); ok {
							group.GroupIdentifier = types.StringValue(groupIdentifierStr)
						}
					}

					if title, exists := groupMap["Title"]; exists {
						if titleStr, ok := title.(string); ok {
							group.Title = types.StringValue(titleStr)
						}
					}

					if bookmarksRaw, exists := groupMap["Bookmarks"]; exists {
						if bookmarksSlice, ok := bookmarksRaw.([]any); ok {
							bookmarks := make([]BookmarkModel, 0, len(bookmarksSlice))

							for _, bookmarkRaw := range bookmarksSlice {
								if bookmarkMap, ok := bookmarkRaw.(map[string]any); ok {
									bookmark := BookmarkModel{}

									if bookmarkType, exists := bookmarkMap["Type"]; exists {
										if bookmarkTypeStr, ok := bookmarkType.(string); ok {
											// Convert API values back to user-friendly values
											switch bookmarkTypeStr {
											case "BOOKMARK":
												bookmark.Type = types.StringValue("bookmark")
											case "FOLDER":
												bookmark.Type = types.StringValue("folder")
											default:
												bookmark.Type = types.StringValue(bookmarkTypeStr) // Pass through as-is
											}
										}
									}

									if bookmarkTitle, exists := bookmarkMap["Title"]; exists {
										if bookmarkTitleStr, ok := bookmarkTitle.(string); ok {
											bookmark.Title = types.StringValue(bookmarkTitleStr)
										}
									}

									if bookmarkURL, exists := bookmarkMap["URL"]; exists {
										if bookmarkURLStr, ok := bookmarkURL.(string); ok {
											bookmark.URL = types.StringValue(bookmarkURLStr)
										}
									}

									if folderRaw, exists := bookmarkMap["Folder"]; exists {
										if folderSlice, ok := folderRaw.([]any); ok {
											folder := make([]UrlBookmarkModel, 0, len(folderSlice))

											for _, urlBookmarkRaw := range folderSlice {
												if urlBookmarkMap, ok := urlBookmarkRaw.(map[string]any); ok {
													urlBookmark := UrlBookmarkModel{}

													if urlTitle, exists := urlBookmarkMap["Title"]; exists {
														if urlTitleStr, ok := urlTitle.(string); ok {
															urlBookmark.Title = types.StringValue(urlTitleStr)
														}
													}

													if urlURL, exists := urlBookmarkMap["URL"]; exists {
														if urlURLStr, ok := urlURL.(string); ok {
															urlBookmark.URL = types.StringValue(urlURLStr)
														}
													}

													folder = append(folder, urlBookmark)
												}
											}

											bookmark.Folder = folder
										}
									}

									bookmarks = append(bookmarks, bookmark)
								}
							}

							group.Bookmarks = bookmarks
						}
					}

					managedBookmarks = append(managedBookmarks, group)
				}
			}

			c.ManagedBookmarks = managedBookmarks
		}
	}

	return nil
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client
func (c *SafariBookmarksComponent) ToClientComponent() (*BlueprintComponentData, error) {
	config, err := c.ToRawConfiguration()
	if err != nil {
		return nil, err
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	return &BlueprintComponentData{
		Identifier:    c.GetIdentifier(),
		Configuration: json.RawMessage(configJSON),
	}, nil
}
