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

// fileItem methods for bubbles/list interface
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

// Initialize application
func initialModel() model {
	config := loadConfig()
	
	// Create initial file list with default dimensions
	var fileList list.Model
	if config != nil {
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

// Bubbletea interface methods
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
			m.message = fmt.Sprintf("Failed to open editor: %v", msg.err)
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
			saveConfig(m.config)
			
			m.message = fmt.Sprintf("Finished editing %s", msg.fileName)
			m.messageType = "success"
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
	// Header
	header := titleStyle.Render("Config Manager") + "\n\n"
	
	// Main content - the file list
	content := m.fileList.View()
	
	// Status/message bar
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

// Event handlers
func (m model) handleAdd() (tea.Model, tea.Cmd) {
	// Use Gum to select files/directories to add
	selectedPath, err := selectFileToAdd(m.config)
	if err != nil {
		m.message = fmt.Sprintf("Add failed: %v", err)
		m.messageType = "error"
		return m, nil
	}
	
	// Check if this file is already managed
	homeDir, _ := os.UserHomeDir()
	var targetPath string
	if strings.HasPrefix(selectedPath, "/") {
		targetPath = selectedPath
	} else if strings.HasPrefix(selectedPath, "~") {
		targetPath = strings.Replace(selectedPath, "~", homeDir, 1)
	} else {
		targetPath = filepath.Join(homeDir, selectedPath)
	}
	
	// Check for duplicates
	for _, file := range m.config.Files {
		if file.Target == targetPath {
			m.message = fmt.Sprintf("File %s is already managed", selectedPath)
			m.messageType = "warning"
			return m, nil
		}
	}
	
	// Create ConfigFile from selected path
	newFile, err := createConfigFileFromPath(selectedPath, m.config)
	if err != nil {
		m.message = fmt.Sprintf("Failed to create config entry: %v", err)
		m.messageType = "error"
		return m, nil
	}
	
	m.config.Files = append(m.config.Files, newFile)
	
	// Update file statuses for all files
	updateFileStatuses(m.config)
	
	// Update the list items properly
	fileItems := make([]list.Item, len(m.config.Files))
	for i, file := range m.config.Files {
		fileItems[i] = fileItem{file: file}
	}
	
	// Use SetItems to update the list and then force a refresh
	m.fileList.SetItems(fileItems)
	
	m.message = fmt.Sprintf("Added %s to configuration", newFile.Name)
	m.messageType = "success"
	saveConfig(m.config)
	
	// Return a command that forces a complete screen refresh
	return m, func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m model) handleRemove() (tea.Model, tea.Cmd) {
	if selected := m.fileList.SelectedItem(); selected != nil {
		selectedFileItem := selected.(fileItem)
		
		// Remove file from config
		for i, file := range m.config.Files {
			if file.Name == selectedFileItem.file.Name && file.Target == selectedFileItem.file.Target {
				m.config.Files = append(m.config.Files[:i], m.config.Files[i+1:]...)
				break
			}
		}
		
		// Update file statuses for remaining files
		updateFileStatuses(m.config)
		
		// Update the list items properly
		fileItems := make([]list.Item, len(m.config.Files))
		for i, file := range m.config.Files {
			fileItems[i] = fileItem{file: file}
		}
		
		// Use SetItems to update the list
		m.fileList.SetItems(fileItems)
		
		m.message = fmt.Sprintf("Removed %s from configuration", selectedFileItem.file.Name)
		m.messageType = "success"
		saveConfig(m.config)
	}
	
	// Return a command that forces a complete screen refresh
	return m, func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m model) handleLinkSelected() (tea.Model, tea.Cmd) {
	if selected := m.fileList.SelectedItem(); selected != nil {
		selectedFileItem := selected.(fileItem)
		
		// Link just the selected file
		msg, err := linkConfigFile(m.config, &selectedFileItem.file)
		if err != nil {
			m.message = fmt.Sprintf("Error linking %s: %v", selectedFileItem.file.Name, err)
			m.messageType = "error"
		} else {
			// Update file statuses
			updateFileStatuses(m.config)
			
			// Update the list items with new statuses
			fileItems := make([]list.Item, len(m.config.Files))
			for i, file := range m.config.Files {
				fileItems[i] = fileItem{file: file}
			}
			
			// Use SetItems to update the list
			m.fileList.SetItems(fileItems)
			
			m.message = msg
			m.messageType = "success"
		}
	} else {
		m.message = "No file selected to link"
		m.messageType = "warning"
	}
	
	// Return a command that forces a complete screen refresh
	return m, func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m model) handleLinkAll() (tea.Model, tea.Cmd) {
	// Apply all configurations (create symlinks)
	messages, err := applyAllConfigs(m.config)
	if err != nil {
		m.message = fmt.Sprintf("Error linking configs: %v", err)
		m.messageType = "error"
	} else {
		// Update file statuses
		updateFileStatuses(m.config)
		
		// Update the list items with new statuses
		fileItems := make([]list.Item, len(m.config.Files))
		for i, file := range m.config.Files {
			fileItems[i] = fileItem{file: file}
		}
		
		// Use SetItems to update the list
		m.fileList.SetItems(fileItems)
		
		// Show summary of what was done
		if len(messages) > 0 {
			// Show just the count and first few actions
			summary := fmt.Sprintf("✅ Successfully processed %d files", len(m.config.Files))
			if len(messages) <= 3 {
				summary += ": " + messages[0]
			} else {
				summary += fmt.Sprintf(" (%s and %d more)", messages[0], len(messages)-1)
			}
			m.message = summary
		} else {
			m.message = fmt.Sprintf("✅ Successfully linked %d configuration files", len(m.config.Files))
		}
		m.messageType = "success"
	}
	
	// Return a command that forces a complete screen refresh
	return m, func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m model) handleEdit() (tea.Model, tea.Cmd) {
	if selected := m.fileList.SelectedItem(); selected != nil {
		selectedFileItem := selected.(fileItem)
		
		// Check if we need to handle directory file selection first
		sourcePath := filepath.Join(m.config.DotfilesDir, selectedFileItem.file.Source)
		
		// Check if the source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			m.message = fmt.Sprintf("Source file/directory does not exist: %s", sourcePath)
			m.messageType = "error"
			return m, nil
		}
		
		// Check if it's a directory
		if info, err := os.Stat(sourcePath); err == nil && info.IsDir() {
			// It's a directory - handle file selection first
			selectedFile, err := handleDirectorySelection(sourcePath)
			if err != nil {
				// Check if it was cancelled
				if strings.Contains(err.Error(), "cancelled") {
					m.message = "Edit operation cancelled"
					m.messageType = "warning"
				} else {
					m.message = fmt.Sprintf("File selection failed: %v", err)
					m.messageType = "error"
				}
				// Return with commands to hide cursor and refresh screen
				return m, tea.Batch(
					tea.HideCursor,
					func() tea.Msg {
						return tea.WindowSizeMsg{Width: m.width, Height: m.height}
					},
				)
			}
			
			// Now open the selected file from the directory
			fullPath := filepath.Join(sourcePath, selectedFile)
			return m, tea.ExecProcess(createSingleFileEditorCommand(m.config.Editor, fullPath), func(err error) tea.Msg {
				if err != nil {
					return editorFinishedMsg{err: err, fileName: selectedFile}
				}
				return editorFinishedMsg{err: nil, fileName: selectedFile}
			})
		} else {
			// It's a single file - open it directly
			return m, tea.ExecProcess(createSingleFileEditorCommand(m.config.Editor, sourcePath), func(err error) tea.Msg {
				if err != nil {
					return editorFinishedMsg{err: err, fileName: selectedFileItem.file.Name}
				}
				return editorFinishedMsg{err: nil, fileName: selectedFileItem.file.Name}
			})
		}
	} else {
		m.message = "No file selected to edit"
		m.messageType = "warning"
		return m, nil
	}
}

// Handle directory file selection (this runs before the TUI suspends)
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
		return "", fmt.Errorf("failed to scan directory: %v", err)
	}
	
	if len(editableFiles) == 0 {
		return "", fmt.Errorf("no editable files found in directory")
	}
	
	// Use the existing file selection logic
	return selectFileToEdit(editableFiles)
}

// Message type for when editor finishes
type editorFinishedMsg struct {
	err      error
	fileName string
}

// Create command for editing a single file
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

func (m model) handleBackup() (tea.Model, tea.Cmd) {
	// Create backup
	backupDir := createBackup(m.config)
	m.message = fmt.Sprintf("Backed up files to %s", backupDir)
	m.messageType = "success"
	
	return m, nil
}

func createBackup(config *Config) string {
	backupDir := fmt.Sprintf("%s/backups/%s", config.ConfigDir, time.Now().Format("2006-01-02_15-04-05"))
	createBackupInDir(config, backupDir)
	return backupDir
}

// Helper function to create a file list
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
