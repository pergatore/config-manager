package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// fileItem methods for bubbles/list interface (unchanged)
func (i fileItem) FilterValue() string { return i.file.Name }

func (i fileItem) Title() string {
	status := "✗"
	if i.file.IsLinked {
		status = "✓"
	} else if i.file.HasConflict {
		status = "⚠️"
	}
	return fmt.Sprintf("%s %s", status, i.file.Name)
}

func (i fileItem) Description() string {
	return fmt.Sprintf("%s → %s", i.file.Target, i.file.Source)
}

// Initialize application with enhanced error handling
func initialModel() model {
	config := loadConfig()
	
	// Create initial file list with default dimensions
	var fileList list.Model
	if config != nil {
		// Ensure directories exist
		if err := config.EnsureDirectoriesExist(); err != nil {
			fmt.Printf("Warning: failed to create directories: %v\n", err)
		}
		
		// Create default templates if they don't exist
		if err := createDefaultTemplates(config); err != nil {
			fmt.Printf("Warning: failed to create default templates: %v", err)
		}
		
		updateFileStatuses(config)
		fileList = createFileList(config.Files, 76, 14) // Default size
	} else {
		fileList = createFileList([]ConfigFile{}, 76, 14)
	}
	
	return model{
		config:      config,
		currentView: "main",
		fileList:    fileList,
		message:     "Welcome to Config Manager! Use 'a' to add configs, 'l' to link them.",
		messageType: "success",
		width:       80,  // Default width
		height:      20,  // Default height
	}
}

// Bubbletea interface methods (Update method enhanced)
func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Update list size - account for header (3 lines), status (2 lines), help (2 lines)
		listHeight := m.height - 7
		if listHeight < 5 {
			listHeight = 5
		}
		listWidth := m.width - 4
		if listWidth < 40 {
			listWidth = 40
		}
		
		m.fileList.SetSize(listWidth, listHeight)
		
	case editorFinishedMsg:
		// Handle the editor finishing
		if msg.err != nil {
			if IsConfigError(msg.err) {
				m.message = fmt.Sprintf("Editor error: %v", msg.err)
			} else {
				m.message = fmt.Sprintf("Failed to open editor: %v", msg.err)
			}
			m.messageType = "error"
		} else {
			// After editing, update file statuses and remove duplicates
			updateFileStatuses(m.config)
			
			// Calculate proper dimensions for the list
			listHeight := m.height - 7
			if listHeight < 5 {
				listHeight = 5
			}
			listWidth := m.width - 4
			if listWidth < 40 {
				listWidth = 40
			}
			
			// Completely recreate the file list to ensure clean display
			m.fileList = createFileList(m.config.Files, listWidth, listHeight)
			
			// Save config to persist any changes
			if err := saveConfigSafe(m.config); err != nil {
				m.message = fmt.Sprintf("Finished editing %s (warning: failed to save config: %v)", msg.fileName, err)
				m.messageType = "warning"
			} else {
				m.message = fmt.Sprintf("Finished editing %s", msg.fileName)
				m.messageType = "success"
			}
		}
		
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
			
		case key.Matches(msg, keys.Add):
			return m.handleAdd()
			
		case key.Matches(msg, keys.Remove):
			return m.handleRemove()
			
		case key.Matches(msg, keys.Link):
			return m.handleLinkSelected()
			
		case key.Matches(msg, keys.LinkAll):
			return m.handleLinkAll()
			
		case key.Matches(msg, keys.Edit):
			return m.handleEdit()
			
		case key.Matches(msg, keys.Backup):
			return m.handleBackup()
		}
	}
	
	// Update the file list
	var cmd tea.Cmd
	m.fileList, cmd = m.fileList.Update(msg)
	return m, cmd
}

