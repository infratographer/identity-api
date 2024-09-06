package crdbx

import (
	"strconv"
	"strings"
	"time"
)

const (
	pgTimestampFormat = "2006-01-02 15:04:05.999999999"
	defaultLimit      = 10
)

var defaultOrderFields = []string{"id"}

type FormatValues struct {
	asOfSystemTime any
	fields         []string
	values         []any
	orderFields    []string
	limit          int
}

// AsOfSystemTime if set will be formatted as `AS OF SYSTEM TIME` followed by a value.
// This value should be included directly after a `FROM` in the query.
func (v FormatValues) AsOfSystemTime() string {
	var value string

	if v.asOfSystemTime != nil {
		switch v := v.asOfSystemTime.(type) {
		case string:
			value = v
		case *string:
			value = *v
		case time.Time:
			// https://github.com/jackc/pgx/blob/9907b874c223887dba628215feae29be69306e73/pgtype/timestamp.go#L203
			value = discardTimeZone(v).Truncate(time.Microsecond).Format(pgTimestampFormat)
		}
	}

	if value == "" {
		return ""
	}

	// https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-STRINGS
	return "AS OF SYSTEM TIME " + `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

func (v FormatValues) Where(next int) string {
	if len(v.fields) == 0 {
		return ""
	}

	where := make([]string, len(v.fields))
	for i, field := range v.fields {
		where[i] = `"` + strings.ReplaceAll(field, `"`, `""`) + `">$` + strconv.Itoa(i+next)
	}

	return "(" + strings.Join(where, " AND ") + ")"
}

func (v FormatValues) WhereClause(next int) string {
	where := v.Where(next)
	if where == "" {
		return ""
	}

	return "WHERE " + where
}

func (v FormatValues) AndWhere(next int) string {
	where := v.Where(next)
	if where == "" {
		return ""
	}

	return "AND " + where
}

func (v FormatValues) LimitClause() string {
	if v.limit <= 0 {
		v.limit = defaultLimit
	}

	return "LIMIT " + strconv.Itoa(v.limit)
}

func (v FormatValues) OrderClause(fields ...string) string {
	order := v.Order(fields...)
	if order == "" {
		return ""
	}

	return "ORDER BY " + order
}

func (v FormatValues) Order(fields ...string) string {
	fields = append(fields, v.orderFields...)

	if len(fields) == 0 {
		return ""
	}

	for i, field := range fields {
		fields[i] = `"` + strings.ReplaceAll(field, `"`, `""`) + `"`
	}

	return strings.Join(fields, ",")
}

func (v FormatValues) Values(values ...any) []any {
	values = append(values, v.values...)

	return values
}

// https://github.com/jackc/pgx/blob/9907b874c223887dba628215feae29be69306e73/pgtype/timestamp.go#L219
func discardTimeZone(t time.Time) time.Time {
	if t.Location() != time.UTC {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	}

	return t
}
