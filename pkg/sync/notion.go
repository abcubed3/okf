package sync

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/jomei/notionapi"
	"gopkg.in/yaml.v3"
)

const (
	notionMaxRichText    = 2000
	notionMaxBlocksBatch = 100
)

type NotionConnector struct {
	state  *StateManager
	config *NotionConfig
	client *rateLimitedClient
	sdk    *notionapi.Client
}

func NewNotionConnector(state *StateManager) *NotionConnector {
	return &NotionConnector{
		state: state,
	}
}

func (c *NotionConnector) Name() string {
	return "notion"
}

func (c *NotionConnector) Initialize(ctx context.Context, cfg *Config) error {
	if cfg.Connectors.Notion == nil {
		return fmt.Errorf("notion configuration missing")
	}
	c.config = cfg.Connectors.Notion
	if strings.TrimSpace(c.config.Token) == "" {
		return fmt.Errorf("notion token is required (set connectors.notion.token or OKF_NOTION_TOKEN)")
	}
	if c.client == nil {
		c.client = newRateLimitedClient(nil, 3.0)
	}
	httpClient := c.client.HTTPClient()
	if base := strings.TrimRight(c.config.BaseURL, "/"); base != "" {
		httpClient = &http.Client{
			Timeout:   httpClient.Timeout,
			Transport: &rewriteBaseURLTransport{baseURL: base, next: httpClient.Transport},
		}
	}
	c.sdk = notionapi.NewClient(notionapi.Token(c.config.Token), notionapi.WithHTTPClient(httpClient))
	return nil
}

type rewriteBaseURLTransport struct {
	baseURL string
	next    http.RoundTripper
}

func (t *rewriteBaseURLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	// Replace scheme+host (and optional base path) when BaseURL is overridden in tests/self-hosted.
	if idx := strings.Index(t.baseURL, "://"); idx != -1 {
		rest := t.baseURL[idx+3:]
		clone.URL.Scheme = t.baseURL[:idx]
		if slash := strings.Index(rest, "/"); slash != -1 {
			clone.URL.Host = rest[:slash]
		} else {
			clone.URL.Host = rest
		}
	}
	return t.next.RoundTrip(clone)
}

