package gesql

import (
	"testing"
)

func TestBuilderWrite(t *testing.T) {
	tests := []struct {
		name    string
		writes  []string
		wantSQL string
	}{
		{
			name:    "single write",
			writes:  []string{"SELECT * FROM users"},
			wantSQL: "SELECT * FROM users",
		},
		{
			name:    "multiple writes concatenate",
			writes:  []string{"SELECT * FROM users", " WHERE ", "age > ?"},
			wantSQL: "SELECT * FROM users WHERE age > ?",
		},
		{
			name:    "empty write",
			writes:  []string{""},
			wantSQL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &builder{}
			for _, s := range tt.writes {
				b.write(s)
			}
			if got := b.String(); got != tt.wantSQL {
				t.Errorf("SQL: got %q, want %q", got, tt.wantSQL)
			}
		})
	}
}

func TestBuilderArg(t *testing.T) {
	tests := []struct {
		name     string
		args     []any
		wantArgs []any
	}{
		{
			name:     "single arg",
			args:     []any{1},
			wantArgs: []any{1},
		},
		{
			name:     "multiple args",
			args:     []any{1, "active", true},
			wantArgs: []any{1, "active", true},
		},
		{
			name:     "no args",
			args:     []any{},
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &builder{}
			for _, a := range tt.args {
				b.arg(a)
			}
			if !equalArgs(b.Args(), tt.wantArgs) {
				t.Errorf("args: got %v, want %v", b.Args(), tt.wantArgs)
			}
		})
	}
}
