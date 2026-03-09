package gesql

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// --- helpers ---

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return path
}

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gesql-test-*")
	if err != nil {
		t.Fatalf("tempDir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func assertQueries(t *testing.T, reg *Registry, want map[string]string) {
	t.Helper()

	if len(reg.queries) != len(want) {
		t.Fatalf("query count: got %d, want %d", len(reg.queries), len(want))
	}

	for name, wantSQL := range want {
		q, err := reg.Get(name)
		if err != nil {
			t.Errorf("query %q not found: %v", name, err)
			continue
		}
		gotSQL, _ := q.RawBuild()
		if gotSQL != wantSQL {
			t.Errorf("query %q: got %q, want %q", name, gotSQL, wantSQL)
		}
	}
}

// --- Load ---

func TestRegistryLoad(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string // filename -> content
		loadFiles   []string          // order to load
		wantQueries map[string]string
		wantErr     bool
	}{
		{
			name: "single file single query",
			files: map[string]string{
				"users.sql": "-- name: listUsers\nSELECT * FROM users",
			},
			loadFiles: []string{"users.sql"},
			wantQueries: map[string]string{
				"listUsers": "SELECT * FROM users",
			},
		},
		{
			name: "single file multiple queries",
			files: map[string]string{
				"users.sql": "-- name: listUsers\nSELECT * FROM users\n\n-- name: getUserByID\nSELECT * FROM users WHERE id = ?",
			},
			loadFiles: []string{"users.sql"},
			wantQueries: map[string]string{
				"listUsers":   "SELECT * FROM users",
				"getUserByID": "SELECT * FROM users WHERE id = ?",
			},
		},
		{
			name: "multiple files",
			files: map[string]string{
				"users.sql":  "-- name: listUsers\nSELECT * FROM users",
				"orders.sql": "-- name: listOrders\nSELECT * FROM orders",
			},
			loadFiles: []string{"users.sql", "orders.sql"},
			wantQueries: map[string]string{
				"listUsers":  "SELECT * FROM users",
				"listOrders": "SELECT * FROM orders",
			},
		},
		{
			name: "duplicate query name across files",
			files: map[string]string{
				"users.sql":  "-- name: listUsers\nSELECT * FROM users",
				"admins.sql": "-- name: listUsers\nSELECT * FROM admins",
			},
			loadFiles: []string{"users.sql", "admins.sql"},
			wantErr:   true,
		},
		{
			name:      "file not found",
			files:     map[string]string{},
			loadFiles: []string{"nonexistent.sql"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tempDir(t)
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}

			reg := NewRegistry()
			var loadErr error
			for _, f := range tt.loadFiles {
				if err := reg.Load(filepath.Join(dir, f)); err != nil {
					loadErr = err
					break
				}
			}

			if tt.wantErr {
				if loadErr == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if loadErr != nil {
				t.Fatalf("unexpected error: %v", loadErr)
			}

			assertQueries(t, reg, tt.wantQueries)
		})
	}
}

// --- LoadFS ---

func TestRegistryLoadFS(t *testing.T) {
	tests := []struct {
		name        string
		fsys        fs.FS
		loadPaths   []string
		wantQueries map[string]string
		wantErr     bool
	}{
		{
			name: "single file single query",
			fsys: fstest.MapFS{
				"users.sql": &fstest.MapFile{
					Data: []byte("-- name: listUsers\nSELECT * FROM users"),
				},
			},
			loadPaths: []string{"users.sql"},
			wantQueries: map[string]string{
				"listUsers": "SELECT * FROM users",
			},
		},
		{
			name: "multiple files",
			fsys: fstest.MapFS{
				"users.sql": &fstest.MapFile{
					Data: []byte("-- name: listUsers\nSELECT * FROM users"),
				},
				"orders.sql": &fstest.MapFile{
					Data: []byte("-- name: listOrders\nSELECT * FROM orders"),
				},
			},
			loadPaths: []string{"users.sql", "orders.sql"},
			wantQueries: map[string]string{
				"listUsers":  "SELECT * FROM users",
				"listOrders": "SELECT * FROM orders",
			},
		},
		{
			name: "duplicate query name across files",
			fsys: fstest.MapFS{
				"users.sql": &fstest.MapFile{
					Data: []byte("-- name: listUsers\nSELECT * FROM users"),
				},
				"admins.sql": &fstest.MapFile{
					Data: []byte("-- name: listUsers\nSELECT * FROM admins"),
				},
			},
			loadPaths: []string{"users.sql", "admins.sql"},
			wantErr:   true,
		},
		{
			name:      "file not found",
			fsys:      fstest.MapFS{},
			loadPaths: []string{"nonexistent.sql"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			var loadErr error
			for _, p := range tt.loadPaths {
				if err := reg.LoadFS(tt.fsys, p); err != nil {
					loadErr = err
					break
				}
			}

			if tt.wantErr {
				if loadErr == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if loadErr != nil {
				t.Fatalf("unexpected error: %v", loadErr)
			}

			assertQueries(t, reg, tt.wantQueries)
		})
	}
}

// --- LoadDir ---

func TestRegistryLoadDir(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		wantQueries map[string]string
		wantErr     bool
	}{
		{
			name: "single sql file",
			files: map[string]string{
				"users.sql": "-- name: listUsers\nSELECT * FROM users",
			},
			wantQueries: map[string]string{
				"listUsers": "SELECT * FROM users",
			},
		},
		{
			name: "multiple sql files",
			files: map[string]string{
				"users.sql":  "-- name: listUsers\nSELECT * FROM users",
				"orders.sql": "-- name: listOrders\nSELECT * FROM orders",
			},
			wantQueries: map[string]string{
				"listUsers":  "SELECT * FROM users",
				"listOrders": "SELECT * FROM orders",
			},
		},
		{
			name: "non sql files are ignored",
			files: map[string]string{
				"users.sql": "-- name: listUsers\nSELECT * FROM users",
				"README.md": "some readme",
				"notes.txt": "some notes",
			},
			wantQueries: map[string]string{
				"listUsers": "SELECT * FROM users",
			},
		},
		{
			name: "duplicate query name across files",
			files: map[string]string{
				"users.sql":  "-- name: listUsers\nSELECT * FROM users",
				"admins.sql": "-- name: listUsers\nSELECT * FROM admins",
			},
			wantErr: true,
		},
		{
			name:    "dir not found",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dir string
			if tt.name == "dir not found" {
				dir = "/nonexistent/path"
			} else {
				dir = tempDir(t)
				for name, content := range tt.files {
					writeFile(t, dir, name, content)
				}
			}

			reg := NewRegistry()
			err := reg.LoadDir(dir)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertQueries(t, reg, tt.wantQueries)
		})
	}
}

