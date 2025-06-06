package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Load configuration
func loadConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "config-manager")
	configFile := filepath.Join(configDir, "config.json")
	
	// Check if this is first run (no config file exists)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Run setup wizard
		config, err := runSetupWizard()
		if err != nil {
			fmt.Printf("Setup wizard failed: %v\n", err)
			fmt.Println("Creating minimal configuration...")
			
			// Fallback to minimal config
			config = &Config{
				ConfigDir:    configDir,
				DotfilesDir:  filepath.Join(configDir, "dotfiles"),
				Variables:    make(map[string]string),
				Categories:   []string{"shell", "editor", "git", "terminal", "misc", "custom"},
				TemplateExts: []string{".tmpl", ".template", ".tpl"},
				Editor:       "vim",
				Shell:        "bash",
				Files:        []ConfigFile{},
			}
			
			os.MkdirAll(configDir, 0755)
			saveConfig(config)
			createDefaultTemplates(config)
		}
		return config
	}
	
	// Load existing config
	config := &Config{}
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			fmt.Printf("Error parsing config file: %v\n", err)
		}
	}
	
	// Ensure default values for new fields
	if len(config.Categories) == 0 {
		config.Categories = []string{"shell", "editor", "git", "terminal", "misc", "custom"}
	}
	if len(config.TemplateExts) == 0 {
		config.TemplateExts = []string{".tmpl", ".template", ".tpl"}
	}
	if config.Variables == nil {
		config.Variables = make(map[string]string)
	}
	if config.Files == nil {
		config.Files = []ConfigFile{}
	}
	
	return config
}

// Save configuration
func saveConfig(config *Config) {
	configFile := filepath.Join(config.ConfigDir, "config.json")
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configFile, data, 0644)
}

// Update file statuses
func updateFileStatuses(config *Config) {
	// First, remove any duplicate entries that might have been added
	uniqueFiles := make([]ConfigFile, 0, len(config.Files))
	seen := make(map[string]bool)
	
	for _, file := range config.Files {
		// Use target path as the unique identifier
		if !seen[file.Target] {
			seen[file.Target] = true
			uniqueFiles = append(uniqueFiles, file)
		}
	}
	
	// Update the config with deduplicated files
	config.Files = uniqueFiles
	
	// Now update statuses for the unique files
	for i := range config.Files {
		file := &config.Files[i]
		
		// Reset status flags
		file.IsLinked = false
		file.HasConflict = false
		
		// Check if target exists and is a symlink to our source
		if info, err := os.Lstat(file.Target); err == nil {
			if info.Mode()&fs.ModeSymlink != 0 {
				if link, err := os.Readlink(file.Target); err == nil {
					expectedSource := filepath.Join(config.DotfilesDir, file.Source)
					file.IsLinked = link == expectedSource
				}
			} else {
				// File exists but is not a symlink
				file.HasConflict = true
			}
		}
	}
}

// Check if a dotfile is a system file that shouldn't be managed
func isSystemFile(filename string) bool {
	systemFiles := []string{
		".DS_Store", ".localized", ".CFUserTextEncoding",
		".bash_history", ".zsh_history", ".lesshst", ".viminfo",
		".sudo_as_admin_successful", ".Trash", ".cache",
	}
	
	for _, sysFile := range systemFiles {
		if filename == sysFile {
			return true
		}
	}
	
	// Skip log files, temp files, etc.
	return strings.HasSuffix(filename, ".log") || 
		   strings.HasSuffix(filename, ".tmp") ||
		   strings.HasSuffix(filename, ".lock")
}

// Categorize dotfile based on filename
func categorizeDotfile(filename string, categories []string) string {
	// Simple categorization rules
	switch {
	case strings.Contains(filename, "zsh") || strings.Contains(filename, "bash") || 
		 strings.Contains(filename, "fish") || filename == ".profile":
		return "shell"
	case strings.Contains(filename, "git"):
		return "git"
	case strings.Contains(filename, "vim") || strings.Contains(filename, "nvim") || 
		 strings.Contains(filename, "emacs"):
		return "editor"
	case strings.Contains(filename, "tmux") || strings.Contains(filename, "screen"):
		return "terminal"
	default:
		return "misc"
	}
}
