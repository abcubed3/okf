package harvester

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitHarvester struct {
	repoPath string
}

type contributorStats struct {
	Name        string
	Email       string
	Commits     int
	FirstCommit time.Time
	LastCommit  time.Time
}

func NewGitHarvester(repoPath string) *GitHarvester {
	return &GitHarvester{
		repoPath: repoPath,
	}
}

func slugifyEmail(email string) string {
	s := strings.ToLower(email)
	s = strings.ReplaceAll(s, "@", "-at-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

func (gh *GitHarvester) Harvest(ctx context.Context) ([]*bundle.Concept, error) {
	absPath, err := filepath.Abs(gh.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for git repo: %w", err)
	}

	repo, err := git.PlainOpen(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository at %s: %w", absPath, err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository HEAD: %w", err)
	}

	// 1. Gather general repo info
	branchName := head.Name().Short()
	var remoteURLs []string
	remotes, err := repo.Remotes()
	if err == nil {
		for _, r := range remotes {
			remoteURLs = append(remoteURLs, r.Config().URLs...)
		}
	}

	cIter, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, fmt.Errorf("failed to get repository logs: %w", err)
	}
	defer cIter.Close()

	var recentCommits []*object.Commit
	contributors := make(map[string]*contributorStats)
	totalCommits := 0

	err = cIter.ForEach(func(c *object.Commit) error {
		totalCommits++

		email := strings.ToLower(strings.TrimSpace(c.Author.Email))
		name := c.Author.Name
		if email != "" {
			stats, exists := contributors[email]
			if !exists {
				stats = &contributorStats{
					Name:        name,
					Email:       email,
					Commits:     0,
					FirstCommit: c.Author.When,
					LastCommit:  c.Author.When,
				}
				contributors[email] = stats
			}
			stats.Commits++
			if c.Author.When.Before(stats.FirstCommit) {
				stats.FirstCommit = c.Author.When
			}
			if c.Author.When.After(stats.LastCommit) {
				stats.LastCommit = c.Author.When
			}
		}

		if len(recentCommits) < 15 {
			recentCommits = append(recentCommits, c)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed parsing commits: %w", err)
	}

	var concepts []*bundle.Concept
	nowStr := time.Now().Format(time.RFC3339)

	// 2. Generate Git Repository Concept
	repoExtra := map[string]interface{}{
		"branch":             branchName,
		"remotes":            remoteURLs,
		"total_commits":      totalCommits,
		"total_contributors": len(contributors),
	}
	if len(recentCommits) > 0 {
		repoExtra["last_commit"] = recentCommits[0].Hash.String()
	}

	var repoBodyBuilder strings.Builder
	repoBodyBuilder.WriteString("# Git Repository Summary\n\n")
	if len(remoteURLs) > 0 {
		repoBodyBuilder.WriteString(fmt.Sprintf("* **Remote Origin**: %s\n", strings.Join(remoteURLs, ", ")))
	}
	repoBodyBuilder.WriteString(fmt.Sprintf("* **Active Branch**: `%s`\n", branchName))
	repoBodyBuilder.WriteString(fmt.Sprintf("* **Total Commits**: %d\n", totalCommits))
	repoBodyBuilder.WriteString(fmt.Sprintf("* **Total Contributors**: %d\n", len(contributors)))
	if len(recentCommits) > 0 {
		shortHash := recentCommits[0].Hash.String()[:8]
		repoBodyBuilder.WriteString(fmt.Sprintf("* **Last Commit**: [%s](commits/%s.md)\n", shortHash, shortHash))
	}

	concepts = append(concepts, &bundle.Concept{
		ID:   "git/repository",
		Path: "git/repository.md",
		Frontmatter: bundle.Frontmatter{
			Type:      "Git Repository",
			Title:     "Git Repository",
			Desc:      "Summary statistics of the git repository status and history.",
			Timestamp: nowStr,
			Extra:     repoExtra,
		},
		Body: repoBodyBuilder.String(),
	})

	// 3. Generate Recent Commit Concepts
	for _, c := range recentCommits {
		shortHash := c.Hash.String()[:8]

		var parentHashes []string
		for _, p := range c.ParentHashes {
			parentHashes = append(parentHashes, p.String())
		}

		commitExtra := map[string]interface{}{
			"hash":         c.Hash.String(),
			"author_name":  c.Author.Name,
			"author_email": c.Author.Email,
			"parents":      parentHashes,
		}

		var commitBodyBuilder strings.Builder
		commitBodyBuilder.WriteString(fmt.Sprintf("# Commit %s\n\n", c.Hash.String()))
		commitBodyBuilder.WriteString(fmt.Sprintf("* **Author**: %s (<%s>)\n", c.Author.Name, c.Author.Email))
		commitBodyBuilder.WriteString(fmt.Sprintf("* **Date**: %s\n\n", c.Author.When.Format(time.RFC3339)))
		commitBodyBuilder.WriteString("## Message\n\n")
		commitBodyBuilder.WriteString(fmt.Sprintf("```\n%s\n```\n\n", strings.TrimSpace(c.Message)))

		if len(parentHashes) > 0 {
			commitBodyBuilder.WriteString("## Parents\n")
			for _, ph := range parentHashes {
				pShort := ph[:8]
				commitBodyBuilder.WriteString(fmt.Sprintf("- [%s](%s.md)\n", pShort, pShort))
			}
			commitBodyBuilder.WriteString("\n")
		}

		// Diff file changes
		currentTree, err := c.Tree()
		if err == nil {
			commitBodyBuilder.WriteString("## File Diffs\n\n")
			var fileChanges []string
			if c.NumParents() > 0 {
				parent, err := c.Parent(0)
				if err == nil {
					parentTree, err := parent.Tree()
					if err == nil {
						changes, err := parentTree.Diff(currentTree)
						if err == nil {
							for _, change := range changes {
								action, _ := change.Action()
								name := change.To.Name
								if name == "" {
									name = change.From.Name
								}
								fileChanges = append(fileChanges, fmt.Sprintf("- **%s**: `%s`", action.String(), name))
							}
						}
					}
				}
			} else {
				// First commit
				fIter := currentTree.Files()
				_ = fIter.ForEach(func(f *object.File) error {
					fileChanges = append(fileChanges, fmt.Sprintf("- **Insert**: `%s`", f.Name))
					return nil
				})
			}

			if len(fileChanges) > 0 {
				// Cap files listed to 30 to avoid blowing up context budget
				limit := 30
				if len(fileChanges) < limit {
					limit = len(fileChanges)
				}
				for i := 0; i < limit; i++ {
					commitBodyBuilder.WriteString(fileChanges[i])
					commitBodyBuilder.WriteString("\n")
				}
				if len(fileChanges) > limit {
					commitBodyBuilder.WriteString(fmt.Sprintf("\n... and %d more files.", len(fileChanges)-limit))
				}
			} else {
				commitBodyBuilder.WriteString("*No file changes detected (possibly a merge commit).*")
			}
		}

		concepts = append(concepts, &bundle.Concept{
			ID:   fmt.Sprintf("git/commits/%s", shortHash),
			Path: fmt.Sprintf("git/commits/%s.md", shortHash),
			Frontmatter: bundle.Frontmatter{
				Type:      "Git Commit",
				Title:     fmt.Sprintf("Commit: %s", shortHash),
				Desc:      fmt.Sprintf("Git commit hash %s by %s.", shortHash, c.Author.Name),
				Timestamp: c.Author.When.Format(time.RFC3339),
				Extra:     commitExtra,
			},
			Body: commitBodyBuilder.String(),
		})
	}

	// 4. Generate Contributor Concepts
	for _, stats := range contributors {
		slug := slugifyEmail(stats.Email)

		contribExtra := map[string]interface{}{
			"commits":      stats.Commits,
			"first_commit": stats.FirstCommit.Format(time.RFC3339),
			"last_commit":  stats.LastCommit.Format(time.RFC3339),
			"email":        stats.Email,
		}

		var contribBodyBuilder strings.Builder
		contribBodyBuilder.WriteString(fmt.Sprintf("# Contributor Profile: %s\n\n", stats.Name))
		contribBodyBuilder.WriteString(fmt.Sprintf("* **Email Address**: %s\n", stats.Email))
		contribBodyBuilder.WriteString(fmt.Sprintf("* **Total Commits**: %d\n", stats.Commits))
		contribBodyBuilder.WriteString(fmt.Sprintf("* **First Contribution**: %s\n", stats.FirstCommit.Format(time.RFC3339)))
		contribBodyBuilder.WriteString(fmt.Sprintf("* **Latest Contribution**: %s\n", stats.LastCommit.Format(time.RFC3339)))

		concepts = append(concepts, &bundle.Concept{
			ID:   fmt.Sprintf("git/contributors/%s", slug),
			Path: fmt.Sprintf("git/contributors/%s.md", slug),
			Frontmatter: bundle.Frontmatter{
				Type:      "Git Contributor",
				Title:     fmt.Sprintf("Contributor: %s", stats.Name),
				Desc:      fmt.Sprintf("Contributor metrics for %s in this project.", stats.Name),
				Timestamp: nowStr,
				Extra:     contribExtra,
			},
			Body: contribBodyBuilder.String(),
		})
	}

	return concepts, nil
}
