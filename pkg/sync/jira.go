package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"gopkg.in/yaml.v3"
)

const jiraMaxDescriptionBytes = 32 * 1024 // 32 KB

type JiraConnector struct {
	state  *StateManager
	config *JiraConfig
	client *rateLimitedClient
	sdk    *jira.Client
}

func NewJiraConnector(state *StateManager) *JiraConnector {
	return &JiraConnector{
		state: state,
	}
}

func (c *JiraConnector) Name() string {
	return "jira"
}

func (c *JiraConnector) Initialize(ctx context.Context, cfg *Config) error {
	if cfg.Connectors.Jira == nil {
		return fmt.Errorf("jira configuration missing")
	}
	c.config = cfg.Connectors.Jira
	if strings.TrimSpace(c.config.Token) == "" {
		return fmt.Errorf("jira token is required (set connectors.jira.token or OKF_JIRA_TOKEN)")
	}
	if strings.TrimSpace(c.config.ProjectKey) == "" && strings.TrimSpace(c.config.JQL) == "" {
		return fmt.Errorf("jira requires project_key or jql in configuration")
	}
	site := strings.TrimRight(c.config.BaseURL, "/")
	site = strings.TrimSuffix(site, "/rest/api/3")
	if site == "" {
		if strings.TrimSpace(c.config.Domain) == "" {
			return fmt.Errorf("jira domain is required")
		}
		domain := strings.TrimPrefix(strings.TrimPrefix(c.config.Domain, "https://"), "http://")
		domain = strings.TrimRight(domain, "/")
		if !strings.Contains(domain, ".") {
			domain += ".atlassian.net"
		}
		site = "https://" + domain
	}
	if c.client == nil {
		c.client = newRateLimitedClient(nil, 5.0)
	}
	sdk, err := jira.New(c.client, site)
	if err != nil {
		return fmt.Errorf("failed to initialize jira SDK client: %w", err)
	}
	if c.config.Email != "" {
		sdk.Auth.SetBasicAuth(c.config.Email, c.config.Token)
	} else {
		sdk.Auth.SetBearerToken(c.config.Token)
	}
	c.sdk = sdk
	return nil
}

func (c *JiraConnector) Push(ctx context.Context, concepts []*bundle.Concept) error {
	if c.config == nil || c.sdk == nil {
		return nil
	}

	allowCreate := c.config.PushNew || c.config.Include.HasRules()

	for _, concept := range concepts {
		extID := c.state.GetExternalID(concept.ID, c.Name())
		if extID == "" && !allowCreate {
			log.Printf("[jira] Skipping new untracked concept %s (set push_new: true or include filter to create Jira issues)", concept.ID)
			continue
		}

		adf, err := buildJiraADF(concept)
		if err != nil {
			log.Printf("[jira] Failed to serialize %s: %v", concept.ID, err)
			continue
		}
		hash, _ := HashConcept(concept)
		summary := concept.Frontmatter.Title
		if summary == "" {
			summary = concept.ID
		}

		if extID == "" {
			if strings.TrimSpace(c.config.ProjectKey) == "" {
				log.Printf("[jira] Cannot create issue for %s without project_key", concept.ID)
				continue
			}
			issueKey, updated, err := c.createIssue(ctx, concept.ID, summary, adf)
			if err != nil {
				log.Printf("[jira] Failed to create issue for %s: %v", concept.ID, err)
				continue
			}
			c.state.SetExternalID(concept.ID, c.Name(), issueKey)
			if hash != "" {
				c.state.SetContentHash(concept.ID, c.Name(), hash)
			}
			if updated != "" {
				c.state.SetRemoteRevision(concept.ID, c.Name(), updated)
			}
			continue
		}

		updated, err := c.updateIssue(ctx, extID, summary, adf)
		if err != nil {
			log.Printf("[jira] Failed to update issue %s (%s): %v", concept.ID, extID, err)
			continue
		}
		if hash != "" {
			c.state.SetContentHash(concept.ID, c.Name(), hash)
		}
		if updated != "" {
			c.state.SetRemoteRevision(concept.ID, c.Name(), updated)
		}
	}
	return nil
}

func (c *JiraConnector) Pull(ctx context.Context) ([]*bundle.Concept, error) {
	if c.config == nil || c.sdk == nil {
		return nil, nil
	}

	jql := strings.TrimSpace(c.config.JQL)
	if jql == "" {
		jql = fmt.Sprintf(`project = "%s" ORDER BY updated DESC`, c.config.ProjectKey)
	}

	var pulled []*bundle.Concept
	nextPageToken := ""
	for {
		res, _, err := c.sdk.Issue.Search.SearchJQL(ctx, jql, []string{"summary", "description", "updated"}, nil, 50, nextPageToken)
		if err != nil {
			return nil, fmt.Errorf("jira SearchJQL failed: %w", err)
		}

		for _, issue := range res.Issues {
			conceptID := c.getIssueConceptID(ctx, issue.Key)
			if conceptID == "" {
				conceptID = c.state.FindConceptByExternalID(c.Name(), issue.Key)
			}
			if conceptID == "" {
				continue
			}

			updated := ""
			var descNode *models.CommentNodeScheme
			if issue.Fields != nil {
				updated = formatJiraDateTime(issue.Fields.Updated)
				descNode = issue.Fields.Description
			}
			if updated != "" && c.state.GetRemoteRevision(conceptID, c.Name()) == updated {
				continue
			}

			rawDoc := extractJiraADFMarkdown(descNode)
			concept, _, parseErr := ParseConceptEnvelope(conceptID, "", rawDoc)
			if parseErr != nil {
				log.Printf("[jira] Quarantine %s (%s): malformed okf-frontmatter YAML: %v", conceptID, issue.Key, parseErr)
				continue
			}
			c.state.SetExternalID(conceptID, c.Name(), issue.Key)
			if updated != "" {
				c.state.SetRemoteRevision(conceptID, c.Name(), updated)
			}
			pulled = append(pulled, concept)
		}

		if res.NextPageToken == "" {
			break
		}
		nextPageToken = res.NextPageToken
	}

	return pulled, nil
}

