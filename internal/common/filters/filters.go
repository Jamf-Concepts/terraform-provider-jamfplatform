// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package filters

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const filterBlockDescription = "Declarative RSQL filter clauses. Each block represents one selector/operator/argument clause."

// FilterModel represents a single declarative filter clause provided by a data source.
type FilterModel struct {
	Selector              types.String `tfsdk:"selector"`
	Operator              types.String `tfsdk:"operator"`
	Argument              types.String `tfsdk:"argument"`
	JoinWith              types.String `tfsdk:"join_with"`
	HasOpeningParenthesis types.Bool   `tfsdk:"has_opening_parenthesis"`
	HasClosingParenthesis types.Bool   `tfsdk:"has_closing_parenthesis"`
}

// SelectorValidator returns true when a selector is permitted for the data source.
type SelectorValidator func(string) bool

// FilterAttribute builds the shared schema for declarative RSQL filter clauses as a nested attribute.
func FilterAttribute(selectorDescription string, validSelectors []string) schema.ListNestedAttribute {
	var selectorValidators []validator.String
	if len(validSelectors) > 0 {
		selectorValidators = append(selectorValidators, stringvalidator.OneOf(validSelectors...))
	}

	return schema.ListNestedAttribute{
		MarkdownDescription: filterBlockDescription,
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"selector": schema.StringAttribute{
					MarkdownDescription: selectorDescription,
					Required:            true,
					Validators:          selectorValidators,
				},
				"operator": schema.StringAttribute{
					MarkdownDescription: "RSQL comparison operator. Valid values are `==`, `!=`, `>`, `<`, `>=`, and `<=`. Defaults to `==` when omitted.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("==", "!=", ">", "<", ">=", "<="),
					},
				},
				"argument": schema.StringAttribute{
					MarkdownDescription: "RSQL argument portion for the selector/operator. Provide the value exactly as required by the API (the provider will escape double quotes automatically).",
					Required:            true,
				},
				"has_opening_parenthesis": schema.BoolAttribute{
					MarkdownDescription: "Whether to prefix this clause with `(` to start a grouped expression.",
					Optional:            true,
				},
				"has_closing_parenthesis": schema.BoolAttribute{
					MarkdownDescription: "Whether to suffix this clause with `)` to close a grouped expression.",
					Optional:            true,
				},
				"join_with": schema.StringAttribute{
					MarkdownDescription: "Logical operator used to join this clause with the previous one. Valid values are `and` and `or`. Defaults to `and` when omitted or for the first clause.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("and", "or"),
					},
				},
			},
		},
	}
}

// ListFilterAttribute builds the shared schema for list resources that support RSQL filters.
func ListFilterAttribute(selectorDescription string, validSelectors []string) listschema.ListNestedAttribute {
	var selectorValidators []validator.String
	if len(validSelectors) > 0 {
		selectorValidators = append(selectorValidators, stringvalidator.OneOf(validSelectors...))
	}

	return listschema.ListNestedAttribute{
		MarkdownDescription: filterBlockDescription,
		Optional:            true,
		NestedObject: listschema.NestedAttributeObject{
			Attributes: map[string]listschema.Attribute{
				"selector": listschema.StringAttribute{
					MarkdownDescription: selectorDescription,
					Required:            true,
					Validators:          selectorValidators,
				},
				"operator": listschema.StringAttribute{
					MarkdownDescription: "RSQL comparison operator. Valid values are `==`, `!=`, `>`, `<`, `>=`, and `<=`. Defaults to `==` when omitted.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("==", "!=", ">", "<", ">=", "<="),
					},
				},
				"argument": listschema.StringAttribute{
					MarkdownDescription: "RSQL argument portion for the selector/operator. Provide the value exactly as required by the API (the provider will escape double quotes automatically).",
					Required:            true,
				},
				"has_opening_parenthesis": listschema.BoolAttribute{
					MarkdownDescription: "Whether to prefix this clause with `(` to start a grouped expression.",
					Optional:            true,
				},
				"has_closing_parenthesis": listschema.BoolAttribute{
					MarkdownDescription: "Whether to suffix this clause with `)` to close a grouped expression.",
					Optional:            true,
				},
				"join_with": listschema.StringAttribute{
					MarkdownDescription: "Logical operator used to join this clause with the previous one. Valid values are `and` and `or`. Defaults to `and` when omitted or for the first clause.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("and", "or"),
					},
				},
			},
		},
	}
}

