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
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/abcubed3/okf/pkg/ai"
	"github.com/abcubed3/okf/pkg/assembly"
	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/export"
	"github.com/abcubed3/okf/pkg/generator"
	"github.com/abcubed3/okf/pkg/harvester"
	"github.com/abcubed3/okf/pkg/lsp"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/publish"
	"github.com/abcubed3/okf/pkg/pull"
	"github.com/abcubed3/okf/pkg/server"
	okfsync "github.com/abcubed3/okf/pkg/sync"
	"github.com/abcubed3/okf/pkg/validator"
	"github.com/go-git/go-git/v5"

	_ "github.com/go-sql-driver/mysql"
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
	case "lsp":
		runLsp()
	case "doc":
		runDoc(os.Args[2:])
	case "sync":
		runSync(os.Args[2:])
	case "export":
		runExport(os.Args[2:])
	case "diff":
		runDiff(os.Args[2:])
	case "merge":
		runMerge(os.Args[2:])
	case "publish":
		runPublish(os.Args[2:])
	case "pull":
		runPull(os.Args[2:])
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
	fmt.Println("  lsp                      Start the OKF Language Server Protocol implementation")
	fmt.Println("  mcp [flags]              Start the OKF MCP (Model Context Protocol) Server")
	fmt.Println("  doc [flags]              Compile OKF bundle into an interactive documentation portal")
	fmt.Println("  sync [flags]             Start the bi-directional knowledge sync daemon")
	fmt.Println("  export <type> [flags]    Export OKF bundles to other structured metadata formats")
	fmt.Println("  diff <path-a> <path-b>   Compare two OKF bundles for drift")
	fmt.Println("  merge <path-a> <path-b>  Merge two OKF bundles together")
	fmt.Println("  publish [path] [flags]   Publish an OKF bundle to the Hub")
	fmt.Println("  pull <uri> [flags]       Pull an OKF bundle from the Hub")
	fmt.Println("  version, -v, --version   Print version information")
	fmt.Println("  help, -h, --help         Display help information")
}

// runLsp starts the Language Server Protocol process.
func runLsp() {
	lsp.Run()
}

// runPublish handles the publish CLI command
func runPublish(args []string) {
	bundlePath := "."
	host := "http://localhost:8080"
	apiKey := os.Getenv("OKF_HUB_API_KEY")

	// simple flag parsing
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	fs.StringVar(&host, "host", host, "OKF Hub host URL")
	fs.StringVar(&apiKey, "api-key", apiKey, "OKF Hub API Key (or set OKF_HUB_API_KEY)")
	_ = fs.Parse(args)

	if fs.NArg() > 0 {
		bundlePath = fs.Arg(0)
	}

	if apiKey == "" {
		fmt.Println("Error: API Key is required for publishing. Use --api-key or set OKF_HUB_API_KEY environment variable.")
		os.Exit(1)
	}

	err := publish.PublishBundle(bundlePath, host, apiKey)
	if err != nil {
		fmt.Printf("Error publishing bundle: %v\n", err)
		os.Exit(1)
	}
}

// runPull handles the pull CLI command
func runPull(args []string) {
	hubURI := ""
	host := "http://localhost:8080"
	apiKey := os.Getenv("OKF_HUB_API_KEY")

	fs := flag.NewFlagSet("pull", flag.ExitOnError)
	fs.StringVar(&host, "host", host, "OKF Hub host URL")
	fs.StringVar(&apiKey, "api-key", apiKey, "OKF Hub API Key (or set OKF_HUB_API_KEY)")
	_ = fs.Parse(args)

	if fs.NArg() > 0 {
		hubURI = fs.Arg(0)
	}

	if hubURI == "" {
		fmt.Println("Error: A hub:// URI is required. Example: okf pull hub://stripe/api")
		os.Exit(1)
	}

	err := pull.PullBundle(hubURI, host, apiKey)
	if err != nil {
		fmt.Printf("Error pulling bundle: %v\n", err)
		os.Exit(1)
	}
}

