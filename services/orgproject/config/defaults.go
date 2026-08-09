// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import "gitea.dev/modules/json"

func rawDefault(value any) RawValue {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

// DefaultSchema returns the configuration compatible with the legacy Workbench project profile.
func DefaultSchema() Schema {
	return Schema{
		SchemaVersion: SchemaVersion,
		Fields: []Field{
			{Key: "stage", Label: "阶段", Type: FieldTypeSingle, Order: 10, Required: true, Default: rawDefault("planned"), Options: []Option{
				{Key: "planned", Label: "规划中", Order: 10},
				{Key: "development", Label: "研发中", Order: 20},
				{Key: "testing", Label: "联调验证", Order: 30},
				{Key: "released", Label: "已上线", Order: 40},
				{Key: "paused", Label: "已暂停", Order: 50},
			}},
			{Key: "progress", Label: "进度", Type: FieldTypePercent, Order: 20, Required: true, Default: rawDefault(0)},
			{Key: "owner", Label: "负责人", Type: FieldTypeMember, Order: 30},
			{Key: "followers", Label: "跟进人", Type: FieldTypeMemberArray, Order: 40, Required: true, Default: rawDefault([]int64{}), MigrationStrategy: MigrationRequireEmpty},
			{Key: "start_date", Label: "开始日期", Type: FieldTypeDate, Order: 50},
			{Key: "target_date", Label: "目标日期", Type: FieldTypeDate, Order: 60},
			{Key: "risk", Label: "风险", Type: FieldTypeSingle, Order: 70, Required: true, Default: rawDefault("normal"), Options: []Option{
				{Key: "normal", Label: "正常", Order: 10},
				{Key: "attention", Label: "需关注", Order: 20},
				{Key: "blocked", Label: "已阻塞", Order: 30},
			}},
			{Key: "summary", Label: "项目摘要", Type: FieldTypeLongText, Order: 80, Default: rawDefault("")},
			{Key: "current_problem", Label: "当前问题", Type: FieldTypeLongText, Order: 90, Default: rawDefault("")},
			{Key: "next_action", Label: "下一步行动", Type: FieldTypeLongText, Order: 100, Default: rawDefault("")},
			{Key: "next_action_owner", Label: "行动负责人", Type: FieldTypeMember, Order: 110},
			{Key: "next_action_due", Label: "行动期限", Type: FieldTypeDate, Order: 120},
		},
		ListView: ListView{
			Columns: []string{"stage", "owner", "followers", "progress", "risk", "target_date"},
			Sort:    []Sort{{FieldKey: "risk", Direction: SortDescending}, {FieldKey: "target_date", Direction: SortAscending}},
		},
		Filters: []Filter{
			{Key: "stage", Label: "阶段", FieldKey: "stage", Operator: FilterEqual},
			{Key: "risk", Label: "风险", FieldKey: "risk", Operator: FilterEqual},
			{Key: "owner", Label: "负责人", FieldKey: "owner", Operator: FilterMember},
		},
		Metrics: []Metric{
			{Key: "total", Label: "项目总数", Aggregation: MetricCount},
			{Key: "by_stage", Label: "按阶段", Aggregation: MetricCount, GroupBy: "stage"},
			{Key: "by_risk", Label: "按风险", Aggregation: MetricCount, GroupBy: "risk"},
		},
	}
}
