package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

type GitConnector struct {
	state     *StateManager
	config    *GitConfig
	repo      *git.Repository
	clonePath string
	auth      transport.AuthMethod
}

func NewGitConnector(state *StateManager) *GitConnector {
	return &GitConnector{
		state: state,
	}
}

func (c *GitConnector) Name() string {
	return "git"
}

func (c *GitConnector) Initialize(ctx context.Context, cfg *Config) error {
	if cfg.Connectors.Git == nil {
		return fmt.Errorf("git configuration missing")
	}
	c.config = cfg.Connectors.Git

	if c.config.Repo == "" {
		return fmt.Errorf("git configuration requires a non-empty 'repo' URL")
	}

	// 1. Configure authentication
	if c.config.PrivateKeyPath != "" {
		sshAuth, err := gitssh.NewPublicKeysFromFile("git", c.config.PrivateKeyPath, "")
		if err != nil {
			return fmt.Errorf("failed to load private key file %s: %w", c.config.PrivateKeyPath, err)
		}

		if c.config.InsecureSkipVerify {
			// Only allowed when user has explicitly opted in via config.
			log.Printf("[git] WARNING: SSH host key verification is disabled (insecure_skip_verify=true). This is a security risk.")
			sshAuth.HostKeyCallback = ssh.InsecureIgnoreHostKey() //nolint:gosec
		} else {
			// Use the system's known_hosts file for host key verification.
			knownHostsPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
			hostKeyCallback, err := gitssh.NewKnownHostsCallback(knownHostsPath)
			if err != nil {
				return fmt.Errorf("failed to load known_hosts from %s (use insecure_skip_verify: true to bypass): %w", knownHostsPath, err)
			}
			sshAuth.HostKeyCallback = hostKeyCallback
		}
		c.auth = sshAuth
	} else if c.config.Token != "" {
		c.auth = &githttp.BasicAuth{
			Username: "token",
			Password: c.config.Token,
		}
	}

	// 2. Generate a deterministic path in TempDir for cloning
	h := sha256.New()
	h.Write([]byte(c.config.Repo))
	hashStr := hex.EncodeToString(h.Sum(nil))[:12]
	c.clonePath = filepath.Join(os.TempDir(), "okf-sync-git-"+hashStr)

	branchName := c.config.Branch
	if branchName == "" {
		branchName = "main"
	}
	refName := plumbing.NewBranchReferenceName(branchName)

	log.Printf("[git] Initializing connector: repo=%s, branch=%s, path=%s", c.config.Repo, branchName, c.clonePath)

	// 3. Open or Clone
	if _, err := os.Stat(c.clonePath); err == nil {
		repo, err := git.PlainOpen(c.clonePath)
		if err == nil {
			c.repo = repo
			w, err := repo.Worktree()
			if err == nil {
				// Clean any uncommitted changes to reset sync state
				_ = w.Reset(&git.ResetOptions{Mode: git.HardReset})
			}
		} else {
			// Directory exists but failed to open (possibly corrupted), remove and clone fresh
			_ = os.RemoveAll(c.clonePath)
		}
	}

	if c.repo == nil {
		repo, err := git.PlainClone(c.clonePath, false, &git.CloneOptions{
			URL:           c.config.Repo,
			ReferenceName: refName,
			SingleBranch:  true,
			Auth:          c.auth,
			Progress:      nil,
		})
		if err != nil {
			return fmt.Errorf("failed to clone git repository: %w", err)
		}
		c.repo = repo
	}

	return nil
}

func (c *GitConnector) Push(ctx context.Context, concepts []*bundle.Concept) error {
	if c.config == nil || c.repo == nil {
		return nil
	}

	log.Printf("[git] Preparing to push %d concepts...", len(concepts))

	// Determine output directories
	subpath := c.config.Path
	if subpath == "" {
		subpath = "concepts"
	}

	w, err := c.repo.Worktree()
	if err != nil {
		return err
	}

	var stagedCount int
	for _, concept := range concepts {
		targetPath := filepath.Join(c.clonePath, subpath, concept.Path)
		parentDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory in git repository %q: %w", parentDir, err)
		}

		// Format concept frontmatter and body
		fmBytes, err := yaml.Marshal(concept.Frontmatter)
		if err != nil {
			log.Printf("failed to marshal YAML frontmatter for %s: %v", concept.ID, err)
			continue
		}

		var builder strings.Builder
		builder.WriteString("---\n")
		builder.Write(fmBytes)
		builder.WriteString("---\n")
		if concept.Body != "" {
			if !strings.HasPrefix(concept.Body, "\n") {
				builder.WriteString("\n")
			}
			builder.WriteString(concept.Body)
			if !strings.HasSuffix(concept.Body, "\n") {
				builder.WriteString("\n")
			}
		}

		// Write file to the cloned git directory
		if err := os.WriteFile(targetPath, []byte(builder.String()), 0644); err != nil {
			return fmt.Errorf("failed to write concept file to repository: %w", err)
		}

		gitRelPath, err := filepath.Rel(c.clonePath, targetPath)
		if err != nil {
			return err
		}

		_, err = w.Add(gitRelPath)
		if err != nil {
			return fmt.Errorf("failed to stage file in git repo: %w", err)
		}
		stagedCount++
	}

	if stagedCount == 0 {
		return nil
	}

	// Commit
	commitMsg := fmt.Sprintf("okf sync: update %d concepts", stagedCount)
	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "OKF Sync Engine",
			Email: "sync@okf.io",
			When:  time.Now(),
		},
	})
	if err != nil && err != git.ErrEmptyCommit {
		return fmt.Errorf("failed to commit: %w", err)
	}

	if err == git.ErrEmptyCommit {
		log.Printf("[git] No modifications. Skipping push.")
		return nil
	}

	// Push
	log.Printf("[git] Pushing changes to remote...")
	err = c.repo.PushContext(ctx, &git.PushOptions{
		RemoteName: "origin",
		Auth:       c.auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push to remote: %w", err)
	}

	log.Printf("[git] Push complete.")
	return nil
}

func (c *GitConnector) Pull(ctx context.Context) ([]*bundle.Concept, error) {
	if c.config == nil || c.repo == nil {
		return nil, nil
	}

	log.Printf("[git] Pulling latest changes from remote origin...")

	w, err := c.repo.Worktree()
	if err != nil {
		return nil, err
	}

	err = w.PullContext(ctx, &git.PullOptions{
		RemoteName: "origin",
		Auth:       c.auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("failed to pull updates from git remote: %w", err)
	}

	subpath := c.config.Path
	if subpath == "" {
		subpath = "concepts"
	}
	pullDir := filepath.Join(c.clonePath, subpath)

	// If the subfolder doesn't exist, return empty list
	if _, err := os.Stat(pullDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Parse all concepts in the pulled bundle subfolder
	b, err := parser.ParseBundle(pullDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pulled bundle from %s: %w", pullDir, err)
	}

	var concepts []*bundle.Concept
	for _, concept := range b.Concepts {
		if concept.ParseError == "" {
			concepts = append(concepts, concept)
		}
	}

	log.Printf("[git] Successfully pulled %d concepts from git remote.", len(concepts))
	return concepts, nil
}
