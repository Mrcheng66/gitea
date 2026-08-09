// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package query

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gitea.dev/modules/json"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"
	"gitea.dev/services/orgproject/config"
)

const maxMetricBuckets = 100

// CompiledQuery contains SQL text and its bound arguments.
type CompiledQuery struct {
	SQL  string
	Args []any
}

// CompiledList contains the data and count queries for a project list.
type CompiledList struct {
	Data     CompiledQuery
	Count    CompiledQuery
	Page     int
	PageSize int
}

// ListOptions selects configured filters, sorting and pagination.
type ListOptions struct {
	OwnerID         int64
	FilterValues    map[string]config.RawValue
	Sort            []config.Sort
	Search          string
	OnlyUserID      int64
	Due             string
	RiskFirst       bool
	Now             timeutil.TimeStamp
	Page            int
	PageSize        int
	IncludeArchived bool
}

// MetricOptions selects a configured metric and optional list filters.
type MetricOptions struct {
	OwnerID      int64
	MetricKey    string
	FilterValues map[string]config.RawValue
}

type compiler struct {
	ast      *AST
	joins    []string
	joinArgs []any
	aliases  map[string]string
}

func newCompiler(ast *AST) *compiler {
	return &compiler{ast: ast, aliases: make(map[string]string)}
}

func (compiler *compiler) join(queryField field) string {
	if alias, ok := compiler.aliases[queryField.key]; ok {
		return alias
	}
	alias := fmt.Sprintf("fv_%d", len(compiler.aliases))
	compiler.aliases[queryField.key] = alias
	compiler.joins = append(compiler.joins,
		fmt.Sprintf("LEFT JOIN org_project_field_value %s ON %s.project_id = org_project.id AND %s.field_key = ?", alias, alias, alias),
	)
	compiler.joinArgs = append(compiler.joinArgs, queryField.key)
	return alias
}

func (compiler *compiler) joinAs(queryField field, alias string) string {
	if existing, ok := compiler.aliases[queryField.key]; ok {
		return existing
	}
	compiler.aliases[queryField.key] = alias
	compiler.joins = append(compiler.joins,
		fmt.Sprintf("LEFT JOIN org_project_field_value %s ON %s.project_id = org_project.id AND %s.field_key = ?", alias, alias, alias),
	)
	compiler.joinArgs = append(compiler.joinArgs, queryField.key)
	return alias
}

// CompileList builds parameterized SQLite list and count queries.
func CompileList(ast *AST, opts ListOptions) (CompiledList, error) {
	if ast == nil {
		return CompiledList{}, errors.New("organization project query AST is required")
	}
	if opts.OwnerID <= 0 {
		return CompiledList{}, errors.New("organization owner ID must be positive")
	}
	page, pageSize, err := normalizePagination(opts.Page, opts.PageSize)
	if err != nil {
		return CompiledList{}, err
	}

	queryCompiler := newCompiler(ast)
	var riskOrder []string
	var riskOrderArgs []any
	if opts.RiskFirst {
		riskOrder, riskOrderArgs = queryCompiler.compileRiskFirstSort(opts.Now)
	}
	where := []string{"org_project.owner_id = ?"}
	whereArgs := []any{opts.OwnerID}
	if !opts.IncludeArchived {
		where = append(where, "org_project.lifecycle = ?")
		whereArgs = append(whereArgs, "active")
	}
	filterSQL, filterArgs, err := queryCompiler.compileFilters(opts.FilterValues)
	if err != nil {
		return CompiledList{}, err
	}
	where = append(where, filterSQL...)
	whereArgs = append(whereArgs, filterArgs...)
	searchSQL, searchArgs := compileSearch(ast, opts.Search)
	if searchSQL != "" {
		where = append(where, searchSQL)
		whereArgs = append(whereArgs, searchArgs...)
	}
	mineSQL, mineArgs := compileOnlyMine(ast, opts.OnlyUserID)
	if mineSQL != "" {
		where = append(where, mineSQL)
		whereArgs = append(whereArgs, mineArgs...)
	}
	dueSQL, dueArgs, err := queryCompiler.compileDue(opts.Due, opts.Now)
	if err != nil {
		return CompiledList{}, err
	}
	if dueSQL != "" {
		where = append(where, dueSQL)
		whereArgs = append(whereArgs, dueArgs...)
	}

	orderBy := riskOrder
	orderArgs := riskOrderArgs
	if len(orderBy) == 0 {
		orderBy, err = queryCompiler.compileSort(opts.Sort)
		if err != nil {
			return CompiledList{}, err
		}
	}
	joins := strings.Join(queryCompiler.joins, " ")
	whereSQL := strings.Join(where, " AND ")
	args := append(slices.Clone(queryCompiler.joinArgs), whereArgs...)

	dataSQL := "SELECT org_project.id, org_project.owner_id, org_project.slug, org_project.name, org_project.description, " +
		"org_project.lifecycle, org_project.version, org_project.created_by, org_project.created_unix, org_project.updated_unix " +
		"FROM org_project " + joins + " WHERE " + whereSQL + " ORDER BY " + strings.Join(orderBy, ", ") + " LIMIT ? OFFSET ?"
	dataArgs := append(slices.Clone(args), orderArgs...)
	dataArgs = append(dataArgs, pageSize, (page-1)*pageSize)
	countSQL := "SELECT COUNT(DISTINCT org_project.id) FROM org_project " + joins + " WHERE " + whereSQL

	return CompiledList{
		Data:  CompiledQuery{SQL: dataSQL, Args: dataArgs},
		Count: CompiledQuery{SQL: countSQL, Args: args},
		Page:  page, PageSize: pageSize,
	}, nil
}

