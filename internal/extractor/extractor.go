package extractor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Document represents extracted content
type Document struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Author      string   `json:"author"`
	SourceType  string   `json:"source_type"`
	SourceURL   string   `json:"source_url"`
	CreatedAt   string   `json:"created_at"`
	CollectedAt string   `json:"collected_at"`
	Tags        []string `json:"tags"`
	Topics      []string `json:"topics"`
	Language    string   `json:"language"`
	RawJSON     string   `json:"raw_json"`
}

// Extractor interface for different source types
type Extractor interface {
	Match(url string) bool
	Extract(url string) (*Document, error)
}

// Registry holds all available extractors
type Registry struct {
	extractors []Extractor
}

func NewRegistry() *Registry {
	return &Registry{
		extractors: []Extractor{
			&XExtractor{},
			&WebExtractor{},
		},
	}
}

func (r *Registry) FindExtractor(urlStr string) (Extractor, error) {
	// Try more specific extractors first
	for _, e := range r.extractors {
		if e.Match(urlStr) {
			return e, nil
		}
	}
	// Default to web extractor
	return &WebExtractor{}, nil
}

// XExtractor handles X.com URLs using xtracticle or fallback
type XExtractor struct{}

func (e *XExtractor) Match(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	return strings.Contains(host, "x.com") || strings.Contains(host, "twitter.com")
}

func (e *XExtractor) Extract(urlStr string) (*Document, error) {
	// Try xtracticle first
	doc, err := e.extractWithXtracticle(urlStr)
	if err == nil && doc.Content != "" {
		return doc, nil
	}

	// Fallback to basic extraction
	return e.extractBasic(urlStr)
}

func (e *XExtractor) extractWithXtracticle(urlStr string) (*Document, error) {
	// Try using npx xtracticle
	cmd := exec.Command("npx", "xtracticle", urlStr)
	cmd.Env = append(os.Environ(), "CI=true")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("xtracticle failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse xtracticle output: %w", err)
	}

	content := ""
	if c, ok := result["content"].(string); ok {
		content = c
	} else if c, ok := result["text"].(string); ok {
		content = c
	} else if c, ok := result["body"].(string); ok {
		content = c
	}

	title := ""
	if t, ok := result["title"].(string); ok {
		title = t
	}

	author := ""
	if a, ok := result["author"].(string); ok {
		author = a
	} else if a, ok := result["username"].(string); ok {
		author = a
	}

	// Get raw JSON for re-processing
	rawJSON, _ := json.Marshal(result)

	return &Document{
		SourceType:  "x",
		SourceURL:   urlStr,
		Title:       title,
		Content:     content,
		Author:      author,
		CollectedAt: time.Now().Format("2006-01-02T15:04:05"),
		RawJSON:     string(rawJSON),
	}, nil
}

func (e *XExtractor) extractBasic(urlStr string) (*Document, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	// Extract post ID from URL
	pathParts := strings.Split(u.Path, "/")
	var postID string
	for i, part := range pathParts {
		if part == "status" && i+1 < len(pathParts) {
			postID = pathParts[i+1]
			break
		}
	}

	return &Document{
		SourceType:  "x",
		SourceURL:   urlStr,
		CollectedAt: time.Now().Format("2006-01-02T15:04:05"),
		Title:       fmt.Sprintf("X Post %s", postID),
		Content:     fmt.Sprintf("# X Content\n\nURL: %s\n\n(Content extraction placeholder - install xtracticle for full extraction)\n\nTo install: npm install -g xtracticle", urlStr),
	}, nil
}

// WebExtractor handles generic web pages
type WebExtractor struct{}

func (e *WebExtractor) Match(urlStr string) bool {
	// Match any HTTP/HTTPS URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func (e *WebExtractor) Extract(urlStr string) (*Document, error) {
	// Fetch the page
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract title
	title := doc.Find("title").First().Text()
	if title == "" {
		title = doc.Find("h1").First().Text()
	}

	// Remove script and style elements
	doc.Find("script, style, nav, header, footer, aside").Remove()

	// Get main content
	content := doc.Find("article, main, .content, .post-content, #content, .article-content").First()
	if content.Length() == 0 {
		content = doc.Find("body")
	}

	// Extract text
	var text strings.Builder
	content.Each(func(i int, s *goquery.Selection) {
		text.WriteString(s.Text())
	})

	// Clean up whitespace
	cleanedContent := cleanWhitespace(text.String())

	// Try to detect author
	author := ""
	doc.Find("meta[name='author'], meta[property='article:author']").Each(func(i int, s *goquery.Selection) {
		if authorAttr, ok := s.Attr("content"); ok && author == "" {
			author = authorAttr
		}
	})

	// Get published date
	publishedAt := ""
	doc.Find("meta[property='article:published_time'], time[datetime]").Each(func(i int, s *goquery.Selection) {
		if dateAttr, ok := s.Attr("content"); ok && publishedAt == "" {
			publishedAt = strings.Split(dateAttr, "T")[0]
		} else if dateAttr, ok := s.Attr("datetime"); ok && publishedAt == "" {
			publishedAt = strings.Split(dateAttr, "T")[0]
		}
	})

	// Get description as summary
	description := ""
	doc.Find("meta[name='description'], meta[property='og:description']").Each(func(i int, s *goquery.Selection) {
		if descAttr, ok := s.Attr("content"); ok && description == "" {
			description = descAttr
		}
	})

	// Build markdown content
	markdownContent := fmt.Sprintf("# %s\n\n", title)
	if author != "" {
		markdownContent += fmt.Sprintf("**Author:** %s\n", author)
	}
	if publishedAt != "" {
		markdownContent += fmt.Sprintf("**Published:** %s\n", publishedAt)
	}
	markdownContent += fmt.Sprintf("**Source:** %s\n\n---\n\n", urlStr)
	markdownContent += description + "\n\n"
	markdownContent += "## Content\n\n" + cleanedContent

	return &Document{
		SourceType:  "web",
		SourceURL:   urlStr,
		Title:       title,
		Content:     markdownContent,
		Author:      author,
		CreatedAt:   publishedAt,
		CollectedAt: time.Now().Format("2006-01-02T15:04:05"),
		Language:    detectLanguage(cleanedContent),
	}, nil
}

func cleanWhitespace(s string) string {
	// Replace multiple whitespace with single space
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return strings.Join(result, "\n\n")
}

func detectLanguage(text string) string {
	// Simple language detection based on common words
	// This is a very basic implementation
	lower := strings.ToLower(text)
	
	chineseChars := 0
	englishWords := 0
	
	for _, r := range lower {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseChars++
		}
	}
	
	words := strings.Fields(lower)
	for _, word := range words {
		if len(word) > 3 {
			englishWords++
		}
	}
	
	if chineseChars > englishWords*2 {
		return "zh"
	}
	return "en"
}

// ExtractFromFile reads URLs from a file and returns them
func ExtractFromFile(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var urls []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}

	return urls, nil
}