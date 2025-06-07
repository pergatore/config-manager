package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateContext holds all variables available to templates
type TemplateContext struct {
	// Built-in system variables
	User     string `json:"user"`
	Hostname string `json:"hostname"`
	Editor   string `json:"editor"`
	Shell    string `json:"shell"`
	
	// Custom variables (merged from global and file-specific)
	Variables map[string]string `json:"variables"`
}

// TemplateResult represents the result of template processing
type TemplateResult struct {
	OutputPath string
	Success    bool
	Error      error
	Variables  map[string]string
}

// Enhanced template functions
func getTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"env": func(key string) string {
			return os.Getenv(key)
		},
		"fileExists": func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"replace": strings.ReplaceAll,
		"join": strings.Join,
		"split": strings.Split,
	}
}

// Create default templates with better error handling
func createDefaultTemplates(config *Config) error {
	templatesDir := filepath.Join(config.ConfigDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return NewConfigError("create templates directory", templatesDir, err)
	}
	
	templates := getDefaultTemplateContent()
	
	var multiErr MultiError
	multiErr.Op = "create default templates"
	
	// Create template files
	for name, content := range templates {
		for _, ext := range config.TemplateExts {
			templatePath := filepath.Join(templatesDir, name+ext)
			if _, err := os.Stat(templatePath); os.IsNotExist(err) {
				if err := os.WriteFile(templatePath, []byte(content), 0644); err != nil {
					multiErr.Add(NewConfigError("write template", templatePath, err))
				}
				break // Only create one template per type
			}
		}
	}
	
	if multiErr.HasErrors() {
		return &multiErr
	}
	
	return nil
}

