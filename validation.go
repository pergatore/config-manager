package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() []ValidationError {
	var errors []ValidationError
	
	// Validate basic config structure
	errors = append(errors, c.validateBasicConfig()...)
	
	// Validate files
	errors = append(errors, c.validateFiles()...)
	
	// Validate templates
	errors = append(errors, c.validateTemplates()...)
	
	// Validate editor
	errors = append(errors, c.validateEditor()...)
	
	return errors
}

func (c *Config) validateBasicConfig() []ValidationError {
	var errors []ValidationError
	
	// Check required directories
	if c.ConfigDir == "" {
		errors = append(errors, *NewValidationError("config_dir", "", "config directory not set", ""))
	} else if !filepath.IsAbs(c.ConfigDir) {
		errors = append(errors, *NewValidationError("config_dir", c.ConfigDir, "must be absolute path", ""))
	}
	
	if c.DotfilesDir == "" {
		errors = append(errors, *NewValidationError("dotfiles_dir", "", "dotfiles directory not set", ""))
	} else if !filepath.IsAbs(c.DotfilesDir) {
		errors = append(errors, *NewValidationError("dotfiles_dir", c.DotfilesDir, "must be absolute path", ""))
	}
	
	// Validate categories
	if len(c.Categories) == 0 {
		errors = append(errors, *NewValidationError("categories", "", "no categories defined", ""))
	}
	
	// Check for duplicate categories
	seen := make(map[string]bool)
	for _, cat := range c.Categories {
		if seen[cat] {
			errors = append(errors, *NewValidationError("categories", cat, "duplicate category", ""))
		}
		seen[cat] = true
	}
	
	return errors
}

func (c *Config) validateFiles() []ValidationError {
	var errors []ValidationError
	
	// Track targets to detect duplicates
	targetsSeen := make(map[string]string)
	
	for i, file := range c.Files {
		fileContext := fmt.Sprintf("files[%d]", i)
		
		// Validate required fields
		if file.Name == "" {
			errors = append(errors, *NewValidationError("name", "", "file name cannot be empty", fileContext))
		}
		
		if file.Source == "" {
			errors = append(errors, *NewValidationError("source", "", "source path cannot be empty", fileContext))
		}
		
		if file.Target == "" {
			errors = append(errors, *NewValidationError("target", "", "target path cannot be empty", fileContext))
		} else {
			// Check for duplicate targets
			if existingFile, exists := targetsSeen[file.Target]; exists {
				errors = append(errors, *NewValidationError("target", file.Target, 
					fmt.Sprintf("duplicate target (also used by %s)", existingFile), fileContext))
			}
			targetsSeen[file.Target] = file.Name
			
			// Validate target path is absolute
			if !filepath.IsAbs(file.Target) {
				errors = append(errors, *NewValidationError("target", file.Target, "must be absolute path", fileContext))
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
				errors = append(errors, *NewValidationError("category", file.Category, "category not defined in config", fileContext))
			}
		}
		
		// Validate source path doesn't escape dotfiles directory
		if file.Source != "" {
			sourcePath := filepath.Join(c.DotfilesDir, file.Source)
			if !strings.HasPrefix(sourcePath, c.DotfilesDir) {
				errors = append(errors, *NewValidationError("source", file.Source, "source path escapes dotfiles directory", fileContext))
			}
		}
	}
	
	return errors
}

func (c *Config) validateTemplates() []ValidationError {
	var errors []ValidationError
	
	for i, file := range c.Files {
		if !file.Template {
			continue
		}
		
		fileContext := fmt.Sprintf("files[%d]", i)
		
		// Find template file
		templatePath := c.findTemplateFile(file.Name, file.Source, file.Category)
		if templatePath == "" {
			errors = append(errors, *NewValidationError("template", file.Name, "template file not found", fileContext))
			continue
		}
		
		// Validate template syntax using the function from templates.go
		if err := validateTemplateFileContent(templatePath); err != nil {
			errors = append(errors, *NewValidationError("template", templatePath, 
				fmt.Sprintf("template syntax error: %v", err), fileContext))
		}
		
		// Validate template variables
		if err := c.validateTemplateVariables(file, templatePath); err != nil {
			errors = append(errors, *NewValidationError("template_variables", file.Name, 
				fmt.Sprintf("template variable error: %v", err), fileContext))
		}
	}
	
	return errors
}

func (c *Config) validateEditor() []ValidationError {
	var errors []ValidationError
	
	if c.Editor == "" {
		errors = append(errors, *NewValidationError("editor", "", "editor not configured", ""))
		return errors
	}
	
	// Check if editor is available in PATH
	if _, err := exec.LookPath(c.Editor); err != nil {
		errors = append(errors, *NewValidationError("editor", c.Editor, 
			fmt.Sprintf("editor not found in PATH: %v", err), ""))
	}
	
	return errors
}

// Remove the duplicate validateTemplateFileContent function since it's in templates.go

func (c *Config) validateTemplateVariables(file ConfigFile, templatePath string) error {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	
	// Parse template to extract variables (simplified check)
	templateStr := string(content)
	
	// Check for common undefined variables
	requiredVars := []string{}
	if strings.Contains(templateStr, "{{ .user }}") {
		requiredVars = append(requiredVars, "user")
	}
	if strings.Contains(templateStr, "{{ .hostname }}") {
		requiredVars = append(requiredVars, "hostname")
	}
	if strings.Contains(templateStr, "{{ .editor }}") {
		requiredVars = append(requiredVars, "editor")
	}
	if strings.Contains(templateStr, "{{ .shell }}") {
		requiredVars = append(requiredVars, "shell")
	}
	
	// Check if variables are available (built-ins are always available)
	for _, varName := range requiredVars {
		if varName == "user" || varName == "hostname" || varName == "editor" || varName == "shell" {
			continue // Built-in variables
		}
		
		// Check if variable is defined in global or file-specific variables
		if _, exists := c.Variables[varName]; !exists {
			if _, exists := file.Variables[varName]; !exists {
				return fmt.Errorf("undefined variable: %s", varName)
			}
		}
	}
	
	return nil
}

func (c *Config) findTemplateFile(fileName, source, category string) string {
	templatesDir := filepath.Join(c.ConfigDir, "templates")
	
	// Try different naming patterns
	baseName := strings.TrimPrefix(fileName, ".")
	
	for _, ext := range c.TemplateExts {
		candidates := []string{
			filepath.Join(templatesDir, baseName+ext),
			filepath.Join(templatesDir, fileName+ext),
			filepath.Join(templatesDir, category+"_"+baseName+ext),
		}
		
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	
	return ""
}

// ValidateBeforeSave performs validation before saving config
func (c *Config) ValidateBeforeSave() error {
	errors := c.Validate()
	if len(errors) > 0 {
		var messages []string
		for _, err := range errors {
			messages = append(messages, err.Error())
		}
		return &MultiError{
			Errors: []error{fmt.Errorf(strings.Join(messages, "; "))},
			Op:     "config validation",
		}
	}
	return nil
}
