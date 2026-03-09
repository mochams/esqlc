package gesql

import "testing"

// --- Helper functions for testing ---

func buildPredicate(p Predicate) (string, []any) {
	b := &builder{}
	p.build(b)
	return b.String(), b.Args()
}

func equalArgs(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- Cond ---

func TestCond(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		args     []any
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "no args",
			expr:     "deleted_at IS NULL",
			wantSQL:  "deleted_at IS NULL",
			wantArgs: nil,
		},
		{
			name:     "single arg",
			expr:     "age > ?",
			args:     []any{30},
			wantSQL:  "age > ?",
			wantArgs: []any{30},
		},
		{
			name:     "multiple args",
			expr:     "age BETWEEN ? AND ?",
			args:     []any{18, 65},
			wantSQL:  "age BETWEEN ? AND ?",
			wantArgs: []any{18, 65},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Cond(tt.expr, tt.args...)
			gotSQL, gotArgs := buildPredicate(p)

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL: got %q, want %q", gotSQL, tt.wantSQL)
			}
			if !equalArgs(gotArgs, tt.wantArgs) {
				t.Errorf("args: got %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

// --- And ---

func TestAnd(t *testing.T) {
	tests := []struct {
		name     string
		preds    []Predicate
		wantSQL  string
		wantArgs []any
	}{
		{
			name:    "empty",
			preds:   []Predicate{},
			wantSQL: "",
		},
		{
			name:     "single predicate skips parens",
			preds:    []Predicate{Cond("x = ?", 1)},
			wantSQL:  "x = ?",
			wantArgs: []any{1},
		},
		{
			name: "two predicates",
			preds: []Predicate{
				Cond("age > ?", 30),
				Cond("status = ?", "active"),
			},
			wantSQL:  "(age > ? AND status = ?)",
			wantArgs: []any{30, "active"},
		},
		{
			name: "three predicates",
			preds: []Predicate{
				Cond("a = ?", 1),
				Cond("b = ?", 2),
				Cond("c = ?", 3),
			},
			wantSQL:  "(a = ? AND b = ? AND c = ?)",
			wantArgs: []any{1, 2, 3},
		},
		{
			name: "nested And",
			preds: []Predicate{
				Cond("a = ?", 1),
				And(Cond("b = ?", 2), Cond("c = ?", 3)),
			},
			wantSQL:  "(a = ? AND (b = ? AND c = ?))",
			wantArgs: []any{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := And(tt.preds...)
			gotSQL, gotArgs := buildPredicate(p)

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL: got %q, want %q", gotSQL, tt.wantSQL)
			}
			if !equalArgs(gotArgs, tt.wantArgs) {
				t.Errorf("args: got %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

// --- Or ---

func TestOr(t *testing.T) {
	tests := []struct {
		name     string
		preds    []Predicate
		wantSQL  string
		wantArgs []any
	}{
		{
			name:    "empty",
			preds:   []Predicate{},
			wantSQL: "",
		},
		{
			name:     "single predicate skips parens",
			preds:    []Predicate{Cond("x = ?", 1)},
			wantSQL:  "x = ?",
			wantArgs: []any{1},
		},
		{
			name: "two predicates",
			preds: []Predicate{
				Cond("age < ?", 18),
				Cond("status = ?", "inactive"),
			},
			wantSQL:  "(age < ? OR status = ?)",
			wantArgs: []any{18, "inactive"},
		},
		{
			name: "three predicates",
			preds: []Predicate{
				Cond("a = ?", 1),
				Cond("b = ?", 2),
				Cond("c = ?", 3),
			},
			wantSQL:  "(a = ? OR b = ? OR c = ?)",
			wantArgs: []any{1, 2, 3},
		},
		{
			name: "nested Or inside And",
			preds: []Predicate{
				Cond("role = ?", "admin"),
				Or(Cond("region = ?", "us"), Cond("region = ?", "eu")),
			},
			wantSQL:  "(role = ? OR (region = ? OR region = ?))",
			wantArgs: []any{"admin", "us", "eu"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Or(tt.preds...)
			gotSQL, gotArgs := buildPredicate(p)

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL: got %q, want %q", gotSQL, tt.wantSQL)
			}
			if !equalArgs(gotArgs, tt.wantArgs) {
				t.Errorf("args: got %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

// --- Not ---

func TestNot(t *testing.T) {
	tests := []struct {
		name     string
		pred     Predicate
		wantSQL  string
		wantArgs []any
	}{
		{
			name:    "simple cond",
			pred:    Cond("deleted_at IS NULL"),
			wantSQL: "NOT (deleted_at IS NULL)",
		},
		{
			name:     "cond with arg",
			pred:     Cond("status = ?", "banned"),
			wantSQL:  "NOT (status = ?)",
			wantArgs: []any{"banned"},
		},
		{
			name: "not of And",
			pred: And(
				Cond("a = ?", 1),
				Cond("b = ?", 2),
			),
			wantSQL:  "NOT ((a = ? AND b = ?))",
			wantArgs: []any{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Not(tt.pred)
			gotSQL, gotArgs := buildPredicate(p)

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL: got %q, want %q", gotSQL, tt.wantSQL)
			}
			if !equalArgs(gotArgs, tt.wantArgs) {
				t.Errorf("args: got %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestConveniencePredicates(t *testing.T) {
	tests := []struct {
		name     string
		pred     Predicate
		wantSQL  string
		wantArgs []any
	}{
		// --- Eq ---
		{
			name:     "Eq",
			pred:     Eq("status", "active"),
			wantSQL:  "status = ?",
			wantArgs: []any{"active"},
		},
		{
			name:     "Eq int",
			pred:     Eq("age", 30),
			wantSQL:  "age = ?",
			wantArgs: []any{30},
		},

		// --- Neq ---
		{
			name:     "Neq",
			pred:     Neq("status", "banned"),
			wantSQL:  "status <> ?",
			wantArgs: []any{"banned"},
		},

		// --- Lt ---
		{
			name:     "Lt",
			pred:     Lt("age", 18),
			wantSQL:  "age < ?",
			wantArgs: []any{18},
		},

		// --- Lte ---
		{
			name:     "Lte",
			pred:     Lte("age", 18),
			wantSQL:  "age <= ?",
			wantArgs: []any{18},
		},

		// --- Gt ---
		{
			name:     "Gt",
			pred:     Gt("age", 65),
			wantSQL:  "age > ?",
			wantArgs: []any{65},
		},

		// --- Gte ---
		{
			name:     "Gte",
			pred:     Gte("age", 65),
			wantSQL:  "age >= ?",
			wantArgs: []any{65},
		},

		// --- Like ---
		{
			name:     "Like prefix wildcard",
			pred:     Like("name", "%john"),
			wantSQL:  "name LIKE ?",
			wantArgs: []any{"%john"},
		},
		{
			name:     "Like suffix wildcard",
			pred:     Like("name", "john%"),
			wantSQL:  "name LIKE ?",
			wantArgs: []any{"john%"},
		},
		{
			name:     "Like both wildcards",
			pred:     Like("name", "%john%"),
			wantSQL:  "name LIKE ?",
			wantArgs: []any{"%john%"},
		},
		{
			name:     "Like no wildcard",
			pred:     Like("name", "john"),
			wantSQL:  "name LIKE ?",
			wantArgs: []any{"john"},
		},

		// --- ILike ---
		{
			name:     "ILike",
			pred:     ILike("name", "%john%"),
			wantSQL:  "name ILIKE ?",
			wantArgs: []any{"%john%"},
		},

		// --- IsNull ---
		{
			name:    "IsNull",
			pred:    IsNull("deleted_at"),
			wantSQL: "deleted_at IS NULL",
		},

		// --- IsNotNull ---
		{
			name:    "IsNotNull",
			pred:    IsNotNull("deleted_at"),
			wantSQL: "deleted_at IS NOT NULL",
		},

		// --- In ---
		{
			name:     "In single value",
			pred:     In("status", "active"),
			wantSQL:  "status IN (?)",
			wantArgs: []any{"active"},
		},
		{
			name:     "In multiple values",
			pred:     In("status", "active", "pending", "review"),
			wantSQL:  "status IN (?, ?, ?)",
			wantArgs: []any{"active", "pending", "review"},
		},
		{
			name:     "In int values",
			pred:     In("id", 1, 2, 3),
			wantSQL:  "id IN (?, ?, ?)",
			wantArgs: []any{1, 2, 3},
		},

		// --- NotIn ---
		{
			name:     "NotIn single value",
			pred:     NotIn("status", "banned"),
			wantSQL:  "NOT (status IN (?))",
			wantArgs: []any{"banned"},
		},
		{
			name:     "NotIn multiple values",
			pred:     NotIn("status", "banned", "suspended"),
			wantSQL:  "NOT (status IN (?, ?))",
			wantArgs: []any{"banned", "suspended"},
		},

		// --- Between ---
		{
			name:     "Between ints",
			pred:     Between("age", 18, 65),
			wantSQL:  "age BETWEEN ? AND ?",
			wantArgs: []any{18, 65},
		},
		{
			name:     "Between strings",
			pred:     Between("name", "a", "m"),
			wantSQL:  "name BETWEEN ? AND ?",
			wantArgs: []any{"a", "m"},
		},
		{
			name:     "Between same value",
			pred:     Between("age", 30, 30),
			wantSQL:  "age BETWEEN ? AND ?",
			wantArgs: []any{30, 30},
		},

		// --- BetweenExclusive ---
		{
			name:     "BetweenExclusive",
			pred:     BetweenExclusive("age", 18, 65),
			wantSQL:  "(age > ? AND age < ?)",
			wantArgs: []any{18, 65},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := buildPredicate(tt.pred)

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL: got %q, want %q", gotSQL, tt.wantSQL)
			}
			if !equalArgs(gotArgs, tt.wantArgs) {
				t.Errorf("args: got %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestInPanicsOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty In, got none")
		}
	}()
	In("id")
}
