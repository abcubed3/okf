package harvester

import (
	"context"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
)

type fakeSchemaProvider struct {
	tables []TableMetadata
	cols   map[string][]ColumnMetadata
	fks    []ForeignKeyMetadata
	dbType string
}

func (f *fakeSchemaProvider) GetTables(ctx context.Context) ([]TableMetadata, error) {
	return f.tables, nil
}

func (f *fakeSchemaProvider) GetColumns(ctx context.Context, tableName string) ([]ColumnMetadata, error) {
	return f.cols[tableName], nil
}

func (f *fakeSchemaProvider) GetForeignKeys(ctx context.Context) ([]ForeignKeyMetadata, error) {
	return f.fks, nil
}

func (f *fakeSchemaProvider) TypeString() string {
	return f.dbType
}

func TestDBHarvester(t *testing.T) {
	provider := &fakeSchemaProvider{
		dbType: "PostgreSQL",
		tables: []TableMetadata{
			{Name: "users", Description: "User accounts table."},
			{Name: "orders", Description: "Customer orders table."},
		},
		cols: map[string][]ColumnMetadata{
			"users": {
				{Name: "id", DataType: "integer", IsNullable: false, DefaultVal: "nextval('users_id_seq')", Description: "Primary key ID"},
				{Name: "email", DataType: "character varying", IsNullable: false, DefaultVal: "", Description: "Unique email"},
			},
			"orders": {
				{Name: "id", DataType: "integer", IsNullable: false, DefaultVal: "", Description: "Order ID"},
				{Name: "user_id", DataType: "integer", IsNullable: true, DefaultVal: "NULL", Description: "References users table"},
			},
		},
		fks: []ForeignKeyMetadata{
			{
				TableName:         "orders",
				ColumnName:        "user_id",
				ForeignTableName:  "users",
				ForeignColumnName: "id",
			},
		},
	}

	harvester := NewDBHarvester(provider)
	concepts, err := harvester.Harvest(context.Background())
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	if len(concepts) != 2 {
		t.Fatalf("expected 2 concepts, got %d", len(concepts))
	}

	// Validate "users" concept
	var usersConcept *bundle.Concept
	var ordersConcept *bundle.Concept
	for _, c := range concepts {
		switch c.ID {
		case "tables/users":
			usersConcept = c
		case "tables/orders":
			ordersConcept = c
		}
	}

	if usersConcept == nil {
		t.Fatal("users concept not found")
	}
	if ordersConcept == nil {
		t.Fatal("orders concept not found")
	}

	// Check frontmatter
	if usersConcept.Frontmatter.Type != "PostgreSQL Table" {
		t.Errorf("expected type 'PostgreSQL Table', got %q", usersConcept.Frontmatter.Type)
	}
	if usersConcept.Frontmatter.Title != "users Table" {
		t.Errorf("expected title 'users Table', got %q", usersConcept.Frontmatter.Title)
	}
	if usersConcept.Frontmatter.Desc != "User accounts table." {
		t.Errorf("expected desc 'User accounts table.', got %q", usersConcept.Frontmatter.Desc)
	}

	// Check body details
	if !strings.Contains(usersConcept.Body, "## Schema") {
		t.Error("users concept body missing Schema section")
	}
	if !strings.Contains(usersConcept.Body, "email | character varying | NO") {
		t.Error("users concept body missing column email info")
	}
	if !strings.Contains(usersConcept.Body, "Referenced by [orders Table](orders.md) via `user_id`") {
		t.Error("users concept body missing relationship back-reference link")
	}

	// Validate "orders" concept
	if ordersConcept.Frontmatter.Type != "PostgreSQL Table" {
		t.Errorf("expected type 'PostgreSQL Table', got %q", ordersConcept.Frontmatter.Type)
	}
	if !strings.Contains(ordersConcept.Body, "Column `user_id` references [users Table](users.md) via `id`") {
		t.Error("orders concept body missing outgoing foreign key relationship description")
	}
}