// runLint executes the linting check on a specified target bundle path.
func runLint(args []string) {
	bundlePath := "."
	host := "http://localhost:8080"
	apiKey := os.Getenv("OKF_HUB_API_KEY")

	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	fs.StringVar(&host, "host", host, "OKF Hub host URL for remote resolution")
	fs.StringVar(&apiKey, "api-key", apiKey, "OKF Hub API Key for remote resolution")
	_ = fs.Parse(args)

	if fs.NArg() > 0 {
		bundlePath = fs.Arg(0)
	}

	localPath, cleanup, err := parser.ResolvePath(bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve bundle path: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Linting OKF bundle at: %s...\n", absPath)

	b, err := parser.ParseBundle(context.Background(), absPath)
	if err != nil {
		fmt.Printf("Error: Failed to parse bundle: %v\n", err)
		os.Exit(1)
	}

	opts := validator.Options{
		Host:   host,
		APIKey: apiKey,
	}
	issues := validator.ValidateBundle(b, opts)

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
	fmt.Println("  db          Harvest database schemas (PostgreSQL, Spanner, BigQuery, MySQL)")
	fmt.Println("  openapi     Harvest OpenAPI v2/v3 spec endpoints")
	fmt.Println("  proto       Harvest Protobuf .proto messages/services")
	fmt.Println("  git         Harvest git repository history, commits, and contributors")
	fmt.Println("  web         Harvest documentation websites into OKF bundles")
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
	case "git":
		runHarvestGit(args[1:])
	case "web":
		runHarvestWeb(args[1:])
	default:
		fmt.Printf("Unknown harvest type: %s\n\n", harvestType)
		printHarvestUsage()
		os.Exit(1)
	}
}

