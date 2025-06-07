package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Enhanced configuration loading with validation and error handling
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
			config = createMinimalConfig(configDir)
			
			// Ensure directories exist
			if err := os.MkdirAll(configDir, 0755); err != nil {
				fmt.Printf("Failed to create config directory: %v\n", err)
				return config // Return config anyway, let user handle errors
			}
			
			// Try to save config
			if err := saveConfigSafe(config); err != nil {
				fmt.Printf("Failed to save minimal config: %v\n", err)
			}
		}
		return config
	}
	
	// Load existing config
	config, err := loadConfigFile(configFile, configDir)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Println("Creating minimal configuration...")
		return createMinimalConfig(configDir)
	}
	
	// Validate loaded config
	if errors := config.Validate(); len(errors) > 0 {
		fmt.Printf("Configuration validation warnings:\n")
		for _, err := range errors {
			fmt.Printf("  - %v\n", err)
		}
		fmt.Println("Continuing with current configuration...")
	}
	
	return config
}

// createMinimalConfig creates a basic working configuration
func createMinimalConfig(configDir string) *Config {
	return &Config{
		ConfigDir:    configDir,
		DotfilesDir:  filepath.Join(configDir, "dotfiles"),
		Variables:    make(map[string]string),
		Categories:   []string{"shell", "editor", "git", "terminal", "misc", "custom"},
		TemplateExts: []string{".tmpl", ".template", ".tpl"},
		Editor:       "vim",
		Shell:        "bash",
		Files:        []ConfigFile{},
	}
}

// loadConfigFile loads and parses the configuration file
func loadConfigFile(configFile, configDir string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, NewConfigError("read config file", configFile, err)
	}
	
	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, NewConfigError("parse config file", configFile, err)
	}
	
	// Ensure config directory is set (for backwards compatibility)
	if config.ConfigDir == "" {
		config.ConfigDir = configDir
	}
	
	// Ensure dotfiles directory is set
	if config.DotfilesDir == "" {
		config.DotfilesDir = filepath.Join(configDir, "dotfiles")
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
	
	return config, nil
}

// Enhanced save configuration with validation and backup
func saveConfig(config *Config) {
	if err := saveConfigSafe(config); err != nil {
		fmt.Printf("Failed to save configuration: %v\n", err)
	}
}

// saveConfigSafe provides safe configuration saving with validation and backup
func saveConfigSafe(config *Config) error {
	// Validate configuration before saving
	if err := config.ValidateBeforeSave(); err != nil {
		return NewConfigError("config validation", config.ConfigDir, err)
	}
	
	configFile := filepath.Join(config.ConfigDir, "config.json")
	
	// Create backup of existing config if it exists
	if _, err := os.Stat(configFile); err == nil {
		backupFile := configFile + ".backup"
		if err := copyFile(configFile, backupFile); err != nil {
			// Log warning but continue
			fmt.Printf("Warning: failed to backup config file: %v\n", err)
		}
	}
	
	// Ensure config directory exists
	if err := os.MkdirAll(config.ConfigDir, 0755); err != nil {
		return NewConfigError("create config directory", config.ConfigDir, err)
	}
	
	// Marshal config to JSON with nice formatting
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return NewConfigError("marshal config", configFile, err)
	}
	
	// Write to temporary file first, then rename (atomic operation)
	tempFile := configFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return NewConfigError("write temp config", tempFile, err)
	}
	
	// Atomically replace the config file
	if err := os.Rename(tempFile, configFile); err != nil {
		// Clean up temp file
		os.Remove(tempFile)
		return NewConfigError("replace config file", configFile, err)
	}
	
	return nil
}

// Enhanced file status updates with better error handling
func updateFileStatuses(config *Config) {
	if config == nil {
		return
	}
	
	// Remove duplicates and update statuses
	config.Files = removeDuplicateFiles(config.Files)
	
	// Update statuses for all files
	for i := range config.Files {
		file := &config.Files[i]
		updateSingleFileStatus(config, file)
	}
}

