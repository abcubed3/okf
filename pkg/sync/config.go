package sync

import (
	"fmt"
	"os"

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

type GitConfig struct {
	Repo           string `yaml:"repo"`
	Branch         string `yaml:"branch"`
	Path           string `yaml:"path"`
	PrivateKeyPath string `yaml:"private_key_path"`
	Token          string `yaml:"token"`
	// InsecureSkipVerify disables SSH host key verification when true.
	// WARNING: This makes the connection vulnerable to man-in-the-middle attacks.
	// Only use in controlled environments where host verification is not possible.
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`
}

type GoogleDriveConfig struct {
	FolderID        string `yaml:"folder_id"`
	ServiceAccount  string `yaml:"service_account"`
	CredentialsFile string `yaml:"credentials_file"`
	TokenFile       string `yaml:"token_file"`
}

type NotionConfig struct {
	Token    string `yaml:"token"`
	ParentID string `yaml:"parent_id"`
}

type ConfluenceConfig struct {
	SpaceKey string `yaml:"space_key"`
	Domain   string `yaml:"domain"`
	Email    string `yaml:"email"`
	Token    string `yaml:"token"`
}

type JiraConfig struct {
	Domain string `yaml:"domain"`
	Email  string `yaml:"email"`
	Token  string `yaml:"token"`
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
	return &cfg, nil
}
