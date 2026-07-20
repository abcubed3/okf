# Harvesting Data

The `okf harvest` command connects to various external systems, extracts their metadata or content, and converts them into structured OKF Concept markdown files.

## Database Harvesting

Extract tables, schema definitions, columns, and foreign key relationships. Supported drivers include `postgres`, `mysql`, `bigquery`, and `spanner`.

**Example: PostgreSQL**
```bash
# Connects to a local Postgres database and extracts the 'public' schema
okf harvest db \
  --driver postgres \
  --conn "postgres://user:pass@localhost:5432/mydb?sslmode=disable" \
  --schema public \
  --output ./okf-bundle
```

**Example: Google Cloud BigQuery**
*(Requires Application Default Credentials. See [Authentication Guide](authentication.md))*
```bash
okf harvest db \
  --driver bigquery \
  --conn my-gcp-project-id \
  --dataset e_commerce_analytics \
  --output ./okf-bundle
```

## Git Harvesting

Extract the commit history of a repository. It analyzes commit messages and file changes, outputting each commit as an OKF `TechArticle` concept.

**Example: Local Repository**
```bash
okf harvest git \
  --repo /path/to/local/repo \
  --output ./okf-bundle
```

**Example: Remote Repository**
```bash
okf harvest git \
  --repo https://github.com/abcubed3/okf.git \
  --output ./okf-bundle
```

## Web Crawler

Extracts textual content from a web page (and recursively from linked pages) and converts it to markdown concepts.

```bash
okf harvest web \
  --url https://example.com/docs \
  --output ./okf-bundle
```

## OpenAPI Specification

Parses an OpenAPI v3 YAML or JSON file. It creates discrete OKF concepts for the API itself, as well as distinct concepts for every endpoint operation (e.g., `GET /users`).

```bash
okf harvest openapi \
  --file ./api-spec.yaml \
  --output ./okf-bundle
```

## Protobuf Schemas

Parses Protocol Buffer (`.proto`) files. It extracts Services, RPC Methods, and Message definitions into interconnected OKF concepts, linking method requests and responses to their respective Message definitions.

```bash
okf harvest proto \
  --dir ./proto \
  --output ./okf-bundle
```
