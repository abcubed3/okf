package sync

import (
	"context"
	"fmt"
	"html"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/ctreminiom/go-atlassian/v2/confluence"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"gopkg.in/yaml.v3"
)

type ConfluenceConnector struct {
	state  *StateManager
	config *ConfluenceConfig
	client *rateLimitedClient
	sdk    *confluence.Client
}

func NewConfluenceConnector(state *StateManager) *ConfluenceConnector {
	return &ConfluenceConnector{
		state: state,
	}
}

func (c *ConfluenceConnector) Name() string {
	return "confluence"
}

func (c *ConfluenceConnector) Initialize(ctx context.Context, cfg *Config) error {
	if cfg.Connectors.Confluence == nil {
		return fmt.Errorf("confluence configuration missing")
	}
	c.config = cfg.Connectors.Confluence
	if strings.TrimSpace(c.config.Token) == "" {
		return fmt.Errorf("confluence token is required (set connectors.confluence.token or OKF_CONFLUENCE_TOKEN)")
	}
	if strings.TrimSpace(c.config.SpaceKey) == "" {
		return fmt.Errorf("confluence space_key is required")
	}
	site := strings.TrimRight(c.config.BaseURL, "/")
	site = strings.TrimSuffix(site, "/wiki/rest/api")
	site = strings.TrimSuffix(site, "/rest/api")
	if site == "" {
		if strings.TrimSpace(c.config.Domain) == "" {
			return fmt.Errorf("confluence domain is required")
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
	sdk, err := confluence.New(c.client, site)
	if err != nil {
		return fmt.Errorf("failed to initialize confluence SDK client: %w", err)
	}
	if c.config.Email != "" {
		sdk.Auth.SetBasicAuth(c.config.Email, c.config.Token)
	} else {
		sdk.Auth.SetBearerToken(c.config.Token)
	}
	c.sdk = sdk
	return nil
}

func (c *ConfluenceConnector) Push(ctx context.Context, concepts []*bundle.Concept) error {
	if c.config == nil || c.sdk == nil {
		return nil
	}
	for _, concept := range concepts {
		storageHTML, err := buildConfluenceStorage(concept)
		if err != nil {
			log.Printf("[confluence] Failed to serialize %s: %v", concept.ID, err)
			continue
		}
		hash, _ := HashConcept(concept)
		title := concept.Frontmatter.Title
		if title == "" {
			title = concept.ID
		}

		extID := c.state.GetExternalID(concept.ID, c.Name())
		if extID == "" {
			pageID, rev, err := c.createPage(ctx, concept.ID, title, storageHTML)
			if err != nil {
				log.Printf("[confluence] Failed to create page for %s: %v", concept.ID, err)
				continue
			}
			c.state.SetExternalID(concept.ID, c.Name(), pageID)
			if hash != "" {
				c.state.SetContentHash(concept.ID, c.Name(), hash)
			}
			if rev != "" {
				c.state.SetRemoteRevision(concept.ID, c.Name(), rev)
			}
			continue
		}

		rev, err := c.updatePage(ctx, extID, title, storageHTML)
		if err != nil {
			log.Printf("[confluence] Failed to update page %s (%s): %v", concept.ID, extID, err)
			continue
		}
		if hash != "" {
			c.state.SetContentHash(concept.ID, c.Name(), hash)
		}
		if rev != "" {
			c.state.SetRemoteRevision(concept.ID, c.Name(), rev)
		}
	}
	return nil
}

func (c *ConfluenceConnector) Pull(ctx context.Context) ([]*bundle.Concept, error) {
	if c.config == nil || c.sdk == nil {
		return nil, nil
	}

	var pulled []*bundle.Concept
	cursor := ""
	const limit = 50
	cql := fmt.Sprintf(`space = "%s" AND type = "page"`, c.config.SpaceKey)
	expand := []string{"body.storage", "version"}

	for {
		pageRes, _, err := c.sdk.Content.Search(ctx, cql, "", expand, cursor, limit)
		if err != nil {
			return nil, fmt.Errorf("confluence search failed: %w", err)
		}

		for _, page := range pageRes.Results {
			conceptID := c.getPageConceptID(ctx, page.ID)
			if conceptID == "" {
				conceptID = c.state.FindConceptByExternalID(c.Name(), page.ID)
			}
			if conceptID == "" {
				continue
			}

			verNum := 0
			if page.Version != nil {
				verNum = page.Version.Number
			}
			rev := strconv.Itoa(verNum)
			if verNum > 0 && c.state.GetRemoteRevision(conceptID, c.Name()) == rev {
				continue
			}

			storageVal := ""
			if page.Body != nil && page.Body.Storage != nil {
				storageVal = page.Body.Storage.Value
			}
			rawDoc := extractConfluenceStorageMarkdown(storageVal)
			concept, _, parseErr := ParseConceptEnvelope(conceptID, "", rawDoc)
			if parseErr != nil {
				log.Printf("[confluence] Quarantine %s (%s): malformed okf-frontmatter YAML: %v", conceptID, page.ID, parseErr)
				continue
			}
			c.state.SetExternalID(conceptID, c.Name(), page.ID)
			if verNum > 0 {
				c.state.SetRemoteRevision(conceptID, c.Name(), rev)
			}
			pulled = append(pulled, concept)
		}

		if pageRes.Links == nil || pageRes.Links.Next == "" {
			break
		}
		// Extract cursor parameter from _links.next if present, otherwise stop.
		nextCursor := extractQueryParam(pageRes.Links.Next, "cursor")
		if nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}
	return pulled, nil
}

func extractQueryParam(rawURL, key string) string {
	idx := strings.Index(rawURL, "?")
	if idx == -1 {
		return ""
	}
	for _, pair := range strings.Split(rawURL[idx+1:], "&") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 && kv[0] == key {
			return kv[1]
		}
	}
	return ""
}

func (c *ConfluenceConnector) createPage(ctx context.Context, conceptID, title, storageHTML string) (string, string, error) {
	payload := &models.ContentScheme{
		Type:  "page",
		Title: title,
		Space: &models.SpaceScheme{Key: c.config.SpaceKey},
		Body: &models.BodyScheme{
			Storage: &models.BodyNodeScheme{
				Value:          storageHTML,
				Representation: "storage",
			},
		},
	}
	created, _, err := c.sdk.Content.Create(ctx, payload)
	if err != nil {
		return "", "", err
	}
	_, _, _ = c.sdk.Content.Property.Create(ctx, created.ID, &models.ContentPropertyPayloadScheme{
		Key:   "okf_concept_id",
		Value: conceptID,
	})
	rev := ""
	if created.Version != nil && created.Version.Number > 0 {
		rev = strconv.Itoa(created.Version.Number)
	}
	return created.ID, rev, nil
}

func (c *ConfluenceConnector) updatePage(ctx context.Context, pageID, title, storageHTML string) (string, error) {
	cur, _, err := c.sdk.Content.Get(ctx, pageID, []string{"version"}, 0)
	if err != nil {
		return "", err
	}
	nextVer := 1
	if cur.Version != nil {
		nextVer = cur.Version.Number + 1
	}
	payload := &models.ContentScheme{
		Type:    "page",
		Title:   title,
		Version: &models.ContentVersionScheme{Number: nextVer},
		Body: &models.BodyScheme{
			Storage: &models.BodyNodeScheme{
				Value:          storageHTML,
				Representation: "storage",
			},
		},
	}
	updated, _, err := c.sdk.Content.Update(ctx, pageID, payload)
	if err != nil {
		return "", err
	}
	if updated.Version != nil && updated.Version.Number > 0 {
		return strconv.Itoa(updated.Version.Number), nil
	}
	return "", nil
}

func buildConfluenceStorage(concept *bundle.Concept) (string, error) {
	fmBytes, err := yaml.Marshal(concept.Frontmatter)
	if err != nil {
		return "", err
	}
	yamlSafe := strings.ReplaceAll(strings.TrimSpace(string(fmBytes)), "]]>", "]]]]><![CDATA[>")
	var b strings.Builder
	b.WriteString(`<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">yaml</ac:parameter><ac:plain-text-body><![CDATA[`)
	b.WriteString(yamlSafe)
	b.WriteString(`]]></ac:plain-text-body></ac:structured-macro>`)
	if strings.TrimSpace(concept.Body) != "" {
		b.WriteString("<p>")
		b.WriteString(html.EscapeString(strings.TrimSpace(concept.Body)))
		b.WriteString("</p>")
	}
	return b.String(), nil
}

var (
	reConfluenceCodeMacro = regexp.MustCompile(`(?s)<ac:structured-macro[^>]*ac:name="code"[^>]*>.*?<ac:plain-text-body><!\[CDATA\[(.*?)\]\]></ac:plain-text-body>.*?</ac:structured-macro>`)
	reHTMLTags            = regexp.MustCompile(`<[^>]+>`)
)

func extractConfluenceStorageMarkdown(storage string) string {
	var sb strings.Builder
	rest := storage
	if m := reConfluenceCodeMacro.FindStringSubmatchIndex(storage); m != nil {
		yamlText := storage[m[2]:m[3]]
		sb.WriteString(okfFrontmatterFence + "\n")
		sb.WriteString(strings.TrimSpace(yamlText) + "\n```\n\n")
		rest = storage[m[1]:]
	}
	plain := reHTMLTags.ReplaceAllString(rest, "\n")
	plain = html.UnescapeString(plain)
	sb.WriteString(strings.TrimSpace(plain))
	return sb.String()
}

func (c *ConfluenceConnector) getPageConceptID(ctx context.Context, pageID string) string {
	prop, _, err := c.sdk.Content.Property.Get(ctx, pageID, "okf_concept_id")
	if err != nil || prop == nil || prop.Value == nil {
		return ""
	}
	if m, ok := prop.Value.(map[string]interface{}); ok {
		if s, ok := m["concept_id"].(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
