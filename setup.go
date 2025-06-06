package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Initial setup wizard using Gum with fallback
func runSetupWizard() (*Config, error) {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "config-manager")
	
	fmt.Println("🎉 Welcome to Config Manager!")
	fmt.Println("Let's set up your configuration management...")
	fmt.Println()
	
	// Check if we're in an interactive terminal and gum works
	useGum := false
	if _, err := exec.LookPath("gum"); err == nil {
		// Test if gum works in this environment
		testCmd := exec.Command("gum", "choose", "--timeout=1s", "test")
		testCmd.Stdin = os.Stdin
		if err := testCmd.Run(); err == nil || strings.Contains(err.Error(), "timeout") {
			useGum = true
		}
	}
	
	if !useGum {
		fmt.Println("Using text-based setup (Gum not available or not working in this environment)")
		return runTextSetup(configDir)
	}
	
	// Step 1: Choose preferred tools with Gum
	fmt.Println("🛠️  Step 1: Tool Preferences")
	
	// Editor selection
	editor := selectEditor()
	fmt.Printf("✅ Editor: %s\n\n", editor)
	
	// Shell selection
	shell := selectShell()
	fmt.Printf("✅ Shell: %s\n\n", shell)
	
	// Step 2: Discover and choose configs to manage
	fmt.Println("📁 Step 2: Configuration Discovery")
	fmt.Println("Scanning for configuration files and directories...")
	
	selectedConfigs := selectConfigs()
	
	return createConfigFromSetup(configDir, editor, shell, selectedConfigs)
}

func selectEditor() string {
	fmt.Println("Choose your preferred editor (use arrow keys and enter):")
	editorCmd := exec.Command("gum", "choose", "vim", "nvim", "emacs", "nano", "code", "other")
	editorCmd.Stdin = os.Stdin
	editorOutput, err := editorCmd.Output()
	if err != nil {
		return "vim"
	}
	
	editor := strings.TrimSpace(string(editorOutput))
	if editor == "" {
		editor = "vim"
	}
	
	if editor == "other" {
		fmt.Print("Enter your editor command: ")
		customEditorCmd := exec.Command("gum", "input", "--placeholder", "editor command")
		customEditorCmd.Stdin = os.Stdin
		customOutput, err := customEditorCmd.Output()
		if err == nil && strings.TrimSpace(string(customOutput)) != "" {
			editor = strings.TrimSpace(string(customOutput))
		} else {
			editor = "vim"
		}
	}
	
	return editor
}

func selectShell() string {
	fmt.Println("Choose your preferred shell (use arrow keys and enter):")
	shellCmd := exec.Command("gum", "choose", "bash", "zsh", "fish", "other")
	shellCmd.Stdin = os.Stdin
	shellOutput, err := shellCmd.Output()
	if err != nil {
		return "bash"
	}
	
	shell := strings.TrimSpace(string(shellOutput))
	if shell == "" {
		shell = "bash"
	}
	
	if shell == "other" {
		fmt.Print("Enter your shell name: ")
		customShellCmd := exec.Command("gum", "input", "--placeholder", "shell name")
		customShellCmd.Stdin = os.Stdin
		customOutput, err := customShellCmd.Output()
		if err == nil && strings.TrimSpace(string(customOutput)) != "" {
			shell = strings.TrimSpace(string(customOutput))
		} else {
			shell = "bash"
		}
	}
	
	return shell
}

func selectConfigs() []string {
	configChoices := discoverAllConfigs()
	fmt.Printf("Found %d potential configurations\n", len(configChoices))
	
	var selectedConfigs []string
	
	if len(configChoices) == 0 {
		fmt.Println("No configuration files found. You can add them later using 'a' in the application.")
		return []string{}
	}
	
	fmt.Println("\nSelect configurations to manage (use space to select, enter to confirm):")
	
	chooseCmd := exec.Command("gum", "choose", "--no-limit")
	chooseCmd.Args = append(chooseCmd.Args, configChoices...)
	chooseCmd.Stdin = os.Stdin
	
	chosenOutput, err := chooseCmd.Output()
	if err != nil {
		fmt.Printf("❌ Config selection cancelled or failed: %v\n", err)
		fmt.Println("Continuing with empty configuration. You can add configs later with 'a'.")
		return []string{}
	}
	
	selectedConfigs = strings.Split(strings.TrimSpace(string(chosenOutput)), "\n")
	var filtered []string
	for _, s := range selectedConfigs {
		if strings.TrimSpace(s) != "" {
			filtered = append(filtered, strings.TrimSpace(s))
		}
	}
	selectedConfigs = filtered
	fmt.Printf("✅ Selected %d configurations\n", len(selectedConfigs))
	
	return selectedConfigs
}

// Text-based setup fallback
func runTextSetup(configDir string) (*Config, error) {
	fmt.Println("\n📝 Text-based Setup")
	
	// Editor selection
	fmt.Println("\n🛠️  Step 1: Tool Preferences")
	editor := selectEditorText()
	fmt.Printf("✅ Editor: %s\n", editor)
	
	// Shell selection
	shell := selectShellText()
	fmt.Printf("✅ Shell: %s\n", shell)
	
	// Config discovery
	selectedConfigs := selectConfigsText()
	
	return createConfigFromSetup(configDir, editor, shell, selectedConfigs)
}

