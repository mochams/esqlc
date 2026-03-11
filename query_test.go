package esqlc

import (
	"testing"
)

func TestQueryBuild(t *testing.T) {
	tests := []struct {
		name     string
		query    *query
		wantSQL  string
		wantArgs []any
	}{
		// --- base SQL only ---
		{
			name:    "base SQL only",
			query:   newQuery("SELECT * FROM users", DialectPostgres),
			wantSQL: "SELECT * FROM users",
		},

		// --- WHERE ---
		{
			name:     "single where condition",
			query:    newQuery("SELECT * FROM users", DialectPostgres).Where(Eq("status", "active")),
			wantSQL:  "SELECT * FROM users WHERE status = $1",
			wantArgs: []any{"active"},
		},
		{
			name: "compound where with And",
			query: newQuery("SELECT * FROM users", DialectPostgres).Where(
				And(Eq("status", "active"), Gt("age", 18)),
			),
			wantSQL:  "SELECT * FROM users WHERE (status = $1 AND age > $2)",
			wantArgs: []any{"active", 18},
		},
		{
			name: "compound where with Or",
			query: newQuery("SELECT * FROM users", DialectPostgres).Where(
				Or(Eq("status", "active"), Eq("status", "pending")),
			),
			wantSQL:  "SELECT * FROM users WHERE (status = $1 OR status = $2)",
			wantArgs: []any{"active", "pending"},
		},
		{
			name: "nested predicates in where",
			query: newQuery("SELECT * FROM users", DialectPostgres).Where(
				And(
					Eq("status", "active"),
					Or(Eq("role", "admin"), Eq("role", "mod")),
				),
			),
			wantSQL:  "SELECT * FROM users WHERE (status = $1 AND (role = $2 OR role = $3))",
			wantArgs: []any{"active", "admin", "mod"},
		},

		// -- Chaining multiple where calls ---
		{
			name: "multiple where calls with And",
			query: newQuery("SELECT * FROM users", DialectPostgres).
				Where(Eq("status", "active")).
				Where(Gt("age", 18)),
			wantSQL:  "SELECT * FROM users WHERE status = $1 AND age > $2",
			wantArgs: []any{"active", 18},
		},
		{
			name: "multiple complex where calls with And",
			query: newQuery("SELECT * FROM users", DialectPostgres).
				Where(Eq("status", "active")).
				Where(Eq("age", 18)).Where(Or(
				Eq("role", "admin"),
				Eq("role", "mod"),
			)),
			wantSQL:  "SELECT * FROM users WHERE status = $1 AND age = $2 AND (role = $3 OR role = $4)",
			wantArgs: []any{"active", 18, "admin", "mod"},
		},
		{
			name: "multiple chains with Exclude",
			query: newQuery("SELECT * FROM users", DialectPostgres).Where(Eq("status", "active")).
				Where(Eq("age", 18)).Exclude(Eq("role", "admin")),
			wantSQL:  "SELECT * FROM users WHERE status = $1 AND age = $2 AND NOT (role = $3)",
			wantArgs: []any{"active", 18, "admin"},
		},

		// --- Exclude ---
		{
			name: "exclude with Not",
			query: newQuery("SELECT * FROM users", DialectPostgres).Exclude(
				Eq("status", "inactive"),
			),
			wantSQL:  "SELECT * FROM users WHERE NOT (status = $1)",
			wantArgs: []any{"inactive"},
		},
		{
			name: "exclude with multiple predicates",
			query: newQuery("SELECT * FROM users", DialectPostgres).Exclude(
				Eq("status", "inactive"),
				Eq("deleted_at", nil),
				IsNotNull("user_group"),
				Or(Eq("role", "admin"), Eq("role", "mod")),
				And(Eq("country", "US"), Eq("age", 30)),
				Not(Eq("email_verified", true)),
			),
			wantSQL:  "SELECT * FROM users WHERE NOT ((status = $1 AND deleted_at = $2 AND user_group IS NOT NULL AND (role = $3 OR role = $4) AND (country = $5 AND age = $6) AND NOT (email_verified = $7)))",
			wantArgs: []any{"inactive", nil, "admin", "mod", "US", 30, true},
		},
		{
			name: "exclude with Not and And",
			query: newQuery("SELECT * FROM users", DialectPostgres).Exclude(
				And(Eq("status", "inactive"), Eq("deleted_at", nil)),
			),
			wantSQL:  "SELECT * FROM users WHERE NOT ((status = $1 AND deleted_at = $2))",
			wantArgs: []any{"inactive", nil},
		},

		// --- GROUP BY ---
		{
			name:    "single group by",
			query:   newQuery("SELECT status, COUNT(*) FROM users", DialectPostgres).GroupBy("status"),
			wantSQL: "SELECT status, COUNT(*) FROM users GROUP BY status",
		},
		{
			name:    "multiple group by",
			query:   newQuery("SELECT status, role, COUNT(*) FROM users", DialectPostgres).GroupBy("status", "role"),
			wantSQL: "SELECT status, role, COUNT(*) FROM users GROUP BY status, role",
		},
		{
			name: "where and group by",
			query: newQuery("SELECT status, COUNT(*) FROM users", DialectPostgres).
				Where(Eq("deleted_at", nil)).
				GroupBy("status"),
			wantSQL:  "SELECT status, COUNT(*) FROM users WHERE deleted_at = $1 GROUP BY status",
			wantArgs: []any{nil},
		},

		// --- HAVING ---
		{
			name: "having",
			query: newQuery("SELECT status, COUNT(*) FROM users", DialectPostgres).
				GroupBy("status").
				Having(Gt("COUNT(*)", 5)),
			wantSQL:  "SELECT status, COUNT(*) FROM users GROUP BY status HAVING COUNT(*) > $1",
			wantArgs: []any{5},
		},
		{
			name: "where group by and having",
			query: newQuery("SELECT status, COUNT(*) FROM users", DialectPostgres).
				Where(IsNotNull("deleted_at")).
				GroupBy("status").
				Having(Gt("COUNT(*)", 5)),
			wantSQL:  "SELECT status, COUNT(*) FROM users WHERE deleted_at IS NOT NULL GROUP BY status HAVING COUNT(*) > $1",
			wantArgs: []any{5},
		},

		// --- ORDER BY ---
		{
			name:    "single order by",
			query:   newQuery("SELECT * FROM users", DialectPostgres).OrderBy("created_at DESC"),
			wantSQL: "SELECT * FROM users ORDER BY created_at DESC",
		},
		{
			name:    "multiple order by",
			query:   newQuery("SELECT * FROM users", DialectPostgres).OrderBy("last_name ASC", "first_name ASC"),
			wantSQL: "SELECT * FROM users ORDER BY last_name ASC, first_name ASC",
		},
		{
			name: "where and order by",
			query: newQuery("SELECT * FROM users", DialectPostgres).
				Where(Eq("status", "active")).
				OrderBy("created_at DESC"),
			wantSQL:  "SELECT * FROM users WHERE status = $1 ORDER BY created_at DESC",
			wantArgs: []any{"active"},
		},

		// --- LIMIT ---
		{
			name:     "limit only",
			query:    newQuery("SELECT * FROM users", DialectPostgres).Limit(10),
			wantSQL:  "SELECT * FROM users LIMIT $1",
			wantArgs: []any{10},
		},

		// --- OFFSET ---
		{
			name:     "offset only",
			query:    newQuery("SELECT * FROM users", DialectPostgres).Offset(20),
			wantSQL:  "SELECT * FROM users OFFSET $1",
			wantArgs: []any{20},
		},
		{
			name:     "limit and offset",
			query:    newQuery("SELECT * FROM users", DialectPostgres).Limit(10).Offset(20),
			wantSQL:  "SELECT * FROM users LIMIT $1 OFFSET $2",
			wantArgs: []any{10, 20},
		},

		// --- full query ---
		{
			name: "full query all clauses",
			query: newQuery("SELECT status, COUNT(*) FROM users", DialectPostgres).
				Where(IsNotNull("deleted_at")).
				GroupBy("status").
				Having(Gt("COUNT(*)", 5)).
				OrderBy("created_at DESC").
				Limit(10).
				Offset(20),
			wantSQL:  "SELECT status, COUNT(*) FROM users WHERE deleted_at IS NOT NULL GROUP BY status HAVING COUNT(*) > $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
			wantArgs: []any{5, 10, 20},
		},

		// --- placeholder numbering continuity ---
		{
			name: "placeholder counter is continuous across clauses",
			query: newQuery("SELECT * FROM users", DialectPostgres).
				Where(Eq("status", "active")).
				Having(Gt("COUNT(*)", 5)).
				Limit(10).
				Offset(20),
			wantSQL:  "SELECT * FROM users WHERE status = $1 HAVING COUNT(*) > $2 LIMIT $3 OFFSET $4",
			wantArgs: []any{"active", 5, 10, 20},
		},

		// --- dialects ---
		{
			name: "mysql dialect",
			query: newQuery("SELECT * FROM users", DialectMySQL).
				Where(Eq("status", "active")).
				Limit(10),
			wantSQL:  "SELECT * FROM users WHERE status = ? LIMIT ?",
			wantArgs: []any{"active", 10},
		},
		{
			name: "sqlserver dialect",
			query: newQuery("SELECT * FROM users", DialectSQLServer).
				Where(Eq("status", "active")).
				Limit(10),
			wantSQL:  "SELECT * FROM users WHERE status = @p1 LIMIT @p2",
			wantArgs: []any{"active", 10},
		},
		{
			name: "oracle dialect",
			query: newQuery("SELECT * FROM users", DialectOracle).
				Where(Eq("status", "active")).
				Limit(10),
			wantSQL:  "SELECT * FROM users WHERE status = :1 LIMIT :2",
			wantArgs: []any{"active", 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := tt.query.Build()

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL:\ngot  %q\nwant %q", gotSQL, tt.wantSQL)
			}
			if !equalArgs(gotArgs, tt.wantArgs) {
				t.Errorf("args:\ngot  %v\nwant %v", gotArgs, tt.wantArgs)
			}
		})
	}
}