func (m model) View() string {
	// Header with stats
	stats := m.config.GetStats()
	header := titleStyle.Render("Config Manager") + 
		fmt.Sprintf(" (%d files, %d linked, %d conflicts)", 
			stats["total_files"], stats["linked_files"], stats["conflicted_files"]) + "\n\n"
	
	// Main content - the file list
	content := m.fileList.View()
	
	// Status/message bar with enhanced styling
	statusStyle := successStyle
	if m.messageType == "error" {
		statusStyle = errorStyle
	} else if m.messageType == "warning" {
		statusStyle = warningStyle
	}
	
	status := "\n" + statusStyle.Render(m.message)
	
	// Fancy help bar at the bottom
	helpItems := []string{
		helpKeyStyle.Render("a") + helpDescStyle.Render(" add"),
		helpKeyStyle.Render("r") + helpDescStyle.Render(" remove"),
		helpKeyStyle.Render("e") + helpDescStyle.Render(" edit"),
		helpKeyStyle.Render("l") + helpDescStyle.Render(" link selected"),
		helpKeyStyle.Render("L") + helpDescStyle.Render(" link all"),
		helpKeyStyle.Render("b") + helpDescStyle.Render(" backup"),
		helpKeyStyle.Render("q") + helpDescStyle.Render(" quit"),
	}
	
	helpContent := strings.Join(helpItems, helpSeparatorStyle.Render(" • "))
	helpBar := "\n" + helpBarStyle.Render(helpContent)
	
	return header + content + status + helpBar
}

// Enhanced event handlers with atomic operations and better error handling

func (m model) handleAdd() (tea.Model, tea.Cmd) {
	// Use enhanced file selection
	selectedPath, err := selectFileToAdd(m.config)
	if err != nil {
		// Handle different error types appropriately
		if IsConfigError(err) {
			if strings.Contains(err.Error(), "cancelled") {
				m.message = "Add operation cancelled"
				m.messageType = "warning"
			} else {
				m.message = fmt.Sprintf("Add failed: %v", err)
				m.messageType = "error"
			}
		} else {
			m.message = fmt.Sprintf("Add failed: %v", err)
			m.messageType = "error"
		}
		
		return m, tea.Batch(
			tea.HideCursor,
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)
	}
	
	// Create ConfigFile from selected path
	newFile, err := createConfigFileFromPath(selectedPath, m.config)
	if err != nil {
		m.message = fmt.Sprintf("Failed to create config entry: %v", err)
		m.messageType = "error"
		return m, tea.Batch(
			tea.HideCursor,
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)
	}
	
	// Add file using the safe method
	if err := m.config.AddConfigFile(newFile); err != nil {
		if IsValidationError(err) {
			m.message = fmt.Sprintf("Validation error: %v", err)
		} else {
			m.message = fmt.Sprintf("Failed to add file: %v", err)
		}
		m.messageType = "error"
		return m, tea.Batch(
			tea.HideCursor,
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)
	}
	
	// Update the list items properly
	fileItems := make([]list.Item, len(m.config.Files))
	for i, file := range m.config.Files {
		fileItems[i] = fileItem{file: file}
	}
	
	m.fileList.SetItems(fileItems)
	
	m.message = fmt.Sprintf("Added %s to configuration", newFile.Name)
	m.messageType = "success"
	
	// Save config safely
	if err := saveConfigSafe(m.config); err != nil {
		m.message += fmt.Sprintf(" (warning: failed to save: %v)", err)
		m.messageType = "warning"
	}
	
	return m, tea.Batch(
		tea.HideCursor,
		func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		},
	)
}

