package esqlc

import "strings"

// query represents a SQL query being built,
// Includes the base SQL, WHERE clause, GROUP BY, HAVING, ORDER BY, LIMIT, OFFSET, and dialect.
// It provides methods for setting these components and building the final SQL string and arguments.
type query struct {
	baseSQL string
	where   []Predicate
	groupBy []string
	having  Predicate
	orderBy []string
	limit   *int
	offset  *int
	dialect Dialect
}

// newQuery creates a new Query with the given base SQL and default dialect (Postgres).
func newQuery(baseSQL string, dialect Dialect) *query {
	return &query{baseSQL: baseSQL, dialect: dialect}
}

// Where sets the WHERE clause of the query using the given Predicate.
// It returns the Query for chaining.
// Chaining multiple Where calls joins them with AND.
func (q *query) Where(pred Predicate) *query {
	// q.where = pred
	q.where = append(q.where, pred)
	return q
}

// Exclude adds a NOT condition to the WHERE clause for the given predicates.
// Multiple predicates are combined with AND before negation.
func (q *query) Exclude(preds ...Predicate) *query {
	switch len(preds) {
	case 0:
		return q
	case 1:
		return q.Where(Not(preds[0]))
	default:
		return q.Where(Not(And(preds...)))
	}
}

// GroupBy adds columns to the GROUP BY clause of the query.
// It returns the Query for chaining.
func (q *query) GroupBy(cols ...string) *query {
	q.groupBy = append(q.groupBy, cols...)
	return q
}

// Having sets the HAVING clause of the query using the given Predicate.
// It returns the Query for chaining.
func (q *query) Having(pred Predicate) *query {
	q.having = pred
	return q
}

// OrderBy adds columns to the ORDER BY clause of the query.
// It returns the Query for chaining.
func (q *query) OrderBy(cols ...string) *query {
	q.orderBy = append(q.orderBy, cols...)
	return q
}

// Limit sets the LIMIT clause of the query to the given number.
// It returns the Query for chaining.
func (q *query) Limit(n int) *query {
	q.limit = &n
	return q
}

// Offset sets the OFFSET clause of the query to the given number.
// It returns the Query for chaining.
func (q *query) Offset(n int) *query {
	q.offset = &n
	return q
}

// Build constructs the final SQL string and arguments for the query.
// It rewrites placeholders according to the specified dialect.
func (q *query) Build() (string, []any) {
	sql, args := q.RawBuild()
	return rewritePlaceholders(sql, q.dialect), args
}

// RawBuild constructs the final SQL string and arguments for the query without rewriting placeholders.
// This is useful for debugging or when the caller wants to handle placeholder rewriting themselves.
func (q *query) RawBuild() (string, []any) {
	b := &builder{args: make([]any, 0, 15)}

	// Start with the base SQL
	b.write(q.baseSQL)

	// Build the WHERE clause
	switch len(q.where) {
	case 0:
		// no WHERE clause
	case 1:
		b.write(" WHERE ")
		q.where[0].build(b)
	default:
		b.write(" WHERE ")
		for i, p := range q.where {
			if i > 0 {
				b.write(" AND ")
			}
			p.build(b)
		}
		// And(q.where...).build(b)
	}

	// Build GROUP BY
	if len(q.groupBy) > 0 {
		b.write(" GROUP BY ")
		b.write(strings.Join(q.groupBy, ", "))
	}

	// Build HAVING
	if q.having != nil {
		b.write(" HAVING ")
		q.having.build(b)
	}

	// Build ORDER BY
	if len(q.orderBy) > 0 {
		b.write(" ORDER BY ")
		b.write(strings.Join(q.orderBy, ", "))
	}

	// Build LIMIT and OFFSET
	if q.limit != nil {
		b.write(" LIMIT ?")
		b.arg(*q.limit)
	}

	// Build OFFSET
	if q.offset != nil {
		b.write(" OFFSET ?")
		b.arg(*q.offset)
	}

	// Return the final SQL string and arguments
	return b.String(), b.Args()
}
