package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Connectors struct {
		GoogleDrive *GoogleDriveConfig `yaml:"google_drive"`
		Notion      *NotionConfig      `yaml:"notion"`
		Confluence  *ConfluenceConfig  `yaml:"confluence"`
		Jira        *JiraConfig        `yaml:"jira"`
		Git         *GitConfig         `yaml:"git"`
	} `yaml:"connectors"`
}

// IncludeFilter scopes which concepts a connector pushes.
// When nil or all slices are empty, all concepts match.
type IncludeFilter struct {
	Types []string `yaml:"types,omitempty"`
	Tags  []string `yaml:"tags,omitempty"`
	Paths []string `yaml:"paths,omitempty"`
}

// HasRules reports whether any include rules are configured.
func (f *IncludeFilter) HasRules() bool {
	return f != nil && (len(f.Types) > 0 || len(f.Tags) > 0 || len(f.Paths) > 0)
}

// Matches returns true if c satisfies the include filter (or if no rules are configured).
func (f *IncludeFilter) Matches(c *bundle.Concept) bool {
	if !f.HasRules() || c == nil {
		return true
	}
	if len(f.Types) > 0 {
		matched := false
		for _, t := range f.Types {
			if strings.EqualFold(strings.TrimSpace(t), strings.TrimSpace(c.Frontmatter.Type)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(f.Tags) > 0 {
		matched := false
		for _, want := range f.Tags {
			for _, have := range c.Frontmatter.Tags {
				if strings.EqualFold(strings.TrimSpace(want), strings.TrimSpace(have)) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(f.Paths) > 0 {
		matched := false
		relSlash := filepath.ToSlash(c.Path)
		for _, pat := range f.Paths {
			if ok, err := filepath.Match(pat, relSlash); err == nil && ok {
				matched = true
				break
			}
			if ok, err := filepath.Match(pat, c.ID); err == nil && ok {
				matched = true
				break
			}
			if strings.HasSuffix(pat, "/**") {
				prefix := strings.TrimSuffix(pat, "**")
				if strings.HasPrefix(relSlash, prefix) || strings.HasPrefix(c.ID, prefix) {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

type GitConfig struct {
	Repo               string         `yaml:"repo"`
	Branch             string         `yaml:"branch"`
	Path               string         `yaml:"path"`
	PrivateKeyPath     string         `yaml:"private_key_path"`
	Token              string         `yaml:"token"`
	InsecureSkipVerify bool           `yaml:"insecure_skip_verify"`
	Include            *IncludeFilter `yaml:"include,omitempty"`
}

type GoogleDriveConfig struct {
	FolderID        string         `yaml:"folder_id"`
	ServiceAccount  string         `yaml:"service_account"`
	CredentialsFile string         `yaml:"credentials_file"`
	TokenFile       string         `yaml:"token_file"`
	Include         *IncludeFilter `yaml:"include,omitempty"`
}

type NotionConfig struct {
	Token    string         `yaml:"token"`
	ParentID string         `yaml:"parent_id"`
	BaseURL  string         `yaml:"base_url,omitempty"`
	Include  *IncludeFilter `yaml:"include,omitempty"`
}

type ConfluenceConfig struct {
	SpaceKey string         `yaml:"space_key"`
	Domain   string         `yaml:"domain"`
	Email    string         `yaml:"email"`
	Token    string         `yaml:"token"`
	BaseURL  string         `yaml:"base_url,omitempty"`
	Include  *IncludeFilter `yaml:"include,omitempty"`
}

type JiraConfig struct {
	Domain     string         `yaml:"domain"`
	Email      string         `yaml:"email"`
	Token      string         `yaml:"token"`
	ProjectKey string         `yaml:"project_key"`
	JQL        string         `yaml:"jql,omitempty"`
	PushNew    bool           `yaml:"push_new,omitempty"`
	BaseURL    string         `yaml:"base_url,omitempty"`
	Include    *IncludeFilter `yaml:"include,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil // Default empty config
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %s: %w", path, err)
	}
	expandConfigSecrets(&cfg)
	return &cfg, nil
}

func expandConfigSecrets(cfg *Config) {
	if cfg == nil {
		return
	}
	if gd := cfg.Connectors.GoogleDrive; gd != nil {
		gd.FolderID = os.ExpandEnv(gd.FolderID)
		gd.ServiceAccount = os.ExpandEnv(gd.ServiceAccount)
		gd.CredentialsFile = os.ExpandEnv(gd.CredentialsFile)
		gd.TokenFile = os.ExpandEnv(gd.TokenFile)
	}
	if n := cfg.Connectors.Notion; n != nil {
		n.Token = os.ExpandEnv(n.Token)
		if n.Token == "" {
			n.Token = os.Getenv("OKF_NOTION_TOKEN")
		}
		n.ParentID = os.ExpandEnv(n.ParentID)
		n.BaseURL = os.ExpandEnv(n.BaseURL)
	}
	if c := cfg.Connectors.Confluence; c != nil {
		c.Domain = os.ExpandEnv(c.Domain)
		c.Email = os.ExpandEnv(c.Email)
		c.Token = os.ExpandEnv(c.Token)
		if c.Token == "" {
			c.Token = os.Getenv("OKF_CONFLUENCE_TOKEN")
		}
		c.SpaceKey = os.ExpandEnv(c.SpaceKey)
		c.BaseURL = os.ExpandEnv(c.BaseURL)
	}
	if j := cfg.Connectors.Jira; j != nil {
		j.Domain = os.ExpandEnv(j.Domain)
		j.Email = os.ExpandEnv(j.Email)
		j.Token = os.ExpandEnv(j.Token)
		if j.Token == "" {
			j.Token = os.Getenv("OKF_JIRA_TOKEN")
		}
		j.ProjectKey = os.ExpandEnv(j.ProjectKey)
		j.JQL = os.ExpandEnv(j.JQL)
		j.BaseURL = os.ExpandEnv(j.BaseURL)
	}
	if g := cfg.Connectors.Git; g != nil {
		g.Repo = os.ExpandEnv(g.Repo)
		g.Branch = os.ExpandEnv(g.Branch)
		g.Token = os.ExpandEnv(g.Token)
		if g.Token == "" {
			g.Token = os.Getenv("OKF_GIT_TOKEN")
		}
	}
}
