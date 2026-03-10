package esqlc

import (
	"strings"
)

// builder is a helper type for building SQL strings and collecting arguments.
type builder struct {
	sql  strings.Builder
	args []any
}

// write appends a string to the SQL being built.
func (b *builder) write(s string) {
	b.sql.WriteString(s)
}

// arg appends an argument to the list of arguments being collected.
func (b *builder) arg(a any) {
	b.args = append(b.args, a)
}

// String returns the built SQL string.
func (b *builder) String() string {
	return b.sql.String()
}

// Args returns the list of arguments being collected.
func (b *builder) Args() []any {
	return b.args
}