func (compiler *compiler) compileDue(scope string, now timeutil.TimeStamp) (string, []any, error) {
	if scope == "" {
		return "", nil, nil
	}
	target, ok := compiler.ast.fields["target_date"]
	if !ok {
		return "0 = 1", nil, nil
	}
	if now <= 0 {
		now = timeutil.TimeStampNow()
	}
	alias := compiler.joinAs(target, "due_filter")
	switch scope {
	case "overdue":
		return alias + ".value_time < ?", []any{now}, nil
	case "week":
		return alias + ".value_time >= ? AND " + alias + ".value_time < ?", []any{now, now.AddDuration(7 * 24 * time.Hour)}, nil
	default:
		return "", nil, fmt.Errorf("unsupported due scope %q", scope)
	}
}

func compileSearch(ast *AST, search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", nil
	}
	pattern := "%" + strings.ToLower(search) + "%"
	clauses := []string{
		"LOWER(org_project.name) LIKE ?",
		"LOWER(org_project.slug) LIKE ?",
		"LOWER(org_project.description) LIKE ?",
	}
	args := []any{pattern, pattern, pattern}
	if _, ok := ast.fields["owner"]; ok {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM org_project_field_value search_owner
			INNER JOIN user search_user ON search_user.id = search_owner.value_user_id
			WHERE search_owner.project_id = org_project.id AND search_owner.field_key = ?
			AND (LOWER(search_user.name) LIKE ? OR LOWER(search_user.full_name) LIKE ?)
		)`)
		args = append(args, "owner", pattern, pattern)
	}
	if _, ok := ast.fields["followers"]; ok {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM org_project_field_value search_followers
			INNER JOIN json_each(search_followers.value_json) search_follower_id
			INNER JOIN user search_follower ON search_follower.id = search_follower_id.value
			WHERE search_followers.project_id = org_project.id AND search_followers.field_key = ?
			AND (LOWER(search_follower.name) LIKE ? OR LOWER(search_follower.full_name) LIKE ?)
		)`)
		args = append(args, "followers", pattern, pattern)
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func compileOnlyMine(ast *AST, userID int64) (string, []any) {
	if userID <= 0 {
		return "", nil
	}
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if _, ok := ast.fields["owner"]; ok {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM org_project_field_value mine_owner
			WHERE mine_owner.project_id = org_project.id AND mine_owner.field_key = ? AND mine_owner.value_user_id = ?
		)`)
		args = append(args, "owner", userID)
	}
	if _, ok := ast.fields["followers"]; ok {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM org_project_field_value mine_followers
			INNER JOIN json_each(mine_followers.value_json) mine_follower
			WHERE mine_followers.project_id = org_project.id AND mine_followers.field_key = ? AND mine_follower.value = ?
		)`)
		args = append(args, "followers", userID)
	}
	if len(clauses) == 0 {
		return "0 = 1", nil
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func (compiler *compiler) compileRiskFirstSort(now timeutil.TimeStamp) ([]string, []any) {
	risk, hasRisk := compiler.ast.fields["risk"]
	target, hasTarget := compiler.ast.fields["target_date"]
	if !hasRisk && !hasTarget {
		return nil, nil
	}
	if now <= 0 {
		now = timeutil.TimeStampNow()
	}
	caseParts := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if hasRisk {
		alias := compiler.joinAs(risk, "risk_sort")
		caseParts = append(caseParts, "WHEN "+alias+".value_text = ? THEN 0")
		args = append(args, "blocked")
	}
	if hasTarget {
		alias := compiler.joinAs(target, "target_sort")
		caseParts = append(caseParts, "WHEN "+alias+".value_time < ? THEN 1")
		args = append(args, now)
	}
	if hasRisk {
		caseParts = append(caseParts, "WHEN risk_sort.value_text = ? THEN 2")
		args = append(args, "attention")
	}
	orderBy := []string{"CASE " + strings.Join(caseParts, " ") + " ELSE 3 END ASC"}
	if hasTarget {
		orderBy = append(orderBy, "target_sort.value_time IS NULL ASC", "target_sort.value_time ASC")
	}
	orderBy = append(orderBy, "org_project.updated_unix DESC", "org_project.id ASC")
	return orderBy, args
}

func (compiler *compiler) compileFilters(values map[string]config.RawValue) ([]string, []any, error) {
	if len(values) == 0 {
		return nil, nil, nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	where := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for _, key := range keys {
		configured, ok := compiler.ast.filters[key]
		if !ok {
			return nil, nil, fmt.Errorf("unknown organization project filter %q", key)
		}
		raw := values[key]
		if len(raw) == 0 {
			raw = configured.value
		}
		clause, clauseArgs, err := compiler.compileFilter(configured, raw)
		if err != nil {
			return nil, nil, fmt.Errorf("filter %q: %w", key, err)
		}
		where = append(where, clause)
		args = append(args, clauseArgs...)
	}
	return where, args, nil
}

func (compiler *compiler) compileFilter(configured filter, raw config.RawValue) (string, []any, error) {
	alias := compiler.join(configured.field)
	column := alias + "." + configured.field.valueColumn
	switch configured.operator {
	case config.FilterIsEmpty:
		if len(raw) > 0 && string(raw) != "null" {
			return "", nil, errors.New("does not accept a value")
		}
		return alias + ".id IS NULL", nil, nil
	case config.FilterIsNotEmpty:
		if len(raw) > 0 && string(raw) != "null" {
			return "", nil, errors.New("does not accept a value")
		}
		return alias + ".id IS NOT NULL", nil, nil
	}
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil, errors.New("requires a value")
	}

	value, err := decodeFilterValue(configured.field, configured.operator, raw)
	if err != nil {
		return "", nil, err
	}
	switch configured.operator {
	case config.FilterEqual:
		return column + " = ?", []any{value}, nil
	case config.FilterNotEqual:
		return alias + ".id IS NOT NULL AND " + column + " != ?", []any{value}, nil
	case config.FilterContains:
		if configured.field.fieldType == config.FieldTypeMulti || configured.field.fieldType == config.FieldTypeMemberArray {
			return "EXISTS (SELECT 1 FROM json_each(" + column + ") filter_value WHERE filter_value.value = ?)", []any{value}, nil
		}
		return "instr(" + column + ", ?) > 0", []any{value}, nil
	case config.FilterGreaterEqual:
		return column + " >= ?", []any{value}, nil
	case config.FilterLessEqual:
		return column + " <= ?", []any{value}, nil
	case config.FilterMember:
		if configured.field.fieldType == config.FieldTypeMemberArray {
			return "EXISTS (SELECT 1 FROM json_each(" + column + ") filter_member WHERE filter_member.value = ?)", []any{value}, nil
		}
		return column + " = ?", []any{value}, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator %q", configured.operator)
	}
}

func decodeFilterValue(queryField field, operator config.FilterOperator, raw config.RawValue) (any, error) {
	switch queryField.fieldType {
	case config.FieldTypeShortText, config.FieldTypeLongText, config.FieldTypeSingle:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, errors.New("must be a string")
		}
		return value, nil
	case config.FieldTypeMulti:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, errors.New("must be an option key")
		}
		return value, nil
	case config.FieldTypeInteger, config.FieldTypeMember, config.FieldTypeMemberArray:
		var value int64
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, errors.New("must be an integer")
		}
		return value, nil
	case config.FieldTypeDecimal, config.FieldTypePercent:
		var value float64
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, errors.New("must be a number")
		}
		return value, nil
	case config.FieldTypeDate, config.FieldTypeDateTime:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, errors.New("must be a date string")
		}
		layout := time.RFC3339
		if queryField.fieldType == config.FieldTypeDate {
			layout = time.DateOnly
		}
		parsed, err := time.Parse(layout, value)
		if err != nil {
			return nil, fmt.Errorf("must use %s format", layout)
		}
		return timeutil.TimeStamp(parsed.Unix()), nil
	case config.FieldTypeBoolean:
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, errors.New("must be a boolean")
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported field type %q for operator %q", queryField.fieldType, operator)
	}
}

