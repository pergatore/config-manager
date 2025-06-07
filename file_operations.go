package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConflictResolution represents user choices for handling conflicts
type ConflictResolution int

const (
	ConflictBackupAndReplace ConflictResolution = iota
	ConflictSkip
	ConflictViewDiff
	ConflictMerge
	ConflictCancel
)

// ConflictInfo provides details about a file conflict
type ConflictInfo struct {
	File        *ConfigFile
	TargetPath  string
	SourcePath  string
	TargetExists bool
	IsSymlink   bool
	LinkTarget  string
	BackupPath  string // Add backup path field
}

// Enhanced file operations with conflict resolution

// Use Gum to select files/directories to add with better error handling
func selectFileToAdd(config *Config) (string, error) {
	// Check if gum is available
	if _, err := exec.LookPath("gum"); err != nil {
		// Fallback to text-based selection
		return selectFileToAddText(config)
	}
	
	homeDir, _ := os.UserHomeDir()
	
	// Find all potential config files and directories
	candidates := []string{}
	
	// Add common dotfiles that exist but aren't managed
	unmanaged := findUnmanagedDotfiles(config)
	for _, file := range unmanaged {
		candidates = append(candidates, file+" (file)")
	}
	
	// Add common config directories
	configDirs := []string{
		".config/nvim", ".config/alacritty", ".config/kitty", ".config/tmux",
		".config/fish", ".config/starship", ".config/rofi", ".config/i3",
		".config/polybar", ".config/dunst", ".config/picom", ".config/sway",
		".config/waybar", ".config/hypr", ".config/wezterm", ".config/helix",
		".ssh", ".gnupg", ".local/bin",
	}
	
	for _, dir := range configDirs {
		fullPath := filepath.Join(homeDir, dir)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			// Check if not already managed
			if !isFileAlreadyManaged(config, fullPath) {
				candidates = append(candidates, dir+" (directory)")
			}
		}
	}
	
	// Add option to browse for custom file/directory
	candidates = append(candidates, "Browse for other file/directory...")
	
	if len(candidates) == 0 {
		return "", NewConfigError("file selection", "", 
			fmt.Errorf("no unmanaged config files or directories found"))
	}
	
	// Use gum choose to select
	cmd := exec.Command("gum", "choose", "--header", "Select config file or directory to add:")
	cmd.Args = append(cmd.Args, candidates...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	
	output, err := cmd.Output()
	if err != nil {
		return "", NewConfigError("file selection", "", fmt.Errorf("selection cancelled or failed: %v", err))
	}
	
	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", NewConfigError("file selection", "", fmt.Errorf("no selection made"))
	}
	
	// Handle "Browse for other..." option
	if strings.Contains(selected, "Browse for other") {
		return browseForFile()
	}
	
	// Remove the " (file)" or " (directory)" suffix
	if strings.HasSuffix(selected, " (file)") {
		selected = strings.TrimSuffix(selected, " (file)")
	} else if strings.HasSuffix(selected, " (directory)") {
		selected = strings.TrimSuffix(selected, " (directory)")
	}
	
	return selected, nil
}

// isFileAlreadyManaged checks if a file is already being managed
func isFileAlreadyManaged(config *Config, fullPath string) bool {
	for _, file := range config.Files {
		if file.Target == fullPath || strings.Contains(file.Target, fullPath) || strings.Contains(fullPath, file.Target) {
			return true
		}
	}
	return false
}

