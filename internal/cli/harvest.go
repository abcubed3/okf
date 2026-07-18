package cli

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	"cloud.google.com/go/bigquery"
	"github.com/abcubed3/okf/pkg/ai"
	"github.com/abcubed3/okf/pkg/harvester"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/go-git/go-git/v5"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/googleapis/go-sql-spanner"
	_ "github.com/lib/pq"
)

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

func RunHarvest(args []string) error {
	if len(args) < 1 {
		printHarvestUsage()
		return fmt.Errorf("missing harvest type")
	}

	harvestType := args[0]
	cmdArgs := args[1:]
	switch harvestType {
	case "db":
		return runHarvestDB(cmdArgs)
	case "openapi":
		return runHarvestOpenAPI(cmdArgs)
	case "proto":
		return runHarvestProto(cmdArgs)
	case "git":
		return runHarvestGit(cmdArgs)
	case "web":
		return runHarvestWeb(cmdArgs)
	default:
		fmt.Printf("Unknown harvest type: %s\n\n", harvestType)
		printHarvestUsage()
		return fmt.Errorf("unknown harvest type: %s", harvestType)
	}
}

func runHarvestDB(args []string) error {
	fs := flag.NewFlagSet("harvest db", flag.ContinueOnError)
	driver := fs.String("driver", "", "Database driver: 'postgres', 'spanner', 'bigquery', or 'mysql'")
	connStr := fs.String("conn", "", "Database connection string / project ID")
	dataset := fs.String("dataset", "", "BigQuery dataset qualifier (required for bigquery)")
	schema := fs.String("schema", "", "Database schema / database name (defaults to 'public' for postgres)")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to generate business descriptions")
	aiModel := fs.String("model", "gemini-2.5-flash", "Gemini model to use for curation")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *driver == "" || *connStr == "" {
		fs.Usage()
		return fmt.Errorf("--driver and --conn are required flags for db harvesting")
	}

	var provider harvester.SchemaProvider

	if *driver == "bigquery" {
		if *dataset == "" {
			return fmt.Errorf("--dataset flag is required for bigquery driver")
		}
		ctx := context.Background()
		client, err := bigquery.NewClient(ctx, *connStr)
		if err != nil {
			return fmt.Errorf("failed to connect to BigQuery: %w", err)
		}
		defer client.Close()
		provider = harvester.NewBigQueryProvider(client, *dataset)
	} else {
		db, err := sql.Open(*driver, *connStr)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
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
			return fmt.Errorf("unsupported database driver %q", *driver)
		}
	}

	h := harvester.NewDBHarvester(provider)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		return fmt.Errorf("harvesting database failed: %w", err)
	}

	if *aiEnrich {
		fmt.Println("Running AI Curation on harvested concepts...")
		opts := ai.CurateOptions{Model: *aiModel, APIKey: *apiKey}
		if err := ai.CurateConcepts(context.Background(), concepts, opts); err != nil {
			fmt.Printf("Warning: AI curation failed: %v\n", err)
		}
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		return fmt.Errorf("failed to write concepts: %w", err)
	}

	fmt.Printf("Successfully harvested %d database table concepts into %q!\n", len(concepts), *output)
	return nil
}

func runHarvestOpenAPI(args []string) error {
	fs := flag.NewFlagSet("harvest openapi", flag.ContinueOnError)
	specPath := fs.String("spec", "", "Path to OpenAPI specification file (YAML/JSON)")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to generate business descriptions")
	aiModel := fs.String("model", "gemini-2.5-flash", "Gemini model to use for curation")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *specPath == "" {
		fs.Usage()
		return fmt.Errorf("--spec flag is required for openapi harvesting")
	}

	h := harvester.NewOpenAPIHarvester(*specPath)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		return fmt.Errorf("harvesting OpenAPI spec failed: %w", err)
	}

	if *aiEnrich {
		fmt.Println("Running AI Curation on harvested concepts...")
		opts := ai.CurateOptions{Model: *aiModel, APIKey: *apiKey}
		if err := ai.CurateConcepts(context.Background(), concepts, opts); err != nil {
			fmt.Printf("Warning: AI curation failed: %v\n", err)
		}
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		return fmt.Errorf("failed to write concepts: %w", err)
	}

	fmt.Printf("Successfully harvested %d API Endpoint concepts into %q!\n", len(concepts), *output)
	return nil
}

