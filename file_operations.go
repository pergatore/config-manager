package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Use Gum to select files/directories to add
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
			isManaged := false
			for _, file := range config.Files {
				// More thorough check for already managed files
				if file.Target == fullPath || strings.Contains(file.Target, fullPath) || strings.Contains(fullPath, file.Target) {
					isManaged = true
					break
				}
			}
			if !isManaged {
				candidates = append(candidates, dir+" (directory)")
			}
		}
	}
	
	// Add option to browse for custom file/directory
	candidates = append(candidates, "Browse for other file/directory...")
	
	if len(candidates) == 0 {
		return "", fmt.Errorf("no unmanaged config files or directories found")
	}
	
	// Use gum choose to select
	cmd := exec.Command("gum", "choose", "--header", "Select config file or directory to add:")
	cmd.Args = append(cmd.Args, candidates...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gum selection failed: %v", err)
	}
	
	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", fmt.Errorf("no selection made")
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

// Text-based fallback for file selection
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
			isManaged := false
			for _, file := range config.Files {
				// More thorough check for already managed files
				if file.Target == fullPath || strings.Contains(file.Target, fullPath) || strings.Contains(fullPath, file.Target) {
					isManaged = true
					break
				}
			}
			if !isManaged {
				candidates = append(candidates, dir+" (directory)")
			}
		}
	}
	
	if len(candidates) == 0 {
		return "", fmt.Errorf("no unmanaged config files or directories found")
	}
	
	// Display options
	for i, candidate := range candidates {
		fmt.Printf("%d. %s\n", i+1, candidate)
	}
	
	fmt.Printf("%d. Enter custom path\n", len(candidates)+1)
	fmt.Print("\nSelect option (number): ")
	
	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil {
		return "", fmt.Errorf("invalid input")
	}
	
	if choice < 1 || choice > len(candidates)+1 {
		return "", fmt.Errorf("invalid choice")
	}
	
	if choice == len(candidates)+1 {
		// Custom path
		fmt.Print("Enter file/directory path (relative to home): ")
		var customPath string
		if _, err := fmt.Scanf("%s", &customPath); err != nil {
			return "", fmt.Errorf("invalid path input")
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

// Browse for a custom file or directory
func browseForFile() (string, error) {
	homeDir, _ := os.UserHomeDir()
	
	// First ask if they want to browse for file or directory
	typeCmd := exec.Command("gum", "choose", "--header", "What do you want to add?", "File", "Directory")
	typeOutput, err := typeCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get selection type: %v", err)
	}
	
	selectionType := strings.TrimSpace(string(typeOutput))
	if selectionType == "" {
		return "", fmt.Errorf("no type selected")
	}
	
	// Use gum file to browse
	var cmd *exec.Cmd
	if selectionType == "Directory" {
		cmd = exec.Command("gum", "file", "--directory", "--height", "15", homeDir)
	} else {
		cmd = exec.Command("gum", "file", "--height", "15", homeDir)
	}
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("file browser failed: %v", err)
	}
	
	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", fmt.Errorf("no file/directory selected")
	}
	
	// Convert to relative path from home directory if possible
	if strings.HasPrefix(selected, homeDir) {
		relative := strings.TrimPrefix(selected, homeDir)
		if strings.HasPrefix(relative, "/") {
			relative = relative[1:]
		}
		if relative != "" {
			selected = relative
		}
	}
	
	return selected, nil
}

// Create ConfigFile from selected path
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
	
	// Check if it's a directory
	isDirectory := false
	if info, err := os.Stat(targetPath); err == nil && info.IsDir() {
		isDirectory = true
		// For directories, we'll manage the entire directory
		fileName = filepath.Base(targetPath)
	}
	
	// Auto-categorize
	category := categorizeDotfile(fileName, config.Categories)
	
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