func (m model) handleRemove() (tea.Model, tea.Cmd) {
	if selected := m.fileList.SelectedItem(); selected != nil {
		selectedFileItem := selected.(fileItem)
		
		// Remove file using the safe method
		if err := m.config.RemoveConfigFile(selectedFileItem.file.Target); err != nil {
			m.message = fmt.Sprintf("Failed to remove %s: %v", selectedFileItem.file.Name, err)
			m.messageType = "error"
		} else {
			// Update the list items properly
			fileItems := make([]list.Item, len(m.config.Files))
			for i, file := range m.config.Files {
				fileItems[i] = fileItem{file: file}
			}
			
			m.fileList.SetItems(fileItems)
			
			m.message = fmt.Sprintf("Removed %s from configuration", selectedFileItem.file.Name)
			m.messageType = "success"
			
			// Save config safely
			if err := saveConfigSafe(m.config); err != nil {
				m.message += fmt.Sprintf(" (warning: failed to save: %v)", err)
				m.messageType = "warning"
			}
		}
	} else {
		m.message = "No file selected to remove"
		m.messageType = "warning"
	}
	
	return m, func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m model) handleLinkSelected() (tea.Model, tea.Cmd) {
	if selected := m.fileList.SelectedItem(); selected != nil {
		selectedFileItem := selected.(fileItem)
		
		// Use atomic linking operation
		msg, err := linkConfigFile(m.config, &selectedFileItem.file)
		if err != nil {
			if IsConfigError(err) {
				m.message = fmt.Sprintf("Link error for %s: %v", selectedFileItem.file.Name, err)
			} else {
				m.message = fmt.Sprintf("Error linking %s: %v", selectedFileItem.file.Name, err)
			}
			m.messageType = "error"
		} else {
			// Update file statuses
			updateFileStatuses(m.config)
			
			// Update the list items with new statuses
			fileItems := make([]list.Item, len(m.config.Files))
			for i, file := range m.config.Files {
				fileItems[i] = fileItem{file: file}
			}
			
			m.fileList.SetItems(fileItems)
			
			m.message = msg
			m.messageType = "success"
		}
	} else {
		m.message = "No file selected to link"
		m.messageType = "warning"
	}
	
	return m, func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m model) handleLinkAll() (tea.Model, tea.Cmd) {
	// Use atomic operations for linking all configs
	messages, err := applyAllConfigs(m.config)
	if err != nil {
		if IsConfigError(err) || IsValidationError(err) {
			m.message = fmt.Sprintf("Configuration error: %v", err)
		} else {
			m.message = fmt.Sprintf("Error linking configs: %v", err)
		}
		m.messageType = "error"
	} else {
		// Update file statuses
		updateFileStatuses(m.config)
		
		// Update the list items with new statuses
		fileItems := make([]list.Item, len(m.config.Files))
		for i, file := range m.config.Files {
			fileItems[i] = fileItem{file: file}
		}
		
		m.fileList.SetItems(fileItems)
		
		// Show summary of what was done
		if len(messages) > 0 {
			summary := fmt.Sprintf("✅ Successfully processed %d files", len(m.config.Files))
			if len(messages) <= 3 {
				if len(messages) == 1 {
					summary = messages[0]
				} else {
					summary += ": " + strings.Join(messages[:2], ", ")
				}
			} else {
				summary += fmt.Sprintf(" (%d operations completed)", len(messages))
			}
			m.message = summary
		} else {
			m.message = fmt.Sprintf("✅ Successfully linked %d configuration files", len(m.config.Files))
		}
		m.messageType = "success"
	}
	
	return m, func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m model) handleEdit() (tea.Model, tea.Cmd) {
	if selected := m.fileList.SelectedItem(); selected != nil {
		selectedFileItem := selected.(fileItem)
		
		// Use enhanced editor opening with better error handling
		sourcePath := filepath.Join(m.config.DotfilesDir, selectedFileItem.file.Source)
		
		// Check if the source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			m.message = fmt.Sprintf("Source file/directory does not exist: %s", sourcePath)
			m.messageType = "error"
			return m, nil
		}
		
		// Check if it's a directory
		if info, err := os.Stat(sourcePath); err == nil && info.IsDir() {
			// Handle directory selection first
			selectedFile, err := handleDirectorySelection(sourcePath)
			if err != nil {
				if IsConfigError(err) && strings.Contains(err.Error(), "cancelled") {
					m.message = "Edit operation cancelled"
					m.messageType = "warning"
				} else {
					m.message = fmt.Sprintf("File selection failed: %v", err)
					m.messageType = "error"
				}
				
				return m, tea.Batch(
					tea.HideCursor,
					func() tea.Msg {
						return tea.WindowSizeMsg{Width: m.width, Height: m.height}
					},
				)
			}
			
			// Open the selected file from the directory
			fullPath := filepath.Join(sourcePath, selectedFile)
			return m, tea.ExecProcess(createSingleFileEditorCommand(m.config.Editor, fullPath), func(err error) tea.Msg {
				return editorFinishedMsg{err: err, fileName: selectedFile}
			})
		} else {
			// Single file - open directly
			return m, tea.ExecProcess(createSingleFileEditorCommand(m.config.Editor, sourcePath), func(err error) tea.Msg {
				return editorFinishedMsg{err: err, fileName: selectedFileItem.file.Name}
			})
		}
	} else {
		m.message = "No file selected to edit"
		m.messageType = "warning"
		return m, nil
	}
}

