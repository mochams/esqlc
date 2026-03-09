# gesql — Go Enhanced SQL

**gesql** is a lightweight SQL toolkit for Go that lets you:

* keep SQL in `.sql` files (yesql-style)
* load queries by name
* compose dynamic predicates safely at runtime

No ORM. No DSL. Just **SQL with powerful runtime composition**.

---

## Why gesql?

Many Go applications prefer keeping SQL **in `.sql` files** instead of embedding it in code. This keeps queries readable, easier to review, and easier to evolve as they grow in complexity.

Libraries like **yesql** popularized this pattern: write SQL in SQL, load it by name, and execute it from your application.

However, real-world APIs often need to **dynamically respond to request parameters**:

* optional filters
* search queries
* pagination
* conditional predicates

This usually leads to manually assembling SQL strings in code.

**gesql bridges this gap.**

You can:

* keep **base queries in `.sql` files**
* load them by name
* safely **compose dynamic predicates at runtime**

```sql
-- name: listUsers
SELECT * FROM users
```

```go
sql, args := reg.MustGet("listUsers").
    Where(gesql.And(
        gesql.Eq("status", "active"),
        gesql.Gt("age", 18),
    )).
    OrderBy("created_at DESC").
    Limit(20).
    Build()
```

This keeps your **core SQL declarative**, while allowing **flexible runtime filtering**.

---

## Features

* Named SQL queries loaded from `.sql` files (yesql-style)
* Composable predicates (`And`, `Or`, `Eq`, `In`, `Between`, etc.)
* Multi-dialect placeholder rewriting

  * Postgres `$1`
  * MySQL / SQLite `?`
  * SQL Server `@p1`
  * Oracle `:1`
* Full clause support

  * `WHERE`
  * `GROUP BY`
  * `HAVING`
  * `ORDER BY`
  * `LIMIT`
  * `OFFSET`
* Works with `fs.FS` and `embed.FS`
* Zero external dependencies

---

## Installation

```bash
go get github.com/mochams/gesql
```

---

## Quick Start

### Ad-hoc queries

```go
sql, args := gesql.NewQuery("SELECT * FROM users").
    Where(gesql.And(
        gesql.Eq("status", "active"),
        gesql.Gt("age", 18),
    )).
    OrderBy("created_at DESC").
    Limit(20).
    Offset(0).
    Build()
```

Result:

```sql
SELECT * FROM users
WHERE (status = $1 AND age > $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4
```

Args:

```
["active", 18, 20, 0]
```

---

### Using SQL files (yesql style)

Write queries in `.sql` files:

```sql
-- name: listUsers
SELECT * FROM users

-- name: getUserByID
SELECT * FROM users WHERE id = ?

-- name: listActiveOrders
SELECT *
FROM orders
WHERE status = ?
AND deleted_at IS NULL
```

Load and use them:

```go
reg := gesql.NewRegistry()

if err := reg.Load("queries/users.sql"); err != nil {
    log.Fatal(err)
}

q := reg.MustGet("listUsers")

sql, args := q.
    Where(gesql.Eq("role", "admin")).
    OrderBy("created_at DESC").
    Limit(10).
    Build()
```

---

### Embedded SQL files

Works with Go's `embed`.

```go
//go:embed queries
var sqlFiles embed.FS

reg := gesql.NewRegistry()

if err := reg.WalkFS(sqlFiles, "queries"); err != nil {
    log.Fatal(err)
}
```

---

## Predicates

### Simple conditions

```go
gesql.Cond("age > ?", 30)

gesql.Eq("status", "active")
gesql.Neq("status", "banned")

gesql.Lt("age", 18)
gesql.Lte("age", 18)

gesql.Gt("age", 65)
gesql.Gte("age", 65)

gesql.Like("name", "%john%")
gesql.ILike("name", "%john%") // Postgres only

gesql.IsNull("deleted_at")
gesql.IsNotNull("deleted_at")
```

---

### Range conditions

```go
gesql.Between("age", 18, 65)

gesql.BetweenExclusive("age", 18, 65)
```

---

### Set conditions

```go
gesql.In("status", "active", "pending")

gesql.NotIn("status", "banned", "deleted")
```

---

### Logical combinators

```go
gesql.And(
    gesql.Eq("status", "active"),
    gesql.Gt("age", 18),
)

gesql.Or(
    gesql.Eq("role", "admin"),
    gesql.Eq("role", "mod"),
)

gesql.Not(
    gesql.Eq("status", "banned"),
)
```

They compose naturally:

```go
gesql.And(
    gesql.Eq("status", "active"),
    gesql.Or(
        gesql.Eq("role", "admin"),
        gesql.Eq("role", "mod"),
    ),
)
```

---

## Query Builder

```go
sql, args := gesql.NewQuery(
    "SELECT status, COUNT(*) FROM users",
).
    Where(gesql.IsNotNull("deleted_at")).
    GroupBy("status").
    Having(gesql.Gt("COUNT(*)", 5)).
    OrderBy("created_at DESC").
    Limit(10).
    Offset(20).
    Build()
```

---

## SQL Dialects

gesql rewrites placeholders automatically.

### Postgres (default)

```
$1, $2, $3
```

### MySQL / SQLite

```
?, ?, ?
```

### SQL Server

```
@p1, @p2
```

### Oracle

```
:1, :2
```

Example:

```go
gesql.NewQuery("SELECT * FROM users").
    Dialect(gesql.DialectSQLServer).
    Where(gesql.Eq("id", 1)).
    Build()
```

---

## Registry

```go
reg := gesql.NewRegistry()

// load single file
reg.Load("queries/users.sql")

// load all .sql files in a directory
reg.LoadDir("queries")

// load from fs.FS
reg.LoadFS(fsys, "queries/users.sql")

// walk directory in fs.FS
reg.WalkFS(fsys, "queries")
```

Get queries:

```go
q, err := reg.Get("listUsers")

q := reg.MustGet("listUsers")
```

---

## SQL File Format

Queries are defined using `-- name:` annotations.

```sql
-- name: listUsers
SELECT * FROM users

-- name: getUserByID
SELECT * FROM users WHERE id = ?

-- name: deleteUser
DELETE FROM users WHERE id = ?
```

Rules:

* Use `?` placeholders regardless of dialect
* gesql rewrites placeholders during `Build()`
* Query names must be unique
* Empty query bodies are rejected

---

## Inspiration

gesql is inspired by [**yesql**](https://github.com/krisajenkins/yesql), a Clojure library by Kris Jenkins that encourages writing SQL in SQL rather than embedding it in application code.

gesql extends the idea with **composable predicates**, making it easier to build dynamic queries for APIs and search endpoints.

---
