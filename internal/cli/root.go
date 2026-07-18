package cli

import (
	"fmt"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func Execute(args []string, version, commit, date string) error {
	Version = version
	Commit = commit
	Date = date

	if len(args) == 0 {
		printUsage()
		return nil
	}

	subcommand := args[0]
	cmdArgs := args[1:]

	switch subcommand {
	case "lint":
		return RunLint(cmdArgs)
	case "harvest":
		return RunHarvest(cmdArgs)
	case "assemble":
		return RunAssemble(cmdArgs)
	case "server", "mcp":
		return RunServer(cmdArgs)
	case "lsp":
		return RunLsp(cmdArgs)
	case "doc":
		return RunDoc(cmdArgs)
	case "sync":
		return RunSync(cmdArgs)
	case "export":
		return RunExport(cmdArgs)
	case "diff":
		return RunDiff(cmdArgs)
	case "merge":
		return RunMerge(cmdArgs)
	case "publish":
		return RunPublish(cmdArgs)
	case "pull":
		return RunPull(cmdArgs)
	case "version", "-v", "--version", "-version":
		printVersion()
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		fmt.Printf("Unknown command: %s\n\n", subcommand)
		printUsage()
		return fmt.Errorf("unknown command: %s", subcommand)
	}
}

func printVersion() {
	fmt.Printf("okf version %s (commit: %s, built at: %s)\n", Version, Commit, Date)
}

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
