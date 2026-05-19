package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourname/kb/internal/config"
	"github.com/yourname/kb/internal/extractor"

	"gopkg.in/yaml.v3"
)

type Storage struct {
	vaultPath string
}

func NewStorage() *Storage {
	return &Storage{
		vaultPath: config.GetVaultPath(),
	}
}

// SetVaultPath sets the vault path
func (s *Storage) SetVaultPath(path string) {
	s.vaultPath = path
}

func (s *Storage) EnsureVaultStructure() error {
	dirs := []string{
		"sources",
		"notes",
		"indexes",
		"cache",
		"config",
		"inbox",
	}

	for _, dir := range dirs {
		path := filepath.Join(s.vaultPath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (s *Storage) SaveDocument(doc *extractor.Document) error {
	// Generate ID based on date
	now := time.Now()
	if doc.ID == "" {
		doc.ID = fmt.Sprintf("kb_%s", now.Format("20060102_150405"))
	}
	if doc.CollectedAt == "" {
		doc.CollectedAt = now.Format("2006-01-02T15:04:05")
	}

	// Create date-based directory structure
	datePath := now.Format("2006/01/02")
	slug := s.generateSlug(doc.Title)
	docDir := filepath.Join(s.vaultPath, "sources", datePath, slug)

	if err := os.MkdirAll(docDir, 0755); err != nil {
		return fmt.Errorf("failed to create document directory: %w", err)
	}

	// Save content.md
	contentPath := filepath.Join(docDir, "content.md")
	content := s.formatContent(doc)
	if err := os.WriteFile(contentPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write content.md: %w", err)
	}

	// Save metadata.yaml
	metadataPath := filepath.Join(docDir, "metadata.yaml")
	metadata := s.formatMetadata(doc)
	if err := os.WriteFile(metadataPath, []byte(metadata), 0644); err != nil {
		return fmt.Errorf("failed to write metadata.yaml: %w", err)
	}

	// Save raw.json
	if doc.RawJSON != "" {
		rawPath := filepath.Join(docDir, "raw.json")
		if err := os.WriteFile(rawPath, []byte(doc.RawJSON), 0644); err != nil {
			return fmt.Errorf("failed to write raw.json: %w", err)
		}
	}

	return nil
}

func (s *Storage) formatContent(doc *extractor.Document) string {
	return fmt.Sprintf(`# %s

Source: %s
Author: %s
Collected At: %s

---

## Content

%s
`, doc.Title, doc.SourceType, doc.Author, doc.CollectedAt, doc.Content)
}

func (s *Storage) formatMetadata(doc *extractor.Document) string {
	data := map[string]interface{}{
		"id":           doc.ID,
		"title":        doc.Title,
		"source_type":  doc.SourceType,
		"source_url":   doc.SourceURL,
		"author":       doc.Author,
		"collected_at": doc.CollectedAt,
		"tags":         doc.Tags,
		"topics":       doc.Topics,
		"language":     doc.Language,
	}

	yamlData, _ := yaml.Marshal(data)
	return string(yamlData)
}

func (s *Storage) generateSlug(title string) string {
	// Simple slug generation
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters
	var result strings.Builder
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result.WriteRune(c)
		}
	}
	return result.String()
}

func (s *Storage) GetVaultPath() string {
	return s.vaultPath
}

// GetAllDocuments returns all documents in the vault
func (s *Storage) GetAllDocuments() ([]*extractor.Document, error) {
	var docs []*extractor.Document

	sourcesDir := filepath.Join(s.vaultPath, "sources")
	err := filepath.Walk(sourcesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "metadata.yaml" {
			doc, err := s.loadDocumentFromPath(path)
			if err != nil {
				return err
			}
			docs = append(docs, doc)
		}
		return nil
	})

	return docs, err
}