// removeDuplicateFiles removes duplicate entries based on target path
func removeDuplicateFiles(files []ConfigFile) []ConfigFile {
	seen := make(map[string]bool)
	uniqueFiles := make([]ConfigFile, 0, len(files))
	
	for _, file := range files {
		// Use target path as the unique identifier
		if !seen[file.Target] {
			seen[file.Target] = true
			uniqueFiles = append(uniqueFiles, file)
		}
	}
	
	return uniqueFiles
}

// updateSingleFileStatus updates the status of a single file
func updateSingleFileStatus(config *Config, file *ConfigFile) {
	// Reset status flags
	file.IsLinked = false
	file.HasConflict = false
	
	// Check if target exists and its status
	info, err := os.Lstat(file.Target)
	if os.IsNotExist(err) {
		// File doesn't exist - no conflict, not linked
		return
	}
	if err != nil {
		// Some other error - treat as conflict
		file.HasConflict = true
		return
	}
	
	// Check if it's a symlink
	if info.Mode()&os.ModeSymlink != 0 {
		// It's a symlink - check where it points
		linkTarget, err := os.Readlink(file.Target)
		if err != nil {
			file.HasConflict = true
			return
		}
		
		expectedSource := filepath.Join(config.DotfilesDir, file.Source)
		file.IsLinked = (linkTarget == expectedSource)
		
		// If it's a symlink but points somewhere else, it's a conflict
		if !file.IsLinked {
			file.HasConflict = true
		}
	} else {
		// File exists but is not a symlink - conflict
		file.HasConflict = true
	}
}

// Enhanced file categorization with better heuristics
func categorizeDotfile(filename string, categories []string) string {
	filename = strings.ToLower(filename)
	
	// Shell configuration files
	shellPatterns := []string{
		"zsh", "bash", "fish", "profile", "bashrc", "zshrc", "fishrc",
		"bash_profile", "bash_aliases", "aliases", "functions",
	}
	for _, pattern := range shellPatterns {
		if strings.Contains(filename, pattern) {
			return "shell"
		}
	}
	
	// Git configuration files
	gitPatterns := []string{"git", "gitconfig", "gitignore", "gitmessage"}
	for _, pattern := range gitPatterns {
		if strings.Contains(filename, pattern) {
			return "git"
		}
	}
	
	// Editor configuration files
	editorPatterns := []string{
		"vim", "nvim", "emacs", "vscode", "code", "sublime",
		"vimrc", "init.vim", "init.lua", "emacs.d",
	}
	for _, pattern := range editorPatterns {
		if strings.Contains(filename, pattern) {
			return "editor"
		}
	}
	
	// Terminal/multiplexer configuration files
	terminalPatterns := []string{
		"tmux", "screen", "alacritty", "kitty", "wezterm",
		"terminator", "gnome-terminal", "konsole",
	}
	for _, pattern := range terminalPatterns {
		if strings.Contains(filename, pattern) {
			return "terminal"
		}
	}
	
	// Window manager / desktop environment
	wmPatterns := []string{
		"i3", "sway", "bspwm", "dwm", "awesome", "xmonad",
		"polybar", "waybar", "rofi", "dmenu", "dunst",
		"picom", "compton", "xorg", "wayland",
	}
	for _, pattern := range wmPatterns {
		if strings.Contains(filename, pattern) {
			// Check if WM category exists, otherwise use misc
			for _, cat := range categories {
				if cat == "wm" || cat == "desktop" {
					return cat
				}
			}
		}
	}
	
	// Development tools
	devPatterns := []string{
		"eslint", "prettier", "editorconfig", "docker",
		"makefile", "cmake", "cargo", "npm", "yarn",
	}
	for _, pattern := range devPatterns {
		if strings.Contains(filename, pattern) {
			// Check if dev category exists, otherwise use misc
			for _, cat := range categories {
				if cat == "dev" || cat == "development" {
					return cat
				}
			}
		}
	}
	
	// Default to misc
	return "misc"
}