func (c *JiraConnector) createIssue(ctx context.Context, conceptID, summary string, adf *models.CommentNodeScheme) (string, string, error) {
	payload := &models.IssueScheme{
		Fields: &models.IssueFieldsScheme{
			Project:     &models.ProjectScheme{Key: c.config.ProjectKey},
			IssueType:   &models.IssueTypeScheme{Name: "Task"},
			Summary:     truncateRunes(summary, 255),
			Description: adf,
		},
	}
	res, _, err := c.sdk.Issue.Create(ctx, payload, nil)
	if err != nil {
		return "", "", err
	}
	_, _ = c.sdk.Issue.Property.Set(ctx, res.Key, "okf_concept_id", map[string]string{"concept_id": conceptID})

	meta, _, err := c.sdk.Issue.Get(ctx, res.Key, []string{"updated"}, nil)
	updated := ""
	if err == nil && meta != nil && meta.Fields != nil {
		updated = formatJiraDateTime(meta.Fields.Updated)
	}
	return res.Key, updated, nil
}

func (c *JiraConnector) updateIssue(ctx context.Context, issueKey, summary string, adf *models.CommentNodeScheme) (string, error) {
	payload := &models.IssueScheme{
		Fields: &models.IssueFieldsScheme{
			Summary:     truncateRunes(summary, 255),
			Description: adf,
		},
	}
	if _, err := c.sdk.Issue.Update(ctx, issueKey, false, payload, nil, nil); err != nil {
		return "", err
	}
	meta, _, err := c.sdk.Issue.Get(ctx, issueKey, []string{"updated"}, nil)
	if err == nil && meta != nil && meta.Fields != nil {
		return formatJiraDateTime(meta.Fields.Updated), nil
	}
	return "", nil
}

func formatJiraDateTime(dt *models.DateTimeScheme) string {
	if dt == nil {
		return ""
	}
	t := time.Time(*dt)
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func (c *JiraConnector) getIssueConceptID(ctx context.Context, issueKey string) string {
	prop, _, err := c.sdk.Issue.Property.Get(ctx, issueKey, "okf_concept_id")
	if err != nil || prop == nil || prop.Value == nil {
		return ""
	}
	raw, err := json.Marshal(prop.Value)
	if err != nil {
		return ""
	}
	var val struct {
		ConceptID string `json:"concept_id"`
	}
	if err := json.Unmarshal(raw, &val); err != nil {
		return ""
	}
	return strings.TrimSpace(val.ConceptID)
}

func buildJiraADF(concept *bundle.Concept) (*models.CommentNodeScheme, error) {
	fmBytes, err := yaml.Marshal(concept.Frontmatter)
	if err != nil {
		return nil, err
	}
	fmText := strings.TrimSpace(string(fmBytes))
	bodyText := strings.TrimSpace(concept.Body)

	// Enforce Jira's ~32 KB description cap on body text while preserving okf-frontmatter.
	budget := jiraMaxDescriptionBytes - len(fmText) - 512
	if budget < 0 {
		budget = 0
	}
	if len(bodyText) > budget {
		const notice = "\n\n[truncated to fit Jira 32 KB limit]"
		if budget > len(notice) {
			bodyText = bodyText[:budget-len(notice)] + notice
		} else {
			bodyText = ""
		}
	}

	doc := &models.CommentNodeScheme{
		Version: 1,
		Type:    "doc",
		Content: []*models.CommentNodeScheme{
			{
				Type:  "codeBlock",
				Attrs: map[string]interface{}{"language": "yaml"},
				Content: []*models.CommentNodeScheme{
					{Type: "text", Text: fmText},
				},
			},
		},
	}
	if bodyText != "" {
		doc.AppendNode(&models.CommentNodeScheme{
			Type: "paragraph",
			Content: []*models.CommentNodeScheme{
				{Type: "text", Text: bodyText},
			},
		})
	}
	return doc, nil
}

func extractJiraADFMarkdown(doc *models.CommentNodeScheme) string {
	if doc == nil {
		return ""
	}
	var sb strings.Builder
	for _, node := range doc.Content {
		if node == nil {
			continue
		}
		var txt strings.Builder
		for _, child := range node.Content {
			if child != nil {
				txt.WriteString(child.Text)
			}
		}
		s := txt.String()
		if node.Type == "codeBlock" && (strings.Contains(s, "type:") || strings.Contains(s, "okf_version:")) {
			sb.WriteString(okfFrontmatterFence + "\n")
			sb.WriteString(strings.TrimSpace(s) + "\n```\n\n")
			continue
		}
		if s != "" {
			sb.WriteString(s + "\n\n")
		}
	}
	return sb.String()
}