// Enhanced conflict detection and resolution
func detectConflict(file *ConfigFile, sourcePath string) (*ConflictInfo, error) {
	conflict := &ConflictInfo{
		File:       file,
		TargetPath: file.Target,
		SourcePath: sourcePath,
	}
	
	// Check if target exists
	info, err := os.Lstat(file.Target)
	if os.IsNotExist(err) {
		// No conflict - target doesn't exist
		return nil, nil
	}
	if err != nil {
		return nil, NewConfigError("stat target", file.Target, err)
	}
	
	conflict.TargetExists = true
	
	// Check if target is a symlink
	if info.Mode()&fs.ModeSymlink != 0 {
		conflict.IsSymlink = true
		linkTarget, err := os.Readlink(file.Target)
		if err != nil {
			return nil, NewConfigError("read symlink", file.Target, err)
		}
		conflict.LinkTarget = linkTarget
		
		// Check if it points to our source
		if linkTarget == sourcePath {
			// Already linked correctly - no conflict
			return nil, nil
		}
	}
	
	// There is a conflict
	return conflict, nil
}

// resolveConflictInteractive presents options to user for conflict resolution
func resolveConflictInteractive(conflict *ConflictInfo) (ConflictResolution, error) {
	// Check if gum is available
	if _, err := exec.LookPath("gum"); err != nil {
		return resolveConflictText(conflict)
	}
	
	// Build gum options
	options := []string{
		"Backup existing and replace",
		"View diff",
		"Skip this file", 
		"Cancel operation",
	}
	
	// Add merge option for text files
	if isTextFile(conflict.TargetPath) {
		options = append(options[:2], append([]string{"Merge interactively"}, options[2:]...)...)
	}
	
	// Show conflict information
	fmt.Printf("🚨 Conflict detected for %s\n", conflict.File.Name)
	fmt.Printf("Target: %s\n", conflict.TargetPath)
	if conflict.IsSymlink {
		fmt.Printf("Current symlink points to: %s\n", conflict.LinkTarget)
		fmt.Printf("Would point to: %s\n", conflict.SourcePath)
	} else {
		fmt.Printf("Target exists as regular file/directory\n")
		fmt.Printf("Would be replaced with symlink to: %s\n", conflict.SourcePath)
	}
	fmt.Println()
	
	cmd := exec.Command("gum", "choose", "--header", "How would you like to resolve this conflict?")
	cmd.Args = append(cmd.Args, options...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	
	output, err := cmd.Output()
	if err != nil {
		return ConflictCancel, NewConfigError("conflict resolution", conflict.File.Name, 
			fmt.Errorf("selection cancelled: %v", err))
	}
	
	choice := strings.TrimSpace(string(output))
	switch {
	case strings.Contains(choice, "Backup"):
		return ConflictBackupAndReplace, nil
	case strings.Contains(choice, "View diff"):
		return ConflictViewDiff, nil
	case strings.Contains(choice, "Merge"):
		return ConflictMerge, nil
	case strings.Contains(choice, "Skip"):
		return ConflictSkip, nil
	default:
		return ConflictCancel, nil
	}
}

// resolveConflictText provides text-based conflict resolution
func resolveConflictText(conflict *ConflictInfo) (ConflictResolution, error) {
	fmt.Printf("\n🚨 Conflict detected for %s\n", conflict.File.Name)
	fmt.Printf("Target: %s\n", conflict.TargetPath)
	if conflict.IsSymlink {
		fmt.Printf("Current symlink points to: %s\n", conflict.LinkTarget)
		fmt.Printf("Would point to: %s\n", conflict.SourcePath)
	} else {
		fmt.Printf("Target exists as regular file/directory\n")
		fmt.Printf("Would be replaced with symlink to: %s\n", conflict.SourcePath)
	}
	
	fmt.Println("\nOptions:")
	fmt.Println("1. Backup existing and replace")
	fmt.Println("2. View diff")
	if isTextFile(conflict.TargetPath) {
		fmt.Println("3. Merge interactively")
		fmt.Println("4. Skip this file")
		fmt.Println("5. Cancel operation")
	} else {
		fmt.Println("3. Skip this file")
		fmt.Println("4. Cancel operation")
	}
	
	fmt.Print("Enter choice: ")
	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil {
		return ConflictCancel, NewConfigError("read choice", "", err)
	}
	
	if isTextFile(conflict.TargetPath) {
		switch choice {
		case 1:
			return ConflictBackupAndReplace, nil
		case 2:
			return ConflictViewDiff, nil
		case 3:
			return ConflictMerge, nil
		case 4:
			return ConflictSkip, nil
		default:
			return ConflictCancel, nil
		}
	} else {
		switch choice {
		case 1:
			return ConflictBackupAndReplace, nil
		case 2:
			return ConflictViewDiff, nil
		case 3:
			return ConflictSkip, nil
		default:
			return ConflictCancel, nil
		}
	}
}

// viewDiff shows differences between files
func viewDiff(file1, file2 string) error {
	// Try different diff tools
	diffTools := [][]string{
		{"diff", "-u", file1, file2},
		{"colordiff", "-u", file1, file2},
		{"git", "diff", "--no-index", file1, file2},
	}
	
	for _, tool := range diffTools {
		if _, err := exec.LookPath(tool[0]); err == nil {
			cmd := exec.Command(tool[0], tool[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			
			// Run and don't treat exit code 1 as error (diff found differences)
			err := cmd.Run()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
					return nil // diff found differences, this is normal
				}
				continue // Try next tool
			}
			return nil
		}
	}
	
	return NewConfigError("view diff", file1, fmt.Errorf("no diff tool available"))
}

// Enhanced link config file with conflict resolution
func linkConfigFile(config *Config, file *ConfigFile) (string, error) {
	// Use atomic operations
	return linkConfigFileAtomic(config, file)
}

// linkConfigFileAtomic uses atomic operations for safe linking
func linkConfigFileAtomic(config *Config, file *ConfigFile) (string, error) {
	// Validate configuration before proceeding
	if errors := config.Validate(); len(errors) > 0 {
		return "", NewConfigError("config validation", file.Name, 
			fmt.Errorf("configuration has validation errors"))
	}
	
	// Create and execute atomic transaction
	if err := atomicLinkSingleConfig(config, file); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("✅ Successfully linked %s", file.Name), nil
}

// Apply all configuration files using atomic operations
func applyAllConfigs(config *Config) ([]string, error) {
	// Validate configuration first
	if errors := config.Validate(); len(errors) > 0 {
		var messages []string
		for _, err := range errors {
			messages = append(messages, err.Error())
		}
		return nil, NewConfigError("config validation", "", 
			fmt.Errorf("configuration validation failed: %s", strings.Join(messages, "; ")))
	}
	
	// Use atomic operations for all configs
	if err := atomicLinkAllConfigs(config); err != nil {
		return nil, err
	}
	
	// Generate success messages
	var messages []string
	for _, file := range config.Files {
		messages = append(messages, fmt.Sprintf("✅ %s", file.Name))
	}
	
	return messages, nil
}

// Enhanced file type detection
func isTextFile(filePath string) bool {
	// Check by extension first
	ext := strings.ToLower(filepath.Ext(filePath))
	textExts := []string{
		".txt", ".md", ".json", ".yaml", ".yml", ".toml", ".ini", ".conf", ".config",
		".sh", ".bash", ".zsh", ".fish", ".rc", ".profile", ".aliases",
		".vim", ".lua", ".py", ".rb", ".js", ".ts", ".go", ".rs", ".c", ".cpp",
		".gitconfig", ".gitignore", ".editorconfig",
	}
	
	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}
	
	// Files without extensions are often config files
	if ext == "" {
		return true
	}
	
	// Check file content (first 512 bytes) for binary content
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()
	
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return false
	}
	
	// Check for null bytes (indicator of binary content)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return false
		}
	}
	
	return true
}