var fixedSortColumns = map[string]string{
	"id": "org_project.id", "name": "org_project.name", "slug": "org_project.slug",
	"created": "org_project.created_unix", "updated": "org_project.updated_unix",
}

func (compiler *compiler) compileSort(requested []config.Sort) ([]string, error) {
	if len(requested) == 0 {
		orderBy := make([]string, 0, len(compiler.ast.defaultSort)+1)
		for _, configured := range compiler.ast.defaultSort {
			alias := compiler.join(configured.field)
			orderBy = append(orderBy, alias+"."+configured.field.valueColumn+" "+strings.ToUpper(string(configured.direction)))
		}
		orderBy = append(orderBy, "org_project.id ASC")
		return orderBy, nil
	}

	orderBy := make([]string, 0, len(requested)+1)
	seen := make(map[string]struct{}, len(requested))
	for _, configured := range requested {
		if _, ok := seen[configured.FieldKey]; ok {
			return nil, fmt.Errorf("duplicate sort field %q", configured.FieldKey)
		}
		if configured.Direction != config.SortAscending && configured.Direction != config.SortDescending {
			return nil, fmt.Errorf("unsupported sort direction %q", configured.Direction)
		}
		column, ok := fixedSortColumns[configured.FieldKey]
		if !ok {
			queryField, exists := compiler.ast.fields[configured.FieldKey]
			if !exists {
				return nil, fmt.Errorf("unknown sort field %q", configured.FieldKey)
			}
			alias := compiler.join(queryField)
			column = alias + "." + queryField.valueColumn
		}
		orderBy = append(orderBy, column+" "+strings.ToUpper(string(configured.Direction)))
		seen[configured.FieldKey] = struct{}{}
	}
	orderBy = append(orderBy, "org_project.id ASC")
	return orderBy, nil
}

func normalizePagination(page, pageSize int) (int, int, error) {
	if page < 0 {
		return 0, 0, errors.New("page must not be negative")
	}
	if page == 0 {
		page = 1
	}
	defaultPageSize := setting.OrgProject.DefaultPageSize
	if defaultPageSize <= 0 {
		defaultPageSize = 25
	}
	maxPageSize := setting.OrgProject.MaxPageSize
	if maxPageSize <= 0 {
		maxPageSize = 100
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize < 0 || pageSize > maxPageSize {
		return 0, 0, fmt.Errorf("page size must be between 1 and %d", maxPageSize)
	}
	return page, pageSize, nil
}