// Enhanced system file detection
func isSystemFile(filename string) bool {
	// macOS system files
	macOSFiles := []string{
		".DS_Store", ".localized", ".CFUserTextEncoding",
		".Trash", ".DocumentRevisions-V100", ".fseventsd",
		".Spotlight-V100", ".TemporaryItems", ".VolumeIcon.icns",
	}
	
	// General system/cache files
	systemFiles := []string{
		".bash_history", ".zsh_history", ".python_history",
		".lesshst", ".viminfo", ".recently-used",
		".sudo_as_admin_successful", ".cache", ".local/share/recently-used.xbel",
		".mozilla", ".chrome", ".chromium", ".firefox",
	}
	
	// Combine all system files
	allSystemFiles := append(macOSFiles, systemFiles...)
	
	for _, sysFile := range allSystemFiles {
		if filename == sysFile {
			return true
		}
	}
	
	// Skip files with certain extensions
	lowerFilename := strings.ToLower(filename)
	skipExtensions := []string{
		".log", ".tmp", ".temp", ".lock", ".pid", ".sock",
		".swp", ".swo", ".backup", ".bak", ".old",
	}
	
	for _, ext := range skipExtensions {
		if strings.HasSuffix(lowerFilename, ext) {
			return true
		}
	}
	
	// Skip hidden directories that are typically not config
	hiddenDirs := []string{
		".git", ".svn", ".hg", ".bzr",
		".npm", ".node_modules", ".cargo", ".rustup",
		".gem", ".bundle", ".rbenv",
		".pyenv", ".virtualenv", ".conda",
	}
	
	for _, dir := range hiddenDirs {
		if filename == dir {
			return true
		}
	}
	
	return false
}

// addConfigFile safely adds a new config file with validation
func (c *Config) AddConfigFile(file ConfigFile) error {
	// Validate the file before adding
	if file.Name == "" {
		return NewValidationError("name", "", "file name cannot be empty", "")
	}
	
	if file.Target == "" {
		return NewValidationError("target", "", "target path cannot be empty", "")
	}
	
	// Check for duplicates
	for _, existing := range c.Files {
		if existing.Target == file.Target {
			return NewValidationError("target", file.Target, 
				fmt.Sprintf("target already managed by %s", existing.Name), "")
		}
		
		if existing.Name == file.Name && existing.Category == file.Category {
			return NewValidationError("name", file.Name, 
				fmt.Sprintf("file with same name already exists in category %s", file.Category), "")
		}
	}
	
	// Validate category exists
	if file.Category != "" {
		found := false
		for _, cat := range c.Categories {
			if cat == file.Category {
				found = true
				break
			}
		}
		if !found {
			return NewValidationError("category", file.Category, 
				"category not defined in configuration", "")
		}
	}
	
	// Add the file
	c.Files = append(c.Files, file)
	
	// Update file status
	updateSingleFileStatus(c, &c.Files[len(c.Files)-1])
	
	return nil
}

// removeConfigFile safely removes a config file
func (c *Config) RemoveConfigFile(targetPath string) error {
	for i, file := range c.Files {
		if file.Target == targetPath {
			// Remove the file from slice
			c.Files = append(c.Files[:i], c.Files[i+1:]...)
			return nil
		}
	}
	
	return NewConfigError("remove config file", targetPath, 
		fmt.Errorf("file not found in configuration"))
}

// getConfigFileByTarget finds a config file by its target path
func (c *Config) GetConfigFileByTarget(targetPath string) (*ConfigFile, error) {
	for i, file := range c.Files {
		if file.Target == targetPath {
			return &c.Files[i], nil
		}
	}
	
	return nil, NewConfigError("find config file", targetPath, 
		fmt.Errorf("file not found in configuration"))
}

// getConfigFileByName finds config files by name (there might be multiple)
func (c *Config) GetConfigFilesByName(name string) []*ConfigFile {
	var files []*ConfigFile
	for i, file := range c.Files {
		if file.Name == name {
			files = append(files, &c.Files[i])
		}
	}
	return files
}

