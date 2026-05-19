package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	VaultPath string   `mapstructure:"vault_path"`
	Git       GitConfig `mapstructure:"git"`
	X         XConfig   `mapstructure:"x"`
	Index     IndexConfig `mapstructure:"index"`
	LLM       LLMConfig  `mapstructure:"llm"`
}

type GitConfig struct {
	AutoCommit bool `mapstructure:"auto_commit"`
}

type XConfig struct {
	Provider string `mapstructure:"provider"`
}

type IndexConfig struct {
	AutoRebuild bool `mapstructure:"auto_rebuild"`
}

type LLMConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

var (
	cfg  *Config
	cfgViper *viper.Viper
)

func Load() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	// Try to load from default locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	defaultPath := filepath.Join(homeDir, "kb-vault", "config.yaml")

	cfgViper = viper.New()
	cfgViper.SetConfigFile(defaultPath)
	cfgViper.SetDefault("vault_path", filepath.Join(homeDir, "kb-vault"))
	cfgViper.SetDefault("git.auto_commit", true)
	cfgViper.SetDefault("x.provider", "xtracticle")
	cfgViper.SetDefault("index.auto_rebuild", true)
	cfgViper.SetDefault("llm.enabled", false)

	// Don't error if config doesn't exist yet - we'll use defaults
	if err := cfgViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	cfg = &Config{}
	if err := cfgViper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

func GetVaultPath() string {
	if cfg == nil {
		Load()
	}
	if cfg != nil && cfg.VaultPath != "" {
		return cfg.VaultPath
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "kb-vault")
}

func Save() error {
	if cfgViper == nil {
		cfgViper = viper.New()
	}
	
	// Ensure config is loaded
	if cfg == nil {
		Load()
	}
	
	vaultPath := GetVaultPath()
	configPath := filepath.Join(vaultPath, "config.yaml")
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	
	cfgViper.SetConfigFile(configPath)
	return cfgViper.WriteConfig()
}

func SetVaultPath(path string) {
	if cfg == nil {
		// Initialize with defaults
		cfg = &Config{
			VaultPath: path,
			Git:       GitConfig{AutoCommit: true},
			X:         XConfig{Provider: "xtracticle"},
			Index:     IndexConfig{AutoRebuild: true},
			LLM:       LLMConfig{Enabled: false},
		}
		return
	}
	cfg.VaultPath = path
}

func SetGitAutoCommit(enabled bool) {
	if cfg == nil {
		Load()
	}
	cfg.Git.AutoCommit = enabled
}