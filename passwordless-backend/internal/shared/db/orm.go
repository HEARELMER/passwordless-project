package shareddb

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

type OrderBy struct {
	Column    string
	Direction OrderDirection
}

type SelectOptions struct {
	OrderBy []OrderBy
	Limit   *int
	Offset  *int
}

type Operator string

const (
	OpEq  Operator = "="
	OpNeq Operator = "!="
	OpGt  Operator = ">"
	OpGte Operator = ">="
	OpLt  Operator = "<"
	OpLte Operator = "<="
	OpIn  Operator = "IN"
)

type Filter struct {
	Column string
	Op     Operator
	Value  any
}

type Table struct {
	name    string
	columns map[string]struct{}
}

func NewTable(name string, columns []string) *Table {
	colSet := make(map[string]struct{}, len(columns))
	for _, col := range columns {
		colSet[col] = struct{}{}
	}
	return &Table{name: name, columns: colSet}
}

func (t *Table) Insert(data map[string]any, returning []string) (string, []any, error) {
	if len(data) == 0 {
		return "", nil, errors.New("insert requires data")
	}

	cols, args, err := t.sortedColumnsAndArgs(data)
	if err != nil {
		return "", nil, err
	}

	placeholders := make([]string, len(cols))
	for i := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		t.name,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	if len(returning) > 0 {
		if err := t.validateColumns(returning); err != nil {
			return "", nil, err
		}
		query += " RETURNING " + strings.Join(returning, ", ")
	}

	return query, args, nil
}

func (t *Table) SelectOne(columns []string, where map[string]any) (string, []any, error) {
	if len(where) == 0 {
		return "", nil, errors.New("select one requires where")
	}

	query, args, err := t.selectQuery(columns, where, nil)
	if err != nil {
		return "", nil, err
	}

	query += " LIMIT 1"
	return query, args, nil
}

func (t *Table) SelectOneWithFilters(columns []string, filters []Filter) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, errors.New("select one requires filters")
	}

	query, args, err := t.selectQueryWithFilters(columns, filters, nil)
	if err != nil {
		return "", nil, err
	}

	query += " LIMIT 1"
	return query, args, nil
}

func (t *Table) SelectMany(columns []string, where map[string]any, opts *SelectOptions) (string, []any, error) {
	return t.selectQuery(columns, where, opts)
}

func (t *Table) SelectManyWithFilters(columns []string, filters []Filter, opts *SelectOptions) (string, []any, error) {
	return t.selectQueryWithFilters(columns, filters, opts)
}

func (t *Table) Update(data map[string]any, where map[string]any) (string, []any, error) {
	if len(data) == 0 {
		return "", nil, errors.New("update requires data")
	}
	if len(where) == 0 {
		return "", nil, errors.New("update requires where")
	}

	cols, args, err := t.sortedColumnsAndArgs(data)
	if err != nil {
		return "", nil, err
	}

	setParts := make([]string, len(cols))
	for i, col := range cols {
		setParts[i] = fmt.Sprintf("%s = $%d", col, i+1)
	}

	whereClause, whereArgs, err := t.buildWhere(where, len(args)+1)
	if err != nil {
		return "", nil, err
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", t.name, strings.Join(setParts, ", "), whereClause)
	return query, append(args, whereArgs...), nil
}

func (t *Table) Delete(where map[string]any) (string, []any, error) {
	if len(where) == 0 {
		return "", nil, errors.New("delete requires where")
	}

	whereClause, args, err := t.buildWhere(where, 1)
	if err != nil {
		return "", nil, err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", t.name, whereClause)
	return query, args, nil
}

func (t *Table) selectQuery(columns []string, where map[string]any, opts *SelectOptions) (string, []any, error) {
	if len(columns) == 0 {
		return "", nil, errors.New("select requires columns")
	}
	if err := t.validateColumns(columns); err != nil {
		return "", nil, err
	}

	whereClause, args, err := t.buildWhere(where, 1)
	if err != nil {
		return "", nil, err
	}

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), t.name)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	if opts != nil {
		orderBy, err := t.buildOrderBy(opts.OrderBy)
		if err != nil {
			return "", nil, err
		}
		if orderBy != "" {
			query += " ORDER BY " + orderBy
		}

		if opts.Limit != nil {
			if *opts.Limit < 0 {
				return "", nil, errors.New("limit must be >= 0")
			}
			query += fmt.Sprintf(" LIMIT %d", *opts.Limit)
		}

		if opts.Offset != nil {
			if *opts.Offset < 0 {
				return "", nil, errors.New("offset must be >= 0")
			}
			query += fmt.Sprintf(" OFFSET %d", *opts.Offset)
		}
	}

	return query, args, nil
}