// ensureDirectoriesExist creates necessary directories
func (c *Config) EnsureDirectoriesExist() error {
	dirs := []string{
		c.ConfigDir,
		c.DotfilesDir,
		filepath.Join(c.ConfigDir, "templates"),
		filepath.Join(c.ConfigDir, "backups"),
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return NewConfigError("create directory", dir, err)
		}
	}
	
	// Create category subdirectories in dotfiles
	for _, category := range c.Categories {
		categoryDir := filepath.Join(c.DotfilesDir, category)
		if err := os.MkdirAll(categoryDir, 0755); err != nil {
			return NewConfigError("create category directory", categoryDir, err)
		}
	}
	
	return nil
}

// getUnlinkedFiles returns files that are not currently linked
func (c *Config) GetUnlinkedFiles() []ConfigFile {
	var unlinked []ConfigFile
	for _, file := range c.Files {
		if !file.IsLinked {
			unlinked = append(unlinked, file)
		}
	}
	return unlinked
}

// getConflictedFiles returns files that have conflicts
func (c *Config) GetConflictedFiles() []ConfigFile {
	var conflicted []ConfigFile
	for _, file := range c.Files {
		if file.HasConflict {
			conflicted = append(conflicted, file)
		}
	}
	return conflicted
}

// getLinkedFiles returns files that are currently linked
func (c *Config) GetLinkedFiles() []ConfigFile {
	var linked []ConfigFile
	for _, file := range c.Files {
		if file.IsLinked {
			linked = append(linked, file)
		}
	}
	return linked
}

// getTemplateFiles returns files that are templates
func (c *Config) GetTemplateFiles() []ConfigFile {
	var templates []ConfigFile
	for _, file := range c.Files {
		if file.Template {
			templates = append(templates, file)
		}
	}
	return templates
}

// getFilesByCategory returns files in a specific category
func (c *Config) GetFilesByCategory(category string) []ConfigFile {
	var files []ConfigFile
	for _, file := range c.Files {
		if file.Category == category {
			files = append(files, file)
		}
	}
	return files
}

// addCategory safely adds a new category
func (c *Config) AddCategory(category string) error {
	if category == "" {
		return NewValidationError("category", "", "category name cannot be empty", "")
	}
	
	// Check for duplicates
	for _, existing := range c.Categories {
		if existing == category {
			return NewValidationError("category", category, "category already exists", "")
		}
	}
	
	c.Categories = append(c.Categories, category)
	
	// Create directory for the new category
	categoryDir := filepath.Join(c.DotfilesDir, category)
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		return NewConfigError("create category directory", categoryDir, err)
	}
	
	return nil
}

// removeCategory safely removes a category (only if no files use it)
func (c *Config) RemoveCategory(category string) error {
	// Check if any files use this category
	for _, file := range c.Files {
		if file.Category == category {
			return NewConfigError("remove category", category, 
				fmt.Errorf("category is used by file %s", file.Name))
		}
	}
	
	// Remove from categories list
	for i, cat := range c.Categories {
		if cat == category {
			c.Categories = append(c.Categories[:i], c.Categories[i+1:]...)
			break
		}
	}
	
	// Optionally remove the directory (commented out for safety)
	// categoryDir := filepath.Join(c.DotfilesDir, category)
	// os.RemoveAll(categoryDir)
	
	return nil
}

// setGlobalVariable sets a global template variable
func (c *Config) SetGlobalVariable(key, value string) {
	if c.Variables == nil {
		c.Variables = make(map[string]string)
	}
	c.Variables[key] = value
}

// getGlobalVariable gets a global template variable
func (c *Config) GetGlobalVariable(key string) (string, bool) {
	if c.Variables == nil {
		return "", false
	}
	value, exists := c.Variables[key]
	return value, exists
}

// removeGlobalVariable removes a global template variable
func (c *Config) RemoveGlobalVariable(key string) {
	if c.Variables != nil {
		delete(c.Variables, key)
	}
}

