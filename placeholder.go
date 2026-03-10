package esqlc

import (
	"strconv"
	"strings"
)

// Dialect represents the SQL dialect to use for building queries.
// It determines how placeholders are formatted (e.g., "?" for MySQL, "$1" for Postgres).
type Dialect int

// Constants for supported SQL dialects.
const (
	DialectPostgres  Dialect = iota // $1, $2
	DialectMySQL                    // ?, ?
	DialectSQLite                   // ?, ?  (same as MySQL)
	DialectSQLServer                // @p1, @p2
	DialectOracle                   // :1, :2
)

// State constants for the placeholder rewriting state machine.
const (
	stateNormal uint8 = iota
	stateSingle
	stateDouble
	stateLineComment
	stateBlockComment
)

// rewritePlaceholders rewrites '?' placeholders for the target dialect,
// safely ignoring string literals and comments
func rewritePlaceholders(sql string, dialect Dialect) string {
	if dialect == DialectMySQL || dialect == DialectSQLite {
		return sql
	}

	var buf strings.Builder
	buf.Grow(len(sql))

	var tmp [20]byte
	state := stateNormal
	n := 1
	i := 0

	for i < len(sql) {
		c := sql[i]

		switch state {
		case stateNormal:
			switch c {
			case '\'':
				state = stateSingle
				buf.WriteByte(c)
			case '"':
				state = stateDouble
				buf.WriteByte(c)
			case '-':
				if i+1 < len(sql) && sql[i+1] == '-' {
					buf.WriteString("--")
					i++
					state = stateLineComment
				} else {
					buf.WriteByte(c)
				}

			case '/':
				if i+1 < len(sql) && sql[i+1] == '*' {
					buf.WriteString("/*")
					i++
					state = stateBlockComment
				} else {
					buf.WriteByte(c)
				}

			case '?':
				switch dialect {
				case DialectPostgres:
					buf.WriteByte('$')
					buf.Write(strconv.AppendInt(tmp[:0], int64(n), 10))
				case DialectSQLServer:
					buf.WriteString("@p")
					buf.Write(strconv.AppendInt(tmp[:0], int64(n), 10))
				case DialectOracle:
					buf.WriteByte(':')
					buf.Write(strconv.AppendInt(tmp[:0], int64(n), 10))
				default:
					buf.WriteByte('?')
				}
				n++

			default:
				buf.WriteByte(c)
			}

		case stateSingle:
			buf.WriteByte(c)
			if c == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					buf.WriteByte('\'')
					i++
				} else {
					state = stateNormal
				}
			}

		case stateDouble:
			buf.WriteByte(c)
			if c == '"' {
				state = stateNormal
			}

		case stateLineComment:
			buf.WriteByte(c)
			if c == '\n' {
				state = stateNormal
			}

		case stateBlockComment:
			buf.WriteByte(c)
			if c == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				buf.WriteByte('/')
				i++
				state = stateNormal
			}
		}

		i++
	}

	return buf.String()
}