// Enhanced backup creation with better organization
func createBackupInDir(config *Config, backupDir string) int {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return 0
	}
	
	backedUp := 0
	for _, file := range config.Files {
		if _, err := os.Stat(file.Target); err == nil {
			// Determine backup filename
			backupName := filepath.Base(file.Target)
			if strings.HasPrefix(backupName, ".") {
				backupName = strings.TrimPrefix(backupName, ".")
			}
			
			backupPath := filepath.Join(backupDir, backupName)
			
			// Handle directories
			if info, err := os.Stat(file.Target); err == nil && info.IsDir() {
				if err := copyDirectory(file.Target, backupPath); err == nil {
					backedUp++
				}
			} else {
				// Handle files
				if data, err := os.ReadFile(file.Target); err == nil {
					if err := os.WriteFile(backupPath, data, 0644); err == nil {
						backedUp++
					}
				}
			}
		}
	}
	return backedUp
}

// Enhanced error handling for editor operations
func openInEditor(config *Config, file *ConfigFile) error {
	sourcePath := filepath.Join(config.DotfilesDir, file.Source)
	
	// Check if the source path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return NewConfigError("open editor", sourcePath, 
			fmt.Errorf("source file/directory does not exist"))
	}
	
	// Validate editor is available
	if _, err := exec.LookPath(config.Editor); err != nil {
		return NewConfigError("open editor", config.Editor, 
			fmt.Errorf("editor not found in PATH: %v", err))
	}
	
	// Check if it's a directory or file
	if info, err := os.Stat(sourcePath); err == nil && info.IsDir() {
		// It's a directory - let user choose which file to edit
		return openDirectoryInEditor(config, sourcePath)
	} else {
		// It's a single file - open it directly
		return openFileInEditor(config.Editor, sourcePath)
	}
}