func (s *Storage) loadDocumentFromPath(metadataPath string) (*extractor.Document, error) {
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}

	var meta map[string]interface{}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	doc := &extractor.Document{
		ID:          getString(meta, "id"),
		Title:       getString(meta, "title"),
		SourceType:  getString(meta, "source_type"),
		SourceURL:   getString(meta, "source_url"),
		Author:      getString(meta, "author"),
		CollectedAt: getString(meta, "collected_at"),
		Language:    getString(meta, "language"),
	}

	if tags, ok := meta["tags"].([]interface{}); ok {
		for _, t := range tags {
			if str, ok := t.(string); ok {
				doc.Tags = append(doc.Tags, str)
			}
		}
	}

	if topics, ok := meta["topics"].([]interface{}); ok {
		for _, t := range topics {
			if str, ok := t.(string); ok {
				doc.Topics = append(doc.Topics, str)
			}
		}
	}

	// Load content
	contentPath := filepath.Join(filepath.Dir(metadataPath), "content.md")
	if content, err := os.ReadFile(contentPath); err == nil {
		doc.Content = string(content)
	}

	// Load raw.json if exists
	rawPath := filepath.Join(filepath.Dir(metadataPath), "raw.json")
	if raw, err := os.ReadFile(rawPath); err == nil {
		doc.RawJSON = string(raw)
	}

	return doc, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// SearchDocuments searches documents by keyword
func (s *Storage) SearchDocuments(keyword string) ([]*extractor.Document, error) {
	allDocs, err := s.GetAllDocuments()
	if err != nil {
		return nil, err
	}

	var results []*extractor.Document
	keyword = strings.ToLower(keyword)

	for _, doc := range allDocs {
		if strings.Contains(strings.ToLower(doc.Title), keyword) ||
			strings.Contains(strings.ToLower(doc.Content), keyword) {
			results = append(results, doc)
		}
	}

	return results, nil
}

// GetStats returns vault statistics
func (s *Storage) GetStats() (map[string]int, error) {
	stats := map[string]int{
		"documents": 0,
		"topics":    0,
		"entities":  0,
	}

	docs, err := s.GetAllDocuments()
	if err != nil {
		return stats, err
	}

	stats["documents"] = len(docs)

	// Count topics and entities
	topicSet := make(map[string]bool)
	entitySet := make(map[string]bool)

	for _, doc := range docs {
		for _, topic := range doc.Topics {
			topicSet[topic] = true
		}
		for _, tag := range doc.Tags {
			entitySet[tag] = true
		}
	}

	stats["topics"] = len(topicSet)
	stats["entities"] = len(entitySet)

	return stats, nil
}

func (s *Storage) SaveIndexes(metadata, topics, entities string) error {
	indexesDir := filepath.Join(s.vaultPath, "indexes")

	if err := os.WriteFile(filepath.Join(indexesDir, "metadata.json"), []byte(metadata), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(indexesDir, "topics.json"), []byte(topics), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(indexesDir, "entities.json"), []byte(entities), 0644); err != nil {
		return err
	}

	return nil
}

func (s *Storage) LoadIndexes() (metadata, topics, entities string, err error) {
	indexesDir := filepath.Join(s.vaultPath, "indexes")

	if data, err := os.ReadFile(filepath.Join(indexesDir, "metadata.json")); err == nil {
		metadata = string(data)
	}
	if data, err := os.ReadFile(filepath.Join(indexesDir, "topics.json")); err == nil {
		topics = string(data)
	}
	if data, err := os.ReadFile(filepath.Join(indexesDir, "entities.json")); err == nil {
		entities = string(data)
	}

	return
}

func (s *Storage) GetLastImportTime() (time.Time, error) {
	sourcesDir := filepath.Join(s.vaultPath, "sources")
	var latest time.Time

	err := filepath.Walk(sourcesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})

	return latest, err
}

// ExportMetadata exports all metadata as JSON
func (s *Storage) ExportMetadata() (string, error) {
	docs, err := s.GetAllDocuments()
	if err != nil {
		return "", err
	}

	type ExportDoc struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		SourceType  string   `json:"source_type"`
		SourceURL   string   `json:"source_url"`
		Author      string   `json:"author"`
		CollectedAt string   `json:"collected_at"`
		Tags        []string `json:"tags"`
		Topics      []string `json:"topics"`
		Language    string   `json:"language"`
	}

	exportDocs := make([]ExportDoc, len(docs))
	for i, doc := range docs {
		exportDocs[i] = ExportDoc{
			ID:          doc.ID,
			Title:       doc.Title,
			SourceType:  doc.SourceType,
			SourceURL:   doc.SourceURL,
			Author:      doc.Author,
			CollectedAt: doc.CollectedAt,
			Tags:        doc.Tags,
			Topics:      doc.Topics,
			Language:    doc.Language,
		}
	}

	data, err := json.MarshalIndent(exportDocs, "", "  ")
	return string(data), err
}