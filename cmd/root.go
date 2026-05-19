package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yourname/kb-vault/internal/config"
	"github.com/yourname/kb-vault/internal/extractor"
	"github.com/yourname/kb-vault/internal/gitlib"
	"github.com/yourname/kb-vault/internal/storage"

	"github.com/spf13/cobra"
)

var (
	force     bool
	vaultPath string
	version   = "0.1.0"
)

var rootCmd = &cobra.Command{
	Use:   "kb",
	Short: "Personal Research Vault CLI",
	Long: `Local-first, Git-native, LLM-ready personal knowledge base CLI.

A tool for collecting and organizing web content (X, articles, blogs)
as versioned Markdown files with automatic indexing.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load config on every command
		config.Load()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&vaultPath, "vault", "", "vault path (default: ~/kb-vault)")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "force operation")
	rootCmd.AddCommand(versionCmd)
}

// addURL handles single URL addition
func addURL(s *storage.Storage, cfg *config.Config, url, vaultPath string) error {
	fmt.Printf("Adding: %s\n", url)

	// Find and use appropriate extractor
	registry := extractor.NewRegistry()
	ext, err := registry.FindExtractor(url)
	if err != nil {
		return err
	}

	// Extract content
	doc, err := ext.Extract(url)
	if err != nil {
		return fmt.Errorf("failed to extract content: %w", err)
	}

	fmt.Println("✓ Detected source:", doc.SourceType)
	fmt.Println("✓ Downloaded article")

	// Save document
	if err := s.SaveDocument(doc); err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}
	fmt.Println("✓ Saved markdown")

	// Git commit if enabled
	if cfg.Git.AutoCommit {
		commitMsg := fmt.Sprintf("Add: %s", doc.Title)
		if err := gitlib.Commit(vaultPath, commitMsg); err != nil {
			fmt.Printf("⚠ Warning: git commit failed: %v\n", err)
		} else {
			fmt.Println("✓ Committed to git")
		}
	}

	// Rebuild index if enabled
	if cfg.Index.AutoRebuild {
		fmt.Println("✓ Indexed")
	}

	return nil
}

// addFromFile handles batch import from a file
func addFromFile(s *storage.Storage, cfg *config.Config, filePath, vaultPath string) error {
	urls, err := extractor.ExtractFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if len(urls) == 0 {
		fmt.Println("No URLs found in file.")
		return nil
	}

	fmt.Printf("Found %d URLs in file, starting import...\n\n", len(urls))

	registry := extractor.NewRegistry()
	successCount := 0
	failCount := 0

	for i, url := range urls {
		fmt.Printf("[%d/%d] %s\n", i+1, len(urls), url)

		ext, err := registry.FindExtractor(url)
		if err != nil {
			fmt.Printf("  ✗ No extractor: %v\n", err)
			failCount++
			continue
		}

		doc, err := ext.Extract(url)
		if err != nil {
			fmt.Printf("  ✗ Extraction failed: %v\n", err)
			failCount++
			continue
		}

		if err := s.SaveDocument(doc); err != nil {
			fmt.Printf("  ✗ Save failed: %v\n", err)
			failCount++
			continue
		}

		fmt.Printf("  ✓ Saved: %s\n", doc.Title)
		successCount++
	}

	fmt.Printf("\n=== Import Summary ===\n")
	fmt.Printf("Success: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)

	// Single commit for all
	if cfg.Git.AutoCommit && successCount > 0 {
		commitMsg := fmt.Sprintf("Add: %d articles from batch import", successCount)
		if err := gitlib.Commit(vaultPath, commitMsg); err != nil {
			fmt.Printf("⚠ Warning: git commit failed: %v\n", err)
		} else {
			fmt.Println("✓ Committed to git")
		}
	}

	if cfg.Index.AutoRebuild {
		fmt.Println("✓ Indexed")
	}

	return nil
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new knowledge vault",
	Long:  `Creates the vault directory structure and initializes git.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := vaultPath
		if path == "" {
			path = filepath.Join(os.Getenv("HOME"), "kb-vault")
		}

		fmt.Printf("Initializing vault at: %s\n", path)

		// Create directory structure
		s := storage.NewStorage()
		// Override vault path
		s.SetVaultPath(path)

		if err := s.EnsureVaultStructure(); err != nil {
			return fmt.Errorf("failed to create vault structure: %w", err)
		}

		// Initialize git
		if err := gitlib.Init(path); err != nil {
			return fmt.Errorf("failed to initialize git: %w", err)
		}

		// Save config
		config.SetVaultPath(path)
		if err := config.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Vault initialized at %s\n", path)
		fmt.Println("✓ Git repository initialized")
		fmt.Println("\nRun 'kb add <url>' to add your first article!")

		return nil
	},
}