// runHarvestDB handles the 'harvest db' subcommand, parsing database options and extracting schema metadata.
func runHarvestDB(args []string) {
	fs := flag.NewFlagSet("harvest db", flag.ExitOnError)
	driver := fs.String("driver", "", "Database driver: 'postgres', 'spanner', 'bigquery', or 'mysql'")
	connStr := fs.String("conn", "", "Database connection string / project ID")
	dataset := fs.String("dataset", "", "BigQuery dataset qualifier (required for bigquery)")
	schema := fs.String("schema", "", "Database schema / database name (defaults to 'public' for postgres)")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to generate business descriptions")
	aiModel := fs.String("model", "gemini-2.5-flash", "Gemini model to use for curation")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
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
		projectID := *connStr
		ds := *dataset

		// Support projects/PROJECT_ID/datasets/DATASET_ID format
		if strings.HasPrefix(*connStr, "projects/") {
			parts := strings.Split(*connStr, "/")
			if len(parts) == 4 && parts[0] == "projects" && parts[2] == "datasets" {
				projectID = parts[1]
				ds = parts[3]
			}
		}

		if ds == "" {
			fmt.Println("Error: --dataset flag is required for bigquery driver, or --conn must be formatted as projects/PROJECT/datasets/DATASET.")
			os.Exit(1)
		}
		ctx := context.Background()
		client, err := bigquery.NewClient(ctx, projectID)
		if err != nil {
			fmt.Printf("Error: Failed to connect to BigQuery: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()
		provider = harvester.NewBigQueryProvider(client, ds)
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
		case "mysql":
			provider = harvester.NewMySQLProvider(db, *schema)
		default:
			fmt.Printf("Error: Unsupported database driver %q. Supported drivers: postgres, spanner, bigquery, mysql\n", *driver)
			os.Exit(1)
		}
	}

	h := harvester.NewDBHarvester(provider)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		fmt.Printf("Error: Harvesting database failed: %v\n", err)
		os.Exit(1)
	}

	if *aiEnrich {
		fmt.Println("Running AI Curation on harvested concepts...")
		opts := ai.CurateOptions{Model: *aiModel, APIKey: *apiKey}
		if err := ai.CurateConcepts(context.Background(), concepts, opts); err != nil {
			fmt.Printf("Warning: AI curation failed: %v\n", err)
		}
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
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to generate business descriptions")
	aiModel := fs.String("model", "gemini-2.5-flash", "Gemini model to use for curation")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
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

	if *aiEnrich {
		fmt.Println("Running AI Curation on harvested concepts...")
		opts := ai.CurateOptions{Model: *aiModel, APIKey: *apiKey}
		if err := ai.CurateConcepts(context.Background(), concepts, opts); err != nil {
			fmt.Printf("Warning: AI curation failed: %v\n", err)
		}
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
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to generate business descriptions")
	aiModel := fs.String("model", "gemini-2.5-flash", "Gemini model to use for curation")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
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

	if *aiEnrich {
		fmt.Println("Running AI Curation on harvested concepts...")
		opts := ai.CurateOptions{Model: *aiModel, APIKey: *apiKey}
		if err := ai.CurateConcepts(context.Background(), concepts, opts); err != nil {
			fmt.Printf("Warning: AI curation failed: %v\n", err)
		}
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		fmt.Printf("Error: Failed to write concepts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully harvested %d Protobuf concepts into %q!\n", len(concepts), *output)
}

// runHarvestGit handles the 'harvest git' subcommand, parsing Git options and extracting repository history metadata.
func runHarvestGit(args []string) {
	fs := flag.NewFlagSet("harvest git", flag.ExitOnError)
	repoPath := fs.String("repo", ".", "Path to the local git repository")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to generate business descriptions")
	aiModel := fs.String("model", "gemini-2.5-flash", "Gemini model to use for curation")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	localRepoPath := *repoPath
	if parser.IsGitURL(*repoPath) {
		fmt.Printf("Cloning remote Git repository %q...\n", *repoPath)
		tempDir, err := os.MkdirTemp("", "okf-harvest-git-*")
		if err != nil {
			fmt.Printf("Error: Failed to create temporary directory for git clone: %v\n", err)
			os.Exit(1)
		}
		defer os.RemoveAll(tempDir)

		_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
			URL:      *repoPath,
			Progress: os.Stdout,
		})
		if err != nil {
			fmt.Printf("Error: Failed to clone remote git repository: %v\n", err)
			os.Exit(1)
		}
		localRepoPath = tempDir
	}

	h := harvester.NewGitHarvester(localRepoPath)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		fmt.Printf("Error: Harvesting git repository failed: %v\n", err)
		os.Exit(1)
	}

	if *aiEnrich {
		fmt.Println("Running AI Curation on harvested concepts...")
		opts := ai.CurateOptions{Model: *aiModel, APIKey: *apiKey}
		if err := ai.CurateConcepts(context.Background(), concepts, opts); err != nil {
			fmt.Printf("Warning: AI curation failed: %v\n", err)
		}
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		fmt.Printf("Error: Failed to write concepts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully harvested %d Git concepts into %q!\n", len(concepts), *output)
}

