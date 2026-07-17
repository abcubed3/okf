// Package harvester defines metadata harvesting abstractions and files generation utilities for OKF.
package harvester

import (
	"context"
	"database/sql"
	"fmt"
)

// MySQLProvider implements SchemaProvider for MySQL databases.
type MySQLProvider struct {
	// DB is the sql.DB database connection.
	DB *sql.DB
	// Schema is the MySQL database/schema name to query.
	Schema string
}

// NewMySQLProvider creates a new MySQLProvider.
func NewMySQLProvider(db *sql.DB, schema string) *MySQLProvider {
	return &MySQLProvider{DB: db, Schema: schema}
}

// TypeString returns "MySQL".
func (p *MySQLProvider) TypeString() string {
	return "MySQL"
}

// resolveSchema returns the configured schema or resolves it by querying DATABASE().
func (p *MySQLProvider) resolveSchema(ctx context.Context) (string, error) {
	if p.Schema != "" {
		return p.Schema, nil
	}
	var dbName sql.NullString
	err := p.DB.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&dbName)
	if err != nil {
		return "", fmt.Errorf("failed to auto-detect current database name: %w", err)
	}
	if !dbName.Valid || dbName.String == "" {
		return "", fmt.Errorf("no database selected, please select a database or provide a schema flag")
	}
	return dbName.String, nil
}

// GetTables queries MySQL tables and comments in the schema.
func (p *MySQLProvider) GetTables(ctx context.Context) ([]TableMetadata, error) {
	schema, err := p.resolveSchema(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			table_name,
			COALESCE(table_comment, '') AS description
		FROM 
			information_schema.tables
		WHERE 
			table_schema = ?
			AND table_type = 'BASE TABLE'
		ORDER BY 
			table_name;
	`

	rows, err := p.DB.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query MySQL tables: %w", err)
	}
	defer rows.Close()

	var tables []TableMetadata
	for rows.Next() {
		var table TableMetadata
		if err := rows.Scan(&table.Name, &table.Description); err != nil {
			return nil, fmt.Errorf("failed to scan MySQL table: %w", err)
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// GetColumns queries MySQL columns for a table.
func (p *MySQLProvider) GetColumns(ctx context.Context, tableName string) ([]ColumnMetadata, error) {
	schema, err := p.resolveSchema(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			column_name,
			column_type,
			is_nullable,
			COALESCE(column_default, '') AS column_default,
			COALESCE(column_comment, '') AS description
		FROM 
			information_schema.columns
		WHERE 
			table_schema = ?
			AND table_name = ?
		ORDER BY 
			ordinal_position;
	`

	rows, err := p.DB.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query MySQL columns for %q: %w", tableName, err)
	}
	defer rows.Close()

	var columns []ColumnMetadata
	for rows.Next() {
		var col ColumnMetadata
		var isNullableStr string
		var defaultVal sql.NullString
		if err := rows.Scan(&col.Name, &col.DataType, &isNullableStr, &defaultVal, &col.Description); err != nil {
			return nil, fmt.Errorf("failed to scan MySQL column: %w", err)
		}
		col.IsNullable = (isNullableStr == "YES")
		if defaultVal.Valid {
			col.DefaultVal = defaultVal.String
		}
		columns = append(columns, col)
	}

	return columns, nil
}

// GetForeignKeys queries MySQL foreign keys from information_schema.
func (p *MySQLProvider) GetForeignKeys(ctx context.Context) ([]ForeignKeyMetadata, error) {
	schema, err := p.resolveSchema(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			table_name,
			column_name,
			referenced_table_name,
			referenced_column_name
		FROM
			information_schema.key_column_usage
		WHERE
			table_schema = ?
			AND referenced_table_name IS NOT NULL;
	`

	rows, err := p.DB.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query MySQL foreign keys: %w", err)
	}
	defer rows.Close()

	var fks []ForeignKeyMetadata
	for rows.Next() {
		var fk ForeignKeyMetadata
		if err := rows.Scan(&fk.TableName, &fk.ColumnName, &fk.ForeignTableName, &fk.ForeignColumnName); err != nil {
			return nil, fmt.Errorf("failed to scan MySQL foreign key: %w", err)
		}
		fks = append(fks, fk)
	}

	return fks, nil
}