// getStats returns statistics about the configuration
func (c *Config) GetStats() map[string]int {
	stats := map[string]int{
		"total_files":     len(c.Files),
		"linked_files":    0,
		"unlinked_files":  0,
		"conflicted_files": 0,
		"template_files":  0,
		"categories":      len(c.Categories),
		"global_variables": len(c.Variables),
	}
	
	for _, file := range c.Files {
		if file.IsLinked {
			stats["linked_files"]++
		} else {
			stats["unlinked_files"]++
		}
		
		if file.HasConflict {
			stats["conflicted_files"]++
		}
		
		if file.Template {
			stats["template_files"]++
		}
	}
	
	return stats
}

// exportConfig exports configuration to a portable format
func (c *Config) ExportConfig() ([]byte, error) {
	// Create a clean copy for export (remove runtime fields)
	export := &Config{
		Files:        make([]ConfigFile, len(c.Files)),
		ConfigDir:    "", // Don't export absolute paths
		DotfilesDir:  "", // Don't export absolute paths
		Variables:    c.Variables,
		Categories:   c.Categories,
		TemplateExts: c.TemplateExts,
		Editor:       c.Editor,
		Shell:        c.Shell,
	}
	
	// Copy files without runtime status
	for i, file := range c.Files {
		export.Files[i] = ConfigFile{
			Name:      file.Name,
			Source:    file.Source,
			Target:    file.Target,
			Category:  file.Category,
			Template:  file.Template,
			Variables: file.Variables,
			// Exclude IsLinked and HasConflict (runtime fields)
		}
	}
	
	return json.MarshalIndent(export, "", "  ")
}

// importConfig imports configuration from exported data
func (c *Config) ImportConfig(data []byte, mergeMode bool) error {
	imported := &Config{}
	if err := json.Unmarshal(data, imported); err != nil {
		return NewConfigError("import config", "", fmt.Errorf("invalid JSON: %v", err))
	}
	
	if mergeMode {
		// Merge imported configuration with existing
		return c.mergeConfig(imported)
	} else {
		// Replace current configuration (keeping paths)
		configDir := c.ConfigDir
		dotfilesDir := c.DotfilesDir
		
		*c = *imported
		c.ConfigDir = configDir
		c.DotfilesDir = dotfilesDir
		
		// Update file statuses
		updateFileStatuses(c)
		
		return nil
	}
}

// mergeConfig merges imported configuration with current configuration
func (c *Config) mergeConfig(imported *Config) error {
	var multiErr MultiError
	multiErr.Op = "merge configuration"
	
	// Merge categories
	for _, cat := range imported.Categories {
		found := false
		for _, existing := range c.Categories {
			if existing == cat {
				found = true
				break
			}
		}
		if !found {
			c.Categories = append(c.Categories, cat)
		}
	}
	
	// Merge global variables
	if c.Variables == nil {
		c.Variables = make(map[string]string)
	}
	for k, v := range imported.Variables {
		c.Variables[k] = v
	}
	
	// Merge template extensions
	for _, ext := range imported.TemplateExts {
		found := false
		for _, existing := range c.TemplateExts {
			if existing == ext {
				found = true
				break
			}
		}
		if !found {
			c.TemplateExts = append(c.TemplateExts, ext)
		}
	}
	
	// Merge files (skip duplicates based on target)
	for _, importedFile := range imported.Files {
		if err := c.AddConfigFile(importedFile); err != nil {
			if IsValidationError(err) {
				// Skip duplicate files but log the issue
				fmt.Printf("Skipping duplicate file %s: %v\n", importedFile.Name, err)
			} else {
				multiErr.Add(err)
			}
		}
	}
	
	// Update editor/shell if not set
	if c.Editor == "" || c.Editor == "vim" {
		c.Editor = imported.Editor
	}
	if c.Shell == "" || c.Shell == "bash" {
		c.Shell = imported.Shell
	}
	
	if multiErr.HasErrors() {
		return &multiErr
	}
	
	return nil
}
