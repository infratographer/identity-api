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

// FormatValues contains the necessary values to extend a query for pagination.
type FormatValues struct {
	asOfSystemTime any
	fields         []string
	values         []any
	orderFields    []string
	limit          int
	qualifier      string
}

// WithQualifier sets all field qualifiers for referenced fields.
func (v FormatValues) WithQualifier(q string) FormatValues {
	v.qualifier = q

	return v
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

	return "AS OF SYSTEM TIME " + quoteValue(value)
}

// Where returns the conditions to be added to a where clause.
// Next provides the offset for defining bind values.
// If your query includes additional values, this value should be set to the next unused number.
// For example if you had the query `SELECT * FROM users WHERE country=$1`. The value you should use here is `2`.
func (v FormatValues) Where(next int) string {
	if len(v.fields) == 0 {
		return ""
	}

	where := make([]string, len(v.fields))
	for i, field := range v.fields {
		where[i] = quoteField(v.qualifier, field) + `>$` + strconv.Itoa(i+next)
	}

	return "(" + strings.Join(where, " AND ") + ")"
}

// WhereClause returns a full WHERE clause including the conditions if defined.
// See [FormatValues.Where] for more details.
func (v FormatValues) WhereClause(next int) string {
	where := v.Where(next)
	if where == "" {
		return ""
	}

	return "WHERE " + where
}

// AndWhere returns the conditions prefixed by `AND` if conditions have been defined.
// See [FormatValues.Where] for more details.
func (v FormatValues) AndWhere(next int) string {
	where := v.Where(next)
	if where == "" {
		return ""
	}

	return "AND " + where
}

// LimitClause returns the `LIMIT` clause.
// If no limit is setup the default limit of `10` is used.
func (v FormatValues) LimitClause() string {
	if v.limit <= 0 {
		v.limit = defaultLimit
	}

	return "LIMIT " + strconv.Itoa(v.limit)
}

// OrderClause provides the `ORDER BY` clause.
// Additional fields may be included, see [FormatValues.Order] for more details.
func (v FormatValues) OrderClause(fields ...string) string {
	order := v.Order(fields...)
	if order == "" {
		return ""
	}

	return "ORDER BY " + order
}

// Order provides the `ORDER BY` clause value.
// Additional fields may be included to build the final value.
// Fields are included in addition to configured order field are used as is.
func (v FormatValues) Order(fields ...string) string {
	fields = append(fields, quoteFields(v.qualifier, v.orderFields...)...)

	if len(fields) == 0 {
		return ""
	}

	return strings.Join(fields, ",")
}

// Values returns the values to be used in a query.
// Additional values may be included and will be first in the combined values slice.
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

// https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS:~:text=To%20include%20a%20double%20quote
func quoteField(q, v string) string {
	var prefix string
	if q != "" {
		prefix = `"` + strings.ReplaceAll(q, `"`, `""`) + `".`
	}

	return prefix + `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}
func quoteFields(qualifier string, fields ...string) []string {
	for i, field := range fields {
		fields[i] = quoteField(qualifier, field)
	}

	return fields
}

// https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-STRINGS
func quoteValue(v string) string {
	return `'` + strings.ReplaceAll(v, `'`, `''`) + `'`
}