// Improved directory editing with better error handling
func openDirectoryInEditor(config *Config, dirPath string) error {
	// Find all editable files in the directory recursively
	var editableFiles []string
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories and system files
		if info.IsDir() || isSystemFile(info.Name()) {
			return nil
		}
		
		// Only include text files that are likely to be config files
		if isEditableFile(info.Name()) {
			// Make path relative to the directory for better display
			relPath, _ := filepath.Rel(dirPath, path)
			editableFiles = append(editableFiles, relPath)
		}
		
		return nil
	})
	
	if err != nil {
		return NewConfigError("scan directory", dirPath, err)
	}
	
	if len(editableFiles) == 0 {
		return NewConfigError("open directory", dirPath, 
			fmt.Errorf("no editable files found in directory"))
	}
	
	// Use the existing file selection logic
	selectedFile, err := selectFileToEdit(editableFiles)
	if err != nil {
		return NewConfigError("select file", dirPath, err)
	}
	
	// Open the selected file
	fullPath := filepath.Join(dirPath, selectedFile)
	return openFileInEditor(config.Editor, fullPath)
}

// Enhanced file selection with better error handling
func selectFileToEdit(files []string) (string, error) {
	// Check if gum is available
	if _, err := exec.LookPath("gum"); err != nil {
		// Fallback to text-based selection
		return selectFileToEditText(files)
	}
	
	// Add cancel option to the files list
	options := append([]string{"🚫 Cancel (Esc)"}, files...)
	
	// Use gum choose to select
	cmd := exec.Command("gum", "choose", "--header", "Select file to edit (Esc to cancel):")
	cmd.Args = append(cmd.Args, options...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	
	output, err := cmd.Output()
	if err != nil {
		return "", NewConfigError("file selection", "", 
			fmt.Errorf("file selection cancelled or failed: %v", err))
	}
	
	selected := strings.TrimSpace(string(output))
	if selected == "" || strings.Contains(selected, "Cancel") {
		return "", NewConfigError("file selection", "", 
			fmt.Errorf("file selection cancelled"))
	}
	
	return selected, nil
}

// Text-based file selection fallback with better error handling
func selectFileToEditText(files []string) (string, error) {
	fmt.Println("\n📝 Select file to edit:")
	
	// Show cancel option first
	fmt.Println("0. Cancel")
	
	for i, file := range files {
		fmt.Printf("%d. %s\n", i+1, file)
	}
	
	fmt.Print("\nSelect file (number, 0 to cancel): ")
	
	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil {
		return "", NewConfigError("read input", "", err)
	}
	
	if choice == 0 {
		return "", NewConfigError("file selection", "", 
			fmt.Errorf("file selection cancelled"))
	}
	
	if choice < 1 || choice > len(files) {
		return "", NewConfigError("file selection", "", 
			fmt.Errorf("invalid choice: %d", choice))
	}
	
	return files[choice-1], nil
}

// Enhanced file editor opening with better error handling
func openFileInEditor(editor, filePath string) error {
	// Validate file exists and is readable
	if _, err := os.Stat(filePath); err != nil {
		return NewConfigError("open file", filePath, err)
	}
	
	// Prepare the command
	var cmd *exec.Cmd
	
	// Handle different editors that might need special arguments
	switch editor {
	case "code", "vscode":
		cmd = exec.Command("code", "--wait", filePath)
	case "subl", "sublime":
		cmd = exec.Command("subl", "--wait", filePath)
	case "atom":
		cmd = exec.Command("atom", "--wait", filePath)
	default:
		// For vim, nvim, emacs, nano, etc.
		cmd = exec.Command(editor, filePath)
	}
	
	// For terminal editors, we need to properly handle the terminal state
	switch editor {
	case "vim", "nvim", "emacs", "nano", "micro", "helix":
		// Set up the command to use the current terminal properly
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		// Run the editor and wait for it to complete
		if err := cmd.Run(); err != nil {
			return NewConfigError("run editor", editor, err)
		}
		
		return nil
	default:
		// For GUI editors, just run them
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			return NewConfigError("run editor", editor, err)
		}
		
		return nil
	}
}