func selectEditorText() string {
	fmt.Println("Choose your preferred editor:")
	fmt.Println("1. vim")
	fmt.Println("2. nvim")
	fmt.Println("3. emacs") 
	fmt.Println("4. nano")
	fmt.Println("5. code")
	fmt.Println("6. other")
	fmt.Print("Enter choice (1-6): ")
	
	var choice string
	fmt.Scanln(&choice)
	
	switch choice {
	case "1":
		return "vim"
	case "2":
		return "nvim"
	case "3":
		return "emacs"
	case "4":
		return "nano"
	case "5":
		return "code"
	case "6":
		fmt.Print("Enter your editor command: ")
		var editor string
		fmt.Scanln(&editor)
		if editor == "" {
			return "vim"
		}
		return editor
	default:
		fmt.Println("Invalid choice, using vim")
		return "vim"
	}
}

func selectShellText() string {
	fmt.Println("\nChoose your preferred shell:")
	fmt.Println("1. bash")
	fmt.Println("2. zsh")
	fmt.Println("3. fish")
	fmt.Println("4. other")
	fmt.Print("Enter choice (1-4): ")
	
	var choice string
	fmt.Scanln(&choice)
	
	switch choice {
	case "1":
		return "bash"
	case "2":
		return "zsh"
	case "3":
		return "fish"
	case "4":
		fmt.Print("Enter your shell name: ")
		var shell string
		fmt.Scanln(&shell)
		if shell == "" {
			return "bash"
		}
		return shell
	default:
		fmt.Println("Invalid choice, using bash")
		return "bash"
	}
}

func selectConfigsText() []string {
	fmt.Println("\n📁 Step 2: Configuration Discovery")
	fmt.Println("Scanning for configuration files and directories...")
	
	configChoices := discoverAllConfigs()
	fmt.Printf("Found %d potential configurations\n", len(configChoices))
	
	if len(configChoices) == 0 {
		fmt.Println("No configuration files found. You can add them later using 'a' in the application.")
		return []string{}
	}
	
	fmt.Println("\nAvailable configurations:")
	for i, choice := range configChoices {
		fmt.Printf("%d. %s\n", i+1, choice)
	}
	
	fmt.Println("\nEnter the numbers of configs to manage (comma-separated, e.g., 1,3,5) or press enter to skip:")
	fmt.Print("Selection: ")
	
	var input string
	fmt.Scanln(&input)
	
	var selectedConfigs []string
	if input != "" {
		choices := strings.Split(input, ",")
		for _, choiceStr := range choices {
			choiceStr = strings.TrimSpace(choiceStr)
			if idx, err := strconv.Atoi(choiceStr); err == nil {
				if idx >= 1 && idx <= len(configChoices) {
					selectedConfigs = append(selectedConfigs, configChoices[idx-1])
				}
			}
		}
	}
	
	fmt.Printf("✅ Selected %d configurations\n", len(selectedConfigs))
	return selectedConfigs
}

// Common config creation logic
func createConfigFromSetup(configDir, editor, shell string, selectedConfigs []string) (*Config, error) {
	config := &Config{
		ConfigDir:    configDir,
		DotfilesDir:  filepath.Join(configDir, "dotfiles"),
		Variables:    make(map[string]string),
		Categories:   []string{"shell", "editor", "git", "terminal", "misc", "custom"},
		TemplateExts: []string{".tmpl", ".template", ".tpl"},
		Editor:       editor,
		Shell:        shell,
		Files:        []ConfigFile{},
	}
	
	// Convert selected configs to ConfigFile structs
	successCount := 0
	for _, selected := range selectedConfigs {
		if selected == "" {
			continue
		}
		configFile, err := createConfigFileFromSelection(selected, config)
		if err == nil {
			config.Files = append(config.Files, configFile)
			successCount++
		} else {
			fmt.Printf("⚠️  Failed to add %s: %v\n", selected, err)
		}
	}
	
	// Create directories and save config
	os.MkdirAll(configDir, 0755)
	saveConfig(config)
	createDefaultTemplates(config)
	
	fmt.Printf("\n🎉 Setup complete! Managing %d configurations.\n", successCount)
	if successCount == 0 {
		fmt.Println("You can add configurations later using 'a' in the application.")
	} else {
		fmt.Println("Use 'l' to link your configurations when ready.")
	}
	fmt.Println("Starting Config Manager...")
	fmt.Println()
	
	return config, nil
}

// Create ConfigFile from user selection
func createConfigFileFromSelection(selection string, config *Config) (ConfigFile, error) {
	homeDir, _ := os.UserHomeDir()
	
	// Parse selection format: "path (type)"
	parts := strings.Split(selection, " (")
	if len(parts) != 2 {
		return ConfigFile{}, fmt.Errorf("invalid selection format")
	}
	
	path := parts[0]
	fileType := strings.TrimSuffix(parts[1], ")")
	
	targetPath := filepath.Join(homeDir, path)
	fileName := filepath.Base(path)
	
	// Auto-categorize
	category := categorizeDotfile(fileName, config.Categories)
	
	// Check if it might be a template (only for files, not directories)
	isTemplate := false
	if fileType == "file" {
		if data, err := os.ReadFile(targetPath); err == nil {
			content := strings.ToLower(string(data))
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
	if strings.Contains(path, ".config/") {
		// For .config subdirectories, preserve the structure
		relativePath := strings.TrimPrefix(path, ".config/")
		sourcePath = filepath.Join("config", relativePath)
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
