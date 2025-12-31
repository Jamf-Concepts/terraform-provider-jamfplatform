// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var benchmarkTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"delete": types.StringType,
}

// WaitForBenchmarkSync polls until the benchmark reaches a terminal state
// (SYNCED or FAILED) or the provided context is canceled. The interval
// controls how often the API is polled.
func waitForBenchmarkSync(ctx context.Context, c *client.Client, id string, interval time.Duration) (*client.CBEngineBenchmarkV2, error) {
	var synced *client.CBEngineBenchmarkV2
	err := helpers.PollUntil(ctx, interval, func(pollCtx context.Context) (bool, error) {
		benchmarks, err := c.GetCBEngineBenchmarksV2(pollCtx)
		if err != nil {
			tflog.Debug(pollCtx, "polling benchmarks failed", map[string]interface{}{"error": err.Error()})
			return false, fmt.Errorf("failed to poll benchmarks: %w", err)
		}
		for _, b := range benchmarks.Benchmarks {
			if b.ID != id {
				continue
			}
			benchCopy := b
			tflog.Debug(pollCtx, "benchmark syncState", map[string]interface{}{"benchmark_id": id, "sync_state": benchCopy.SyncState})
			switch benchCopy.SyncState {
			case "PENDING":
				return false, nil
			case "SYNCED":
				synced = &benchCopy
				return true, nil
			case "FAILED":
				return false, fmt.Errorf("benchmark %s in FAILED state", id)
			default:
				return false, fmt.Errorf("unexpected syncState for benchmark %s: %s", id, benchCopy.SyncState)
			}
		}
		tflog.Debug(pollCtx, "benchmark not present yet", map[string]interface{}{"benchmark_id": id})
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return synced, nil
}

// WaitForBenchmarkDeletion polls until the benchmark is no longer present or
// the context is canceled. Returns nil when the benchmark is absent. If the
// API reports a DELETE_FAILED state an error is returned.
func waitForBenchmarkDeletion(ctx context.Context, c *client.Client, id string, interval time.Duration) error {
	return helpers.PollUntil(ctx, interval, func(pollCtx context.Context) (bool, error) {
		benchmarks, err := c.GetCBEngineBenchmarksV2(pollCtx)
		if err != nil {
			tflog.Debug(pollCtx, "polling benchmarks failed", map[string]interface{}{"error": err.Error()})
			return false, fmt.Errorf("failed to poll benchmarks: %w", err)
		}
		for _, b := range benchmarks.Benchmarks {
			if b.ID != id {
				continue
			}
			tflog.Debug(pollCtx, "benchmark still present during deletion poll", map[string]interface{}{
				"benchmark_id": b.ID,
				"sync_state":   b.SyncState,
			})
			switch b.SyncState {
			case "DELETING":
				return false, nil
			case "DELETE_FAILED":
				return false, fmt.Errorf("benchmark %s deletion failed: syncState=DELETE_FAILED", id)
			default:
				return false, fmt.Errorf("benchmark %s still present after delete, syncState=%s", id, b.SyncState)
			}
		}
		tflog.Debug(pollCtx, "benchmark absent after delete", map[string]interface{}{"benchmark_id": id})
		return true, nil
	})
}

// assignBenchmarkModelFromResponse maps the API response into the Terraform benchmark model.
func assignBenchmarkModelFromResponse(model *BenchmarkResourceModel, bench *client.CBEngineBenchmarkResponseV2) {
	if model == nil || bench == nil {
		return
	}

	model.ID = types.StringValue(bench.BenchmarkID)
	model.Title = types.StringValue(bench.Title)
	model.Description = types.StringValue(bench.Description)
	model.TenantID = types.StringValue(bench.TenantID)
	model.Deleted = types.BoolValue(bench.Deleted)
	model.UpdateAvailable = types.BoolValue(bench.UpdateAvailable)
	model.LastUpdatedAt = types.StringValue(bench.LastUpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	model.Sources = make([]SourceModel, len(bench.Sources))
	for i, s := range bench.Sources {
		model.Sources[i] = SourceModel{
			Branch:   types.StringValue(s.Branch),
			Revision: types.StringValue(s.Revision),
		}
	}

	osInfoObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"os_type":         types.StringType,
			"os_version":      types.Int64Type,
			"management_type": types.StringType,
		},
	}
	osspecificObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"title":       types.StringType,
			"description": types.StringType,
			"odv_value":   types.StringType,
			"odv_hint":    types.StringType,
		},
	}

	model.Rules = make([]RuleModel, len(bench.Rules))
	for i, r := range bench.Rules {
		references := types.ListNull(types.StringType)
		if len(r.References) > 0 {
			vals := make([]attr.Value, len(r.References))
			for j, ref := range r.References {
				vals[j] = types.StringValue(ref)
			}
			references, _ = types.ListValue(types.StringType, vals)
		}

		supportedOS := types.ListNull(osInfoObjType)
		if len(r.SupportedOS) > 0 {
			osVals := make([]attr.Value, len(r.SupportedOS))
			for j, osInfo := range r.SupportedOS {
				osVals[j], _ = types.ObjectValue(
					osInfoObjType.AttrTypes,
					map[string]attr.Value{
						"os_type":         types.StringValue(osInfo.OSType),
						"os_version":      types.Int64Value(int64(osInfo.OSVersion)),
						"management_type": types.StringValue(osInfo.ManagementType),
					},
				)
			}
			supportedOS, _ = types.ListValue(osInfoObjType, osVals)
		}

		osSpecificDefaults := types.MapNull(osspecificObjType)
		if len(r.OSSpecificDefaults) > 0 {
			vals := make(map[string]attr.Value, len(r.OSSpecificDefaults))
			for key, def := range r.OSSpecificDefaults {
				var odvValue, odvHint types.String
				if def.ODV != nil {
					odvValue = types.StringValue(def.ODV.Value)
					odvHint = types.StringValue(def.ODV.Hint)
				} else {
					odvValue = types.StringNull()
					odvHint = types.StringNull()
				}
				vals[key], _ = types.ObjectValue(
					osspecificObjType.AttrTypes,
					map[string]attr.Value{
						"title":       types.StringValue(def.Title),
						"description": types.StringValue(def.Description),
						"odv_value":   odvValue,
						"odv_hint":    odvHint,
					},
				)
			}
			osSpecificDefaults, _ = types.MapValue(osspecificObjType, vals)
		}

		var (
			odvValue, odvHint, odvPlaceholder, odvType, odvValidationRegex types.String
			odvValidationMin, odvValidationMax                             types.Int64
			odvValidationEnumValues                                        types.List
		)
		if r.ODV != nil {
			odvValue = types.StringValue(r.ODV.Value)
			odvHint = types.StringValue(r.ODV.Hint)
			odvPlaceholder = types.StringValue(r.ODV.Placeholder)
			odvType = types.StringValue(r.ODV.Type)
			if r.ODV.Validation != nil {
				if r.ODV.Validation.Min != nil {
					odvValidationMin = types.Int64Value(int64(*r.ODV.Validation.Min))
				} else {
					odvValidationMin = types.Int64Null()
				}
				if r.ODV.Validation.Max != nil {
					odvValidationMax = types.Int64Value(int64(*r.ODV.Validation.Max))
				} else {
					odvValidationMax = types.Int64Null()
				}
				enumVals := make([]attr.Value, len(r.ODV.Validation.EnumValues))
				for j, v := range r.ODV.Validation.EnumValues {
					enumVals[j] = types.StringValue(v)
				}
				if len(enumVals) == 0 {
					odvValidationEnumValues = types.ListNull(types.StringType)
				} else {
					odvValidationEnumValues, _ = types.ListValue(types.StringType, enumVals)
				}
				odvValidationRegex = types.StringValue(r.ODV.Validation.Regex)
			} else {
				odvValidationMin = types.Int64Null()
				odvValidationMax = types.Int64Null()
				odvValidationEnumValues = types.ListNull(types.StringType)
				odvValidationRegex = types.StringNull()
			}
		} else {
			odvValue = types.StringNull()
			odvHint = types.StringNull()
			odvPlaceholder = types.StringNull()
			odvType = types.StringNull()
			odvValidationMin = types.Int64Null()
			odvValidationMax = types.Int64Null()
			odvValidationEnumValues = types.ListNull(types.StringType)
			odvValidationRegex = types.StringNull()
		}

		dependsOn := types.ListNull(types.StringType)
		if r.RuleRelation != nil && len(r.RuleRelation.DependsOn) > 0 {
			vals := make([]attr.Value, len(r.RuleRelation.DependsOn))
			for j, dep := range r.RuleRelation.DependsOn {
				vals[j] = types.StringValue(dep)
			}
			dependsOn, _ = types.ListValue(types.StringType, vals)
		}

		model.Rules[i] = RuleModel{
			ID:                      types.StringValue(r.ID),
			SectionName:             types.StringValue(r.SectionName),
			Enabled:                 types.BoolValue(r.Enabled),
			Title:                   types.StringValue(r.Title),
			Description:             types.StringValue(r.Description),
			References:              references,
			SupportedOS:             supportedOS,
			OSSpecificDefaults:      osSpecificDefaults,
			ODVValue:                odvValue,
			ODVHint:                 odvHint,
			ODVPlaceholder:          odvPlaceholder,
			ODVType:                 odvType,
			ODVValidationMin:        odvValidationMin,
			ODVValidationMax:        odvValidationMax,
			ODVValidationEnumValues: odvValidationEnumValues,
			ODVValidationRegex:      odvValidationRegex,
			DependsOn:               dependsOn,
		}
	}

	if len(bench.Target.DeviceGroups) > 0 {
		model.TargetDeviceGroup = types.StringValue(bench.Target.DeviceGroups[0])
	} else {
		model.TargetDeviceGroup = types.StringNull()
	}

	model.EnforcementMode = types.StringValue(bench.EnforcementMode)
}