// Text-based fallback for file selection with enhanced error handling
func selectFileToAddText(config *Config) (string, error) {
	homeDir, _ := os.UserHomeDir()
	
	fmt.Println("\n📁 Add Configuration File/Directory")
	fmt.Println("Available options:")
	
	candidates := []string{}
	
	// Add common dotfiles that exist but aren't managed
	unmanaged := findUnmanagedDotfiles(config)
	for _, file := range unmanaged {
		candidates = append(candidates, file+" (file)")
	}
	
	// Add common config directories
	configDirs := []string{
		".config/nvim", ".config/alacritty", ".config/kitty", ".config/tmux",
		".config/fish", ".config/starship", ".config/rofi", ".config/i3",
		".config/polybar", ".config/dunst", ".config/picom", ".config/sway",
		".config/waybar", ".config/hypr", ".config/wezterm", ".config/helix",
		".ssh", ".gnupg", ".local/bin",
	}
	
	for _, dir := range configDirs {
		fullPath := filepath.Join(homeDir, dir)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			// Check if not already managed
			if !isFileAlreadyManaged(config, fullPath) {
				candidates = append(candidates, dir+" (directory)")
			}
		}
	}
	
	if len(candidates) == 0 {
		return "", NewConfigError("file discovery", "", 
			fmt.Errorf("no unmanaged config files or directories found"))
	}
	
	// Display options
	for i, candidate := range candidates {
		fmt.Printf("%d. %s\n", i+1, candidate)
	}
	
	fmt.Printf("%d. Enter custom path\n", len(candidates)+1)
	fmt.Print("\nSelect option (number): ")
	
	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil {
		return "", NewConfigError("read input", "", err)
	}
	
	if choice < 1 || choice > len(candidates)+1 {
		return "", NewConfigError("file selection", "", 
			fmt.Errorf("invalid choice: %d", choice))
	}
	
	if choice == len(candidates)+1 {
		// Custom path
		fmt.Print("Enter file/directory path (relative to home): ")
		var customPath string
		if _, err := fmt.Scanf("%s", &customPath); err != nil {
			return "", NewConfigError("read path", "", err)
		}
		return customPath, nil
	}
	
	// Selected from list
	selected := candidates[choice-1]
	
	// Remove the " (file)" or " (directory)" suffix
	if strings.HasSuffix(selected, " (file)") {
		selected = strings.TrimSuffix(selected, " (file)")
	} else if strings.HasSuffix(selected, " (directory)") {
		selected = strings.TrimSuffix(selected, " (directory)")
	}
	
	return selected, nil
}