func runHarvestProto(args []string) error {
	fs := flag.NewFlagSet("harvest proto", flag.ContinueOnError)
	protoPath := fs.String("path", "", "Path to .proto file or directory containing .proto files")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to generate business descriptions")
	aiModel := fs.String("model", "gemini-2.5-flash", "Gemini model to use for curation")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *protoPath == "" {
		fs.Usage()
		return fmt.Errorf("--path flag is required for proto harvesting")
	}

	h := harvester.NewProtobufHarvester(*protoPath)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		return fmt.Errorf("harvesting protobuf schemas failed: %w", err)
	}

	if *aiEnrich {
		fmt.Println("Running AI Curation on harvested concepts...")
		opts := ai.CurateOptions{Model: *aiModel, APIKey: *apiKey}
		if err := ai.CurateConcepts(context.Background(), concepts, opts); err != nil {
			fmt.Printf("Warning: AI curation failed: %v\n", err)
		}
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		return fmt.Errorf("failed to write concepts: %w", err)
	}

	fmt.Printf("Successfully harvested %d Protobuf concepts into %q!\n", len(concepts), *output)
	return nil
}

func runHarvestGit(args []string) error {
	fs := flag.NewFlagSet("harvest git", flag.ContinueOnError)
	repoPath := fs.String("repo", ".", "Path to the local git repository")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to generate business descriptions")
	aiModel := fs.String("model", "gemini-2.5-flash", "Gemini model to use for curation")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	localRepoPath := *repoPath
	if parser.IsGitURL(*repoPath) {
		fmt.Printf("Cloning remote Git repository %q...\n", *repoPath)
		tempDir, err := os.MkdirTemp("", "okf-harvest-git-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory for git clone: %w", err)
		}
		defer os.RemoveAll(tempDir)

		_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
			URL:      *repoPath,
			Progress: os.Stdout,
		})
		if err != nil {
			return fmt.Errorf("failed to clone remote git repository: %w", err)
		}
		localRepoPath = tempDir
	}

	h := harvester.NewGitHarvester(localRepoPath)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		return fmt.Errorf("harvesting git repository failed: %w", err)
	}

	if *aiEnrich {
		fmt.Println("Running AI Curation on harvested concepts...")
		opts := ai.CurateOptions{Model: *aiModel, APIKey: *apiKey}
		if err := ai.CurateConcepts(context.Background(), concepts, opts); err != nil {
			fmt.Printf("Warning: AI curation failed: %v\n", err)
		}
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		return fmt.Errorf("failed to write concepts: %w", err)
	}

	fmt.Printf("Successfully harvested %d Git concepts into %q!\n", len(concepts), *output)
	return nil
}

func runHarvestWeb(args []string) error {
	fs := flag.NewFlagSet("harvest web", flag.ContinueOnError)
	url := fs.String("url", "", "Starting URL to crawl (e.g., https://docs.example.com)")
	depth := fs.Int("depth", 1, "Maximum crawl depth (default 1)")
	output := fs.String("output", "concepts", "Output directory for generated concepts")
	aiEnrich := fs.Bool("ai-enrich", false, "Use AI (Gemini) to extract structured metadata frontmatter")
	apiKey := fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *url == "" {
		fs.Usage()
		return fmt.Errorf("--url flag is required for web harvesting")
	}

	var llm *harvester.LLMClassifier
	if *aiEnrich {
		key := *apiKey
		if key == "" {
			key = os.Getenv("GEMINI_API_KEY")
		}
		if key == "" {
			return fmt.Errorf("--api-key or GEMINI_API_KEY environment variable is required when --ai-enrich is true")
		}
		var llmErr error
		llm, llmErr = harvester.NewLLMClassifier(context.Background(), key)
		if llmErr != nil {
			return fmt.Errorf("failed to initialize LLM classifier: %w", llmErr)
		}
		defer llm.Close()
	}

	h := harvester.NewWebHarvester(*url, *depth, llm)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		return fmt.Errorf("harvesting website failed: %w", err)
	}

	if err := harvester.WriteConcepts(concepts, *output); err != nil {
		return fmt.Errorf("failed to write concepts: %w", err)
	}

	fmt.Printf("Successfully harvested %d Web concepts into %q!\n", len(concepts), *output)
	return nil
}
