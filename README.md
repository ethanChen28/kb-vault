# kb-vault - Personal Research Vault CLI

Local-first, Git-native, LLM-ready personal knowledge base CLI.

## Features

- **URL Import**: Collect content from X.com, web articles, and more
- **Git-native**: Automatic version control for all content
- **Full-text Search**: Search through all collected content
- **Batch Import**: Import multiple URLs from a file
- **Indexing**: Automatic metadata and topic indexing
- **Extensible**: Plugin architecture for new content sources

## Installation

```bash
# From source
go install github.com/ethanChen28/kb-vault@latest

# Or build manually
git clone https://github.com/ethanChen28/kb-vault.git
cd kb-vault
go build -o kb-vault .
sudo cp kb-vault /usr/local/bin/
```

## Quick Start

```bash
# Initialize vault
kb init

# Add a single URL
kb add https://x.com/user/status/123456
kb add https://example.com/article

# Batch import from file
kb add urls.txt

# Search content
kb search agent

# View statistics
kb stats

# View git history
kb history
```

## Commands

| Command | Description |
|---------|-------------|
| `kb init` | Initialize a new knowledge vault |
| `kb add <url>` | Add a URL to the vault |
| `kb add <file>` | Batch import URLs from a file |
| `kb search <keyword>` | Search vault content |
| `kb stats` | Show vault statistics |
| `kb rebuild-index` | Rebuild search indexes |
| `kb history` | Show git history |
| `kb rollback <commit>` | Rollback to a previous commit |
| `kb version` | Show version information |

## Configuration

Config file is stored at `~/kb-vault/config.yaml`:

```yaml
vault_path: ~/kb-vault
git:
  auto_commit: true
x:
  provider: xtracticle
index:
  auto_rebuild: true
llm:
  enabled: false
```

## Project Structure

```
kb-vault/
├── cmd/              # CLI commands
├── internal/
│   ├── config/       # Configuration
│   ├── extractor/    # Content extractors
│   ├── gitlib/       # Git operations
│   └── storage/      # Local storage
├── main.go           # Entry point
└── go.mod            # Go module
```

## License

MIT