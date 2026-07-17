// Package harvester defines metadata harvesting abstractions and files generation utilities for OKF.
package harvester

import (
	"context"
	"database/sql"
	"fmt"
)

// PostgresProvider implements SchemaProvider for PostgreSQL databases.
type PostgresProvider struct {
	// DB is the sql.DB pointer database connection.
	DB *sql.DB
	// Schema is the PostgreSQL schema name to query (defaults to "public").
	Schema string
}

// NewPostgresProvider creates a new PostgresProvider.
func NewPostgresProvider(db *sql.DB, schema string) *PostgresProvider {
	if schema == "" {
		schema = "public"
	}
	return &PostgresProvider{DB: db, Schema: schema}
}

// TypeString returns "PostgreSQL".
func (p *PostgresProvider) TypeString() string {
	return "PostgreSQL"
}

// GetTables queries PostgreSQL tables and their descriptions in the configured schema.
func (p *PostgresProvider) GetTables(ctx context.Context) ([]TableMetadata, error) {
	query := `
		SELECT 
			t.table_name,
			COALESCE(pg_catalog.obj_description(pgc.oid, 'pg_class'), '') as description
		FROM 
			information_schema.tables t
		JOIN 
			pg_catalog.pg_class pgc ON pgc.relname = t.table_name
		JOIN 
			pg_catalog.pg_namespace pgn ON pgn.oid = pgc.relnamespace AND pgn.nspname = t.table_schema
		WHERE 
			t.table_schema = $1 
			AND t.table_type = 'BASE TABLE'
		ORDER BY 
			t.table_name;
	`

	rows, err := p.DB.QueryContext(ctx, query, p.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []TableMetadata
	for rows.Next() {
		var table TableMetadata
		if err := rows.Scan(&table.Name, &table.Description); err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("table row iteration error: %w", err)
	}

	return tables, nil
}

// GetColumns queries PostgreSQL column schemas and comments for a given table.
func (p *PostgresProvider) GetColumns(ctx context.Context, tableName string) ([]ColumnMetadata, error) {
	query := `
		SELECT 
			cols.column_name,
			cols.data_type,
			cols.is_nullable,
			COALESCE(cols.column_default, '') as column_default,
			COALESCE(pg_catalog.col_description(pgc.oid, cols.ordinal_position), '') as description
		FROM 
			information_schema.columns cols
		JOIN 
			pg_catalog.pg_class pgc ON pgc.relname = cols.table_name
		JOIN 
			pg_catalog.pg_namespace pgn ON pgn.oid = pgc.relnamespace AND pgn.nspname = cols.table_schema
		WHERE 
			cols.table_schema = $1 
			AND cols.table_name = $2
		ORDER BY 
			cols.ordinal_position;
	`

	rows, err := p.DB.QueryContext(ctx, query, p.Schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns for table %q: %w", tableName, err)
	}
	defer rows.Close()

	var columns []ColumnMetadata
	for rows.Next() {
		var col ColumnMetadata
		var isNullableStr string
		if err := rows.Scan(&col.Name, &col.DataType, &isNullableStr, &col.DefaultVal, &col.Description); err != nil {
			return nil, fmt.Errorf("failed to scan column row: %w", err)
		}
		col.IsNullable = (isNullableStr == "YES")
		columns = append(columns, col)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("column row iteration error: %w", err)
	}

	return columns, nil
}

// GetForeignKeys queries foreign key relationships in the configured schema.
func (p *PostgresProvider) GetForeignKeys(ctx context.Context) ([]ForeignKeyMetadata, error) {
	query := `
		SELECT
			kcu.table_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name
		FROM
			information_schema.table_constraints AS tc
			JOIN information_schema.key_column_usage AS kcu
			  ON tc.constraint_name = kcu.constraint_name
			  AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage AS ccu
			  ON ccu.constraint_name = tc.constraint_name
			  AND ccu.table_schema = tc.table_schema
		WHERE 
			tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1;
	`

	rows, err := p.DB.QueryContext(ctx, query, p.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer rows.Close()

	var fks []ForeignKeyMetadata
	for rows.Next() {
		var fk ForeignKeyMetadata
		if err := rows.Scan(&fk.TableName, &fk.ColumnName, &fk.ForeignTableName, &fk.ForeignColumnName); err != nil {
			return nil, fmt.Errorf("failed to scan foreign key row: %w", err)
		}
		fks = append(fks, fk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("foreign key row iteration error: %w", err)
	}

	return fks, nil
}
