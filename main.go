// Package main is the entrypoint for the OKF (Open Knowledge Format) CLI tool.
// It provides commands to lint and validate OKF bundle directories, and to
// harvest schema metadata from databases (PostgreSQL, Spanner, BigQuery), OpenAPI specifications,
// and Protobuf files.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"cloud.google.com/go/bigquery"
	"github.com/abcubed3/okf/pkg/assembly"
	"github.com/abcubed3/okf/pkg/generator"
	"github.com/abcubed3/okf/pkg/harvester"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/server"
	"github.com/abcubed3/okf/pkg/validator"

	_ "github.com/googleapis/go-sql-spanner"
	_ "github.com/lib/pq"
)

var (
	// Version is the current version of the OKF CLI (injected at build time).
	Version = "dev"
	// Commit is the git commit hash at build time (injected).
	Commit = "none"
	// Date is the build date (injected).
	Date = "unknown"
)

// main parses the command line arguments and routes to the appropriate subcommand.
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "lint":
		runLint(os.Args[2:])
	case "harvest":
		runHarvest(os.Args[2:])
	case "assemble":
		runAssemble(os.Args[2:])
	case "server", "mcp":
		runServer(os.Args[2:])
	case "doc":
		runDoc(os.Args[2:])
	case "version", "-v", "--version", "-version":
		printVersion()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

// printVersion outputs the build version metadata.
func printVersion() {
	fmt.Printf("okf version %s (commit: %s, built at: %s)\n", Version, Commit, Date)
}

// printUsage prints general usage instructions and available subcommands for the OKF tool.
func printUsage() {
	fmt.Println("OKF CLI — Open Knowledge Format Tool")
	fmt.Println("\nUsage:")
	fmt.Println("  okf <command> [arguments]")
	fmt.Println("\nAvailable Commands:")
	fmt.Println("  lint [path]              Validate an OKF knowledge bundle directory (defaults to '.')")
	fmt.Println("  harvest <type> [flags]   Harvest metadata and generate OKF concepts")
	fmt.Println("  assemble <id> [flags]    Assemble dynamic context from a concept ID")
	fmt.Println("  mcp [flags]              Start the OKF MCP (Model Context Protocol) Server")
	fmt.Println("  doc [flags]              Compile OKF bundle into an interactive documentation portal")
	fmt.Println("  version, -v, --version   Print version information")
	fmt.Println("  help, -h, --help         Display help information")
}

