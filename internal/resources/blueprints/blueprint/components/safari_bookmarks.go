// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

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
func SafariBookmarksComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"managed_bookmarks": schema.ListNestedBlock{
				Description: "List of managed bookmark groups.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"group_identifier": schema.StringAttribute{
							Description: "Unique identifier for this group of managed bookmarks.",
							Required:    true,
						},
						"title": schema.StringAttribute{
							Description: "The name of the bookmarks folder.",
							Required:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"bookmarks": schema.ListNestedBlock{
							Description: "List of bookmarks in this group.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Description: "Type of bookmark. Valid values: 'bookmark' (URL bookmark) or 'folder' (bookmark folder).",
										Optional:    true,
										Validators:  []validator.String{stringvalidator.OneOf("bookmark", "folder")},
									},
									"title": schema.StringAttribute{
										Description: "The title of the folder shown in Safari.",
										Required:    true,
									},
									"url": schema.StringAttribute{
										Description: "The URL for direct bookmarks (not used for folders).",
										Optional:    true,
									},
								},
								Blocks: map[string]schema.Block{
									"folder": schema.ListNestedBlock{
										Description: "Bookmarks within this folder.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"title": schema.StringAttribute{
													Description: "The title of the bookmark shown in Safari.",
													Required:    true,
												},
												"url": schema.StringAttribute{
													Description: "The URL for the bookmark item.",
													Required:    true,
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
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI SafariBookmarksConfiguration schema
func (c *SafariBookmarksComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	if len(c.ManagedBookmarks) > 0 {
		managedBookmarks := make([]interface{}, 0, len(c.ManagedBookmarks))

		for _, group := range c.ManagedBookmarks {
			groupMap := make(map[string]interface{})

			if !group.GroupIdentifier.IsNull() && !group.GroupIdentifier.IsUnknown() {
				groupMap["GroupIdentifier"] = group.GroupIdentifier.ValueString()
			}

			if !group.Title.IsNull() && !group.Title.IsUnknown() {
				groupMap["Title"] = group.Title.ValueString()
			}

			if len(group.Bookmarks) > 0 {
				bookmarks := make([]interface{}, 0, len(group.Bookmarks))

				for _, bookmark := range group.Bookmarks {
					bookmarkMap := make(map[string]interface{})

					if !bookmark.Type.IsNull() && !bookmark.Type.IsUnknown() {
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

					if !bookmark.Title.IsNull() && !bookmark.Title.IsUnknown() {
						bookmarkMap["Title"] = bookmark.Title.ValueString()
					}

					if !bookmark.URL.IsNull() && !bookmark.URL.IsUnknown() {
						bookmarkMap["URL"] = bookmark.URL.ValueString()
					}

					if len(bookmark.Folder) > 0 {
						folder := make([]interface{}, 0, len(bookmark.Folder))
						for _, urlBookmark := range bookmark.Folder {
							urlBookmarkMap := make(map[string]interface{})
							urlBookmarkMap["Type"] = "BOOKMARK"
							if !urlBookmark.Title.IsNull() && !urlBookmark.Title.IsUnknown() {
								urlBookmarkMap["Title"] = urlBookmark.Title.ValueString()
							}
							if !urlBookmark.URL.IsNull() && !urlBookmark.URL.IsUnknown() {
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
func (c *SafariBookmarksComponent) FromRawConfiguration(raw map[string]interface{}) error {
	if managedBookmarksRaw, exists := raw["ManagedBookmarks"]; exists {
		if managedBookmarksSlice, ok := managedBookmarksRaw.([]interface{}); ok {
			managedBookmarks := make([]BookmarkGroupModel, 0, len(managedBookmarksSlice))

			for _, groupRaw := range managedBookmarksSlice {
				if groupMap, ok := groupRaw.(map[string]interface{}); ok {
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
						if bookmarksSlice, ok := bookmarksRaw.([]interface{}); ok {
							bookmarks := make([]BookmarkModel, 0, len(bookmarksSlice))

							for _, bookmarkRaw := range bookmarksSlice {
								if bookmarkMap, ok := bookmarkRaw.(map[string]interface{}); ok {
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
										if folderSlice, ok := folderRaw.([]interface{}); ok {
											folder := make([]UrlBookmarkModel, 0, len(folderSlice))

											for _, urlBookmarkRaw := range folderSlice {
												if urlBookmarkMap, ok := urlBookmarkRaw.(map[string]interface{}); ok {
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