func (m model) handleBackup() (tea.Model, tea.Cmd) {
	// Create enhanced backup
	backupDir := createBackupWithStats(m.config)
	if backupDir == "" {
		m.message = "Failed to create backup"
		m.messageType = "error"
	} else {
		stats := m.config.GetStats()
		m.message = fmt.Sprintf("Backed up %d files to %s", stats["total_files"], filepath.Base(backupDir))
		m.messageType = "success"
	}
	
	return m, nil
}

// Enhanced backup creation with statistics
func createBackupWithStats(config *Config) string {
	backupDir := fmt.Sprintf("%s/backups/%s", config.ConfigDir, time.Now().Format("2006-01-02_15-04-05"))
	backedUp := createBackupInDir(config, backupDir)
	
	if backedUp == 0 {
		return ""
	}
	
	return backupDir
}

// Message type for when editor finishes (unchanged)
type editorFinishedMsg struct {
	err      error
	fileName string
}

// Enhanced directory selection handling
func handleDirectorySelection(dirPath string) (string, error) {
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
		return "", NewConfigError("scan directory", dirPath, err)
	}
	
	if len(editableFiles) == 0 {
		return "", NewConfigError("find editable files", dirPath, 
			fmt.Errorf("no editable files found in directory"))
	}
	
	// Use the existing file selection logic with enhanced error handling
	return selectFileToEdit(editableFiles)
}

// Create command for editing a single file (unchanged)
func createSingleFileEditorCommand(editor, filePath string) *exec.Cmd {
	// Handle different editors that might need special arguments
	switch editor {
	case "code", "vscode":
		return exec.Command("code", "--wait", filePath)
	case "subl", "sublime":
		return exec.Command("subl", "--wait", filePath)
	case "atom":
		return exec.Command("atom", "--wait", filePath)
	default:
		// For vim, nvim, emacs, nano, etc.
		return exec.Command(editor, filePath)
	}
}

// Enhanced file list creation with better sizing
func createFileList(files []ConfigFile, width, height int) list.Model {
	fileItems := make([]list.Item, len(files))
	for i, file := range files {
		fileItems[i] = fileItem{file: file}
	}
	
	// Ensure minimum dimensions
	if width < 40 {
		width = 40
	}
	if height < 5 {
		height = 5
	}
	
	fileList := list.New(fileItems, list.NewDefaultDelegate(), width, height)
	fileList.Title = "Managed Configuration Files"
	fileList.SetShowStatusBar(false)
	fileList.SetShowHelp(false) // We'll show our own help
	fileList.SetFilteringEnabled(false) // Disable filtering to avoid interference
	
	return fileList
}

// Enhanced error checking for editable files
func isEditableFile(filename string) bool {
	// Skip binary files and temporary files
	if strings.HasSuffix(filename, ".lock") ||
		strings.HasSuffix(filename, ".tmp") ||
		strings.HasSuffix(filename, ".log") ||
		strings.HasSuffix(filename, ".pid") ||
		strings.HasSuffix(filename, ".sock") ||
		strings.HasSuffix(filename, ".swp") ||
		strings.HasSuffix(filename, ".swo") ||
		strings.HasSuffix(filename, "~") {
		return false
	}
	
	// Include common config file extensions and files without extensions
	editableExts := []string{
		".conf", ".config", ".cfg", ".ini", ".yaml", ".yml", ".toml", ".json",
		".sh", ".bash", ".zsh", ".fish", ".vim", ".lua", ".py", ".rb", ".js", ".ts",
		".md", ".txt", ".rc", ".profile", ".aliases", ".env", ".gitignore",
		".tmpl", ".template", ".tpl", ".service", ".timer", ".desktop",
		".xml", ".html", ".css", ".scss", ".less", ".properties",
	}
	
	// Files without extensions are often config files
	if !strings.Contains(filename, ".") {
		return true
	}
	
	// Check known config file extensions
	lowerFilename := strings.ToLower(filename)
	for _, ext := range editableExts {
		if strings.HasSuffix(lowerFilename, ext) {
			return true
		}
	}
	
	// Files starting with dot are often config files (but exclude some system files)
	if strings.HasPrefix(filename, ".") && !isSystemFile(filename) {
		return true
	}
	
	return false
}