// runHarvestWeb handles the 'harvest web' subcommand, parsing options and scraping websites.
func runHarvestWeb(args []string) {
	fs := flag.NewFlagSet("harvest web", flag.ExitOnError)
	url := fs.String("url", "", "Starting URL to crawl (e.g., https://docs.example.com)")
	depth := fs.Int("depth", 1, "Maximum crawl depth (default 1)")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to extract structured metadata frontmatter")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}
	if *url == "" {
		fmt.Println("Error: --url flag is required for web harvesting.")
		fs.Usage()
		os.Exit(1)
	}

	var llm *harvester.LLMClassifier
	if *aiEnrich {
		key := *apiKey
		if key == "" {
			key = os.Getenv("GEMINI_API_KEY")
		}
		if key == "" {
			fmt.Println("Error: --api-key or GEMINI_API_KEY environment variable is required when --ai-enrich is true.")
			os.Exit(1)
		}
		var llmErr error
		llm, llmErr = harvester.NewLLMClassifier(context.Background(), key)
		if llmErr != nil {
			fmt.Printf("Error: Failed to initialize LLM classifier: %v\n", llmErr)
			os.Exit(1)
		}
		defer llm.Close()
	}

	h := harvester.NewWebHarvester(*url, *depth, llm)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		fmt.Printf("Error: Harvesting website failed: %v\n", err)
		os.Exit(1)
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		fmt.Printf("Error: Failed to write concepts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully harvested %d Web concepts into %q!\n", len(concepts), *output)
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

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve bundle path: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for bundle: %v\n", err)
		os.Exit(1)
	}

	b, err := parser.ParseBundle(context.Background(), absPath)
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

	res, err := assembly.AssembleContext(g, startID, opts)
	if err != nil {
		fmt.Printf("Error: Failed to assemble context: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(res.Context)
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

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve bundle path: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	absPath, err := filepath.Abs(localPath)
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

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve bundle path: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	absBundlePath, err := filepath.Abs(localPath)
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

// runSync handles the 'sync' subcommand to start the bi-directional knowledge sync daemon.
func runSync(args []string) {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	configPath := fs.String("config", "okf.yaml", "Path to the sync configuration file")
	daemon := fs.Bool("daemon", false, "Run continuously as a daemon")
	interval := fs.Int("interval", 300, "Sync interval in seconds (when running as daemon)")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve bundle path: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	absBundlePath, err := filepath.Abs(localPath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for bundle: %v\n", err)
		os.Exit(1)
	}

	// Dynamic import of the sync package (we'll implement this next).
	// Calling the engine start method.
	fmt.Printf("Starting OKF Sync Engine for bundle at %s (Config: %s)\n", absBundlePath, *configPath)
	// We'll import okfsync "github.com/abcubed3/okf/pkg/sync"
	err = okfsync.Run(absBundlePath, *configPath, *daemon, *interval)
	if err != nil {
		fmt.Printf("Error: Sync engine failed: %v\n", err)
		os.Exit(1)
	}
}

// runExport routes the export subcommand execution to the appropriate format exporter.
func runExport(args []string) {
	if len(args) < 1 {
		printExportUsage()
		os.Exit(1)
	}

	exportType := args[0]
	switch exportType {
	case "jsonld":
		runExportJSONLD(args[1:])
	default:
		fmt.Printf("Unknown export type: %s\n\n", exportType)
		printExportUsage()
		os.Exit(1)
	}
}

// printExportUsage prints usage information for the export command.
func printExportUsage() {
	fmt.Println("Usage: okf export <type> [flags]")
	fmt.Println("\nAvailable export types:")
	fmt.Println("  jsonld     Export OKF bundle to Schema.org JSON-LD")
}

// runExportJSONLD handles the 'export jsonld' subcommand.
func runExportJSONLD(args []string) {
	fs := flag.NewFlagSet("export jsonld", flag.ExitOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	output := fs.String("output", "schema.jsonld", "Output path for the generated JSON-LD schema file")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve bundle path: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	absBundlePath, err := filepath.Abs(localPath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for bundle: %v\n", err)
		os.Exit(1)
	}

	b, err := parser.ParseBundle(context.Background(), absBundlePath)
	if err != nil {
		fmt.Printf("Error: Failed to parse bundle: %v\n", err)
		os.Exit(1)
	}

	bytes, err := export.ExportBundleToJSONLD(b)
	if err != nil {
		fmt.Printf("Error: Failed to export bundle to JSON-LD: %v\n", err)
		os.Exit(1)
	}

	absOutputPath, err := filepath.Abs(*output)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for output: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(absOutputPath, bytes, 0644); err != nil {
		fmt.Printf("Error: Failed to write output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully exported OKF bundle to Schema.org JSON-LD at %s! 🎉\n", absOutputPath)
}

// runDiff compares two bundles (local path or git remote) and prints structural changes.
func runDiff(args []string) {
	if len(args) < 2 {
		fmt.Println("Error: two bundle paths are required.")
		fmt.Println("Usage: okf diff <bundle-path-a> <bundle-path-b>")
		os.Exit(1)
	}

	pathA := args[0]
	pathB := args[1]

	localPathA, cleanupA, err := parser.ResolvePath(pathA)
	if err != nil {
		fmt.Printf("Error: Failed to resolve bundle path %q: %v\n", pathA, err)
		os.Exit(1)
	}
	defer cleanupA()

	localPathB, cleanupB, err := parser.ResolvePath(pathB)
	if err != nil {
		fmt.Printf("Error: Failed to resolve bundle path %q: %v\n", pathB, err)
		os.Exit(1)
	}
	defer cleanupB()

	absPathA, err := filepath.Abs(localPathA)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for %q: %v\n", pathA, err)
		os.Exit(1)
	}

	absPathB, err := filepath.Abs(localPathB)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for %q: %v\n", pathB, err)
		os.Exit(1)
	}

	bundleA, err := parser.ParseBundle(context.Background(), absPathA)
	if err != nil {
		fmt.Printf("Error: Failed to parse bundle %q: %v\n", pathA, err)
		os.Exit(1)
	}

	bundleB, err := parser.ParseBundle(context.Background(), absPathB)
	if err != nil {
		fmt.Printf("Error: Failed to parse bundle %q: %v\n", pathB, err)
		os.Exit(1)
	}

	d, err := bundle.Diff(bundleA, bundleB)
	if err != nil {
		fmt.Printf("Error: Failed to diff bundles: %v\n", err)
		os.Exit(1)
	}

	d.PrettyPrint(os.Stdout)

	if len(d.Changes) > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

// runMerge merges two OKF bundles together based on a specified strategy.
func runMerge(args []string) {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	strategy := fs.String("strategy", "union", "Merge strategy: 'union', 'ours', or 'theirs'")
	output := fs.String("output", "", "Output directory for the merged bundle (defaults to source bundle path)")
	err := fs.Parse(args)
	if err != nil {
		fmt.Printf("Error: Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	if len(fs.Args()) < 2 {
		fmt.Println("Error: two bundle paths are required.")
		fmt.Println("Usage: okf merge <bundle-path-source> <bundle-path-target> [flags]")
		os.Exit(1)
	}

	sourcePath := fs.Arg(0)
	targetPath := fs.Arg(1)

	localSource, cleanupSource, err := parser.ResolvePath(sourcePath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve source bundle path %q: %v\n", sourcePath, err)
		os.Exit(1)
	}
	defer cleanupSource()

	localTarget, cleanupTarget, err := parser.ResolvePath(targetPath)
	if err != nil {
		fmt.Printf("Error: Failed to resolve target bundle path %q: %v\n", targetPath, err)
		os.Exit(1)
	}
	defer cleanupTarget()

	absSource, err := filepath.Abs(localSource)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for source: %v\n", err)
		os.Exit(1)
	}

	absTarget, err := filepath.Abs(localTarget)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute path for target: %v\n", err)
		os.Exit(1)
	}

	bundleSource, err := parser.ParseBundle(context.Background(), absSource)
	if err != nil {
		fmt.Printf("Error: Failed to parse source bundle: %v\n", err)
		os.Exit(1)
	}

	bundleTarget, err := parser.ParseBundle(context.Background(), absTarget)
	if err != nil {
		fmt.Printf("Error: Failed to parse target bundle: %v\n", err)
		os.Exit(1)
	}

	strat := bundle.MergeStrategy(*strategy)
	merged, err := bundle.Merge(bundleSource, bundleTarget, strat)
	if err != nil {
		fmt.Printf("Error merging bundles: %v\n", err)
		os.Exit(1)
	}

	outPath := *output
	if outPath == "" {
		if parser.IsGitURL(sourcePath) {
			fmt.Println("Warning: The source bundle is a remote Git URL and no --output was specified. Merged output will not be saved locally.")
		}
		outPath = sourcePath
	}

	localOut, cleanupOut, err := parser.ResolvePath(outPath)
	if err != nil {
		fmt.Printf("Error resolving output path: %v\n", err)
		os.Exit(1)
	}
	defer cleanupOut()

	absOut, err := filepath.Abs(localOut)
	if err != nil {
		fmt.Printf("Error: Failed to resolve absolute output path: %v\n", err)
		os.Exit(1)
	}

	var concepts []*bundle.Concept
	for _, c := range merged.Concepts {
		concepts = append(concepts, c)
	}

	if err := harvester.WriteConcepts(concepts, absOut); err != nil {
		fmt.Printf("Error: Failed to write merged concepts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully merged bundles! Written into %q\n", absOut)
}
