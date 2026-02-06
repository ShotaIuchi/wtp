// Package config defines the configuration schema and helpers for wtp.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Config represents the wtp configuration
type Config struct {
	Version  string   `yaml:"version"`
	Defaults Defaults `yaml:"defaults,omitempty"`
	Hooks    Hooks    `yaml:"hooks,omitempty"`
}

// Defaults represents default configuration values
type Defaults struct {
	BaseDir string `yaml:"base_dir,omitempty"`
}

// Hooks represents the post-create hooks configuration
type Hooks struct {
	PostCreate []Hook `yaml:"post_create,omitempty"`
}

// Hook represents a single hook configuration
type Hook struct {
	Type    string            `yaml:"type"` // "copy", "command", or "symlink"
	From    string            `yaml:"from,omitempty"`
	To      string            `yaml:"to,omitempty"`
	Command string            `yaml:"command,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	WorkDir string            `yaml:"work_dir,omitempty"`
}

const (
	// ConfigFileName is the default filename for the wtp configuration.
	ConfigFileName = ".wtp.yml"
	// CurrentVersion represents the current configuration version written to disk.
	CurrentVersion = "1.0"
	// DefaultBaseDir is the default directory for new worktrees relative to a repository.
	DefaultBaseDir = "../worktrees"
	// HookTypeCopy identifies a hook that copies files.
	HookTypeCopy = "copy"
	// HookTypeCommand identifies a hook that executes a command.
	HookTypeCommand = "command"
	// HookTypeSymlink identifies a hook that creates symlinks.
	HookTypeSymlink       = "symlink"
	configFilePermissions = 0o600
)

// userHomeDir is a package-level variable for testability.
var userHomeDir = os.UserHomeDir

// loadConfigFromFile reads and unmarshals a config file.
// Returns nil, nil if the file does not exist.
func loadConfigFromFile(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	// #nosec G304 -- path is derived from validated locations
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// MergeConfig merges override into base and returns the result.
// Scalar fields (Version, BaseDir) use override when non-empty.
// Hooks.PostCreate is concatenated: base hooks first, then override hooks.
func MergeConfig(base, override *Config) *Config {
	result := *base

	if override.Version != "" {
		result.Version = override.Version
	}

	if override.Defaults.BaseDir != "" {
		result.Defaults.BaseDir = override.Defaults.BaseDir
	}

	if len(override.Hooks.PostCreate) > 0 {
		merged := make([]Hook, 0, len(base.Hooks.PostCreate)+len(override.Hooks.PostCreate))
		merged = append(merged, base.Hooks.PostCreate...)
		merged = append(merged, override.Hooks.PostCreate...)
		result.Hooks.PostCreate = merged
	}

	return &result
}

// LoadConfig loads configuration from ~/.wtp.yml (global) and <repoRoot>/.wtp.yml (repo),
// merging them with repo config taking precedence for scalar fields.
func LoadConfig(repoRoot string) (*Config, error) {
	cleanedRoot := filepath.Clean(repoRoot)
	if !filepath.IsAbs(cleanedRoot) {
		absRoot, err := filepath.Abs(cleanedRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve repository root: %w", err)
		}
		cleanedRoot = absRoot
	}

	// Load global config from ~/.wtp.yml
	var globalCfg *Config
	if home, err := userHomeDir(); err == nil {
		globalPath := filepath.Join(home, ConfigFileName)
		globalCfg, err = loadConfigFromFile(globalPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load global config: %w", err)
		}
	}

	// Load repo config from <repoRoot>/.wtp.yml
	repoPath := filepath.Join(cleanedRoot, ConfigFileName)
	repoCfg, err := loadConfigFromFile(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load repo config: %w", err)
	}

	// Start with defaults, layer global, then repo
	result := &Config{}
	if globalCfg != nil {
		result = MergeConfig(result, globalCfg)
	}
	if repoCfg != nil {
		result = MergeConfig(result, repoCfg)
	}

	// Apply defaults, then validate configuration.
	result.ApplyDefaults()
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return result, nil
}

// SaveConfig saves configuration to .git-worktree-plus.yml in the repository root
func SaveConfig(repoRoot string, config *Config) error {
	config.ApplyDefaults()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	configPath := filepath.Join(repoRoot, ConfigFileName)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, configFilePermissions); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ApplyDefaults applies default values to the configuration in-place.
func (c *Config) ApplyDefaults() {
	if c.Version == "" {
		c.Version = CurrentVersion
	}

	if c.Defaults.BaseDir == "" {
		c.Defaults.BaseDir = DefaultBaseDir
	}

	for i := range c.Hooks.PostCreate {
		c.Hooks.PostCreate[i].ApplyDefaults()
	}
}

// Validate validates the configuration without mutating it.
func (c *Config) Validate() error {
	for i := range c.Hooks.PostCreate {
		if err := c.Hooks.PostCreate[i].Validate(); err != nil {
			return fmt.Errorf("invalid hook %d: %w", i+1, err)
		}
	}

	return nil
}

// ApplyDefaults applies default values to a single hook in-place.
func (h *Hook) ApplyDefaults() {
	if h.Type != HookTypeCopy {
		return
	}
	if h.To != "" || h.From == "" {
		return
	}
	// Only default to=from for relative paths. Absolute paths must be explicit.
	if filepath.IsAbs(h.From) {
		return
	}
	h.To = h.From
}

// Validate validates a single hook configuration without mutating it.
func (h *Hook) Validate() error {
	switch h.Type {
	case HookTypeCopy:
		if h.From == "" {
			return fmt.Errorf("copy hook requires 'from' field")
		}
		if h.To == "" && filepath.IsAbs(h.From) {
			return fmt.Errorf("copy hook with absolute 'from' requires 'to' field")
		}
		if h.Command != "" {
			return fmt.Errorf("copy hook should not have 'command' field")
		}
	case HookTypeCommand:
		if h.Command == "" {
			return fmt.Errorf("command hook requires 'command' field")
		}
		if h.From != "" || h.To != "" {
			return fmt.Errorf("command hook should not have 'from' or 'to' fields")
		}
	case HookTypeSymlink:
		if h.From == "" || h.To == "" {
			return fmt.Errorf("symlink hook requires both 'from' and 'to' fields")
		}
		if h.Command != "" {
			return fmt.Errorf("symlink hook should not have 'command' field")
		}
	default:
		return fmt.Errorf("invalid hook type '%s', must be 'copy', 'command', or 'symlink'", h.Type)
	}

	return nil
}

// HasHooks returns true if the configuration has any post-create hooks
func (c *Config) HasHooks() bool {
	return len(c.Hooks.PostCreate) > 0
}

// slugify converts a branch name to a slug (replaces / with -)
func slugify(s string) string {
	return strings.ReplaceAll(s, "/", "-")
}

// ExpandVariables expands placeholder variables in the given string.
// Supported variables:
//   - ${DIRNAME} - Directory name (basename) of the repository root
//   - ${PATHNAME} - Absolute path of the repository root
//   - ${BRANCH} - Target branch name (alias: ${TARGET_BRANCH})
//   - ${BRANCH_SLUG} - Slugified branch name (alias: ${TARGET_SLUG})
func ExpandVariables(s, repoRoot, branchName string) string {
	// Get absolute path of repoRoot
	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		absRepoRoot = repoRoot
	}

	// Get directory name (basename)
	dirName := filepath.Base(absRepoRoot)

	// Create slug from branch name
	branchSlug := slugify(branchName)

	// Replace variables
	result := s
	result = strings.ReplaceAll(result, "${DIRNAME}", dirName)
	result = strings.ReplaceAll(result, "${PATHNAME}", absRepoRoot)
	result = strings.ReplaceAll(result, "${BRANCH}", branchName)
	result = strings.ReplaceAll(result, "${TARGET_BRANCH}", branchName)
	result = strings.ReplaceAll(result, "${BRANCH_SLUG}", branchSlug)
	result = strings.ReplaceAll(result, "${TARGET_SLUG}", branchSlug)

	return result
}

// ResolveWorktreePath resolves the full path for a worktree given a name
func (c *Config) ResolveWorktreePath(repoRoot, worktreeName string) string {
	baseDir := c.Defaults.BaseDir

	// Expand variables in baseDir
	baseDir = ExpandVariables(baseDir, repoRoot, worktreeName)

	if !filepath.IsAbs(baseDir) {
		baseDir = filepath.Join(repoRoot, baseDir)
	}
	return filepath.Join(baseDir, worktreeName)
}
