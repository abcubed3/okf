package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/skill"
)

func printSkillUsage() {
	fmt.Println("Usage: okf skill <subcommand> [flags]")
	fmt.Println("\nAvailable Subcommands:")
	fmt.Println("  export   Compile OKF bundle into a domain-specific Agent Skill (SKILL.md)")
	fmt.Println("  install  Install official OKF Agent Skill into workspace (.agents/.claude/.cursor)")
	fmt.Println("\nFlags for 'export':")
	fmt.Println("  --bundle <path>      Path to the OKF bundle (default '.')")
	fmt.Println("  --output <path>      Destination file path for SKILL.md (default 'SKILL.md')")
	fmt.Println("  --name <name>        Skill name for frontmatter (e.g. 'finance-metrics')")
	fmt.Println("  --desc <description> Custom description for agent activation")
	fmt.Println("  --tags <tags>        Comma-separated tags to filter concepts (e.g. 'finance,revenue')")
	fmt.Println("  --types <types>      Comma-separated types to filter concepts (e.g. 'Metric,Attestation')")
	fmt.Println("\nFlags for 'install':")
	fmt.Println("  --dir <path>         Explicit target directory for skill")
	fmt.Println("  --platform <name>    Target agent platform: 'antigravity', 'claude', or 'cursor'")
}

// RunSkill handles the 'okf skill' subcommand.
func RunSkill(args []string) error {
	if len(args) < 1 {
		printSkillUsage()
		return fmt.Errorf("missing skill subcommand")
	}

	subcommand := args[0]
	cmdArgs := args[1:]

	switch subcommand {
	case "export":
		return runSkillExport(cmdArgs)
	case "install", "init":
		return runSkillInstall(cmdArgs)
	case "help", "-h", "--help":
		printSkillUsage()
		return nil
	default:
		printSkillUsage()
		return fmt.Errorf("unknown skill subcommand: %s", subcommand)
	}
}

func runSkillExport(args []string) error {
	fs := flag.NewFlagSet("skill export", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	output := fs.String("output", "SKILL.md", "Output path for the generated SKILL.md file")
	name := fs.String("name", "", "Name of the skill (default inferred from bundle directory)")
	desc := fs.String("desc", "", "Description for agent skill activation")
	tags := fs.String("tags", "", "Comma-separated tag filter")
	types := fs.String("types", "", "Comma-separated type filter")
	noAttestations := fs.Bool("no-attestations", false, "Exclude attested computations section")

	if err := fs.Parse(args); err != nil {
		return err
	}

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path: %w", err)
	}
	defer cleanup()

	absBundlePath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute bundle path: %w", err)
	}

	b, err := parser.ParseBundle(context.Background(), absBundlePath)
	if err != nil {
		return fmt.Errorf("failed to parse bundle: %w", err)
	}

	skillName := *name
	if skillName == "" {
		skillName = filepath.Base(absBundlePath)
		if skillName == "." || skillName == "/" {
			skillName = "okf-knowledge"
		}
	}

	var tagList []string
	if *tags != "" {
		for _, t := range strings.Split(*tags, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				tagList = append(tagList, trimmed)
			}
		}
	}

	var typeList []string
	if *types != "" {
		for _, t := range strings.Split(*types, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				typeList = append(typeList, trimmed)
			}
		}
	}

	opts := skill.ExportOptions{
		Name:                skillName,
		Description:         *desc,
		Tags:                tagList,
		Types:               typeList,
		IncludeAttestations: !*noAttestations,
	}

	rendered, err := skill.ExportBundleToSkill(b, opts)
	if err != nil {
		return fmt.Errorf("failed to export skill: %w", err)
	}

	absOut, err := filepath.Abs(*output)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(absOut), 0755); err != nil {
		return fmt.Errorf("failed to create output parent directory: %w", err)
	}

	if err := os.WriteFile(absOut, []byte(rendered), 0644); err != nil {
		return fmt.Errorf("failed to write skill file %s: %w", absOut, err)
	}

	fmt.Printf("Successfully compiled OKF bundle into Agent Skill at: %s 🎉\n", absOut)
	return nil
}

func runSkillInstall(args []string) error {
	fs := flag.NewFlagSet("skill install", flag.ContinueOnError)
	dir := fs.String("dir", "", "Explicit target directory to install skill")
	platform := fs.String("platform", "agents", "Target agent platform: 'agents', 'antigravity', 'claude', or 'cursor'")

	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := skill.InstallOptions{
		TargetDir: *dir,
		Platform:  *platform,
	}

	dest, err := skill.InstallOfficialSkill(opts)
	if err != nil {
		return fmt.Errorf("failed to install official OKF skill: %w", err)
	}

	fmt.Printf("Successfully installed the OKF Agent Skill into: %s 🎉\n", dest)
	fmt.Printf("AI agent harness can now discover and use the OKF skill with the `/okf` command!\n")
	return nil
}