func (t *Table) selectQueryWithFilters(columns []string, filters []Filter, opts *SelectOptions) (string, []any, error) {
	if len(columns) == 0 {
		return "", nil, errors.New("select requires columns")
	}
	if err := t.validateColumns(columns); err != nil {
		return "", nil, err
	}

	whereClause, args, err := t.buildWhereFilters(filters, 1)
	if err != nil {
		return "", nil, err
	}

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), t.name)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	if opts != nil {
		orderBy, err := t.buildOrderBy(opts.OrderBy)
		if err != nil {
			return "", nil, err
		}
		if orderBy != "" {
			query += " ORDER BY " + orderBy
		}

		if opts.Limit != nil {
			if *opts.Limit < 0 {
				return "", nil, errors.New("limit must be >= 0")
			}
			query += fmt.Sprintf(" LIMIT %d", *opts.Limit)
		}

		if opts.Offset != nil {
			if *opts.Offset < 0 {
				return "", nil, errors.New("offset must be >= 0")
			}
			query += fmt.Sprintf(" OFFSET %d", *opts.Offset)
		}
	}

	return query, args, nil
}

func (t *Table) buildOrderBy(order []OrderBy) (string, error) {
	if len(order) == 0 {
		return "", nil
	}

	parts := make([]string, len(order))
	for i, item := range order {
		if item.Column == "" {
			return "", errors.New("order by requires column")
		}
		if err := t.validateColumns([]string{item.Column}); err != nil {
			return "", err
		}
		if item.Direction != OrderAsc && item.Direction != OrderDesc {
			return "", fmt.Errorf("invalid order direction: %s", item.Direction)
		}
		parts[i] = fmt.Sprintf("%s %s", item.Column, item.Direction)
	}

	return strings.Join(parts, ", "), nil
}

func (t *Table) validateColumns(columns []string) error {
	for _, col := range columns {
		if _, ok := t.columns[col]; !ok {
			return fmt.Errorf("invalid column: %s", col)
		}
	}
	return nil
}

func (t *Table) sortedColumnsAndArgs(data map[string]any) ([]string, []any, error) {
	cols := make([]string, 0, len(data))
	for col := range data {
		cols = append(cols, col)
	}
	if err := t.validateColumns(cols); err != nil {
		return nil, nil, err
	}

	sort.Strings(cols)
	args := make([]any, len(cols))
	for i, col := range cols {
		args[i] = data[col]
	}

	return cols, args, nil
}

func (t *Table) buildWhere(where map[string]any, startIndex int) (string, []any, error) {
	if len(where) == 0 {
		return "", nil, nil
	}

	cols, args, err := t.sortedColumnsAndArgs(where)
	if err != nil {
		return "", nil, err
	}

	clauses := make([]string, len(cols))
	for i, col := range cols {
		if args[i] == nil {
			return "", nil, fmt.Errorf("where value for %s is nil", col)
		}
		clauses[i] = fmt.Sprintf("%s = $%d", col, startIndex+i)
	}

	return strings.Join(clauses, " AND "), args, nil
}

func (t *Table) buildWhereFilters(filters []Filter, startIndex int) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	clauses := make([]string, 0, len(filters))
	args := make([]any, 0, len(filters))
	argIndex := startIndex

	for _, filter := range filters {
		if filter.Column == "" {
			return "", nil, errors.New("filter requires column")
		}
		if err := t.validateColumns([]string{filter.Column}); err != nil {
			return "", nil, err
		}
		if filter.Value == nil {
			return "", nil, fmt.Errorf("filter value for %s is nil", filter.Column)
		}
		if !isAllowedOperator(filter.Op) {
			return "", nil, fmt.Errorf("invalid operator: %s", filter.Op)
		}

		if filter.Op == OpIn {
			values, err := sliceValues(filter.Value)
			if err != nil {
				return "", nil, err
			}
			if len(values) == 0 {
				return "", nil, errors.New("IN requires at least one value")
			}

			placeholders := make([]string, len(values))
			for i, value := range values {
				placeholders[i] = fmt.Sprintf("$%d", argIndex+i)
				args = append(args, value)
			}
			clauses = append(clauses, fmt.Sprintf("%s IN (%s)", filter.Column, strings.Join(placeholders, ", ")))
			argIndex += len(values)
			continue
		}

		clauses = append(clauses, fmt.Sprintf("%s %s $%d", filter.Column, filter.Op, argIndex))
		args = append(args, filter.Value)
		argIndex++
	}

	return strings.Join(clauses, " AND "), args, nil
}

func isAllowedOperator(op Operator) bool {
	switch op {
	case OpEq, OpNeq, OpGt, OpGte, OpLt, OpLte, OpIn:
		return true
	default:
		return false
	}
}

func sliceValues(value any) ([]any, error) {
	if value == nil {
		return nil, errors.New("IN value is nil")
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, errors.New("IN value must be slice or array")
	}

	length := rv.Len()
	values := make([]any, 0, length)
	for i := 0; i < length; i++ {
		values = append(values, rv.Index(i).Interface())
	}
	return values, nil
}