// Enhanced browse for file with better error handling
func browseForFile() (string, error) {
	// Check if gum is available for the selection, but use text input for path
	if _, err := exec.LookPath("gum"); err != nil {
		// Fallback to text input
		return browseForFileText()
	}
	
	// Ask what type, but then use text input for the path
	typeCmd := exec.Command("gum", "choose", "--header", "What do you want to add?", "File", "Directory", "Cancel")
	typeCmd.Stdin = os.Stdin
	typeCmd.Stderr = os.Stderr
	
	typeOutput, err := typeCmd.Output()
	if err != nil {
		return "", NewConfigError("browse type selection", "", 
			fmt.Errorf("selection cancelled: %v", err))
	}
	
	selectionType := strings.TrimSpace(string(typeOutput))
	if selectionType == "" || selectionType == "Cancel" {
		return "", NewConfigError("browse type selection", "", 
			fmt.Errorf("selection cancelled"))
	}
	
	// Use gum input for the actual path
	return browseForFileGumInput(selectionType)
}

// Enhanced gum input for path entry
func browseForFileGumInput(fileType string) (string, error) {
	var placeholder string
	if fileType == "File" {
		placeholder = ".gitconfig, .zshrc, ~/.vimrc, etc."
	} else {
		placeholder = ".config/nvim, ~/.ssh, ~/Documents, etc."
	}
	
	inputCmd := exec.Command("gum", "input", 
		"--placeholder", placeholder,
		"--prompt", fmt.Sprintf("Enter %s path: ", strings.ToLower(fileType)))
	inputCmd.Stdin = os.Stdin
	inputCmd.Stderr = os.Stderr
	
	output, err := inputCmd.Output()
	if err != nil {
		return "", NewConfigError("path input", "", 
			fmt.Errorf("input cancelled: %v", err))
	}
	
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", NewConfigError("path input", "", 
			fmt.Errorf("no path entered"))
	}
	
	return validateAndNormalizePath(path)
}

// Text-based file browsing fallback with enhanced error handling
func browseForFileText() (string, error) {
	fmt.Println("\n📁 Enter file or directory path")
	fmt.Println("Examples of common config files:")
	fmt.Println("  .gitconfig          (file)")
	fmt.Println("  .zshrc              (file)")
	fmt.Println("  .config/nvim        (directory)")
	fmt.Println("  ~/.ssh              (directory)")
	fmt.Println("  ~/Documents/configs (directory)")
	fmt.Print("\nEnter path (relative to home, or use ~/): ")
	
	var path string
	if _, err := fmt.Scanln(&path); err != nil {
		return "", NewConfigError("read path", "", err)
	}
	
	if strings.TrimSpace(path) == "" {
		return "", NewConfigError("path input", "", 
			fmt.Errorf("no path entered"))
	}
	
	return validateAndNormalizePath(path)
}

// validateAndNormalizePath validates and normalizes the entered path
func validateAndNormalizePath(path string) (string, error) {
	// Expand home directory if path starts with ~/
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		path = filepath.Join(homeDir, path[2:]) // Remove ~/ and join with home
	}
	
	// Determine full path for validation
	var fullPath string
	if strings.HasPrefix(path, "/") {
		// Absolute path
		fullPath = path
	} else {
		// Relative path - relative to home directory
		homeDir, _ := os.UserHomeDir()
		fullPath = filepath.Join(homeDir, path)
	}
	
	// Validate the path exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		// Ask for confirmation
		confirmed, err := confirmNonExistentPath(path)
		if err != nil {
			return "", err
		}
		if !confirmed {
			return "", NewConfigError("path validation", path, 
				fmt.Errorf("path does not exist and not confirmed"))
		}
	}
	
	// Convert back to relative path if it was within home directory
	homeDir, _ := os.UserHomeDir()
	if strings.HasPrefix(fullPath, homeDir) {
		relativePath := strings.TrimPrefix(fullPath, homeDir)
		if strings.HasPrefix(relativePath, "/") {
			relativePath = relativePath[1:]
		}
		if relativePath != "" {
			return relativePath, nil
		}
	}
	
	return path, nil
}