// getDefaultTemplateContent returns the default template content
func getDefaultTemplateContent() map[string]string {
	return map[string]string{
		"gitconfig": `[user]
    name = {{ .User }}
    email = {{ .User }}@{{ if contains .Hostname "work" }}company.com{{ else }}gmail.com{{ end }}
[core]
    editor = {{ .Editor }}
    autocrlf = input

# Work-specific settings
{{ if contains .Hostname "work" }}
[url "git@github.com:company/"]
    insteadOf = https://github.com/company/
[commit]
    gpgsign = true
{{ end }}

# Personal settings
{{ if not (contains .Hostname "work") }}
[github]
    user = {{ .User }}
[pull]
    rebase = true
{{ end }}

[push]
    default = simple
[alias]
    st = status
    co = checkout
    br = branch
    # Configuration for {{ .User }} on {{ .Hostname }}`,
		
		"zshrc": `# {{ .User }}'s {{ .Shell }} configuration on {{ .Hostname }}
export EDITOR="{{ .Editor }}"
export PATH="$HOME/bin:$PATH"

# Work-specific settings
{{ if contains .Hostname "work" }}
export WORK_PROJECT_DIR="$HOME/work"
alias vpn="sudo openvpn /etc/openvpn/work.conf"
{{ end }}

# Personal settings
{{ if contains .Hostname "home" }}
export PERSONAL_PROJECT_DIR="$HOME/projects"
alias backup="rsync -av $HOME/Documents/ /backup/"
{{ else if contains .Hostname "macbook" }}
export PERSONAL_PROJECT_DIR="$HOME/Code"
alias brew-update="brew update && brew upgrade"
{{ end }}

# Common aliases for all machines
alias ll="ls -la"
alias la="ls -la"
alias l="ls -l"
alias ..="cd .."
alias ...="cd ../.."
alias grep="grep --color=auto"

# History
HISTSIZE=10000
SAVEHIST=10000
setopt HIST_IGNORE_DUPS
setopt HIST_IGNORE_SPACE

# Prompt
autoload -U colors && colors
PS1="%{$fg[blue]%}%n@%m%{$reset_color%}:%{$fg[green]%}%~%{$reset_color%}$ "`,

		"bashrc": `# {{ .User }}'s {{ .Shell }} configuration on {{ .Hostname }}
export EDITOR="{{ .Editor }}"
export PATH="$HOME/bin:$PATH"

# Work-specific settings
{{ if contains .Hostname "work" }}
export WORK_PROJECT_DIR="$HOME/work"
alias vpn="sudo openvpn /etc/openvpn/work.conf"
{{ end }}

# Personal settings
{{ if contains .Hostname "home" }}
export PERSONAL_PROJECT_DIR="$HOME/projects"
alias backup="rsync -av $HOME/Documents/ /backup/"
{{ end }}

# Common aliases
alias ll="ls -la"
alias la="ls -la"
alias l="ls -l"
alias ..="cd .."
alias ...="cd ../.."
alias grep="grep --color=auto"

# History
HISTSIZE=10000
HISTFILESIZE=20000
export HISTCONTROL=ignoredups:erasedups

# Prompt
PS1="\[\033[01;34m\]\u@\h\[\033[00m\]:\[\033[01;32m\]\w\[\033[00m\]$ "`,

		"vimrc": `" {{ .User }}'s Vim configuration on {{ .Hostname }}
set nocompatible
set number
set relativenumber
set tabstop=4
set shiftwidth=4
set expandtab
set autoindent
set smartindent
set hlsearch
set incsearch
set ignorecase
set smartcase
set ruler
set showcmd
set wildmenu
set laststatus=2

" Color scheme
syntax enable
set background=dark

" Key mappings
let mapleader = ","
nnoremap <leader>w :w<CR>
nnoremap <leader>q :q<CR>

" Work-specific settings
{{ if contains .Hostname "work" }}
" Company-specific vim settings
set textwidth=100
{{ end }}`,

		"tmux": `# {{ .User }}'s tmux configuration on {{ .Hostname }}
# Change prefix key
set -g prefix C-a
unbind C-b
bind C-a send-prefix

# Split panes
bind | split-window -h
bind - split-window -v

# Move between panes
bind h select-pane -L
bind j select-pane -D
bind k select-pane -U
bind l select-pane -R

# Enable mouse mode
set -g mouse on

# Status bar
set -g status-bg black
set -g status-fg white
set -g status-left '[#S] '
set -g status-right '#H %Y-%m-%d %H:%M'

# Work-specific settings
{{ if contains .Hostname "work" }}
set -g status-right-length 50
set -g status-right '#(uptime | cut -d "," -f 1) #H %Y-%m-%d %H:%M'
{{ end }}`,
	}
}

// Enhanced createFromTemplate with better error handling and validation
func createFromTemplate(config *Config, file *ConfigFile, outputPath string) error {
	// First, validate that this should be a template
	if !file.Template {
		return NewConfigError("create from template", file.Name, 
			fmt.Errorf("file is not marked as template"))
	}
	
	// Find template file
	templatePath := findTemplateFile(config, file.Name, file.Source, file.Category)
	if templatePath == "" {
		// If no template found, try to create a basic config file
		return createBasicConfigFile(file, outputPath)
	}
	
	// Validate template before processing
	if err := validateTemplateFileContent(templatePath); err != nil {
		return NewConfigError("validate template", templatePath, err)
	}
	
	// Create template context
	context, err := createTemplateContext(config, file)
	if err != nil {
		return NewConfigError("create template context", file.Name, err)
	}
	
	// Process template
	result, err := processTemplate(templatePath, context, outputPath)
	if err != nil {
		return err
	}
	
	if !result.Success {
		return NewConfigError("process template", templatePath, result.Error)
	}
	
	return nil
}

