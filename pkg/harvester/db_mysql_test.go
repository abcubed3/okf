package harvester

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"
)

// Define mock DB structures for testing database schema queries without real MySQL server.
type mockMySQLDriver struct {
	dbName string
}

func (d *mockMySQLDriver) Open(name string) (driver.Conn, error) {
	return &mockMySQLConn{dbName: d.dbName}, nil
}

type mockMySQLConn struct {
	dbName string
}

func (c *mockMySQLConn) Prepare(query string) (driver.Stmt, error) {
	return &mockMySQLStmt{query: query, dbName: c.dbName}, nil
}

func (c *mockMySQLConn) Close() error { return nil }

func (c *mockMySQLConn) Begin() (driver.Tx, error) { return nil, nil }

type mockMySQLStmt struct {
	query  string
	dbName string
}

func (s *mockMySQLStmt) Close() error { return nil }

func (s *mockMySQLStmt) NumInput() int { return -1 }

func (s *mockMySQLStmt) Exec(args []driver.Value) (driver.Result, error) { return nil, nil }

func (s *mockMySQLStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockMySQLRows{query: s.query, dbName: s.dbName}, nil
}

type mockMySQLRows struct {
	query  string
	dbName string
	cursor int
}

func (r *mockMySQLRows) Columns() []string {
	if r.query == "SELECT DATABASE()" {
		return []string{"DATABASE()"}
	}
	return []string{}
}

func (r *mockMySQLRows) Close() error { return nil }

func (r *mockMySQLRows) Next(dest []driver.Value) error {
	if r.query == "SELECT DATABASE()" {
		if r.cursor == 0 {
			dest[0] = r.dbName
			r.cursor++
			return nil
		}
	}
	return io.EOF
}

func TestMySQLProviderResolveSchema(t *testing.T) {
	sql.Register("mock_mysql_driver", &mockMySQLDriver{dbName: "auto_detected_db"})
	db, err := sql.Open("mock_mysql_driver", "")
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	defer db.Close()

	t.Run("explicit schema set", func(t *testing.T) {
		p := NewMySQLProvider(db, "my_custom_schema")
		if p.TypeString() != "MySQL" {
			t.Errorf("expected TypeString to be MySQL, got %q", p.TypeString())
		}
		schema, err := p.resolveSchema(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if schema != "my_custom_schema" {
			t.Errorf("expected schema 'my_custom_schema', got %q", schema)
		}
	})

	t.Run("auto-detected schema", func(t *testing.T) {
		p := NewMySQLProvider(db, "")
		schema, err := p.resolveSchema(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if schema != "auto_detected_db" {
			t.Errorf("expected auto-detected schema 'auto_detected_db', got %q", schema)
		}
	})
}
