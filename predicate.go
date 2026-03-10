package esqlc

import "strings"

// Predicate represents a SQL condition that can be built into a query.
type Predicate interface {
	build(*builder)
}

// cond represents a simple condition with an expression and arguments.
// For example, "status = ?" with argument "active".
// It implements the Predicate interface.
type cond struct {
	expr string
	args []any
}

// build implements the Predicate interface for cond.
func (c *cond) build(b *builder) {
	b.write(c.expr)
	b.args = append(b.args, c.args...)
}

// Cond is a helper function to create a simple condition predicate.
// Example usage: Cond("age > ?", 30) creates condition "age > ?" with argument 30.
func Cond(expr string, args ...any) Predicate {
	return &cond{expr: expr, args: args}
}

// and represents a logical AND of multiple predicates.
// It implements the Predicate interface.
type and struct {
	preds []Predicate
}

// build implements the Predicate interface for and.
func (a *and) build(b *builder) {
	switch len(a.preds) {
	case 0:
		return
	case 1:
		a.preds[0].build(b)
		return
	}

	b.write("(")
	for i, p := range a.preds {
		if i > 0 {
			b.write(" AND ")
		}
		p.build(b)
	}
	b.write(")")
}

// And is a helper function to combine multiple predicates with a logical AND.
// Example usage: And(Cond("age > ?", 30), Cond("status = ?", "active"))
// creates "(age > ? AND status = ?)" with arguments 30 and "active".
func And(preds ...Predicate) Predicate {
	return &and{preds: preds}
}

// or represents a logical OR of multiple predicates.
// It implements the Predicate interface.
type or struct {
	preds []Predicate
}

// build implements the Predicate interface for or.
func (o *or) build(b *builder) {
	switch len(o.preds) {
	case 0:
		return
	case 1:
		o.preds[0].build(b)
		return
	}

	b.write("(")
	for i, p := range o.preds {
		if i > 0 {
			b.write(" OR ")
		}
		p.build(b)
	}
	b.write(")")
}

// Or is a helper function to combine multiple predicates with a logical OR.
// Example usage: Or(Cond("age < ?", 18), Cond("status = ?", "inactive"))
// creates "(age < ? OR status = ?)" with arguments 18 and "inactive".
func Or(preds ...Predicate) Predicate {
	return &or{preds: preds}
}

// not represents a logical NOT of a predicate.
// It implements the Predicate interface.
type not struct {
	pred Predicate
}

// build implements the Predicate interface for not.
func (n *not) build(b *builder) {
	b.write("NOT (")
	n.pred.build(b)
	b.write(")")
}

// Not is a helper function to negate a predicate with a logical NOT.
// Example usage: Not(Cond("status = ?", "active"))
// creates "NOT (status = ?)" with argument "active".
func Not(pred Predicate) Predicate {
	return &not{pred: pred}
}

// IsNull checks if a column is NULL.
// It builds "col IS NULL"
func IsNull(col string) Predicate {
	return Cond(col + " IS NULL")
}

// IsNotNull checks if a column is NOT NULL.
// It builds "col IS NOT NULL"
func IsNotNull(col string) Predicate {
	return Cond(col + " IS NOT NULL")
}

// Eq checks if a column equals a value.
// Eq builds "col = ?"
func Eq(col string, val any) Predicate {
	return Cond(col+" = ?", val)
}

// Neq checks if a column does not equal a value.
// Neq builds "col <> ?"
func Neq(col string, val any) Predicate {
	return Cond(col+" <> ?", val)
}

// Gt checks if a column is greater than a value.
// Gt builds "col > ?"
func Gt(col string, val any) Predicate {
	return Cond(col+" > ?", val)
}

// Gte checks if a column is greater than or equal to a value.
// Gte builds "col >= ?"
func Gte(col string, val any) Predicate {
	return Cond(col+" >= ?", val)
}

// Lt checks if a column is less than a value.
// Lt builds "col < ?"
func Lt(col string, val any) Predicate {
	return Cond(col+" < ?", val)
}

// Lte checks if a column is less than or equal to a value.
// Lte builds "col <= ?"
func Lte(col string, val any) Predicate {
	return Cond(col+" <= ?", val)
}

// Like checks if a column matches a pattern using SQL LIKE.
// Like builds "col LIKE ?"
func Like(col string, val string) Predicate {
	return Cond(col+" LIKE ?", val)
}

// ILike checks if a column matches a pattern using SQL ILIKE (case-insensitive).
// ILike builds "col ILIKE ?"
func ILike(col string, pattern string) Predicate {
	return Cond(col+" ILIKE ?", pattern)
}

// In checks if a column's value is in a list of values.
// In builds "col IN (?, ?, ...)" with the appropriate number of placeholders.
func In(col string, vals ...any) Predicate {
	if len(vals) == 0 {
		panic("In: at least one value required")
	}

	placeholders := make([]string, len(vals))
	for i := range vals {
		placeholders[i] = "?"
	}
	return Cond(col+" IN ("+strings.Join(placeholders, ", ")+")", vals...)
}

// NotIn checks if a column's value is not in a list of values.
// NotIn builds "col NOT IN (?, ?, ...)" with the appropriate number of placeholders.
func NotIn(col string, vals ...any) Predicate {
	return Not(In(col, vals...))
}

// Between checks if a column's value is between two values (inclusive).
// Between builds "col BETWEEN ? AND ?" with the start and end values as arguments.
func Between(col string, from, to any) Predicate {
	return Cond(col+" BETWEEN ? AND ?", from, to)
}

// BetweenExclusive checks if a column's value is between two values (exclusive).
// It builds "(col > ? AND col < ?)" with the start and end values as arguments.
func BetweenExclusive(col string, from, to any) Predicate {
	return And(Gt(col, from), Lt(col, to))
}