// confirmNonExistentPath asks user to confirm adding a non-existent path
func confirmNonExistentPath(path string) (bool, error) {
	// Try gum first
	if _, err := exec.LookPath("gum"); err == nil {
		confirmCmd := exec.Command("gum", "confirm", 
			fmt.Sprintf("Path '%s' does not exist. Add anyway?", path))
		confirmCmd.Stdin = os.Stdin
		confirmCmd.Stderr = os.Stderr
		
		if err := confirmCmd.Run(); err != nil {
			return false, nil // User said no or cancelled
		}
		return true, nil
	}
	
	// Fallback to text input
	fmt.Printf("Warning: %s does not exist yet. Add anyway? (y/N): ", path)
	var confirm string
	if _, err := fmt.Scanln(&confirm); err != nil {
		return false, NewConfigError("read confirmation", "", err)
	}
	
	return strings.ToLower(confirm) == "y" || strings.ToLower(confirm) == "yes", nil
}

// Enhanced createConfigFileFromPath with better error handling
func createConfigFileFromPath(selectedPath string, config *Config) (ConfigFile, error) {
	homeDir, _ := os.UserHomeDir()
	
	var targetPath string
	var fileName string
	
	// Handle different path formats
	if strings.HasPrefix(selectedPath, "/") {
		// Absolute path
		targetPath = selectedPath
		fileName = filepath.Base(selectedPath)
	} else if strings.HasPrefix(selectedPath, "~") {
		// Home directory path
		targetPath = strings.Replace(selectedPath, "~", homeDir, 1)
		fileName = filepath.Base(targetPath)
	} else {
		// Relative to home directory
		targetPath = filepath.Join(homeDir, selectedPath)
		fileName = selectedPath
		if strings.HasPrefix(fileName, ".") {
			fileName = filepath.Base(fileName)
		}
	}
	
	// Validate target path is within reasonable bounds
	if !strings.HasPrefix(targetPath, homeDir) && !strings.HasPrefix(targetPath, "/etc") {
		return ConfigFile{}, NewConfigError("create config file", selectedPath,
			fmt.Errorf("target path outside of home directory or /etc"))
	}
	
	// Check if it's a directory
	isDirectory := false
	if info, err := os.Stat(targetPath); err == nil && info.IsDir() {
		isDirectory = true
		fileName = filepath.Base(targetPath)
	}
	
	// Auto-categorize with validation
	category := categorizeDotfile(fileName, config.Categories)
	if category == "" {
		category = "misc" // Default fallback
	}
	
	// Check if it might be a template
	isTemplate := false
	if !isDirectory {
		if data, err := os.ReadFile(targetPath); err == nil {
			content := strings.ToLower(string(data))
			// Look for template patterns
			if strings.Contains(content, "{{") || 
			   strings.Contains(content, "$user") || 
			   strings.Contains(content, "$email") ||
			   strings.Contains(content, "$editor") {
				isTemplate = true
			}
		}
	}
	
	// Determine source path in dotfiles directory
	sourcePath := filepath.Join(category, strings.TrimPrefix(fileName, "."))
	if isDirectory {
		sourcePath = filepath.Join(category, fileName)
	}
	
	return ConfigFile{
		Name:      fileName,
		Source:    sourcePath,
		Target:    targetPath,
		Category:  category,
		Template:  isTemplate,
		Variables: make(map[string]string),
	}, nil
}