// findTemplateFile locates the template file for a given config
func findTemplateFile(config *Config, fileName, source, category string) string {
	templatesDir := filepath.Join(config.ConfigDir, "templates")
	
	// Try different naming patterns
	baseName := strings.TrimPrefix(fileName, ".")
	
	for _, ext := range config.TemplateExts {
		candidates := []string{
			filepath.Join(templatesDir, baseName+ext),
			filepath.Join(templatesDir, fileName+ext),
			filepath.Join(templatesDir, category+"_"+baseName+ext),
			filepath.Join(templatesDir, category, baseName+ext),
		}
		
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	
	return ""
}

// createTemplateContext builds the context for template execution
func createTemplateContext(config *Config, file *ConfigFile) (*TemplateContext, error) {
	context := &TemplateContext{
		Variables: make(map[string]string),
	}
	
	// Set built-in system variables
	if user := os.Getenv("USER"); user != "" {
		context.User = user
	} else {
		context.User = "unknown"
	}
	
	if hostname, err := os.Hostname(); err == nil {
		context.Hostname = hostname
	} else {
		context.Hostname = "localhost"
	}
	
	context.Editor = config.Editor
	context.Shell = config.Shell
	
	// Merge variables: global < file-specific
	for k, v := range config.Variables {
		context.Variables[k] = v
	}
	
	for k, v := range file.Variables {
		context.Variables[k] = v
	}
	
	return context, nil
}

// processTemplate executes the template with the given context
func processTemplate(templatePath string, context *TemplateContext, outputPath string) (*TemplateResult, error) {
	result := &TemplateResult{
		OutputPath: outputPath,
		Variables:  context.Variables,
	}
	
	// Read template content
	content, err := os.ReadFile(templatePath)
	if err != nil {
		result.Error = NewConfigError("read template", templatePath, err)
		return result, result.Error
	}
	
	// Create template with functions
	tmpl, err := template.New(filepath.Base(templatePath)).
		Funcs(getTemplateFunctions()).
		Parse(string(content))
	if err != nil {
		result.Error = NewConfigError("parse template", templatePath, err)
		return result, result.Error
	}
	
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		result.Error = NewConfigError("create output directory", filepath.Dir(outputPath), err)
		return result, result.Error
	}
	
	// Create output file
	output, err := os.Create(outputPath)
	if err != nil {
		result.Error = NewConfigError("create output file", outputPath, err)
		return result, result.Error
	}
	defer output.Close()
	
	// Execute template
	if err := tmpl.Execute(output, context); err != nil {
		result.Error = NewConfigError("execute template", templatePath, err)
		return result, result.Error
	}
	
	result.Success = true
	return result, nil
}

// createBasicConfigFile creates a basic config file when no template is found
func createBasicConfigFile(file *ConfigFile, outputPath string) error {
	basicContent := fmt.Sprintf("# %s configuration\n# Generated by config-manager\n# No template found, please customize as needed\n", file.Name)
	
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return NewConfigError("create output directory", filepath.Dir(outputPath), err)
	}
	
	if err := os.WriteFile(outputPath, []byte(basicContent), 0644); err != nil {
		return NewConfigError("write basic config", outputPath, err)
	}
	
	return nil
}

// validateTemplateFileContent checks template syntax and common issues
func validateTemplateFileContent(templatePath string) error {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return NewConfigError("read template", templatePath, err)
	}
	
	// Parse template to check syntax
	tmpl, err := template.New(filepath.Base(templatePath)).
		Funcs(getTemplateFunctions()).
		Parse(string(content))
	if err != nil {
		return NewConfigError("parse template", templatePath, err)
	}
	
	// Try to execute with dummy data to catch runtime errors
	dummyContext := &TemplateContext{
		User:     "testuser",
		Hostname: "testhost",
		Editor:   "vim",
		Shell:    "bash",
		Variables: map[string]string{
			"email_domain": "example.com",
			"environment": "test",
		},
	}
	
	// Execute to a dummy writer to validate template logic
	var buf strings.Builder
	if err := tmpl.Execute(&buf, dummyContext); err != nil {
		return NewConfigError("execute template", templatePath, err)
	}
	
	return nil
}