func (c *NotionConnector) Push(ctx context.Context, concepts []*bundle.Concept) error {
	if c.config == nil || c.sdk == nil {
		return nil
	}
	for _, concept := range concepts {
		blocks, err := buildNotionBlocks(concept)
		if err != nil {
			log.Printf("[notion] Failed to serialize %s: %v", concept.ID, err)
			continue
		}
		hash, _ := HashConcept(concept)
		title := concept.Frontmatter.Title
		if title == "" {
			title = concept.ID
		}

		extID := c.state.GetExternalID(concept.ID, c.Name())
		if extID == "" {
			pageID, rev, err := c.createPage(ctx, concept.ID, title, blocks)
			if err != nil {
				log.Printf("[notion] Failed to create page for %s: %v", concept.ID, err)
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

		rev, err := c.updatePage(ctx, extID, blocks)
		if err != nil {
			log.Printf("[notion] Failed to update page %s (%s): %v", concept.ID, extID, err)
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

func (c *NotionConnector) Pull(ctx context.Context) ([]*bundle.Concept, error) {
	if c.config == nil || c.sdk == nil {
		return nil, nil
	}

	pages, err := c.listPages(ctx)
	if err != nil {
		return nil, err
	}

	var pulled []*bundle.Concept
	for _, p := range pages {
		conceptID := p.ConceptID
		if conceptID == "" {
			conceptID = c.state.FindConceptByExternalID(c.Name(), p.ID)
		}
		if conceptID == "" {
			continue
		}
		if p.LastEditedTime != "" && c.state.GetRemoteRevision(conceptID, c.Name()) == p.LastEditedTime {
			continue
		}

		rawDoc, err := c.fetchPageMarkdown(ctx, p.ID)
		if err != nil {
			log.Printf("[notion] Failed to fetch blocks for %s (%s): %v", conceptID, p.ID, err)
			continue
		}
		concept, _, parseErr := ParseConceptEnvelope(conceptID, "", rawDoc)
		if parseErr != nil {
			log.Printf("[notion] Quarantine %s (%s): malformed okf-frontmatter YAML: %v", conceptID, p.ID, parseErr)
			continue
		}
		c.state.SetExternalID(conceptID, c.Name(), p.ID)
		if p.LastEditedTime != "" {
			c.state.SetRemoteRevision(conceptID, c.Name(), p.LastEditedTime)
		}
		pulled = append(pulled, concept)
	}
	return pulled, nil
}

type notionPageMeta struct {
	ID             string
	ConceptID      string
	LastEditedTime string
}

func (c *NotionConnector) createPage(ctx context.Context, conceptID, title string, blocks []notionapi.Block) (string, string, error) {
	firstBatch := blocks
	var remaining []notionapi.Block
	if len(blocks) > notionMaxBlocksBatch {
		firstBatch = blocks[:notionMaxBlocksBatch]
		remaining = blocks[notionMaxBlocksBatch:]
	}

	req := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(c.config.ParentID),
		},
		Properties: notionapi.Properties{
			"title": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{Text: &notionapi.Text{Content: truncateRunes(title, notionMaxRichText)}},
				},
			},
			"okf_concept_id": notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{Text: &notionapi.Text{Content: truncateRunes(conceptID, 124)}},
				},
			},
		},
		Children: firstBatch,
	}

	page, err := c.sdk.Page.Create(ctx, req)
	if err != nil {
		return "", "", err
	}
	pageID := string(page.ID)
	rev := page.LastEditedTime.UTC().Format(time.RFC3339Nano)
	if page.LastEditedTime.IsZero() {
		rev = ""
	}

	for len(remaining) > 0 {
		batch := remaining
		if len(batch) > notionMaxBlocksBatch {
			batch = remaining[:notionMaxBlocksBatch]
			remaining = remaining[notionMaxBlocksBatch:]
		} else {
			remaining = nil
		}
		if _, err := c.sdk.Block.AppendChildren(ctx, notionapi.BlockID(pageID), &notionapi.AppendBlockChildrenRequest{
			Children: batch,
		}); err != nil {
			return pageID, rev, err
		}
	}
	return pageID, rev, nil
}

func (c *NotionConnector) updatePage(ctx context.Context, pageID string, blocks []notionapi.Block) (string, error) {
	oldIDs, err := c.listBlockIDs(ctx, pageID)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(blocks); i += notionMaxBlocksBatch {
		end := i + notionMaxBlocksBatch
		if end > len(blocks) {
			end = len(blocks)
		}
		if _, err := c.sdk.Block.AppendChildren(ctx, notionapi.BlockID(pageID), &notionapi.AppendBlockChildrenRequest{
			Children: blocks[i:end],
		}); err != nil {
			return "", err
		}
	}

	// Delete old blocks only after new blocks are safely appended.
	for _, bid := range oldIDs {
		if _, err := c.sdk.Block.Delete(ctx, bid); err != nil {
			log.Printf("[notion] Warning: failed to delete old block %s: %v", bid, err)
		}
	}

	page, err := c.sdk.Page.Get(ctx, notionapi.PageID(pageID))
	if err == nil && !page.LastEditedTime.IsZero() {
		return page.LastEditedTime.UTC().Format(time.RFC3339Nano), nil
	}
	return "", nil
}

func (c *NotionConnector) listBlockIDs(ctx context.Context, blockID string) ([]notionapi.BlockID, error) {
	var ids []notionapi.BlockID
	var cursor notionapi.Cursor
	for {
		res, err := c.sdk.Block.GetChildren(ctx, notionapi.BlockID(blockID), &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    100,
		})
		if err != nil {
			return nil, err
		}
		for _, blk := range res.Results {
			ids = append(ids, blk.GetID())
		}
		if !res.HasMore || res.NextCursor == "" {
			break
		}
		cursor = notionapi.Cursor(res.NextCursor)
	}
	return ids, nil
}

