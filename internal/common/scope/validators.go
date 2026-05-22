// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AllFlagConflictsWith is a value-discriminated bool validator: when the
// attribute is true, every path in conflicts must be null, unknown, or
// empty. It is the scope-specific replacement for boolvalidator.ConflictsWith,
// which triggers on any value (true or false) and so cannot express the
// "only when true" semantics that all_computers, all_mobile_devices, and
// all_jss_users require.
//
// Errors attach to each conflicting attribute path per STYLE_GUIDE
// §Cross-field validation. The validator emits one diagnostic per
// violating path so the user sees every conflict in a single pass.
func AllFlagConflictsWith(conflicts ...path.Expression) validator.Bool {
	return allFlagConflictsWithValidator{conflicts: conflicts}
}

// allFlagConflictsWithValidator is the concrete validator returned by
// AllFlagConflictsWith.
type allFlagConflictsWithValidator struct {
	conflicts []path.Expression
}

// Description returns a plain-text description of the validator's rule.
func (v allFlagConflictsWithValidator) Description(_ context.Context) string {
	paths := make([]string, 0, len(v.conflicts))
	for _, expr := range v.conflicts {
		paths = append(paths, expr.String())
	}
	return fmt.Sprintf("when set to true, conflicts with: %s", strings.Join(paths, ", "))
}

// MarkdownDescription returns the markdown description of the validator's rule.
func (v allFlagConflictsWithValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateBool implements validator.Bool. It is a no-op when the attribute
// is null, unknown, or false; when the attribute is true, it inspects each
// conflict path and records an attribute error per populated sibling Set.
func (v allFlagConflictsWithValidator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if !req.ConfigValue.ValueBool() {
		return
	}
	for _, expr := range v.conflicts {
		// Relative path expressions (path.MatchRelative()) are anchored to the
		// attribute under validation. PathMatches requires an absolute path
		// expression — merge against req.PathExpression to resolve.
		absolute := req.PathExpression.MergeExpressions(expr)
		if len(absolute) == 0 {
			continue
		}
		matches, diags := req.Config.PathMatches(ctx, absolute[0])
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			continue
		}
		for _, p := range matches {
			var target types.Set
			if d := req.Config.GetAttribute(ctx, p, &target); d.HasError() {
				resp.Diagnostics.Append(d...)
				continue
			}
			if target.IsNull() || target.IsUnknown() {
				continue
			}
			if len(target.Elements()) == 0 {
				continue
			}
			resp.Diagnostics.AddAttributeError(
				p,
				"Conflicts with all-flag",
				fmt.Sprintf("%s must be null or empty when %s is true.", p, req.Path),
			)
		}
	}
}