// SelectorDescription builds the Markdown sentence that advertises the valid selectors.
func SelectorDescription(selectors []string) string {
	return fmt.Sprintf("RSQL selector. Valid values are %s.", selectorsMarkdownList(selectors))
}

// selectorsMarkdownList formats a list of selectors as a Markdown inline code list.
func selectorsMarkdownList(selectors []string) string {
	quoted := make([]string, len(selectors))
	for i, selector := range selectors {
		quoted[i] = fmt.Sprintf("`%s`", selector)
	}
	return strings.Join(quoted, ", ")
}

// BuildRSQLExpression concatenates filter clauses into a reusable RSQL string.
func BuildRSQLExpression(filters []FilterModel, selectorValidator SelectorValidator) string {
	if selectorValidator == nil {
		selectorValidator = func(string) bool { return true }
	}

	clauses := []string{}
	joiners := []string{}

	for _, filter := range filters {
		selector, hasSelector := configuredFilterValue(filter.Selector)
		argument, hasArgument := configuredFilterValue(filter.Argument)
		if !hasSelector || !hasArgument {
			continue
		}
		if !selectorValidator(selector) {
			continue
		}

		operator := "=="
		if value, ok := configuredFilterValue(filter.Operator); ok {
			operator = value
		}

		clause := Clause(selector, operator, argument)
		if isTrue(filter.HasOpeningParenthesis) {
			clause = "(" + clause
		}
		if isTrue(filter.HasClosingParenthesis) {
			clause = clause + ")"
		}
		clauses = append(clauses, clause)

		joinWith := "and"
		if value, ok := configuredFilterValue(filter.JoinWith); ok {
			logic := strings.ToLower(value)
			if logic == "or" {
				joinWith = "or"
			}
		}
		joiners = append(joiners, joinWith)
	}

	if len(clauses) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(clauses[0])
	for i := 1; i < len(clauses); i++ {
		logic := "and"
		if i < len(joiners) {
			logic = joiners[i]
		}
		builder.WriteString(" ")
		builder.WriteString(logic)
		builder.WriteString(" ")
		builder.WriteString(clauses[i])
	}

	return builder.String()
}

// AllowList returns a selector validator that permits only selectors from the provided list.
func AllowList(validSelectors []string) SelectorValidator {
	if len(validSelectors) == 0 {
		return nil
	}
	return func(selector string) bool {
		return slices.Contains(validSelectors, selector)
	}
}

// Clause builds a selector/operator/argument clause using Terraform-style escaping.
func Clause(selector, operator, argument string) string {
	if operator == "" {
		operator = "=="
	}
	return fmt.Sprintf("%s%s%s", selector, operator, FormatArgument(argument))
}

// configuredFilterValue extracts the string value from a types.String, returning
func configuredFilterValue(value types.String) (string, bool) {
	if !helpers.IsConfiguredValue(value) {
		return "", false
	}
	str := value.ValueString()
	if str == "" {
		return "", false
	}
	return str, true
}

// FormatArgument prepares an RSQL argument value, adding quotes/escapes when needed.
func FormatArgument(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if alreadyWrappedArgument(trimmed) || looksLikeListArgument(trimmed) {
		return trimmed
	}
	escaped := strings.ReplaceAll(trimmed, `\\`, `\\\\`)
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	if argumentNeedsQuoting(trimmed) {
		return fmt.Sprintf("\"%s\"", escaped)
	}
	return escaped
}

// alreadyWrappedArgument checks if the value is already enclosed in quotes.
func alreadyWrappedArgument(value string) bool {
	if len(value) < 2 {
		return false
	}
	return (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'"))
}

// looksLikeListArgument checks if the value appears to be a list argument enclosed in parentheses.
func looksLikeListArgument(value string) bool {
	return strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")")
}

// argumentNeedsQuoting determines if the argument contains characters that require it to be quoted.
func argumentNeedsQuoting(value string) bool {
	return strings.ContainsAny(value, " ,;()\t")
}

// isTrue checks if a types.Bool is explicitly true.
func isTrue(value types.Bool) bool {
	if !helpers.IsConfiguredValue(value) {
		return false
	}
	return value.ValueBool()
}