func (c *NotionConnector) listPages(ctx context.Context) ([]notionPageMeta, error) {
	var pages []notionPageMeta
	var cursor notionapi.Cursor
	for {
		res, err := c.sdk.Search.Do(ctx, &notionapi.SearchRequest{
			Filter: notionapi.SearchFilter{
				Property: "object",
				Value:    "page",
			},
			StartCursor: cursor,
			PageSize:    100,
		})
		if err != nil {
			return nil, err
		}
		for _, obj := range res.Results {
			page, ok := obj.(*notionapi.Page)
			if !ok || page == nil {
				continue
			}
			cid := ""
			if prop, exists := page.Properties["okf_concept_id"]; exists {
				if rtProp, ok := prop.(*notionapi.RichTextProperty); ok && len(rtProp.RichText) > 0 {
					cid = strings.TrimSpace(rtProp.RichText[0].PlainText)
				}
			}
			rev := ""
			if !page.LastEditedTime.IsZero() {
				rev = page.LastEditedTime.UTC().Format(time.RFC3339Nano)
			}
			pages = append(pages, notionPageMeta{
				ID:             string(page.ID),
				ConceptID:      cid,
				LastEditedTime: rev,
			})
		}
		if !res.HasMore || res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}
	return pages, nil
}

func (c *NotionConnector) fetchPageMarkdown(ctx context.Context, pageID string) (string, error) {
	var sb strings.Builder
	var cursor notionapi.Cursor
	for {
		res, err := c.sdk.Block.GetChildren(ctx, notionapi.BlockID(pageID), &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    100,
		})
		if err != nil {
			return "", err
		}
		for _, blk := range res.Results {
			switch b := blk.(type) {
			case *notionapi.CodeBlock:
				var txt strings.Builder
				for _, rt := range b.Code.RichText {
					txt.WriteString(rt.PlainText)
				}
				codeStr := txt.String()
				if strings.Contains(codeStr, "type:") || strings.Contains(codeStr, "okf_version:") {
					sb.WriteString(okfFrontmatterFence + "\n")
					sb.WriteString(strings.TrimSpace(codeStr) + "\n```\n\n")
				} else {
					sb.WriteString("```\n" + codeStr + "\n```\n\n")
				}
			case *notionapi.ParagraphBlock:
				for _, rt := range b.Paragraph.RichText {
					sb.WriteString(rt.PlainText)
				}
				sb.WriteString("\n\n")
			}
		}
		if !res.HasMore || res.NextCursor == "" {
			break
		}
		cursor = notionapi.Cursor(res.NextCursor)
	}
	return sb.String(), nil
}

func buildNotionBlocks(concept *bundle.Concept) ([]notionapi.Block, error) {
	fmBytes, err := yaml.Marshal(concept.Frontmatter)
	if err != nil {
		return nil, err
	}
	var blocks []notionapi.Block
	blocks = append(blocks, notionapi.CodeBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: notionapi.ObjectTypeBlock,
			Type:   notionapi.BlockTypeCode,
		},
		Code: notionapi.Code{
			Language: "yaml",
			RichText: chunkNotionRichText(strings.TrimSpace(string(fmBytes))),
		},
	})
	for _, chunk := range chunkString(strings.TrimSpace(concept.Body), notionMaxRichText) {
		if chunk == "" {
			continue
		}
		blocks = append(blocks, notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeParagraph,
			},
			Paragraph: notionapi.Paragraph{
				RichText: []notionapi.RichText{
					{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: chunk}},
				},
			},
		})
	}
	return blocks, nil
}

func chunkNotionRichText(s string) []notionapi.RichText {
	chunks := chunkString(s, notionMaxRichText)
	res := make([]notionapi.RichText, 0, len(chunks))
	for _, ch := range chunks {
		res = append(res, notionapi.RichText{
			Type: notionapi.ObjectTypeText,
			Text: &notionapi.Text{Content: ch},
		})
	}
	return res
}

func chunkString(s string, maxLen int) []string {
	if s == "" {
		return nil
	}
	var out []string
	runes := []rune(s)
	for len(runes) > 0 {
		n := maxLen
		if len(runes) < n {
			n = len(runes)
		}
		out = append(out, string(runes[:n]))
		runes = runes[n:]
	}
	return out
}

func truncateRunes(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen])
}
