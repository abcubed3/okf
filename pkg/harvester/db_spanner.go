// Package harvester defines metadata harvesting abstractions and files generation utilities for OKF.
package harvester

import (
	"context"
	"database/sql"
	"fmt"
)

// SpannerProvider implements SchemaProvider for Cloud Spanner.
type SpannerProvider struct {
	// DB is the spanner database driver sql.DB connection.
	DB *sql.DB
}

// NewSpannerProvider creates a new SpannerProvider.
func NewSpannerProvider(db *sql.DB) *SpannerProvider {
	return &SpannerProvider{DB: db}
}

// TypeString returns "Cloud Spanner".
func (p *SpannerProvider) TypeString() string {
	return "Cloud Spanner"
}

// GetTables queries Spanner tables, capturing parent-child interleaved relationships.
func (p *SpannerProvider) GetTables(ctx context.Context) ([]TableMetadata, error) {
	// Spanner stores user-defined tables in the empty string ("") schema.
	// We also fetch parent_table_name to describe interleaving.
	query := `
		SELECT 
			table_name,
			COALESCE(parent_table_name, '') AS parent_table
		FROM 
			information_schema.tables
		WHERE 
			table_schema = '' 
			AND table_type = 'BASE TABLE'
		ORDER BY 
			table_name;
	`

	rows, err := p.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query Spanner tables: %w", err)
	}
	defer rows.Close()

	var tables []TableMetadata
	for rows.Next() {
		var name string
		var parentTable string
		if err := rows.Scan(&name, &parentTable); err != nil {
			return nil, fmt.Errorf("failed to scan Spanner table: %w", err)
		}

		desc := ""
		if parentTable != "" {
			desc = fmt.Sprintf("Interleaved parent table: %s.", parentTable)
		}

		tables = append(tables, TableMetadata{
			Name:        name,
			Description: desc,
		})
	}

	return tables, nil
}

// GetColumns queries Spanner columns. SPANNER_TYPE represents the type.
func (p *SpannerProvider) GetColumns(ctx context.Context, tableName string) ([]ColumnMetadata, error) {
	query := `
		SELECT 
			column_name,
			spanner_type,
			is_nullable,
			COALESCE(column_default, '') AS column_default
		FROM 
			information_schema.columns
		WHERE 
			table_schema = '' 
			AND table_name = @tableName
		ORDER BY 
			ordinal_position;
	`

	rows, err := p.DB.QueryContext(ctx, query, sql.Named("tableName", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to query Spanner columns for %q: %w", tableName, err)
	}
	defer rows.Close()

	var columns []ColumnMetadata
	for rows.Next() {
		var col ColumnMetadata
		var isNullableStr string
		if err := rows.Scan(&col.Name, &col.DataType, &isNullableStr, &col.DefaultVal); err != nil {
			return nil, fmt.Errorf("failed to scan Spanner column: %w", err)
		}
		col.IsNullable = (isNullableStr == "YES")
		columns = append(columns, col)
	}

	return columns, nil
}

// GetForeignKeys queries Spanner foreign keys from INFORMATION_SCHEMA.
func (p *SpannerProvider) GetForeignKeys(ctx context.Context) ([]ForeignKeyMetadata, error) {
	// Spanner does not support constraint_column_usage.
	// We join referential_constraints with key_column_usage twice to map referencing to referenced columns.
	query := `
		SELECT
			kcu.table_name AS table_name,
			kcu.column_name AS column_name,
			kcu2.table_name AS foreign_table_name,
			kcu2.column_name AS foreign_column_name
		FROM
			information_schema.referential_constraints AS rc
		JOIN
			information_schema.key_column_usage AS kcu
			ON rc.constraint_name = kcu.constraint_name
			AND rc.constraint_schema = kcu.constraint_schema
		JOIN
			information_schema.key_column_usage AS kcu2
			ON rc.unique_constraint_name = kcu2.constraint_name
			AND rc.unique_constraint_schema = kcu2.constraint_schema
			AND kcu.position_in_unique_constraint = kcu2.ordinal_position
		WHERE 
			rc.constraint_schema = ''
			AND kcu.table_schema = ''
			AND kcu2.table_schema = '';
	`

	rows, err := p.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query Spanner foreign keys: %w", err)
	}
	defer rows.Close()

	var fks []ForeignKeyMetadata
	for rows.Next() {
		var fk ForeignKeyMetadata
		if err := rows.Scan(&fk.TableName, &fk.ColumnName, &fk.ForeignTableName, &fk.ForeignColumnName); err != nil {
			return nil, fmt.Errorf("failed to scan Spanner foreign key: %w", err)
		}
		fks = append(fks, fk)
	}

	return fks, nil
}
