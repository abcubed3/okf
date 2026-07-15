// Package harvester defines metadata harvesting abstractions and files generation utilities for OKF.
package harvester

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// BigQueryProvider implements SchemaProvider for Google Cloud BigQuery using the official BigQuery SDK.
type BigQueryProvider struct {
	// Client is the official BigQuery client instance.
	Client *bigquery.Client
	// Dataset is the dataset name (e.g. "my_dataset").
	Dataset string
}

// NewBigQueryProvider creates a new BigQueryProvider.
func NewBigQueryProvider(client *bigquery.Client, dataset string) *BigQueryProvider {
	return &BigQueryProvider{Client: client, Dataset: dataset}
}

// TypeString returns "BigQuery".
func (p *BigQueryProvider) TypeString() string {
	return "BigQuery"
}

// GetTables queries BigQuery tables and their descriptions.
func (p *BigQueryProvider) GetTables(ctx context.Context) ([]TableMetadata, error) {
	var tables []TableMetadata
	ds := p.Client.Dataset(p.Dataset)
	it := ds.Tables(ctx)

	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list BigQuery tables: %w", err)
		}

		// Fetch table metadata to retrieve the type and description
		meta, err := t.Metadata(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch metadata for BigQuery table %s: %w", t.TableID, err)
		}

		// Only include standard base tables (exclude views, external tables, materialised views, etc. if required)
		if meta.Type == bigquery.RegularTable {
			tables = append(tables, TableMetadata{
				Name:        t.TableID,
				Description: meta.Description,
			})
		}
	}

	return tables, nil
}

// GetColumns queries BigQuery columns and column descriptions using table schemas.
func (p *BigQueryProvider) GetColumns(ctx context.Context, tableName string) ([]ColumnMetadata, error) {
	t := p.Client.Dataset(p.Dataset).Table(tableName)
	meta, err := t.Metadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata for BigQuery table %q: %w", tableName, err)
	}

	var columns []ColumnMetadata
	for _, field := range meta.Schema {
		dataType := string(field.Type)
		if field.Repeated {
			dataType = "ARRAY<" + dataType + ">"
		}
		isNullable := !field.Required

		columns = append(columns, ColumnMetadata{
			Name:        field.Name,
			DataType:    dataType,
			IsNullable:  isNullable,
			DefaultVal:  "", // BigQuery does not expose default values in standard table schemas
			Description: field.Description,
		})
	}

	return columns, nil
}

// GetForeignKeys returns nil, nil as BigQuery does not enforce standard relational foreign keys.
func (p *BigQueryProvider) GetForeignKeys(ctx context.Context) ([]ForeignKeyMetadata, error) {
	return nil, nil
}
