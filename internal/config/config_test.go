package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeConfig(t *testing.T) {
	t.Run("override scalar fields", func(t *testing.T) {
		base := &Config{
			Version:  "1.0",
			Defaults: Defaults{BaseDir: "../base-dir"},
		}
		override := &Config{
			Version:  "2.0",
			Defaults: Defaults{BaseDir: "../override-dir"},
		}
		result := MergeConfig(base, override)
		if result.Version != "2.0" {
			t.Errorf("Expected version 2.0, got %s", result.Version)
		}
		if result.Defaults.BaseDir != "../override-dir" {
			t.Errorf("Expected base_dir ../override-dir, got %s", result.Defaults.BaseDir)
		}
	})

	t.Run("empty override keeps base", func(t *testing.T) {
		base := &Config{
			Version:  "1.0",
			Defaults: Defaults{BaseDir: "../base-dir"},
		}
		override := &Config{}
		result := MergeConfig(base, override)
		if result.Version != "1.0" {
			t.Errorf("Expected version 1.0, got %s", result.Version)
		}
		if result.Defaults.BaseDir != "../base-dir" {
			t.Errorf("Expected base_dir ../base-dir, got %s", result.Defaults.BaseDir)
		}
	})

	t.Run("hooks concatenated", func(t *testing.T) {
		base := &Config{
			Hooks: Hooks{
				PostCreate: []Hook{
					{Type: HookTypeCommand, Command: "echo A"},
					{Type: HookTypeCommand, Command: "echo B"},
				},
			},
		}
		override := &Config{
			Hooks: Hooks{
				PostCreate: []Hook{
					{Type: HookTypeCommand, Command: "echo C"},
				},
			},
		}
		result := MergeConfig(base, override)
		if len(result.Hooks.PostCreate) != 3 {
			t.Fatalf("Expected 3 hooks, got %d", len(result.Hooks.PostCreate))
		}
		if result.Hooks.PostCreate[0].Command != "echo A" {
			t.Errorf("Expected first hook 'echo A', got %s", result.Hooks.PostCreate[0].Command)
		}
		if result.Hooks.PostCreate[1].Command != "echo B" {
			t.Errorf("Expected second hook 'echo B', got %s", result.Hooks.PostCreate[1].Command)
		}
		if result.Hooks.PostCreate[2].Command != "echo C" {
			t.Errorf("Expected third hook 'echo C', got %s", result.Hooks.PostCreate[2].Command)
		}
	})

	t.Run("override without hooks keeps base hooks", func(t *testing.T) {
		base := &Config{
			Hooks: Hooks{
				PostCreate: []Hook{
					{Type: HookTypeCommand, Command: "echo A"},
				},
			},
		}
		override := &Config{}
		result := MergeConfig(base, override)
		if len(result.Hooks.PostCreate) != 1 {
			t.Fatalf("Expected 1 hook, got %d", len(result.Hooks.PostCreate))
		}
		if result.Hooks.PostCreate[0].Command != "echo A" {
			t.Errorf("Expected hook 'echo A', got %s", result.Hooks.PostCreate[0].Command)
		}
	})
}