// Apply all configuration files (create symlinks)
func applyAllConfigs(config *Config) ([]string, error) {
	var messages []string
	var errors []string
	
	for _, file := range config.Files {
		if msg, err := linkConfigFile(config, &file); err != nil {
			errors = append(errors, fmt.Sprintf("❌ %s: %v", file.Name, err))
		} else {
			messages = append(messages, msg)
		}
	}
	
	if len(errors) > 0 {
		return messages, fmt.Errorf("linking failed for some files: %s", strings.Join(errors, "; "))
	}
	
	return messages, nil
}

// Link a config file
func linkConfigFile(config *Config, file *ConfigFile) (string, error) {
	sourceDir := filepath.Dir(filepath.Join(config.DotfilesDir, file.Source))
	os.MkdirAll(sourceDir, 0755)
	
	sourcePath := filepath.Join(config.DotfilesDir, file.Source)
	var messages []string
	
	// Check if the target is a directory or file
	isDirectory := false
	if info, err := os.Stat(file.Target); err == nil && info.IsDir() {
		isDirectory = true
	}
	
	// Handle existing target file/directory
	if _, err := os.Lstat(file.Target); err == nil {
		// Target exists, check what it is
		if info, err := os.Lstat(file.Target); err == nil {
			if info.Mode()&fs.ModeSymlink != 0 {
				// It's already a symlink, check if it points to our source
				if link, err := os.Readlink(file.Target); err == nil {
					if link == sourcePath {
						// Already linked correctly, nothing to do
						return fmt.Sprintf("✅ %s already linked correctly", file.Name), nil
					}
				}
				// Remove the old symlink
				os.Remove(file.Target)
				messages = append(messages, fmt.Sprintf("🔄 Updated symlink for %s", file.Name))
			} else {
				// It's a regular file or directory, back it up first
				backupPath := file.Target + ".backup." + time.Now().Format("20060102-150405")
				if err := os.Rename(file.Target, backupPath); err != nil {
					return "", fmt.Errorf("failed to backup existing file to %s: %v", backupPath, err)
				}
				messages = append(messages, fmt.Sprintf("📁 Backed up %s", filepath.Base(file.Target)))
				
				// Copy the original file/directory to our dotfiles directory if source doesn't exist
				if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
					if isDirectory {
						// Copy entire directory structure
						if err := copyDirectory(backupPath, sourcePath); err != nil {
							return "", fmt.Errorf("failed to copy directory to dotfiles: %v", err)
						}
						messages = append(messages, fmt.Sprintf("📋 Preserved %s directory", file.Name))
					} else {
						// Copy single file
						if data, err := os.ReadFile(backupPath); err == nil {
							os.MkdirAll(filepath.Dir(sourcePath), 0755)
							if err := os.WriteFile(sourcePath, data, 0644); err != nil {
								return "", fmt.Errorf("failed to copy original to dotfiles: %v", err)
							}
							messages = append(messages, fmt.Sprintf("📋 Preserved %s content", file.Name))
						}
					}
				}
			}
		}
	}
	
	// Create source file/directory from template if needed and doesn't exist
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		if file.Template && !isDirectory {
			if err := createFromTemplate(config, file, sourcePath); err != nil {
				return "", err
			}
		} else if !isDirectory {
			// If source doesn't exist and it's not a template, create a basic file
			basicContent := fmt.Sprintf("# %s configuration\n# Managed by config-manager\n", file.Name)
			if err := os.WriteFile(sourcePath, []byte(basicContent), 0644); err != nil {
				return "", fmt.Errorf("failed to create source file: %v", err)
			}
		} else {
			// For directories, create the directory structure
			if err := os.MkdirAll(sourcePath, 0755); err != nil {
				return "", fmt.Errorf("failed to create source directory: %v", err)
			}
		}
	}
	
	// Create target directory (parent of the target)
	os.MkdirAll(filepath.Dir(file.Target), 0755)
	
	// Create symlink
	if err := os.Symlink(sourcePath, file.Target); err != nil {
		return "", fmt.Errorf("failed to create symlink: %v", err)
	}
	
	messages = append(messages, fmt.Sprintf("🔗 Linked %s", file.Name))
	
	// Join all messages with " • "
	if len(messages) > 0 {
		return strings.Join(messages, " • "), nil
	}
	
	return fmt.Sprintf("✅ Processed %s", file.Name), nil
}

