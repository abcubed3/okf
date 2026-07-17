package harvester

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/gocolly/colly/v2"
)

// WebHarvester crawls public websites and converts them into OKF bundles.
type WebHarvester struct {
	startURL string
	maxDepth int
	llm      *LLMClassifier
}

// NewWebHarvester creates a new WebHarvester instance.
func NewWebHarvester(startURL string, maxDepth int, llm *LLMClassifier) *WebHarvester {
	return &WebHarvester{
		startURL: startURL,
		maxDepth: maxDepth,
		llm:      llm,
	}
}

// Harvest executes the crawler and returns a list of OKF concepts.
func (h *WebHarvester) Harvest(ctx context.Context) ([]*bundle.Concept, error) {
	parsedURL, err := url.Parse(h.startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
	}

	c := colly.NewCollector(
		colly.AllowedDomains(parsedURL.Host),
		colly.MaxDepth(h.maxDepth),
		colly.Async(true),
	)

	// Rate limiting to avoid getting blocked
	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       500 * time.Millisecond,
	}); err != nil {
		return nil, fmt.Errorf("failed to set rate limit: %w", err)
	}

	var concepts []*bundle.Concept
	var mu sync.Mutex

	c.OnHTML("html", func(e *colly.HTMLElement) {
		doc := e.DOM

		// Strip boilerplate/chrome
		doc.Find("script, style, nav, footer, header, aside, .sidebar").Remove()

		htmlContent, err := doc.Html()
		if err != nil {
			fmt.Printf("Warning: failed to get HTML for %s: %v\n", e.Request.URL, err)
			return
		}

		// Convert HTML to Markdown
		converter := md.NewConverter(parsedURL.Host, true, nil)
		markdown, err := converter.ConvertString(htmlContent)
		if err != nil {
			fmt.Printf("Warning: failed to convert markdown for %s: %v\n", e.Request.URL, err)
			return
		}

		title := strings.TrimSpace(doc.Find("title").Text())

		// Generate Concept ID and Path
		urlPath := e.Request.URL.Path
		urlPath = strings.TrimPrefix(urlPath, "/")
		if urlPath == "" {
			urlPath = "index"
		} else if strings.HasSuffix(urlPath, "/") {
			urlPath = urlPath + "index"
		}
		
		id := "web/" + urlPath
		path := id + ".md"

		// Extract Metadata using LLM (if provided)
		var fm bundle.Frontmatter
		if h.llm != nil {
			extractedFm, llmErr := h.llm.Classify(ctx, title, markdown)
			if llmErr != nil {
				fmt.Printf("Warning: LLM classification failed for %s: %v\n", e.Request.URL, llmErr)
				// Fallback
				fm = bundle.Frontmatter{
					Type:      "Documentation",
					Title:     title,
				}
			} else {
				fm = extractedFm
			}
		} else {
			fm = bundle.Frontmatter{
				Type:  "Documentation",
				Title: title,
			}
		}

		// Always set the resource and timestamp
		fm.Resource = e.Request.URL.String()
		fm.Timestamp = time.Now().Format(time.RFC3339)

		concept := &bundle.Concept{
			ID:          id,
			Path:        path,
			Frontmatter: fm,
			Body:        markdown,
		}

		// Thread-safe append
		mu.Lock()
		concepts = append(concepts, concept)
		mu.Unlock()

		// Attempt to crawl linked pages
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			link, exists := s.Attr("href")
			if exists {
				e.Request.Visit(link)
			}
		})
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Crawling: %s\n", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request Error %s: %v\n", r.Request.URL.String(), err)
	})

	fmt.Printf("Starting web harvest at %s with max depth %d...\n", h.startURL, h.maxDepth)
	if err := c.Visit(h.startURL); err != nil {
		return nil, fmt.Errorf("visit failed: %w", err)
	}
	c.Wait()

	return concepts, nil
}