func TestLoadConfig_GlobalOnly(t *testing.T) {
	globalDir := t.TempDir()
	repoDir := t.TempDir() // no config file here

	globalConfig := `version: "1.0"
defaults:
  base_dir: "../global-wt"
hooks:
  post_create:
    - type: command
      command: "echo global"
`
	if err := os.WriteFile(filepath.Join(globalDir, ConfigFileName), []byte(globalConfig), 0o644); err != nil {
		t.Fatalf("Failed to write global config: %v", err)
	}

	original := userHomeDir
	userHomeDir = func() (string, error) { return globalDir, nil }
	t.Cleanup(func() { userHomeDir = original })

	config, err := LoadConfig(repoDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Defaults.BaseDir != "../global-wt" {
		t.Errorf("Expected base_dir '../global-wt', got %s", config.Defaults.BaseDir)
	}
	if len(config.Hooks.PostCreate) != 1 {
		t.Fatalf("Expected 1 hook, got %d", len(config.Hooks.PostCreate))
	}
	if config.Hooks.PostCreate[0].Command != "echo global" {
		t.Errorf("Expected hook command 'echo global', got %s", config.Hooks.PostCreate[0].Command)
	}
}

func TestLoadConfig_RepoOverridesGlobalBaseDir(t *testing.T) {
	globalDir := t.TempDir()
	repoDir := t.TempDir()

	globalConfig := `version: "1.0"
defaults:
  base_dir: "../global-wt"
`
	repoConfig := `version: "1.0"
defaults:
  base_dir: "../repo-wt"
`
	if err := os.WriteFile(filepath.Join(globalDir, ConfigFileName), []byte(globalConfig), 0o644); err != nil {
		t.Fatalf("Failed to write global config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ConfigFileName), []byte(repoConfig), 0o644); err != nil {
		t.Fatalf("Failed to write repo config: %v", err)
	}

	original := userHomeDir
	userHomeDir = func() (string, error) { return globalDir, nil }
	t.Cleanup(func() { userHomeDir = original })

	config, err := LoadConfig(repoDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Defaults.BaseDir != "../repo-wt" {
		t.Errorf("Expected base_dir '../repo-wt', got %s", config.Defaults.BaseDir)
	}
}

func TestLoadConfig_HooksConcatenated(t *testing.T) {
	globalDir := t.TempDir()
	repoDir := t.TempDir()

	globalConfig := `hooks:
  post_create:
    - type: command
      command: "echo A"
    - type: command
      command: "echo B"
`
	repoConfig := `hooks:
  post_create:
    - type: command
      command: "echo C"
`
	if err := os.WriteFile(filepath.Join(globalDir, ConfigFileName), []byte(globalConfig), 0o644); err != nil {
		t.Fatalf("Failed to write global config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ConfigFileName), []byte(repoConfig), 0o644); err != nil {
		t.Fatalf("Failed to write repo config: %v", err)
	}

	original := userHomeDir
	userHomeDir = func() (string, error) { return globalDir, nil }
	t.Cleanup(func() { userHomeDir = original })

	config, err := LoadConfig(repoDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Hooks.PostCreate) != 3 {
		t.Fatalf("Expected 3 hooks, got %d", len(config.Hooks.PostCreate))
	}
	if config.Hooks.PostCreate[0].Command != "echo A" {
		t.Errorf("Expected first hook 'echo A', got %s", config.Hooks.PostCreate[0].Command)
	}
	if config.Hooks.PostCreate[1].Command != "echo B" {
		t.Errorf("Expected second hook 'echo B', got %s", config.Hooks.PostCreate[1].Command)
	}
	if config.Hooks.PostCreate[2].Command != "echo C" {
		t.Errorf("Expected third hook 'echo C', got %s", config.Hooks.PostCreate[2].Command)
	}
}

func TestLoadConfig_GlobalHooksOnlyWhenRepoHasNone(t *testing.T) {
	globalDir := t.TempDir()
	repoDir := t.TempDir()

	globalConfig := `hooks:
  post_create:
    - type: command
      command: "echo global"
`
	repoConfig := `defaults:
  base_dir: "../repo-wt"
`
	if err := os.WriteFile(filepath.Join(globalDir, ConfigFileName), []byte(globalConfig), 0o644); err != nil {
		t.Fatalf("Failed to write global config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ConfigFileName), []byte(repoConfig), 0o644); err != nil {
		t.Fatalf("Failed to write repo config: %v", err)
	}

	original := userHomeDir
	userHomeDir = func() (string, error) { return globalDir, nil }
	t.Cleanup(func() { userHomeDir = original })

	config, err := LoadConfig(repoDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Defaults.BaseDir != "../repo-wt" {
		t.Errorf("Expected base_dir '../repo-wt', got %s", config.Defaults.BaseDir)
	}
	if len(config.Hooks.PostCreate) != 1 {
		t.Fatalf("Expected 1 hook, got %d", len(config.Hooks.PostCreate))
	}
	if config.Hooks.PostCreate[0].Command != "echo global" {
		t.Errorf("Expected hook 'echo global', got %s", config.Hooks.PostCreate[0].Command)
	}
}

func TestLoadConfig_NeitherExists(t *testing.T) {
	globalDir := t.TempDir() // no config
	repoDir := t.TempDir()   // no config

	original := userHomeDir
	userHomeDir = func() (string, error) { return globalDir, nil }
	t.Cleanup(func() { userHomeDir = original })

	config, err := LoadConfig(repoDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Version != CurrentVersion {
		t.Errorf("Expected version %s, got %s", CurrentVersion, config.Version)
	}
	if config.Defaults.BaseDir != DefaultBaseDir {
		t.Errorf("Expected base_dir %s, got %s", DefaultBaseDir, config.Defaults.BaseDir)
	}
	if len(config.Hooks.PostCreate) != 0 {
		t.Errorf("Expected no hooks, got %d", len(config.Hooks.PostCreate))
	}
}

func stubHomeDir(t *testing.T) {
	t.Helper()
	emptyDir := t.TempDir()
	original := userHomeDir
	userHomeDir = func() (string, error) { return emptyDir, nil }
	t.Cleanup(func() { userHomeDir = original })
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	stubHomeDir(t)
	tempDir := t.TempDir()

	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Version != CurrentVersion {
		t.Errorf("Expected version %s, got %s", CurrentVersion, config.Version)
	}

	if config.Defaults.BaseDir != "../worktrees" {
		t.Errorf("Expected default base_dir '../worktrees', got %s", config.Defaults.BaseDir)
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	stubHomeDir(t)
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ConfigFileName)

	configContent := `version: "1.0"
defaults:
  base_dir: "../my-worktrees"
hooks:
  post_create:
    - type: copy
      from: ".env.example"
      to: ".env"
    - type: command
      command: "echo test"
    - type: symlink
      from: ".bin"
      to: ".bin"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", config.Version)
	}

	if config.Defaults.BaseDir != "../my-worktrees" {
		t.Errorf("Expected base_dir '../my-worktrees', got %s", config.Defaults.BaseDir)
	}

	if len(config.Hooks.PostCreate) != 3 {
		t.Errorf("Expected 3 hooks, got %d", len(config.Hooks.PostCreate))
	}

	if config.Hooks.PostCreate[0].Type != HookTypeCopy {
		t.Errorf("Expected first hook type 'copy', got %s", config.Hooks.PostCreate[0].Type)
	}

	if config.Hooks.PostCreate[1].Type != HookTypeCommand {
		t.Errorf("Expected second hook type 'command', got %s", config.Hooks.PostCreate[1].Type)
	}

	if config.Hooks.PostCreate[2].Type != HookTypeSymlink {
		t.Errorf("Expected third hook type 'symlink', got %s", config.Hooks.PostCreate[2].Type)
	}
}

func TestLoadConfig_CopyHookDefaultsToFrom(t *testing.T) {
	stubHomeDir(t)
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ConfigFileName)

	configContent := `version: "1.0"
hooks:
  post_create:
    - type: copy
      from: ".env"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Hooks.PostCreate) != 1 {
		t.Fatalf("Expected 1 hook, got %d", len(config.Hooks.PostCreate))
	}

	if got := config.Hooks.PostCreate[0].To; got != ".env" {
		t.Errorf("Expected hook.To to default to %q, got %q", ".env", got)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	stubHomeDir(t)
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ConfigFileName)

	invalidContent := `version: "1.0"
hooks:
  post_create:
    - type: copy
      from: ".env.example"
      # Invalid YAML syntax
      to: ".env"
    invalid_structure
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = LoadConfig(tempDir)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestSaveConfig(t *testing.T) {
	stubHomeDir(t)
	tempDir := t.TempDir()

	config := &Config{
		Version: "1.0",
		Defaults: Defaults{
			BaseDir: "../test-worktrees",
		},
		Hooks: Hooks{
			PostCreate: []Hook{
				{
					Type: HookTypeCopy,
					From: ".env.example",
					To:   ".env",
				},
			},
		},
	}

	err := SaveConfig(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tempDir, ConfigFileName)
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		t.Error("Config file was not created")
	}

	// Load it back and verify
	loadedConfig, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Version != config.Version {
		t.Errorf("Expected version %s, got %s", config.Version, loadedConfig.Version)
	}

	if loadedConfig.Defaults.BaseDir != config.Defaults.BaseDir {
		t.Errorf("Expected base_dir %s, got %s", config.Defaults.BaseDir, loadedConfig.Defaults.BaseDir)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				Version: "1.0",
				Defaults: Defaults{
					BaseDir: "../worktrees",
				},
				Hooks: Hooks{
					PostCreate: []Hook{
						{
							Type: HookTypeCopy,
							From: ".env.example",
							To:   ".env",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty version gets default",
			config: &Config{
				Defaults: Defaults{
					BaseDir: "../worktrees",
				},
			},
			expectError: false,
		},
		{
			name: "empty base_dir gets default",
			config: &Config{
				Version: "1.0",
			},
			expectError: false,
		},
		{
			name: "invalid copy hook - missing from",
			config: &Config{
				Version: "1.0",
				Hooks: Hooks{
					PostCreate: []Hook{
						{
							Type: HookTypeCopy,
							To:   ".env",
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid command hook - missing command",
			config: &Config{
				Version: "1.0",
				Hooks: Hooks{
					PostCreate: []Hook{
						{
							Type: HookTypeCommand,
							// Missing Command field - should cause validation error
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.ApplyDefaults()
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check defaults are set
			if !tt.expectError {
				if tt.config.Version == "" {
					t.Error("Version should be set to default")
				}
				if tt.config.Defaults.BaseDir == "" {
					t.Error("BaseDir should be set to default")
				}
			}
		})
	}
}

func TestHookValidate(t *testing.T) {
	tests := []struct {
		name        string
		hook        Hook
		expectError bool
	}{
		{
			name: "valid copy hook",
			hook: Hook{
				Type: HookTypeCopy,
				From: ".env.example",
				To:   ".env",
			},
			expectError: false,
		},
		{
			name: "valid command hook",
			hook: Hook{
				Type:    HookTypeCommand,
				Command: "echo test",
			},
			expectError: false,
		},
		{
			name: "valid symlink hook",
			hook: Hook{
				Type: HookTypeSymlink,
				From: ".bin",
				To:   ".bin",
			},
			expectError: false,
		},
		{
			name: "copy hook missing from",
			hook: Hook{
				Type: HookTypeCopy,
				To:   ".env",
			},
			expectError: true,
		},
		{
			name: "copy hook missing to",
			hook: Hook{
				Type: HookTypeCopy,
				From: ".env.example",
			},
			expectError: false,
		},
		{
			name: "copy hook missing to with absolute from",
			hook: Hook{
				Type: HookTypeCopy,
				From: filepath.Join(string(os.PathSeparator), "tmp", "source.txt"),
			},
			expectError: true,
		},
		{
			name: "copy hook with command field",
			hook: Hook{
				Type:    HookTypeCopy,
				From:    ".env.example",
				To:      ".env",
				Command: "echo", // Should not have command
			},
			expectError: true,
		},
		{
			name: "command hook missing command",
			hook: Hook{
				Type: HookTypeCommand,
			},
			expectError: true,
		},
		{
			name: "symlink hook missing from",
			hook: Hook{
				Type: HookTypeSymlink,
				To:   ".bin",
			},
			expectError: true,
		},
		{
			name: "symlink hook missing to",
			hook: Hook{
				Type: HookTypeSymlink,
				From: ".bin",
			},
			expectError: true,
		},
		{
			name: "symlink hook with command field",
			hook: Hook{
				Type:    HookTypeSymlink,
				From:    ".bin",
				To:      ".bin",
				Command: "echo", // Should not have command
			},
			expectError: true,
		},
		{
			name: "command hook with from/to fields",
			hook: Hook{
				Type:    HookTypeCommand,
				Command: "echo",
				From:    ".env.example", // Should not have from/to
				To:      ".env",
			},
			expectError: true,
		},
		{
			name: "invalid hook type",
			hook: Hook{
				Type: "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.hook.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestHookValidate_DoesNotMutateTo(t *testing.T) {
	hook := Hook{
		Type: HookTypeCopy,
		From: ".env",
	}

	if err := hook.Validate(); err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	if hook.To != "" {
		t.Errorf("Expected hook.To to remain empty, got %q", hook.To)
	}
}

func TestHookApplyDefaults_CopyToDefaultsToFrom(t *testing.T) {
	hook := Hook{
		Type: HookTypeCopy,
		From: ".env",
	}

	hook.ApplyDefaults()

	if hook.To != hook.From {
		t.Errorf("Expected hook.To to default to %q, got %q", hook.From, hook.To)
	}

	if err := hook.Validate(); err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}
}

func TestConfigApplyDefaults_CopyToDefaultsToFrom(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Hooks: Hooks{
			PostCreate: []Hook{
				{
					Type: HookTypeCopy,
					From: ".env",
				},
			},
		},
	}

	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	if got := config.Hooks.PostCreate[0].To; got != ".env" {
		t.Errorf("Expected hook.To to default to %q, got %q", ".env", got)
	}
}

func TestConfigValidate_CopyAbsoluteFromRequiresTo(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Hooks: Hooks{
			PostCreate: []Hook{
				{
					Type: HookTypeCopy,
					From: filepath.Join(string(os.PathSeparator), "tmp", "source.txt"),
				},
			},
		},
	}

	config.ApplyDefaults()

	if err := config.Validate(); err == nil {
		t.Fatalf("Expected error but got nil")
	}
}

func TestResolveWorktreePath(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		repoRoot     string
		worktreeName string
		expected     string
	}{
		{
			name: "relative base_dir",
			config: &Config{
				Defaults: Defaults{
					BaseDir: "../worktrees",
				},
			},
			repoRoot:     "/home/user/project",
			worktreeName: "feature/auth",
			expected:     "/home/user/worktrees/feature/auth",
		},
		{
			name: "absolute base_dir",
			config: &Config{
				Defaults: Defaults{
					BaseDir: "/tmp/worktrees",
				},
			},
			repoRoot:     "/home/user/project",
			worktreeName: "feature/auth",
			expected:     "/tmp/worktrees/feature/auth",
		},
		{
			name: "simple worktree name",
			config: &Config{
				Defaults: Defaults{
					BaseDir: "../worktrees",
				},
			},
			repoRoot:     "/home/user/project",
			worktreeName: "main",
			expected:     "/home/user/worktrees/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ResolveWorktreePath(tt.repoRoot, tt.worktreeName)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExpandVariables(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		repoRoot   string
		branchName string
		expected   string
	}{
		{
			name:       "expand DIRNAME",
			input:      "../${DIRNAME}-worktrees",
			repoRoot:   "/home/user/myproject",
			branchName: "feature/auth",
			expected:   "../myproject-worktrees",
		},
		{
			name:       "expand PATHNAME",
			input:      "${PATHNAME}/worktrees",
			repoRoot:   "/home/user/myproject",
			branchName: "feature/auth",
			expected:   "/home/user/myproject/worktrees",
		},
		{
			name:       "expand BRANCH",
			input:      "../worktrees-${BRANCH}",
			repoRoot:   "/home/user/myproject",
			branchName: "feature/auth",
			expected:   "../worktrees-feature/auth",
		},
		{
			name:       "expand TARGET_BRANCH (alias)",
			input:      "../worktrees-${TARGET_BRANCH}",
			repoRoot:   "/home/user/myproject",
			branchName: "feature/auth",
			expected:   "../worktrees-feature/auth",
		},
		{
			name:       "expand BRANCH_SLUG",
			input:      "../${DIRNAME}-${BRANCH_SLUG}",
			repoRoot:   "/home/user/myproject",
			branchName: "feature/auth",
			expected:   "../myproject-feature-auth",
		},
		{
			name:       "expand TARGET_SLUG (alias)",
			input:      "../${DIRNAME}-${TARGET_SLUG}",
			repoRoot:   "/home/user/myproject",
			branchName: "feature/auth",
			expected:   "../myproject-feature-auth",
		},
		{
			name:       "multiple variables",
			input:      "../${DIRNAME}___${BRANCH_SLUG}",
			repoRoot:   "/home/user/myproject",
			branchName: "feature/auth",
			expected:   "../myproject___feature-auth",
		},
		{
			name:       "no variables",
			input:      "../worktrees",
			repoRoot:   "/home/user/myproject",
			branchName: "feature/auth",
			expected:   "../worktrees",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandVariables(tt.input, tt.repoRoot, tt.branchName)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feature/auth", "feature-auth"},
		{"hotfix/bug-123", "hotfix-bug-123"},
		{"main", "main"},
		{"feature/nested/deep", "feature-nested-deep"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveWorktreePath_WithVariables(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		repoRoot     string
		worktreeName string
		expected     string
	}{
		{
			name: "base_dir with DIRNAME variable",
			config: &Config{
				Defaults: Defaults{
					BaseDir: "../${DIRNAME}-worktrees",
				},
			},
			repoRoot:     "/home/user/myproject",
			worktreeName: "feature/auth",
			expected:     "/home/user/myproject-worktrees/feature/auth",
		},
		{
			name: "base_dir with BRANCH_SLUG variable",
			config: &Config{
				Defaults: Defaults{
					BaseDir: "../${DIRNAME}___${BRANCH_SLUG}",
				},
			},
			repoRoot:     "/home/user/myproject",
			worktreeName: "feature/auth",
			expected:     "/home/user/myproject___feature-auth/feature/auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ResolveWorktreePath(tt.repoRoot, tt.worktreeName)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHasHooks(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "config with hooks",
			config: &Config{
				Hooks: Hooks{
					PostCreate: []Hook{
						{Type: HookTypeCopy, From: "a", To: "b"},
					},
				},
			},
			expected: true,
		},
		{
			name: "config without hooks",
			config: &Config{
				Hooks: Hooks{},
			},
			expected: false,
		},
		{
			name: "config with empty hooks slice",
			config: &Config{
				Hooks: Hooks{
					PostCreate: []Hook{},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HasHooks()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
