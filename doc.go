// Package esqlc provides a lightweight SQL query builder for Go with support
// for named queries loaded from .sql files and composable dynamic predicates.
//
// # Overview
//
// esqlc is inspired by yesql (https://github.com/krisajenkins/yesql) — the idea
// that SQL is a good language and should be written in SQL files rather than
// embedded in application code. esqlc brings that philosophy to Go and extends
// it with a composable predicate system for the dynamic filtering that API
// endpoints typically require.
//
// The library has three core components:
//
//   - Registry — loads and caches named queries from .sql files
//   - Query — a chainable builder for attaching dynamic clauses
//   - Predicates — composable conditions for WHERE and HAVING clauses
//
// # Quick Start
//
// Write SQL in .sql files using yesql-style annotations:
//
//	-- name: listUsers
//	SELECT * FROM users
//
//	-- name: getUserByID
//	SELECT * FROM users WHERE id = ?
//
// Load at startup and query at runtime:
//
//	reg := esqlc.NewRegistry(esqlc.DialectPostgres)
//	if err := reg.LoadDir("queries/"); err != nil {
//	    log.Fatal(err)
//	}
//
//	sql, args := reg.MustGet("listUsers").
//	    Where(esqlc.And(
//	        esqlc.Eq("status", "active"),
//	        esqlc.Gt("age", 18),
//	    )).
//	    OrderBy("created_at DESC").
//	    Limit(10).
//	    Build()
//
//	db.QueryContext(ctx, sql, args...)
//
// Ad-hoc queries are supported through the registry as well:
//
//	sql, args := reg.Query("SELECT * FROM orders").
//	    Where(esqlc.Eq("user_id", userID)).
//	    Build()
//
// # SQL Injection Safety
//
// esqlc never interpolates values into SQL strings. Every value passed to a
// predicate becomes a bound argument. Placeholders are rewritten to the correct
// dialect format at build time:
//
//	reg.Query("SELECT * FROM users").
//	    Where(esqlc.Eq("name", "robert'); DROP TABLE users;--")).
//	    Build()
//	// SELECT * FROM users WHERE name = $1
//	// args: ["robert'); DROP TABLE users;--"]
//
// # Predicates
//
// Predicates are composable conditions that can be combined with And, Or, and Not:
//
//	esqlc.And(
//	    esqlc.Eq("status", "active"),
//	    esqlc.Or(
//	        esqlc.Eq("role", "admin"),
//	        esqlc.Eq("role", "mod"),
//	    ),
//	    esqlc.Not(esqlc.IsNull("deleted_at")),
//	)
//	// (status = ? AND (role = ? OR role = ?) AND NOT (deleted_at IS NULL))
//
// Available predicates: Cond, Eq, Neq, Gt, Gte, Lt, Lte, Like, ILike,
// IsNull, IsNotNull, In, NotIn, Between, RangeOpen, RangeClosedOpen,
// RangeOpenClosed, And, Or, Not.
//
// # Dialects
//
// Dialect is configured once on the registry and applied to all queries:
//
//	esqlc.NewRegistry(esqlc.DialectPostgres)   // $1, $2, ...
//	esqlc.NewRegistry(esqlc.DialectMySQL)      // ?, ?, ...
//	esqlc.NewRegistry(esqlc.DialectSQLServer)  // @p1, @p2, ...
//	esqlc.NewRegistry(esqlc.DialectOracle)     // :1, :2, ...
//
// # SQL File Format
//
// Files use yesql-style -- name: annotations. A single file can contain
// multiple named queries. Use ? as the placeholder regardless of dialect —
// esqlc rewrites them at build time.
//
//	-- name: listUsers
//	SELECT * FROM users
//
//	-- name: getUserByID
//	SELECT * FROM users WHERE id = ?
//
// Query names must be unique within a file and across all loaded files.
// Duplicates and empty query bodies are caught at load time.
//
// # Embedded Files
//
// The registry supports embedded SQL files via fs.FS and embed.FS:
//
//	//go:embed queries
//	var sqlFiles embed.FS
//
//	reg := esqlc.NewRegistry(esqlc.DialectPostgres)
//	if err := reg.WalkFS(sqlFiles, "queries"); err != nil {
//	    log.Fatal(err)
//	}
package esqlc
