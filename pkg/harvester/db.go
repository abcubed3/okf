// Package harvester defines metadata harvesting abstractions and files generation utilities for OKF.
package harvester

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
)

// TableMetadata contains high-level information about a database table.
type TableMetadata struct {
	// Name is the database table name.
	Name string
	// Description is the comment or description attached to the table.
	Description string
}

// ColumnMetadata contains detailed schema info for a database column.
type ColumnMetadata struct {
	// Name is the column name.
	Name string
	// DataType is the database type string (e.g. VARCHAR, INT, TIMESTAMP).
	DataType string
	// IsNullable specifies whether the column permits NULL values.
	IsNullable bool
	// DefaultVal is the default value expression, if any.
	DefaultVal string
	// Description is the comment or description attached to the column.
	Description string
}

// ForeignKeyMetadata describes a foreign key constraint linking two tables.
type ForeignKeyMetadata struct {
	// TableName is the referencing table name.
	TableName string
	// ColumnName is the referencing column name.
	ColumnName string
	// ForeignTableName is the referenced table name.
	ForeignTableName string
	// ForeignColumnName is the referenced column name.
	ForeignColumnName string
}

// SchemaProvider abstracts the dialect-specific schema extraction queries.
type SchemaProvider interface {
	// GetTables returns a list of tables and their descriptions.
	GetTables(ctx context.Context) ([]TableMetadata, error)
	// GetColumns returns a list of columns for a specific table.
	GetColumns(ctx context.Context, tableName string) ([]ColumnMetadata, error)
	// GetForeignKeys returns all foreign key constraints in the schema.
	GetForeignKeys(ctx context.Context) ([]ForeignKeyMetadata, error)
	// TypeString returns the brand/type name, e.g. "PostgreSQL", "BigQuery", "Cloud Spanner".
	TypeString() string
}

// DBHarvester coordinates metadata extraction from a database schema.
type DBHarvester struct {
	// Provider handles dialect-specific database metadata queries.
	Provider SchemaProvider
}

// NewDBHarvester creates a new DBHarvester using the given provider.
func NewDBHarvester(provider SchemaProvider) *DBHarvester {
	return &DBHarvester{Provider: provider}
}

// Harvest extracts the schema metadata and maps it to OKF Concept documents.
func (h *DBHarvester) Harvest(ctx context.Context) ([]*bundle.Concept, error) {
	tables, err := h.Provider.GetTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tables: %w", err)
	}

	foreignKeys, err := h.Provider.GetForeignKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch foreign keys: %w", err)
	}

	// Map of outgoing foreign keys: TableName -> list of FKs
	outgoingFKs := make(map[string][]ForeignKeyMetadata)
	// Map of incoming foreign keys: ForeignTableName -> list of FKs
	incomingFKs := make(map[string][]ForeignKeyMetadata)
	for _, fk := range foreignKeys {
		outgoingFKs[fk.TableName] = append(outgoingFKs[fk.TableName], fk)
		incomingFKs[fk.ForeignTableName] = append(incomingFKs[fk.ForeignTableName], fk)
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	var concepts []*bundle.Concept

	for _, table := range tables {
		columns, err := h.Provider.GetColumns(ctx, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch columns for table %q: %w", table.Name, err)
		}

		concept := h.buildTableConcept(table, columns, outgoingFKs[table.Name], incomingFKs[table.Name], timestamp)
		concepts = append(concepts, concept)
	}

	return concepts, nil
}

// buildTableConcept formats a table schema and its foreign keys into a markdown-structured Concept.
func (h *DBHarvester) buildTableConcept(
	table TableMetadata,
	columns []ColumnMetadata,
	outgoing []ForeignKeyMetadata,
	incoming []ForeignKeyMetadata,
	timestamp string,
) *bundle.Concept {
	var body strings.Builder

	// Title header
	body.WriteString(fmt.Sprintf("# %s Table\n\n", table.Name))

	// Description
	if table.Description != "" {
		body.WriteString(table.Description)
		body.WriteString("\n\n")
	} else {
		body.WriteString(fmt.Sprintf("Metadata representation of the %s table.\n\n", table.Name))
	}

	// Columns Schema Table
	body.WriteString("## Schema\n")
	body.WriteString("| Column | Type | Nullable | Default | Description |\n")
	body.WriteString("| ------ | ---- | -------- | ------- | ----------- |\n")
	for _, col := range columns {
		nullableStr := "YES"
		if !col.IsNullable {
			nullableStr = "NO"
		}
		defaultStr := col.DefaultVal
		if defaultStr == "" {
			defaultStr = "NULL"
		}
		descStr := col.Description
		if descStr == "" {
			descStr = "-"
		}
		body.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", col.Name, col.DataType, nullableStr, defaultStr, descStr))
	}
	body.WriteString("\n")

	// Key Relationships
	hasRelationships := len(outgoing) > 0 || len(incoming) > 0
	if hasRelationships {
		body.WriteString("## Key Relationships\n")
		for _, fk := range outgoing {
			body.WriteString(fmt.Sprintf("*   Column `%s` references [%s Table](%s.md) via `%s`.\n",
				fk.ColumnName, fk.ForeignTableName, strings.ToLower(fk.ForeignTableName), fk.ForeignColumnName))
		}
		for _, fk := range incoming {
			body.WriteString(fmt.Sprintf("*   Referenced by [%s Table](%s.md) via `%s`.\n",
				fk.TableName, strings.ToLower(fk.TableName), fk.ColumnName))
		}
	}

	conceptID := fmt.Sprintf("tables/%s", strings.ToLower(table.Name))
	conceptPath := fmt.Sprintf("tables/%s.md", strings.ToLower(table.Name))

	dbType := h.Provider.TypeString()
	title := fmt.Sprintf("%s Table", table.Name)
	tags := []string{"database", "table", strings.ToLower(dbType)}

	return &bundle.Concept{
		ID:   conceptID,
		Path: conceptPath,
		Frontmatter: bundle.Frontmatter{
			Type:      fmt.Sprintf("%s Table", dbType),
			Title:     title,
			Desc:      table.Description,
			Resource:  table.Name,
			Tags:      tags,
			Timestamp: timestamp,
		},
		Body: body.String(),
	}
}