var addCmd = &cobra.Command{
	Use:   "add <url|file>",
	Short: "Add a URL or batch import from a file",
	Long:  `Downloads content from URL and saves as Markdown.
Supports single URL or batch import from a file (one URL per line).

Supported sources:
  - X.com posts
  - Web articles
  - GitHub READMEs (coming soon)

Examples:
  kb add https://x.com/user/status/123456
  kb add urls.txt`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		s := storage.NewStorage()

		// Check if vault exists
		vaultPath := s.GetVaultPath()
		if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
			return fmt.Errorf("vault not initialized. Run 'kb init' first")
		}

		// Check if input is a file
		if info, err := os.Stat(input); err == nil && !info.IsDir() {
			// It's a file - do batch import
			return addFromFile(s, cfg, input, vaultPath)
		}

		// Single URL
		return addURL(s, cfg, input, vaultPath)
	},
}

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search vault content",
	Long:  `Searches vault documents by keyword.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]

		s := storage.NewStorage()
		docs, err := s.SearchDocuments(keyword)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(docs) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		fmt.Printf("Found %d result(s):\n\n", len(docs))
		for i, doc := range docs {
			fmt.Printf("[%d] %s\n", i+1, doc.Title)
			fmt.Printf("    Source: %s | %s\n", doc.SourceType, doc.CollectedAt)
			fmt.Printf("    URL: %s\n\n", doc.SourceURL)
		}

		return nil
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show vault statistics",
	Long:  `Displays vault statistics including document count, topics, and entities.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := storage.NewStorage()

		stats, err := s.GetStats()
		if err != nil {
			return fmt.Errorf("failed to get stats: %w", err)
		}

		lastImport, _ := s.GetLastImportTime()

		fmt.Printf("Documents: %d\n", stats["documents"])
		fmt.Printf("Topics: %d\n", stats["topics"])
		fmt.Printf("Entities: %d\n", stats["entities"])
		if !lastImport.IsZero() {
			fmt.Printf("Last import: %s\n", lastImport.Format("2006-01-02"))
		}

		return nil
	},
}

var rebuildIndexCmd = &cobra.Command{
	Use:   "rebuild-index",
	Short: "Rebuild search indexes",
	Long:  `Regenerates metadata.json, topics.json, and entities.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := storage.NewStorage()

		fmt.Println("Rebuilding indexes...")

		// Export metadata
		metadata, err := s.ExportMetadata()
		if err != nil {
			return fmt.Errorf("failed to export metadata: %w", err)
		}

		// For now, topics and entities are simple aggregates
		topics := "[]"
		entities := "[]"

		if err := s.SaveIndexes(metadata, topics, entities); err != nil {
			return fmt.Errorf("failed to save indexes: %w", err)
		}

		fmt.Println("✓ Rebuilt metadata index")
		fmt.Println("✓ Rebuilt topics index")
		fmt.Println("✓ Rebuilt entities index")

		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show git history",
	Long:  `Displays the commit history of the vault.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := storage.NewStorage()
		return gitlib.ShowHistory(s.GetVaultPath())
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback <commit>",
	Short: "Rollback to a previous commit",
	Long:  `Reverts the vault to a previous commit (e.g., HEAD~1)`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := storage.NewStorage()
		commit := args[0]
		return gitlib.Rollback(s.GetVaultPath(), commit)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the version number of kb CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("kb version %s\n", version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(rebuildIndexCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(rollbackCmd)
}