// Copy directory recursively
func copyDirectory(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}
	
	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	
	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		
		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// Copy single file
func copyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	
	// Get source file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	// Write to destination with same permissions
	return os.WriteFile(dst, data, srcInfo.Mode())
}

// Open a config file or directory in the editor
func openInEditor(config *Config, file *ConfigFile) error {
	sourcePath := filepath.Join(config.DotfilesDir, file.Source)
	
	// Check if the source path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source file/directory does not exist: %s", sourcePath)
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

// Open a directory - let user choose which file to edit
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
		return fmt.Errorf("failed to scan directory: %v", err)
	}
	
	if len(editableFiles) == 0 {
		return fmt.Errorf("no editable files found in directory")
	}
	
	// Use gum to let user choose which file to edit
	selectedFile, err := selectFileToEdit(editableFiles)
	if err != nil {
		return err
	}
	
	// Open the selected file
	fullPath := filepath.Join(dirPath, selectedFile)
	return openFileInEditor(config.Editor, fullPath)
}

// Let user select which file to edit from a directory
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
		return "", fmt.Errorf("file selection cancelled")
	}
	
	selected := strings.TrimSpace(string(output))
	if selected == "" || strings.Contains(selected, "Cancel") {
		return "", fmt.Errorf("file selection cancelled")
	}
	
	return selected, nil
}

// Text-based file selection fallback
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
		return "", fmt.Errorf("invalid input")
	}
	
	if choice == 0 {
		return "", fmt.Errorf("file selection cancelled")
	}
	
	if choice < 1 || choice > len(files) {
		return "", fmt.Errorf("invalid choice")
	}
	
	return files[choice-1], nil
}

// Check if a file is editable (text-based config file)
func isEditableFile(filename string) bool {
	// Skip binary files and temporary files
	if strings.HasSuffix(filename, ".lock") ||
		strings.HasSuffix(filename, ".tmp") ||
		strings.HasSuffix(filename, ".log") ||
		strings.HasSuffix(filename, ".pid") ||
		strings.HasSuffix(filename, ".sock") {
		return false
	}
	
	// Include common config file extensions and files without extensions
	editableExts := []string{
		".conf", ".config", ".cfg", ".ini", ".yaml", ".yml", ".toml", ".json",
		".sh", ".bash", ".zsh", ".fish", ".vim", ".lua", ".py", ".rb", ".js", ".ts",
		".md", ".txt", ".rc", ".profile", ".aliases", ".env", ".gitignore",
		".tmpl", ".template", ".tpl",
	}
	
	// Files without extensions are often config files
	if !strings.Contains(filename, ".") {
		return true
	}
	
	// Check known config file extensions
	for _, ext := range editableExts {
		if strings.HasSuffix(strings.ToLower(filename), ext) {
			return true
		}
	}
	
	// Files starting with dot are often config files
	if strings.HasPrefix(filename, ".") {
		return true
	}
	
	return false
}

// Open a single file in the configured editor
func openFileInEditor(editor, filePath string) error {
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
		err := cmd.Run()
		
		// After the editor exits, give the terminal a moment to settle
		// and ensure stdin is in a good state
		if err == nil {
			// Try to reset terminal state by reading any pending input
			// This helps prevent the "resource temporarily unavailable" error
		}
		
		return err
	default:
		// For GUI editors, just run them
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

func createBackupInDir(config *Config, backupDir string) int {
	os.MkdirAll(backupDir, 0755)
	
	backedUp := 0
	for _, file := range config.Files {
		if _, err := os.Stat(file.Target); err == nil {
			// Copy file to backup
			if data, err := os.ReadFile(file.Target); err == nil {
				backupPath := filepath.Join(backupDir, filepath.Base(file.Target))
				if os.WriteFile(backupPath, data, 0644) == nil {
					backedUp++
				}
			}
		}
	}
	return backedUp
}
