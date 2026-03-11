# esqlc — Go Enhanced SQL

**esqlc** is a lightweight SQL toolkit for Go that lets you:

* keep SQL in `.sql` files (yesql-style)
* load queries by name
* compose dynamic predicates safely at runtime

No ORM. No DSL. Just **SQL with powerful runtime composition**.

---

## Why esqlc?

Many Go applications prefer keeping SQL **in `.sql` files** instead of embedding it in code. This keeps queries readable, easier to review, and easier to evolve as they grow in complexity.

Libraries like **yesql** popularized this pattern: write SQL in SQL, load it by name, and execute it from your application.

However, real-world APIs often need to **dynamically respond to request parameters**:

* optional filters
* search queries
* pagination
* conditional predicates

This usually leads to manually assembling SQL strings in code.

**esqlc bridges this gap.**

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
    Where(esqlc.And(
        esqlc.Eq("status", "active"),
        esqlc.Gt("age", 18),
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
go get github.com/mochams/esqlc
```

---

## Quick Start

### Ad-hoc queries

```go
reg := esqlc.NewRegistry(esqlc.DialectPostgres)

sql, args := reg.Query("SELECT * FROM users").
    Where(esqlc.And(
        esqlc.Eq("status", "active"),
        esqlc.Gt("age", 18),
    )).
    OrderBy("created_at DESC").
    Limit(20).
    Offset(0).
    Build()
```

Result:

```sql
SELECT * FROM users
WHERE status = $1 AND age > $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4
```

Args:

```txt
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
reg := esqlc.NewRegistry(esqlc.DialectPostgres)

if err := reg.Load("queries/users.sql"); err != nil {
    log.Fatal(err)
}

query := reg.MustGet("listUsers")

sql, args := query.
    Where(esqlc.Eq("role", "admin")).
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

reg := esqlc.NewRegistry(esqlc.DialectPostgres)

if err := reg.WalkFS(sqlFiles, "queries"); err != nil {
    log.Fatal(err)
}
```

---

## Query Methods

| Method             | Description                                                   |
| ------------------ | ------------------------------------------------------------- |
| `Where(pred)`      | Add a WHERE condition. Multiple calls are ANDed together      |
| `Exclude(pred)`    | Add a negated WHERE condition — equivalent to `AND NOT (...)` |
| `GroupBy(cols...)` | Add a GROUP BY clause                                         |
| `Having(pred)`     | Add a HAVING condition                                        |
| `OrderBy(cols...)` | Add an ORDER BY clause                                        |
| `Limit(n)`         | Add a LIMIT clause                                            |
| `Offset(n)`        | Add an OFFSET clause                                          |
| `Build()`          | Finalize and return the SQL string and arguments              |

---

## Predicates

### Simple conditions

```go
esqlc.Cond("age > ?", 30)

esqlc.Eq("status", "active")
esqlc.Neq("status", "banned")

esqlc.Lt("age", 18)
esqlc.Lte("age", 18)

esqlc.Gt("age", 65)
esqlc.Gte("age", 65)

esqlc.Like("name", "%john%")
esqlc.ILike("name", "%john%") // Postgres only

esqlc.IsNull("deleted_at")
esqlc.IsNotNull("deleted_at")
```

---

### Range conditions

```go
esqlc.Between("age", 18, 65)

esqlc.BetweenExclusive("age", 18, 65)
```

---

### Set conditions

```go
esqlc.In("status", "active", "pending")

esqlc.NotIn("status", "banned", "deleted")
```

---

### Logical combinators

```go
esqlc.And(
    esqlc.Eq("status", "active"),
    esqlc.Gt("age", 18),
)

esqlc.Or(
    esqlc.Eq("role", "admin"),
    esqlc.Eq("role", "mod"),
)

esqlc.Not(
    esqlc.Eq("status", "banned"),
)
```

They compose naturally:

```go
esqlc.And(
    esqlc.Eq("status", "active"),
    esqlc.Or(
        esqlc.Eq("role", "admin"),
        esqlc.Eq("role", "mod"),
    ),
)
```

---

## SQL Dialects

esqlc rewrites placeholders automatically.

### Postgres

```sql
$1, $2, $3
```

### MySQL / SQLite

```sql
?, ?, ?
```

### SQL Server

```sql
@p1, @p2
```

### Oracle

```sql
:1, :2
```

Example:

```go
reg := esqlc.NewRegistry(esqlc.DialectSQLServer)

reg.Query("SELECT * FROM users").
    Where(esqlc.Eq("id", 1)).
    Build()
```

---

## Registry

```go
reg := esqlc.NewRegistry(esqlc.DialectPostgres)

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

q := reg.Query("Select * FROM users")
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
* esqlc rewrites placeholders during `Build()`
* Query names must be unique
* Empty query bodies are rejected

---

## Inspiration

esqlc is inspired by [**yesql**](https://github.com/krisajenkins/yesql), a Clojure library by Kris Jenkins that encourages writing SQL in SQL rather than embedding it in application code.

esqlc extends the idea with **composable predicates**, making it easier to build dynamic queries for APIs and search endpoints.

---