// --- WalkFS ---

func TestRegistryWalkFS(t *testing.T) {
	tests := []struct {
		name        string
		fsys        fs.FS
		dir         string
		wantQueries map[string]string
		wantErr     bool
	}{
		{
			name: "single sql file",
			fsys: fstest.MapFS{
				"queries/users.sql": &fstest.MapFile{
					Data: []byte("-- name: listUsers\nSELECT * FROM users"),
				},
			},
			dir: "queries",
			wantQueries: map[string]string{
				"listUsers": "SELECT * FROM users",
			},
		},
		{
			name: "multiple sql files",
			fsys: fstest.MapFS{
				"queries/users.sql": &fstest.MapFile{
					Data: []byte("-- name: listUsers\nSELECT * FROM users"),
				},
				"queries/orders.sql": &fstest.MapFile{
					Data: []byte("-- name: listOrders\nSELECT * FROM orders"),
				},
			},
			dir: "queries",
			wantQueries: map[string]string{
				"listUsers":  "SELECT * FROM users",
				"listOrders": "SELECT * FROM orders",
			},
		},
		{
			name: "non sql files are ignored",
			fsys: fstest.MapFS{
				"queries/users.sql": &fstest.MapFile{
					Data: []byte("-- name: listUsers\nSELECT * FROM users"),
				},
				"queries/README.md": &fstest.MapFile{
					Data: []byte("some readme"),
				},
			},
			dir: "queries",
			wantQueries: map[string]string{
				"listUsers": "SELECT * FROM users",
			},
		},
		{
			name: "subdirectories are ignored",
			fsys: fstest.MapFS{
				"queries/users.sql": &fstest.MapFile{
					Data: []byte("-- name: listUsers\nSELECT * FROM users"),
				},
				"queries/sub/orders.sql": &fstest.MapFile{
					Data: []byte("-- name: listOrders\nSELECT * FROM orders"),
				},
			},
			dir: "queries",
			wantQueries: map[string]string{
				"listUsers": "SELECT * FROM users",
			},
		},
		{
			name:    "dir not found",
			fsys:    fstest.MapFS{},
			dir:     "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			err := reg.WalkFS(tt.fsys, tt.dir)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertQueries(t, reg, tt.wantQueries)
		})
	}
}

// --- Get ---

func TestRegistryGet(t *testing.T) {
	tests := []struct {
		name    string
		queries map[string]string
		get     string
		wantSQL string
		wantErr bool
	}{
		{
			name:    "existing query",
			queries: map[string]string{"listUsers": "SELECT * FROM users"},
			get:     "listUsers",
			wantSQL: "SELECT * FROM users",
		},
		{
			name:    "query not found",
			queries: map[string]string{},
			get:     "listUsers",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			maps.Copy(reg.queries, tt.queries)

			q, err := reg.Get(tt.get)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotSQL, _ := q.Build()
			if gotSQL != tt.wantSQL {
				t.Errorf("SQL: got %q, want %q", gotSQL, tt.wantSQL)
			}
		})
	}
}

// --- MustGet ---

func TestRegistryMustGet(t *testing.T) {
	tests := []struct {
		name    string
		queries map[string]string
		get     string
		wantSQL string
	}{
		{
			name:    "existing query",
			queries: map[string]string{"listUsers": "SELECT * FROM users"},
			get:     "listUsers",
			wantSQL: "SELECT * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			maps.Copy(reg.queries, tt.queries)

			q := reg.MustGet(tt.get)
			gotSQL, _ := q.Build()
			if gotSQL != tt.wantSQL {
				t.Errorf("SQL: got %q, want %q", gotSQL, tt.wantSQL)
			}
		})
	}
}

func TestRegistryMustGetPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got none")
		}
	}()

	reg := NewRegistry()
	reg.MustGet("nonexistent")
}