// runLint executes the linting check on a specified target bundle path.
func runLint(args []string) {
	bundlePath := "."
	if len(args) > 0 {
		bundlePath = args[0]
	}

	absPath, err := filepath.Abs(bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Linting OKF bundle at: %s...\n", absPath)

	b, err := parser.ParseBundle(absPath)
	if err != nil {
		fmt.Printf("Error: Failed to parse bundle: %v\n", err)
		os.Exit(1)
	}

	issues := validator.ValidateBundle(b)

	errorsCount := 0
	warningsCount := 0

	for _, issue := range issues {
		severityPrefix := ""
		if issue.Severity == validator.SeverityError {
			severityPrefix = "[ERROR]"
			errorsCount++
		} else {
			severityPrefix = "[WARN] "
			warningsCount++
		}

		fmt.Printf("  %s %s: %s\n", severityPrefix, issue.Path, issue.Message)
	}

	fmt.Println()
	if errorsCount > 0 || warningsCount > 0 {
		fmt.Printf("Validation complete: %d errors, %d warnings found.\n", errorsCount, warningsCount)
	} else {
		fmt.Println("Validation complete: OKF bundle is perfectly valid! 🎉")
	}

	if errorsCount > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

// printHarvestUsage prints usage information for the harvest command.
func printHarvestUsage() {
	fmt.Println("Usage: okf harvest <type> [flags]")
	fmt.Println("\nAvailable harvest types:")
	fmt.Println("  db          Harvest database schemas (PostgreSQL, Spanner, BigQuery)")
	fmt.Println("  openapi     Harvest OpenAPI v2/v3 spec endpoints")
	fmt.Println("  proto       Harvest Protobuf .proto messages/services")
	fmt.Println("\nRun 'okf harvest <type> --help' to see flags for a specific type.")
}

// runHarvest routes the harvest command execution to the appropriate source type.
func runHarvest(args []string) {
	if len(args) < 1 {
		printHarvestUsage()
		os.Exit(1)
	}

	harvestType := args[0]
	switch harvestType {
	case "db":
		runHarvestDB(args[1:])
	case "openapi":
		runHarvestOpenAPI(args[1:])
	case "proto":
		runHarvestProto(args[1:])
	default:
		fmt.Printf("Unknown harvest type: %s\n\n", harvestType)
		printHarvestUsage()
		os.Exit(1)
	}
}

// runHarvestDB handles the 'harvest db' subcommand, parsing database options and extracting schema metadata.
func runHarvestDB(args []string) {
	fs := flag.NewFlagSet("harvest db", flag.ExitOnError)
	driver := fs.String("driver", "", "Database driver: 'postgres', 'spanner', or 'bigquery'")
	connStr := fs.String("conn", "", "Database connection string / project ID")
	dataset := fs.String("dataset", "", "BigQuery dataset qualifier (required for bigquery)")
	schema := fs.String("schema", "public", "PostgreSQL schema name")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}
	if *driver == "" || *connStr == "" {
		fmt.Println("Error: --driver and --conn are required flags for db harvesting.")
		fs.Usage()
		os.Exit(1)
	}

	var provider harvester.SchemaProvider

	if *driver == "bigquery" {
		if *dataset == "" {
			fmt.Println("Error: --dataset flag is required for bigquery driver.")
			os.Exit(1)
		}
		ctx := context.Background()
		client, err := bigquery.NewClient(ctx, *connStr)
		if err != nil {
			fmt.Printf("Error: Failed to connect to BigQuery: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()
		provider = harvester.NewBigQueryProvider(client, *dataset)
	} else {
		db, err := sql.Open(*driver, *connStr)
		if err != nil {
			fmt.Printf("Error: Failed to connect to database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		switch *driver {
		case "postgres":
			provider = harvester.NewPostgresProvider(db, *schema)
		case "spanner":
			provider = harvester.NewSpannerProvider(db)
		default:
			fmt.Printf("Error: Unsupported database driver %q. Supported drivers: postgres, spanner, bigquery\n", *driver)
			os.Exit(1)
		}
	}

	h := harvester.NewDBHarvester(provider)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		fmt.Printf("Error: Harvesting database failed: %v\n", err)
		os.Exit(1)
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		fmt.Printf("Error: Failed to write concepts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully harvested %d database table concepts into %q!\n", len(concepts), *output)
}

// runHarvestOpenAPI handles the 'harvest openapi' subcommand, parsing OpenAPI options and generating concepts.
func runHarvestOpenAPI(args []string) {
	fs := flag.NewFlagSet("harvest openapi", flag.ExitOnError)
	specPath := fs.String("spec", "", "Path to OpenAPI specification file (YAML/JSON)")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}
	if *specPath == "" {
		fmt.Println("Error: --spec flag is required for openapi harvesting.")
		fs.Usage()
		os.Exit(1)
	}

	h := harvester.NewOpenAPIHarvester(*specPath)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		fmt.Printf("Error: Harvesting OpenAPI spec failed: %v\n", err)
		os.Exit(1)
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		fmt.Printf("Error: Failed to write concepts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully harvested %d API Endpoint concepts into %q!\n", len(concepts), *output)
}

// runHarvestProto handles the 'harvest proto' subcommand, parsing Proto options and extracting protobuf message/service metadata.
func runHarvestProto(args []string) {
	fs := flag.NewFlagSet("harvest proto", flag.ExitOnError)
	protoPath := fs.String("path", "", "Path to .proto file or directory containing .proto files")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}
	if *protoPath == "" {
		fmt.Println("Error: --path flag is required for proto harvesting.")
		fs.Usage()
		os.Exit(1)
	}

	h := harvester.NewProtobufHarvester(*protoPath)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		fmt.Printf("Error: Harvesting protobuf schemas failed: %v\n", err)
		os.Exit(1)
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		fmt.Printf("Error: Failed to write concepts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully harvested %d Protobuf concepts into %q!\n", len(concepts), *output)
}

// runAssemble processes a concept ID and assembles its surrounding context based on a bundle graph.
func runAssemble(args []string) {
	fs := flag.NewFlagSet("assemble", flag.ExitOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	depth := fs.Int("depth", 2, "Maximum depth of link traversal")
	maxChars := fs.Int("max-chars", 16000, "Maximum character budget for assembled context (0 for unlimited)")
	direction := fs.String("direction", "bidirectional", "Link traversal direction: 'outbound', 'inbound', or 'bidirectional'")
	format := fs.String("format", "xml", "Output format: 'xml' or 'markdown'")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	if len(fs.Args()) < 1 {
		fmt.Println("Error: Start Concept ID is required.")
		fmt.Println("Usage: okf assemble <concept-id> [flags]")
		os.Exit(1)
	}
	startID := fs.Arg(0)

	absPath, err := filepath.Abs(*bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for bundle: %v\n", err)
		os.Exit(1)
	}

	b, err := parser.ParseBundle(absPath)
	if err != nil {
		fmt.Printf("Error: Failed to parse bundle: %v\n", err)
		os.Exit(1)
	}

	g := assembly.BuildGraph(b)

	dir := assembly.Direction(*direction)
	if dir != assembly.DirectionOutbound && dir != assembly.DirectionInbound && dir != assembly.DirectionBidirectional {
		fmt.Printf("Error: Invalid direction %q. Must be 'outbound', 'inbound', or 'bidirectional'.\n", *direction)
		os.Exit(1)
	}

	opts := assembly.AssemblyOptions{
		MaxDepth:      *depth,
		MaxCharacters: *maxChars,
		MaxTokens:     *maxChars / 4,
		Direction:     dir,
		Format:        *format,
	}

	ctxStr, err := assembly.AssembleContext(g, startID, opts)
	if err != nil {
		fmt.Printf("Error: Failed to assemble context: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(ctxStr)
}

// runServer starts the MCP (Model Context Protocol) server over either Stdio or SSE transport.
func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	transport := fs.String("transport", "stdio", "Transport mode: 'stdio' or 'sse'")
	port := fs.Int("port", 8080, "Port for SSE transport server")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	absPath, err := filepath.Abs(*bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for bundle: %v\n", err)
		os.Exit(1)
	}

	srv, err := server.NewMCPServer(absPath)
	if err != nil {
		fmt.Printf("Error: Failed to initialize MCP server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if *transport == "sse" {
		addr := fmt.Sprintf(":%d", *port)
		fmt.Fprintf(os.Stderr, "Starting OKF MCP Server over SSE at http://localhost%s...\n", addr)
		if err := srv.StartSSE(addr); err != nil {
			fmt.Printf("Error: Server failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Starting OKF MCP Server over Stdio...")
		if err := srv.StartStdio(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Server failed: %v\n", err)
			os.Exit(1)
		}
	}
}

// runDoc handles the 'doc' subcommand, compiling a bundle into interactive HTML docs.
func runDoc(args []string) {
	fs := flag.NewFlagSet("doc", flag.ExitOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	output := fs.String("output", "docs", "Output directory for generated documentation portal")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	absBundlePath, err := filepath.Abs(*bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for bundle: %v\n", err)
		os.Exit(1)
	}

	absOutputPath, err := filepath.Abs(*output)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generating OKF documentation from %s to %s...\n", absBundlePath, absOutputPath)

	if err := generator.Generate(absBundlePath, absOutputPath); err != nil {
		fmt.Printf("Error: Documentation generation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Documentation generated successfully! 🎉")
}
